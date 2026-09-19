package content

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultLocale is the fallback content language.
const DefaultLocale = "en"

// NormalizeLocale reduces a BCP-47-ish tag to the supported set {en, zh} or its
// primary subtag.
func NormalizeLocale(tag string) string {
	t := strings.ToLower(strings.TrimSpace(tag))
	if t == "" {
		return DefaultLocale
	}
	if i := strings.IndexAny(t, "-_."); i > 0 {
		t = t[:i]
	}
	switch t {
	case "en":
		return "en"
	case "zh":
		return "zh"
	default:
		return t
	}
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
		l.Values = m
		return nil
	default:
		return fmt.Errorf("localized string must be a scalar or a map, got kind %d", node.Kind)
	}
}

func (l localizedString) resolve(locale string) string {
	if s := l.Values[locale]; s != "" {
		return s
	}
	if s := l.Values[DefaultLocale]; s != "" {
		return s
	}
	for _, s := range l.Values {
		if s != "" {
			return s
		}
	}
	return ""
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

// resolveLocalizedList picks the requested locale from a locale->[]string map.
func resolveLocalizedList(in map[string][]string, locale string) []string {
	if in == nil {
		return nil
	}
	if v := in[locale]; len(v) > 0 {
		return v
	}
	if v := in[DefaultLocale]; len(v) > 0 {
		return v
	}
	for _, v := range in {
		if len(v) > 0 {
			return v
		}
	}
	return nil
}
