package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Reading is one structured RFC reading slice.
type Reading struct {
	RFC          int    `yaml:"rfc"`
	Title        string `yaml:"title"`
	Section      string `yaml:"section"`
	SectionTitle string `yaml:"section_title"`
	Kind         string `yaml:"kind"` // required | optional
	URL          string `yaml:"url"`
	Intro        string `yaml:"-"`
	Question     string `yaml:"-"`
}

type readingYAML struct {
	Intro    string `yaml:"intro"`
	Question string `yaml:"question"`
	Reading  `yaml:",inline"`
}

// LoadReading loads reading.yaml for a chapter directory. A missing file yields (nil, nil).
func LoadReading(chapterDir string) ([]Reading, error) {
	path := filepath.Join(chapterDir, "reading.yaml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw []readingYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse reading: %w", err)
	}
	out := make([]Reading, 0, len(raw))
	for _, r := range raw {
		rd := r.Reading
		rd.Intro = r.Intro
		rd.Question = r.Question
		out = append(out, rd)
	}
	return out, nil
}
