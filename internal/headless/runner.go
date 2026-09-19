// Package headless runs a whole library without an interactive terminal.
package headless

import (
	"context"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/watch"
)

// Verify runs one exercise's full verification (including code-implement build).
type Verify func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error)

// OnResult is called after each exercise completes.
type OnResult func(exerciseID string, results []assert.Result)

// RunAll verifies every exercise in order: deploy/destroy per chapter, and a
// hard reset between exercises.
func RunAll(ctx context.Context, lib *content.Library, eng watch.Engine, res watch.Resetter, verify Verify, onResult OnResult) error {
	var current *content.Chapter
	defer func() {
		if current != nil {
			_ = eng.Destroy(context.Background(), current)
		}
	}()

	for _, ex := range lib.Order {
		if current == nil || current.ID != ex.ChapterRef.ID {
			if current != nil {
				if err := eng.Destroy(ctx, current); err != nil {
					return err
				}
			}
			if err := eng.Deploy(ctx, ex.ChapterRef); err != nil {
				return err
			}
			current = ex.ChapterRef
		}
		if err := res.Hard(ctx, ex); err != nil {
			return err
		}
		results, err := verify(ctx, ex)
		if err != nil {
			return err
		}
		if onResult != nil {
			onResult(ex.FullID(), results)
		}
	}
	return nil
}

// RunOne verifies a single exercise (deploys its chapter first, destroys after).
func RunOne(ctx context.Context, lib *content.Library, id string, eng watch.Engine, res watch.Resetter, verify Verify) ([]assert.Result, error) {
	ex := lib.ByID[id]
	if ex == nil {
		return nil, errUnknownExercise(id)
	}
	if err := eng.Deploy(ctx, ex.ChapterRef); err != nil {
		return nil, err
	}
	defer func() { _ = eng.Destroy(context.Background(), ex.ChapterRef) }()
	if err := res.Hard(ctx, ex); err != nil {
		return nil, err
	}
	return verify(ctx, ex)
}
