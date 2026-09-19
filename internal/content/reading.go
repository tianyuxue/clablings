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
	Intro    localizedString `yaml:"intro"`
	Question localizedString `yaml:"question"`
	Reading  `yaml:",inline"`
}

// LoadReading loads reading.<locale>.yaml for a chapter directory, falling back
// to reading.<DefaultLocale>.yaml. A missing file yields (nil, nil).
func LoadReading(chapterDir, locale string) ([]Reading, error) {
	locale = NormalizeLocale(locale)
	candidates := []string{
		filepath.Join(chapterDir, "reading."+locale+".yaml"),
		filepath.Join(chapterDir, "reading."+DefaultLocale+".yaml"),
	}
	var data []byte
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			data = b
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
	}
	if data == nil {
		return nil, nil
	}
	var raw []readingYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse reading: %w", err)
	}
	out := make([]Reading, 0, len(raw))
	for _, r := range raw {
		rd := r.Reading
		rd.Intro = r.Intro.resolve(locale)
		rd.Question = r.Question.resolve(locale)
		out = append(out, rd)
	}
	return out, nil
}
