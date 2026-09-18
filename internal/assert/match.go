// Package assert executes declarative assertions and produces structured results.
package assert

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tianyuxue/clablings/internal/content"
)

// matchText reports whether got satisfies tm. An invalid regex is a config
// error and is surfaced to the caller rather than treated as a mismatch.
func matchText(got string, tm content.TextMatch) (bool, error) {
	switch {
	case tm.Equals != "":
		return strings.TrimSpace(got) == strings.TrimSpace(tm.Equals), nil
	case tm.Contains != "":
		return strings.Contains(got, tm.Contains), nil
	case tm.Regex != "":
		re, err := regexp.Compile(tm.Regex)
		if err != nil {
			return false, fmt.Errorf("invalid regex %q: %w", tm.Regex, err)
		}
		return re.MatchString(got), nil
	default:
		return true, nil
	}
}

func textMatchEmpty(tm content.TextMatch) bool {
	return tm.Equals == "" && tm.Contains == "" && tm.Regex == ""
}

func describeTextMatch(tm content.TextMatch) string {
	switch {
	case tm.Equals != "":
		return "equals " + strconv.Quote(tm.Equals)
	case tm.Contains != "":
		return "contains " + strconv.Quote(tm.Contains)
	case tm.Regex != "":
		return "regex " + strconv.Quote(tm.Regex)
	default:
		return "(no text match condition)"
	}
}

type pathToken struct {
	key      string
	wildcard bool
	index    int
	isIndex  bool
}

func evalJSONPath(data any, path string) ([]any, error) {
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("JSONPath must start with $: %q", path)
	}
	toks, err := tokenizePath(path[1:])
	if err != nil {
		return nil, err
	}
	cur := []any{data}
	for _, tok := range toks {
		var next []any
		for _, v := range cur {
			next = append(next, applyToken(v, tok)...)
		}
		cur = next
	}
	return cur, nil
}

func tokenizePath(s string) ([]pathToken, error) {
	var toks []pathToken
	i := 0
	for i < len(s) {
		switch s[i] {
		case '.':
			i++
			j := i
			for j < len(s) && s[j] != '.' && s[j] != '[' {
				j++
			}
			seg := s[i:j]
			if seg == "" {
				return nil, fmt.Errorf("invalid JSONPath segment (empty)")
			}
			if seg == "*" {
				toks = append(toks, pathToken{wildcard: true})
			} else {
				toks = append(toks, pathToken{key: seg})
			}
			i = j
		case '[':
			end := strings.IndexByte(s[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unclosed '[' in JSONPath")
			}
			inner := s[i+1 : i+end]
			if inner == "*" {
				toks = append(toks, pathToken{wildcard: true})
			} else {
				n, err := strconv.Atoi(inner)
				if err != nil || n < 0 {
					return nil, fmt.Errorf("invalid array index %q", inner)
				}
				toks = append(toks, pathToken{index: n, isIndex: true})
			}
			i += end + 1
		default:
			return nil, fmt.Errorf("invalid JSONPath character %q", s[i])
		}
	}
	return toks, nil
}

func applyToken(v any, tok pathToken) []any {
	switch node := v.(type) {
	case map[string]any:
		if tok.wildcard {
			out := make([]any, 0, len(node))
			for _, val := range node {
				out = append(out, val)
			}
			return out
		}
		if tok.isIndex {
			return nil
		}
		if val, ok := node[tok.key]; ok {
			return []any{val}
		}
		return nil
	case []any:
		if tok.isIndex {
			if tok.index >= 0 && tok.index < len(node) {
				return []any{node[tok.index]}
			}
			return nil
		}
		if tok.wildcard {
			return append([]any{}, node...)
		}
		return nil
	default:
		return nil
	}
}

// matchJSON extracts values at path and requires every extracted value to match
// tm. Extracted values are sorted so the reported actual string is deterministic.
func matchJSON(stdout, path string, tm content.TextMatch) (bool, string, error) {
	var data any
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return false, "", fmt.Errorf("stdout is not valid JSON: %w", err)
	}
	vals, err := evalJSONPath(data, path)
	if err != nil {
		return false, "", err
	}
	if len(vals) == 0 {
		return false, "(JSONPath matched nothing)", nil
	}
	strs := make([]string, 0, len(vals))
	for _, v := range vals {
		s, err := scalarString(v)
		if err != nil {
			return false, "", err
		}
		strs = append(strs, s)
	}
	sort.Strings(strs)
	actual := strings.Join(strs, ", ")
	for _, s := range strs {
		ok, err := matchText(s, tm)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return false, actual, nil
		}
	}
	return true, actual, nil
}

func scalarString(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(x), nil
	case nil:
		return "null", nil
	default:
		b, err := json.Marshal(x)
		return string(b), err
	}
}
