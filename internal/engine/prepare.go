package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tianyuxue/clablings/internal/content"
)

const derivedTopologyName = "clablings.generated.clab.yml"

// DerivedTopologyPath returns the generated topology path for ch, placed beside
// the original so relative references keep the same base directory.
func DerivedTopologyPath(ch *content.Chapter) string {
	return filepath.Join(filepath.Dir(ch.TopologyPath), derivedTopologyName)
}

// PrepareTopology writes a derived topology whose top-level name is the
// chapter's instance LabName and returns its path. When LabName is empty it
// returns the original path unchanged.
func PrepareTopology(ch *content.Chapter) (string, error) {
	if ch.LabName == "" {
		return ch.TopologyPath, nil
	}
	data, err := os.ReadFile(ch.TopologyPath)
	if err != nil {
		return "", fmt.Errorf("read topology %s: %w", ch.TopologyPath, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse topology %s: %w", ch.TopologyPath, err)
	}
	setTopLevelName(&doc, ch.LabName)
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("render topology %s: %w", ch.TopologyPath, err)
	}
	path := DerivedTopologyPath(ch)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", fmt.Errorf("write derived topology %s: %w", path, err)
	}
	return path, nil
}

func setTopLevelName(doc *yaml.Node, name string) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "name" {
			root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"}
			return
		}
	}
	root.Content = append([]*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
	}, root.Content...)
}
