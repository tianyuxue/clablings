// Package watch implements the triggering and advancement logic of `clablings watch`.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
)

// Sentinel is the completion marker string (kept in sync with assert.Sentinel).
const Sentinel = "I AM NOT DONE"

type fingerprint struct {
	size    int64
	modTime time.Time
}

// Watcher polls an exercise's declared files, detecting changes and gating on the sentinel.
type Watcher struct {
	ex          *content.Exercise
	debounce    time.Duration
	last        map[string]fingerprint
	changed     bool
	changedAt   time.Time
	missingSent bool
}

// NewWatcher builds a Watcher. debounce is the stability window after the last change.
func NewWatcher(ex *content.Exercise, debounce time.Duration) *Watcher {
	return &Watcher{ex: ex, debounce: debounce, last: map[string]fingerprint{}}
}

// Scan checks file fingerprints once and reports whether a new change was detected.
func (w *Watcher) Scan() (bool, error) {
	now := time.Now()
	newlyChanged := false
	missingSentinel := false

	for _, f := range w.ex.Files {
		path := filepath.Join(w.ex.Dir, f.Source)
		st, err := os.Stat(path)
		if err != nil {
			return false, fmt.Errorf("watched file %s: %w", f.Source, err)
		}
		fp := fingerprint{size: st.Size(), modTime: st.ModTime()}
		if prev, ok := w.last[path]; !ok || prev != fp {
			newlyChanged = true
		}
		w.last[path] = fp

		data, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		if !strings.Contains(string(data), Sentinel) {
			missingSentinel = true
		} else {
			missingSentinel = false
			break
		}
	}

	w.missingSent = missingSentinel
	if newlyChanged {
		w.changed = true
		w.changedAt = now
	}
	return newlyChanged, nil
}

// ReadyToVerify reports true once the last change has been stable for the debounce
// window and every watched file has the sentinel removed.
func (w *Watcher) ReadyToVerify() bool {
	if !w.changed || !w.missingSent {
		return false
	}
	return time.Since(w.changedAt) >= w.debounce
}

// Consume clears the pending-change flag after a verification.
func (w *Watcher) Consume() {
	w.changed = false
}
