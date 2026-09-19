package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tianyuxue/clablings/internal/content"
)

// DerivedTopologyPath returns the generated topology path for ch, placed beside
// the original so relative references keep the same base directory. The name
// includes the instance LabName so concurrent instances sharing a library do
// not clobber each other.
func DerivedTopologyPath(ch *content.Chapter) string {
	return filepath.Join(filepath.Dir(ch.TopologyPath), "clablings."+ch.LabName+".clab.yml")
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
	if err := setTopLevelName(&doc, ch.LabName); err != nil {
		return "", fmt.Errorf("set topology name in %s: %w", ch.TopologyPath, err)
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("render topology %s: %w", ch.TopologyPath, err)
	}
	path := DerivedTopologyPath(ch)
	tmp, err := os.CreateTemp(filepath.Dir(path), "clablings.*.clab.yml")
	if err != nil {
		return "", fmt.Errorf("create derived topology: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("write derived topology %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("sync derived topology %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("close derived topology %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("commit derived topology %s: %w", path, err)
	}
	// CreateTemp makes the file 0600; make it world-readable like a normal
	// topology so other tools/users can inspect it.
	if err := os.Chmod(path, 0o644); err != nil {
		return "", fmt.Errorf("chmod derived topology %s: %w", path, err)
	}
	return path, nil
}

func setTopLevelName(doc *yaml.Node, name string) error {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("topology document is empty")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("topology root is not a mapping")
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "name" {
			root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"}
			return nil
		}
	}
	root.Content = append([]*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
	}, root.Content...)
	return nil
}
