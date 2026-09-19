package content

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Answers maps an exercise FullID to per-question lists of chosen option indices.
type Answers map[string][][]int

// LoadAnswers reads an answers YAML file.
func LoadAnswers(path string) (Answers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a Answers
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse answers %s: %w", path, err)
	}
	return a, nil
}
