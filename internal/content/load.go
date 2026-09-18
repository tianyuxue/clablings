package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Load 解析题库根。
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

	for _, rel := range manifest.Chapters {
		chDir := filepath.Join(root, rel)
		ch, err := loadChapter(chDir)
		if err != nil {
			return nil, fmt.Errorf("章节 %s: %w", rel, err)
		}
		lib.Chapters = append(lib.Chapters, ch)

		exDir := filepath.Join(chDir, "exercises")
		entries, err := os.ReadDir(exDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("章节 %s 的 exercises 目录: %w", ch.ID, err)
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
				return nil, fmt.Errorf("题目 %s/%s: %w", ch.ID, name, err)
			}
			ex.Chapter = ch.ID
			if _, dup := lib.ByID[ex.FullID()]; dup {
				return nil, fmt.Errorf("题目 ID 重复: %s", ex.FullID())
			}
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
		return m, fmt.Errorf("解析 %s: %w", path, err)
	}
	if m.Version == 0 {
		return m, fmt.Errorf("%s: 缺少 version", path)
	}
	if len(m.Chapters) == 0 {
		return m, fmt.Errorf("%s: chapters 为空", path)
	}
	return m, nil
}

func loadChapter(dir string) (*Chapter, error) {
	var ch Chapter
	if err := decodeYAML(filepath.Join(dir, "chapter.yaml"), &ch); err != nil {
		return nil, err
	}
	if ch.ID == "" {
		return nil, fmt.Errorf("%s: 缺少 id", dir)
	}
	if ch.Topology == "" {
		return nil, fmt.Errorf("章节 %s: 缺少 topology", ch.ID)
	}
	ch.Dir = dir
	ch.TopologyPath = filepath.Join(dir, ch.Topology)
	if _, err := os.Stat(ch.TopologyPath); err != nil {
		return nil, fmt.Errorf("章节 %s: topology 文件不存在: %w", ch.ID, err)
	}
	name, err := readTopologyName(ch.TopologyPath)
	if err != nil {
		return nil, fmt.Errorf("章节 %s: %w", ch.ID, err)
	}
	if name == "" {
		return nil, fmt.Errorf("章节 %s: 拓扑缺少顶层 name", ch.ID)
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
		return nil, fmt.Errorf("%s: 缺少 id", dir)
	}
	switch ex.Type {
	case TypeConfigFix, TypeCodeImplement:
	default:
		return nil, fmt.Errorf("题目 %s: 非法 type %q", ex.ID, ex.Type)
	}
	if ex.Type == TypeCodeImplement && (ex.Build == nil || ex.Run == nil) {
		return nil, fmt.Errorf("题目 %s: code-implement 需要 build 与 run", ex.ID)
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
