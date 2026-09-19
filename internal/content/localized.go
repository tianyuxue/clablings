package content

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultLocale is the fallback content language.
const DefaultLocale = "en"

// NormalizeLocale reduces a BCP-47-ish tag to its primary subtag, lowercased.
func NormalizeLocale(tag string) string {
	t := strings.ToLower(strings.TrimSpace(tag))
	t = strings.TrimLeft(t, "-_.")
	if t == "" {
		return DefaultLocale
	}
	if i := strings.IndexAny(t, "-_."); i > 0 {
		t = t[:i]
	}
	if t == "" {
		return DefaultLocale
	}
	return t
}

// localizedString accepts either a scalar (default language) or a locale->text map.
type localizedString struct {
	Values map[string]string `yaml:"-"`
}

func (l *localizedString) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		l.Values = map[string]string{DefaultLocale: node.Value}
		return nil
	case yaml.MappingNode:
		m := map[string]string{}
		if err := node.Decode(&m); err != nil {
			return err
		}
		normalized := make(map[string]string, len(m))
		for k, v := range m {
			normalized[NormalizeLocale(k)] = v
		}
		l.Values = normalized
		return nil
	default:
		return fmt.Errorf("localized string must be a scalar or a map, got kind %d", node.Kind)
	}
}

func (l localizedString) resolve(locale string) string {
	if s := l.Values[locale]; s != "" {
		return s
	}
	return l.Values[DefaultLocale]
}

// chapterYAML is the on-disk shape of chapter.yaml. Human-text fields are
// localized; non-text metadata is copied into the runtime Chapter.
type chapterYAML struct {
	Title            localizedString     `yaml:"title"`
	Objectives       map[string][]string `yaml:"objectives"`
	Difficulty       int                 `yaml:"difficulty"`
	Tags             []string            `yaml:"tags"`
	EstimatedMinutes int                 `yaml:"estimated_minutes"`
	Prerequisites    []string            `yaml:"prerequisites"`
	Chapter          `yaml:",inline"`
}

// exerciseYAML is the on-disk shape of exercise.yaml.
type exerciseYAML struct {
	Title      localizedString   `yaml:"title"`
	Hints      []localizedString `yaml:"hints"`
	Assertions []assertionYAML   `yaml:"assertions"`
	Exercise   `yaml:",inline"`
}

// assertionYAML is the on-disk shape of a single assertion.
type assertionYAML struct {
	Name      localizedString `yaml:"name"`
	Hint      localizedString `yaml:"hint"`
	Assertion `yaml:",inline"`
}

func resolveStrings(in []localizedString, locale string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, s.resolve(locale))
	}
	return out
}

// resolveLocalizedList picks the requested locale from a locale->[]string map,
// falling back to the default language. Lookups are deterministic: raw keys are
// normalized, and ties are broken by sorted key order.
func resolveLocalizedList(in map[string][]string, locale string) []string {
	if len(in) == 0 {
		return nil
	}
	locale = NormalizeLocale(locale)
	if v, ok := in[locale]; ok && len(v) > 0 {
		return v
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if NormalizeLocale(k) == locale && len(in[k]) > 0 {
			return in[k]
		}
	}
	if v, ok := in[DefaultLocale]; ok && len(v) > 0 {
		return v
	}
	for _, k := range keys {
		if NormalizeLocale(k) == DefaultLocale && len(in[k]) > 0 {
			return in[k]
		}
	}
	return nil
}
