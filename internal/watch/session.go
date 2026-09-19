package watch

import (
	"context"
	"fmt"
	"time"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

// Engine is the minimal lifecycle capability watch needs.
type Engine interface {
	Deploy(ctx context.Context, ch *content.Chapter) error
	Destroy(ctx context.Context, ch *content.Chapter) error
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
	Status  func(string)
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

// Current returns the current exercise, or nil when the library is complete.
func (s *Session) Current() *content.Exercise { return s.current }

// Start locates the current exercise, ensures its topology is deployed, and runs setup.
func (s *Session) Start(ctx context.Context) error {
	ex, err := s.locateCurrent()
	if err != nil {
		return err
	}
	return s.activate(ctx, ex)
}

// Run jumps to the given exercise and loads it. id may be a fully qualified ID
// or an exercise-local ID that is unique across the library.
func (s *Session) Run(ctx context.Context, id string) error {
	ex, err := s.opts.Library.Resolve(id)
	if err != nil {
		return err
	}
	return s.activate(ctx, ex)
}

func (s *Session) status(format string, args ...any) {
	if s.opts.Status != nil {
		s.opts.Status(fmt.Sprintf(format, args...))
	}
}

// ResetCurrent re-arms the current exercise via fast or hard reset. It does not
// change progress state.
func (s *Session) ResetCurrent(ctx context.Context, hard bool) error {
	if s.current == nil {
		return nil
	}
	mode := "fast"
	if hard {
		mode = "hard"
	}
	start := time.Now()
	s.status("→ resetting exercise %s (%s) ...", s.current.FullID(), mode)
	var err error
	if hard {
		err = s.opts.Reset.Hard(ctx, s.current)
	} else {
		err = s.opts.Reset.Fast(ctx, s.current)
	}
	if err != nil {
		return err
	}
	s.status("✓ reset complete (%s)", time.Since(start).Round(time.Millisecond))
	return nil
}

// Cleanup destroys every chapter in the library (idempotent).
func (s *Session) Cleanup(ctx context.Context) error {
	for _, ch := range s.opts.Library.Chapters {
		if err := s.destroyChapter(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) deployChapter(ctx context.Context, ch *content.Chapter) error {
	start := time.Now()
	s.status("→ deploying chapter %s ...", ch.ID)
	if err := s.opts.Engine.Deploy(ctx, ch); err != nil {
		return err
	}
	s.status("✓ chapter %s ready (%s)", ch.ID, time.Since(start).Round(time.Millisecond))
	return nil
}

func (s *Session) destroyChapter(ctx context.Context, ch *content.Chapter) error {
	start := time.Now()
	s.status("→ destroying chapter %s ...", ch.ID)
	if err := s.opts.Engine.Destroy(ctx, ch); err != nil {
		return err
	}
	s.status("✓ chapter %s destroyed (%s)", ch.ID, time.Since(start).Round(time.Millisecond))
	return nil
}

// VerifyNow runs one verification; on pass it advances to the next exercise.
func (s *Session) VerifyNow(ctx context.Context) error {
	if s.current == nil {
		return fmt.Errorf("no active exercise")
	}
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
		return s.opts.State.Save(s.opts.Library.Root)
	}

	es := s.opts.State.Get(ex.FullID())
	s.opts.State.Upsert(ex.FullID(), state.StatusPassed, es.Attempts+1)

	next := s.next(ex)
	if next == nil {
		s.opts.State.SetCurrent("")
		s.current = nil
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
func (s *Session) activate(ctx context.Context, ex *content.Exercise) error {
	if s.chapter == nil || s.chapter.ID != ex.ChapterRef.ID {
		if s.chapter != nil {
			if err := s.destroyChapter(ctx, s.chapter); err != nil {
				return err
			}
			s.chapter = nil
		}
		if err := s.deployChapter(ctx, ex.ChapterRef); err != nil {
			return err
		}
	}
	s.chapter = ex.ChapterRef
	return s.setActiveAndReset(ctx, ex)
}

// setActiveAndReset records ex as the current exercise, persists state (without
// downgrading an already-passed exercise), and runs the fast reset.
func (s *Session) setActiveAndReset(ctx context.Context, ex *content.Exercise) error {
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

// advance switches to the next exercise after a pass and persists state.
func (s *Session) advance(ctx context.Context, from, next *content.Exercise) error {
	if from.ChapterRef.ID == next.ChapterRef.ID {
		return s.setActiveAndReset(ctx, next)
	}
	// Cross chapter: destroy the current chapter (if any), then deploy the next.
	if s.chapter != nil {
		if err := s.destroyChapter(ctx, s.chapter); err != nil {
			return err
		}
		s.chapter = nil
	}
	if err := s.deployChapter(ctx, next.ChapterRef); err != nil {
		return err
	}
	s.chapter = next.ChapterRef
	return s.setActiveAndReset(ctx, next)
}
