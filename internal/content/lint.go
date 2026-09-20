package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tianyuxue/clablings/internal/i18n"
)

func (l localizedString) has(locale string) bool { return l.Values[locale] != "" }

// missingText reports a field that lacks locale but was authored in some locale.
func missingText(l localizedString, locale string) bool {
	if l.has(locale) {
		return false
	}
	return len(l.Values) > 0
}

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
		if missingText(ch.Title, locale) {
			problems = append(problems, fmt.Sprintf("chapter %s: title missing %s", ch.ID, locale))
		}
		if len(ch.Objectives) > 0 && !hasLocaleList(ch.Objectives, locale) {
			problems = append(problems, fmt.Sprintf("chapter %s: objectives missing %s", ch.ID, locale))
		}
		exDir := filepath.Join(chDir, "exercises")
		entries, err := os.ReadDir(exDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read exercises dir %s: %w", exDir, err)
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
			if missingText(ex.Title, locale) {
				problems = append(problems, fmt.Sprintf("exercise %s: title missing %s", full, locale))
			}
			if missingText(ex.Brief, locale) {
				problems = append(problems, fmt.Sprintf("exercise %s: brief missing %s", full, locale))
			}
			for i, h := range ex.Hints {
				if missingText(h, locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: hint %d missing %s", full, i+1, locale))
				}
			}
			for i, a := range ex.Assertions {
				if missingText(a.Name, locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: assertion %d name missing %s", full, i+1, locale))
				}
				if missingText(a.Hint, locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: assertion %d hint missing %s", full, i+1, locale))
				}
			}
			for qi, q := range ex.Questions {
				if missingText(q.Prompt, locale) {
					problems = append(problems, fmt.Sprintf("exercise %s: question %d prompt missing %s", full, qi+1, locale))
				}
				for oi, o := range q.Options {
					if missingText(o.Text, locale) {
						problems = append(problems, fmt.Sprintf("exercise %s: question %d option %d text missing %s", full, qi+1, oi+1, locale))
					}
					if missingText(o.Explanation, locale) {
						problems = append(problems, fmt.Sprintf("exercise %s: question %d option %d explanation missing %s", full, qi+1, oi+1, locale))
					}
				}
			}
		}
	}
	return problems, nil
}

// TranslationParity checks that a locale's content stays structurally in sync
// with the default locale: markdown parity for concept files, plus a git-based
// freshness warning when the source was committed more recently.
func TranslationParity(root, locale string) (problems, warns []string, err error) {
	return translationParity(root, locale, i18n.GitLastCommit(root))
}

func translationParity(root, locale string, last i18n.LastCommit) (problems, warns []string, err error) {
	locale = NormalizeLocale(locale)
	manifest, err := loadManifest(filepath.Join(root, ManifestName))
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range manifest.Chapters {
		chDir := filepath.Join(root, rel)
		enPath := filepath.Join(chDir, "concept."+DefaultLocale+".md")
		locPath := filepath.Join(chDir, "concept."+locale+".md")
		enData, enOK, err := readIfExists(enPath)
		if err != nil {
			return nil, nil, err
		}
		locData, locOK, err := readIfExists(locPath)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case !enOK && !locOK:
			continue
		case enOK && !locOK:
			problems = append(problems, fmt.Sprintf("%s: concept.%s.md missing", rel, locale))
			continue
		case !enOK && locOK:
			problems = append(problems, fmt.Sprintf("%s: concept.%s.md missing", rel, DefaultLocale))
			continue
		}
		for _, p := range i18n.MarkdownParity(string(enData), string(locData)) {
			problems = append(problems, fmt.Sprintf("%s: concept.%s.md %s", rel, locale, p))
		}
		stale, comparable, err := i18n.Stale(last, enPath, locPath)
		if err != nil {
			return nil, nil, err
		}
		if comparable && stale {
			warns = append(warns, fmt.Sprintf("%s/concept.%s.md may be outdated (concept.%s.md changed more recently)", rel, locale, DefaultLocale))
		}
	}
	return problems, warns, nil
}

func readIfExists(path string) ([]byte, bool, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		return b, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}
