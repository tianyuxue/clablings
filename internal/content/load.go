package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Load parses the library using the default locale.
func Load(root string) (*Library, error) { return LoadWithLocale(root, DefaultLocale) }

// LoadWithLocale parses the library, resolving human text to locale.
func LoadWithLocale(root, locale string) (*Library, error) {
	locale = NormalizeLocale(locale)
	manifest, err := loadManifest(filepath.Join(root, ManifestName))
	if err != nil {
		return nil, err
	}

	lib := &Library{
		Root:     root,
		Manifest: manifest,
		ByID:     map[string]*Exercise{},
	}

	instance := InstanceID(root)

	for _, rel := range manifest.Chapters {
		chDir := filepath.Join(root, rel)
		ch, err := loadChapter(chDir, locale)
		if err != nil {
			return nil, fmt.Errorf("chapter %s: %w", rel, err)
		}
		if ch.TopologyName != "" {
			ch.LabName = LabName(root, instance, ch.TopologyName)
		}
		lib.Chapters = append(lib.Chapters, ch)

		exDir := filepath.Join(chDir, "exercises")
		entries, err := os.ReadDir(exDir)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("exercises directory of chapter %s: %w", ch.ID, err)
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			ex, err := loadExercise(filepath.Join(exDir, name), locale)
			if err != nil {
				return nil, fmt.Errorf("exercise %s/%s: %w", ch.ID, name, err)
			}
			ex.Chapter = ch.ID
			if _, dup := lib.ByID[ex.FullID()]; dup {
				return nil, fmt.Errorf("duplicate exercise ID: %s", ex.FullID())
			}
			ex.ChapterRef = ch
			ch.Exercises = append(ch.Exercises, ex)
			lib.ByID[ex.FullID()] = ex
			lib.Order = append(lib.Order, ex)
		}
	}
	return lib, nil
}

func loadManifest(path string) (Manifest, error) {
	var m Manifest
	if err := decodeYAML(path, &m); err != nil {
		return m, fmt.Errorf("parse %s: %w", path, err)
	}
	if m.Version == 0 {
		return m, fmt.Errorf("%s: missing version", path)
	}
	if len(m.Chapters) == 0 {
		return m, fmt.Errorf("%s: chapters is empty", path)
	}
	return m, nil
}

func loadChapter(dir, locale string) (*Chapter, error) {
	var dto chapterYAML
	if err := decodeYAML(filepath.Join(dir, "chapter.yaml"), &dto); err != nil {
		return nil, err
	}
	ch := dto.Chapter
	ch.Title = dto.Title.resolve(locale)
	ch.Objectives = resolveLocalizedList(dto.Objectives, locale)
	ch.Difficulty = dto.Difficulty
	ch.Tags = dto.Tags
	ch.EstimatedMinutes = dto.EstimatedMinutes
	ch.Prerequisites = dto.Prerequisites
	if ch.ID == "" {
		return nil, fmt.Errorf("%s: missing id", dir)
	}
	ch.Dir = dir
	if ch.Topology != "" {
		ch.TopologyPath = filepath.Join(dir, ch.Topology)
		if _, err := os.Stat(ch.TopologyPath); err != nil {
			return nil, fmt.Errorf("chapter %s: topology file does not exist: %w", ch.ID, err)
		}
		name, err := readTopologyName(ch.TopologyPath)
		if err != nil {
			return nil, fmt.Errorf("chapter %s: %w", ch.ID, err)
		}
		if name == "" {
			return nil, fmt.Errorf("chapter %s: topology missing top-level name", ch.ID)
		}
		ch.TopologyName = name
	}
	return &ch, nil
}

func readTopologyName(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var raw struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", err
	}
	return raw.Name, nil
}

func loadExercise(dir, locale string) (*Exercise, error) {
	var dto exerciseYAML
	if err := decodeYAML(filepath.Join(dir, "exercise.yaml"), &dto); err != nil {
		return nil, err
	}
	ex := dto.Exercise
	ex.Title = dto.Title.resolve(locale)
	ex.Hints = resolveStrings(dto.Hints, locale)
	ex.Difficulty = dto.Difficulty
	ex.Tags = dto.Tags
	if len(dto.Assertions) > 0 {
		ex.Assertions = make([]Assertion, len(dto.Assertions))
		for i := range dto.Assertions {
			ex.Assertions[i] = dto.Assertions[i].Assertion
			ex.Assertions[i].Name = dto.Assertions[i].Name.resolve(locale)
			ex.Assertions[i].Hint = dto.Assertions[i].Hint.resolve(locale)
		}
	}
	if len(dto.Questions) > 0 {
		ex.Questions = make([]Question, len(dto.Questions))
		for i := range dto.Questions {
			q := Question{
				Prompt:   dto.Questions[i].Prompt.resolve(locale),
				Multiple: dto.Questions[i].Multiple,
			}
			if n := len(dto.Questions[i].Options); n > 0 {
				q.Options = make([]Option, n)
				for j := range dto.Questions[i].Options {
					o := dto.Questions[i].Options[j]
					q.Options[j] = Option{
						Text:        o.Text.resolve(locale),
						Explanation: o.Explanation.resolve(locale),
						Correct:     o.Correct,
					}
				}
			}
			ex.Questions[i] = q
		}
	}
	if ex.ID == "" {
		return nil, fmt.Errorf("%s: missing id", dir)
	}
	switch ex.Type {
	case TypeConfigFix, TypeCodeImplement, TypeQuiz:
	default:
		return nil, fmt.Errorf("exercise %s: invalid type %q", ex.ID, ex.Type)
	}
	if ex.Type == TypeCodeImplement && (ex.Build == nil || ex.Run == nil) {
		return nil, fmt.Errorf("exercise %s: code-implement requires build and run", ex.ID)
	}
	if ex.Type == TypeQuiz {
		if len(ex.Questions) == 0 {
			return nil, fmt.Errorf("exercise %s: quiz requires at least one question", ex.ID)
		}
		for qi := range ex.Questions {
			q := &ex.Questions[qi]
			if len(q.Options) < 2 {
				return nil, fmt.Errorf("exercise %s: question %d needs at least two options", ex.ID, qi+1)
			}
			correct := 0
			for _, o := range q.Options {
				if o.Correct {
					correct++
				}
			}
			if correct == 0 {
				return nil, fmt.Errorf("exercise %s: question %d needs a correct option", ex.ID, qi+1)
			}
			if !q.Multiple && correct != 1 {
				return nil, fmt.Errorf("exercise %s: question %d single-choice question must have exactly one correct option", ex.ID, qi+1)
			}
		}
	}
	ex.Dir = dir
	return &ex, nil
}

func decodeYAML(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
