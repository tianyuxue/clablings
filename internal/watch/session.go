package watch

import (
	"context"
	"fmt"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/state"
)

// Engine is the minimal lifecycle capability watch needs.
type Engine interface {
	Deploy(ctx context.Context, ch *content.Chapter) error
	Destroy(ctx context.Context, ch *content.Chapter) error
	Topology(ctx context.Context, ch *content.Chapter) (*engine.Topology, error)
}

// Resetter is the two-tier reset capability.
type Resetter interface {
	Fast(ctx context.Context, ex *content.Exercise) error
	Hard(ctx context.Context, ex *content.Exercise) error
}

// Verifier runs one full verification (including code-implement build) and
// returns assertion results.
type Verifier func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error)

// Reporter renders assertion results.
type Reporter func(results []assert.Result)

// Options configures a Session.
type Options struct {
	Library *content.Library
	State   *state.State
	Engine  Engine
	Reset   Resetter
	Verify  Verifier
	Report  Reporter
}

// Session orchestrates watch's lifecycle and advancement.
type Session struct {
	opts    Options
	current *content.Exercise
	chapter *content.Chapter
	busy    bool
}

// New builds a Session.
func New(opts Options) *Session { return &Session{opts: opts} }

// Current returns the current exercise.
func (s *Session) Current() *content.Exercise { return s.current }

// Start locates the current exercise, ensures its topology is deployed, and runs setup.
func (s *Session) Start(ctx context.Context) error {
	ex, err := s.locateCurrent()
	if err != nil {
		return err
	}
	return s.activate(ctx, ex, true)
}

// Run jumps to the given exercise and loads it. id may be a fully qualified ID
// or an exercise-local ID that is unique across the library.
func (s *Session) Run(ctx context.Context, id string) error {
	ex, err := s.resolveExercise(id)
	if err != nil {
		return err
	}
	return s.activate(ctx, ex, true)
}

func (s *Session) resolveExercise(id string) (*content.Exercise, error) {
	if ex := s.opts.Library.ByID[id]; ex != nil {
		return ex, nil
	}
	var found *content.Exercise
	for _, ex := range s.opts.Library.Order {
		if ex.ID == id {
			if found != nil {
				return nil, fmt.Errorf("exercise ID %q is not unique; use chapter-id/exercise-id", id)
			}
			found = ex
		}
	}
	if found == nil {
		return nil, fmt.Errorf("unknown exercise %s", id)
	}
	return found, nil
}

// VerifyNow runs one verification; on pass it advances to the next exercise.
func (s *Session) VerifyNow(ctx context.Context) error {
	if s.busy {
		return nil
	}
	s.busy = true
	defer func() { s.busy = false }()

	ex := s.current
	results, err := s.opts.Verify(ctx, ex)
	if err != nil {
		return err
	}
	if s.opts.Report != nil {
		s.opts.Report(results)
	}
	passed := len(results) > 0
	for _, r := range results {
		if !r.Passed {
			passed = false
			break
		}
	}
	if !passed {
		es := s.opts.State.Get(ex.FullID())
		s.opts.State.Upsert(ex.FullID(), state.StatusInProgress, es.Attempts+1)
		_ = s.opts.State.Save(s.opts.Library.Root)
		return nil
	}

	es := s.opts.State.Get(ex.FullID())
	s.opts.State.Upsert(ex.FullID(), state.StatusPassed, es.Attempts+1)

	next := s.next(ex)
	if next == nil {
		s.opts.State.SetCurrent("")
		return s.opts.State.Save(s.opts.Library.Root)
	}
	return s.advance(ctx, ex, next)
}

func (s *Session) locateCurrent() (*content.Exercise, error) {
	if s.opts.State.Current != "" {
		if ex := s.opts.Library.ByID[s.opts.State.Current]; ex != nil {
			return ex, nil
		}
	}
	if len(s.opts.Library.Order) == 0 {
		return nil, fmt.Errorf("library is empty")
	}
	return s.opts.Library.Order[0], nil
}

func (s *Session) next(ex *content.Exercise) *content.Exercise {
	for i, o := range s.opts.Library.Order {
		if o.FullID() == ex.FullID() {
			if i+1 < len(s.opts.Library.Order) {
				return s.opts.Library.Order[i+1]
			}
			return nil
		}
	}
	return nil
}

// activate ensures the exercise's chapter is deployed and runs setup (fast reset).
func (s *Session) activate(ctx context.Context, ex *content.Exercise, ensureDeploy bool) error {
	if ensureDeploy && (s.chapter == nil || s.chapter.ID != ex.ChapterRef.ID) {
		if s.chapter != nil {
			if err := s.opts.Engine.Destroy(ctx, s.chapter); err != nil {
				return err
			}
		}
		if err := s.opts.Engine.Deploy(ctx, ex.ChapterRef); err != nil {
			return err
		}
	}
	s.chapter = ex.ChapterRef
	s.current = ex
	s.opts.State.SetCurrent(ex.FullID())
	es := s.opts.State.Get(ex.FullID())
	if es.Status != state.StatusPassed {
		s.opts.State.Upsert(ex.FullID(), state.StatusInProgress, es.Attempts)
	}
	if err := s.opts.State.Save(s.opts.Library.Root); err != nil {
		return err
	}
	return s.opts.Reset.Fast(ctx, ex)
}

// advance handles switching to the next exercise after a pass and persists state.
func (s *Session) advance(ctx context.Context, from, next *content.Exercise) error {
	sameChapter := from.ChapterRef.ID == next.ChapterRef.ID
	if sameChapter {
		s.current = next
		s.opts.State.SetCurrent(next.FullID())
		es := s.opts.State.Get(next.FullID())
		if es.Status != state.StatusPassed {
			s.opts.State.Upsert(next.FullID(), state.StatusInProgress, es.Attempts)
		}
		if err := s.opts.State.Save(s.opts.Library.Root); err != nil {
			return err
		}
		return s.opts.Reset.Fast(ctx, next)
	}
	// Cross chapter: destroy the current chapter, deploy the next.
	if err := s.opts.Engine.Destroy(ctx, s.chapter); err != nil {
		return err
	}
	if err := s.opts.Engine.Deploy(ctx, next.ChapterRef); err != nil {
		return err
	}
	s.chapter = next.ChapterRef
	s.current = next
	s.opts.State.SetCurrent(next.FullID())
	es := s.opts.State.Get(next.FullID())
	if es.Status != state.StatusPassed {
		s.opts.State.Upsert(next.FullID(), state.StatusInProgress, es.Attempts)
	}
	if err := s.opts.State.Save(s.opts.Library.Root); err != nil {
		return err
	}
	return s.opts.Reset.Fast(ctx, next)
}
