package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LegacyLocalizedFiles reports content files that still use a language suffix.
func LegacyLocalizedFiles(root string) ([]string, error) {
	manifest, err := loadManifest(filepath.Join(root, ManifestName))
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, rel := range manifest.Chapters {
		chDir := filepath.Join(root, rel)
		err := filepath.WalkDir(chDir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			if name == "concept.md" || name == "README.md" || name == "reading.yaml" {
				return nil
			}
			base := ""
			switch {
			case strings.HasPrefix(name, "concept.") && strings.HasSuffix(name, ".md"):
				base = "concept.md"
			case strings.HasPrefix(name, "README.") && strings.HasSuffix(name, ".md"):
				base = "README.md"
			case strings.HasPrefix(name, "reading.") && strings.HasSuffix(name, ".yaml"):
				base = "reading.yaml"
			}
			if base == "" {
				return nil
			}
			stem := strings.TrimSuffix(strings.TrimPrefix(name, strings.SplitN(base, ".", 2)[0]+"."), filepath.Ext(name))
			if stem == "" {
				return nil
			}
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			problems = append(problems, fmt.Sprintf("%s: legacy localized file; use %s", filepath.ToSlash(relPath), base))
			return nil
		})
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
	}
	sort.Strings(problems)
	return problems, nil
}
