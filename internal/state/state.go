// Package state persists learner progress to <root>/.clablings_state.json.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const StateFileName = ".clablings_state.json"

const currentVersion = 1

// Status is an exercise's progress.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusPassed     Status = "passed"
)

// ExerciseState is the progress of a single exercise.
type ExerciseState struct {
	Status    Status    `json:"status"`
	Attempts  int       `json:"attempts"`
	UpdatedAt time.Time `json:"updated_at"`
}

// State is the complete progress state.
type State struct {
	Version   int                      `json:"version"`
	Current   string                   `json:"current"`
	Exercises map[string]ExerciseState `json:"exercises"`
}

// CorruptError indicates the state file is corrupt; RenameErr is nil if quarantine succeeded.
type CorruptError struct {
	Path      string
	Err       error
	RenameErr error
}

func (e *CorruptError) Error() string {
	if e.RenameErr != nil {
		return fmt.Sprintf("state file corrupt, failed to quarantine to %s.bak: %v (original error: %v)", e.Path, e.RenameErr, e.Err)
	}
	return fmt.Sprintf("state file corrupt, quarantined to %s.bak: %v", e.Path, e.Err)
}

func (e *CorruptError) Unwrap() error { return e.Err }

// Load reads the state; a missing file returns an empty state; if corrupt, it quarantines
// the file and returns an empty state together with a CorruptError.
func Load(root string) (*State, error) {
	path := filepath.Join(root, StateFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newState(), nil
	}
	if err != nil {
		return nil, err
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		renameErr := os.Rename(path, path+".bak")
		return newState(), &CorruptError{Path: path, Err: err, RenameErr: renameErr}
	}
	if st.Exercises == nil {
		st.Exercises = map[string]ExerciseState{}
	}
	if st.Version == 0 {
		st.Version = currentVersion
	}
	return &st, nil
}

// Save atomically writes the state (temp file + fsync + rename).
func (s *State) Save(root string) error {
	if s.Version == 0 {
		s.Version = currentVersion
	}
	if s.Exercises == nil {
		s.Exercises = map[string]ExerciseState{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(root, StateFileName+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(root, StateFileName)); err != nil {
		return err
	}
	dir, err := os.Open(root)
	if err != nil {
		return fmt.Errorf("open parent dir for fsync: %w", err)
	}
	if err := dir.Sync(); err != nil {
		dir.Close()
		return fmt.Errorf("fsync parent dir: %w", err)
	}
	return dir.Close()
}

// SetCurrent records the current exercise.
func (s *State) SetCurrent(id string) { s.Current = id }

// Get returns the exercise progress; if absent, it returns todo status.
func (s *State) Get(id string) ExerciseState {
	if es, ok := s.Exercises[id]; ok {
		return es
	}
	return ExerciseState{Status: StatusTodo}
}

// Upsert updates exercise progress (attempts is cumulative).
func (s *State) Upsert(id string, status Status, attempts int) {
	if s.Exercises == nil {
		s.Exercises = map[string]ExerciseState{}
	}
	s.Exercises[id] = ExerciseState{Status: status, Attempts: attempts, UpdatedAt: time.Now().UTC()}
}

func newState() *State {
	return &State{Version: currentVersion, Exercises: map[string]ExerciseState{}}
}
