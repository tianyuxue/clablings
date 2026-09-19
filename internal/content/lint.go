package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func (l localizedString) has(locale string) bool { return l.Values[locale] != "" }

func hasLocaleList(in map[string][]string, locale string) bool {
	for k, v := range in {
		if NormalizeLocale(k) == locale && len(v) > 0 {
			return true
		}
	}
	return false
}

// MissingTranslations reports human-text fields with no value for locale.
func MissingTranslations(root, locale string) ([]string, error) {
	locale = NormalizeLocale(locale)
	manifest, err := loadManifest(filepath.Join(root, ManifestName))
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, rel := range manifest.Chapters {
		chDir := filepath.Join(root, rel)
		var ch chapterYAML
		if err := decodeYAML(filepath.Join(chDir, "chapter.yaml"), &ch); err != nil {
			return nil, err
		}
		if !ch.Title.has(locale) {
			problems = append(problems, fmt.Sprintf("chapter %s: title missing %s", ch.ID, locale))
		}
		if !hasLocaleList(ch.Objectives, locale) {
			problems = append(problems, fmt.Sprintf("chapter %s: objectives missing %s", ch.ID, locale))
		}
		exDir := filepath.Join(chDir, "exercises")
		entries, err := os.ReadDir(exDir)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			var ex exerciseYAML
			if err := decodeYAML(filepath.Join(exDir, name, "exercise.yaml"), &ex); err != nil {
				return nil, err
			}
			full := ch.ID + "/" + ex.ID
			if !ex.Title.has(locale) {
				problems = append(problems, fmt.Sprintf("exercise %s: title missing %s", full, locale))
			}
			for i, h := range ex.Hints {
				if !h.has(locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: hint %d missing %s", full, i+1, locale))
				}
			}
			for i, a := range ex.Assertions {
				if !a.Name.has(locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: assertion %d name missing %s", full, i+1, locale))
				}
				if !a.Hint.has(locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: assertion %d hint missing %s", full, i+1, locale))
				}
			}
			for qi, q := range ex.Questions {
				if !q.Prompt.has(locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: question %d prompt missing %s", full, qi+1, locale))
				}
				for oi, o := range q.Options {
					if !o.Text.has(locale) {
						problems = append(problems, fmt.Sprintf("exercise %s: question %d option %d text missing %s", full, qi+1, oi+1, locale))
					}
				}
			}
		}
	}
	return problems, nil
}
