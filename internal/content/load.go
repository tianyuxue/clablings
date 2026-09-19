package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Load parses the library root.
func Load(root string) (*Library, error) {
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
		ch, err := loadChapter(chDir)
		if err != nil {
			return nil, fmt.Errorf("chapter %s: %w", rel, err)
		}
		ch.LabName = LabName(root, instance, ch.TopologyName)
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
			ex, err := loadExercise(filepath.Join(exDir, name))
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

func loadChapter(dir string) (*Chapter, error) {
	var ch Chapter
	if err := decodeYAML(filepath.Join(dir, "chapter.yaml"), &ch); err != nil {
		return nil, err
	}
	if ch.ID == "" {
		return nil, fmt.Errorf("%s: missing id", dir)
	}
	if ch.Topology == "" {
		return nil, fmt.Errorf("chapter %s: missing topology", ch.ID)
	}
	ch.Dir = dir
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

func loadExercise(dir string) (*Exercise, error) {
	var ex Exercise
	if err := decodeYAML(filepath.Join(dir, "exercise.yaml"), &ex); err != nil {
		return nil, err
	}
	if ex.ID == "" {
		return nil, fmt.Errorf("%s: missing id", dir)
	}
	switch ex.Type {
	case TypeConfigFix, TypeCodeImplement:
	default:
		return nil, fmt.Errorf("exercise %s: invalid type %q", ex.ID, ex.Type)
	}
	if ex.Type == TypeCodeImplement && (ex.Build == nil || ex.Run == nil) {
		return nil, fmt.Errorf("exercise %s: code-implement requires build and run", ex.ID)
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
