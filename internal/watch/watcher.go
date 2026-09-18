// Package watch implements the triggering and advancement logic of `clablings watch`.
package watch

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
)

// Sentinel is the completion marker string.
const Sentinel = assert.Sentinel

type fingerprint struct {
	size    int64
	modTime time.Time
}

// Watcher polls an exercise's declared files, detecting changes and gating on the sentinel.
type Watcher struct {
	ex                  *content.Exercise
	debounce            time.Duration
	last                map[string]fingerprint
	changed             bool
	changedAt           time.Time
	allSentinelsRemoved bool
}

// NewWatcher builds a Watcher. debounce is the stability window after the last change.
func NewWatcher(ex *content.Exercise, debounce time.Duration) *Watcher {
	return &Watcher{ex: ex, debounce: debounce, last: map[string]fingerprint{}}
}

// Scan checks all watched file fingerprints once. A change is detected when a
// file is new to the watcher or its size/modTime differ. The watcher is ready
// only when every watched file has the sentinel removed. Fingerprints are
// committed only on a fully successful scan, so a transient error does not
// swallow a change: the next scan re-detects it.
func (w *Watcher) Scan() (bool, error) {
	now := time.Now()
	next := make(map[string]fingerprint, len(w.ex.Files))
	newlyChanged := false
	allRemoved := true

	for _, f := range w.ex.Files {
		path := filepath.Join(w.ex.Dir, f.Source)
		st, err := os.Stat(path)
		if err != nil {
			return false, fmt.Errorf("watched file %s: %w", f.Source, err)
		}
		fp := fingerprint{size: st.Size(), modTime: st.ModTime()}
		prev, ok := w.last[path]
		if !ok || prev.size != fp.size || !prev.modTime.Equal(fp.modTime) {
			newlyChanged = true
		}
		next[path] = fp

		data, err := os.ReadFile(path)
		if err != nil {
			return false, fmt.Errorf("watched file %s: %w", f.Source, err)
		}
		if bytes.Contains(data, []byte(Sentinel)) {
			allRemoved = false
		}
	}

	w.last = next
	w.allSentinelsRemoved = allRemoved
	if newlyChanged {
		w.changed = true
		w.changedAt = now
	}
	return newlyChanged, nil
}

// ReadyToVerify reports true once the last change has been stable for the debounce
// window and every watched file has the sentinel removed.
func (w *Watcher) ReadyToVerify() bool {
	if !w.changed || !w.allSentinelsRemoved {
		return false
	}
	return time.Since(w.changedAt) >= w.debounce
}

// Consume clears the pending-change flag after a verification.
func (w *Watcher) Consume() {
	w.changed = false
}
