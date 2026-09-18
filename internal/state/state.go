// Package state 持久化学员进度到 <root>/.clablings_state.json。
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

// Status 是题目进度。
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusPassed     Status = "passed"
)

// ExerciseState 是单题进度。
type ExerciseState struct {
	Status    Status    `json:"status"`
	Attempts  int       `json:"attempts"`
	UpdatedAt time.Time `json:"updated_at"`
}

// State 是完整进度状态。
type State struct {
	Version   int                      `json:"version"`
	Current   string                   `json:"current"`
	Exercises map[string]ExerciseState `json:"exercises"`
}

// CorruptError 表示状态文件损坏并已被隔离为 .bak。
type CorruptError struct {
	Path string
	Err  error
}

func (e *CorruptError) Error() string {
	return fmt.Sprintf("状态文件损坏，已隔离到 %s.bak: %v", e.Path, e.Err)
}

func (e *CorruptError) Unwrap() error { return e.Err }

// Load 读取状态；文件不存在返回空状态；损坏则隔离并将空状态与 CorruptError 一起返回。
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
		_ = os.Rename(path, path+".bak")
		return newState(), &CorruptError{Path: path, Err: err}
	}
	if st.Exercises == nil {
		st.Exercises = map[string]ExerciseState{}
	}
	if st.Version == 0 {
		st.Version = currentVersion
	}
	return &st, nil
}

// Save 原子写状态（临时文件 + fsync + rename）。
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
	return os.Rename(tmpName, filepath.Join(root, StateFileName))
}

// SetCurrent 记录当前题目。
func (s *State) SetCurrent(id string) { s.Current = id }

// Get 返回题目进度；不存在时返回 todo 状态。
func (s *State) Get(id string) ExerciseState {
	if es, ok := s.Exercises[id]; ok {
		return es
	}
	return ExerciseState{Status: StatusTodo}
}

// Upsert 更新题目进度（attempts 为累计值）。
func (s *State) Upsert(id string, status Status, attempts int) {
	if s.Exercises == nil {
		s.Exercises = map[string]ExerciseState{}
	}
	s.Exercises[id] = ExerciseState{Status: status, Attempts: attempts, UpdatedAt: time.Now().UTC()}
}

func newState() *State {
	return &State{Version: currentVersion, Exercises: map[string]ExerciseState{}}
}
