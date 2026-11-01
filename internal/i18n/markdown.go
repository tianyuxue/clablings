// Package i18n checks that localized content keeps the same technical facts.
package i18n

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type codeBlock struct {
	lang string
	body string
}

type docModel struct {
	code  []codeBlock
	links []string
}

// TechnicalParity compares the English and Chinese versions of the same
// markdown document. Only language-independent technical parts must match:
// normalized fenced code blocks (as a multiset, order-independent) and link
// destinations. Headings, prose, paragraph count, and inline code may differ.
func TechnicalParity(en, zh string) []string {
	e := parseMarkdown(en)
	z := parseMarkdown(zh)
	var p []string

	if len(e.code) != len(z.code) {
		p = append(p, fmt.Sprintf("code block count differs: en=%d zh=%d", len(e.code), len(z.code)))
	} else if !sameCounter(counter(codeKeys(e.code)), counter(codeKeys(z.code))) {
		p = append(p, "code block content or language differs between en and zh")
	}

	p = append(p, diffReport("link", e.links, z.links)...)
	return p
}

func parseMarkdown(src string) docModel {
	data := []byte(src)
	root := goldmark.New().Parser().Parse(text.NewReader(data))
	var m docModel
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.FencedCodeBlock:
			lang := string(v.Language(data))
			m.code = append(m.code, codeBlock{lang: lang, body: normalizeCode(blockText(v, data), lang)})
		case *ast.CodeBlock:
			m.code = append(m.code, codeBlock{body: normalizeCode(blockText(v, data), "")})
		case *ast.Link:
			m.links = append(m.links, string(v.Destination))
		case *ast.Image:
			m.links = append(m.links, string(v.Destination))
		case *ast.AutoLink:
			m.links = append(m.links, string(v.URL(data)))
		}
		return ast.WalkContinue, nil
	})
	return m
}

type hasLines interface{ Lines() *text.Segments }

func blockText(n hasLines, src []byte) string {
	var b strings.Builder
	segs := n.Lines()
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

func codeKeys(blocks []codeBlock) []string {
	keys := make([]string, len(blocks))
	for i, b := range blocks {
		keys[i] = b.lang + "\x00" + b.body
	}
	return keys
}

// normalizeCode compares code by its commands: comments are prose and may be
// localized, so comment-only lines are dropped and inline comments removed
// before trailing whitespace and blank lines are trimmed.
func normalizeCode(s, lang string) string {
	s = stripComments(s, lang)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func stripComments(s, lang string) string {
	prefixes := commentPrefixes(lang)
	if len(prefixes) == 0 {
		return s
	}
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(ln)
		if hasAnyPrefix(trimmed, prefixes) {
			continue
		}
		out = append(out, stripInlineComment(ln, prefixes))
	}
	return strings.Join(out, "\n")
}

func commentPrefixes(lang string) []string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "sh", "bash", "shell", "zsh", "console", "yaml", "yml",
		"py", "python", "ruby", "rb", "make", "makefile", "dockerfile",
		"toml", "ini", "conf", "properties":
		return []string{"#"}
	case "c", "cpp", "c++", "go", "java", "js", "javascript", "ts",
		"typescript", "rust", "rs":
		return []string{"//"}
	}
	return nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// stripInlineComment cuts a trailing comment introduced by whitespace plus a
// comment prefix, leaving command text intact.
func stripInlineComment(line string, prefixes []string) string {
	cut := -1
	for _, p := range prefixes {
		for i := 0; i < len(line); {
			j := strings.Index(line[i:], p)
			if j < 0 {
				break
			}
			pos := i + j
			if pos > 0 && (line[pos-1] == ' ' || line[pos-1] == '\t') {
				if cut < 0 || pos < cut {
					cut = pos
				}
				break
			}
			i = pos + len(p)
		}
	}
	if cut >= 0 {
		return line[:cut]
	}
	return line
}

func diffReport(kind string, en, zh []string) []string {
	ce := counter(en)
	cz := counter(zh)
	var onlyEn, onlyZh []string
	for k, n := range ce {
		if cz[k] < n {
			onlyEn = append(onlyEn, k)
		}
	}
	for k, n := range cz {
		if ce[k] < n {
			onlyZh = append(onlyZh, k)
		}
	}
	sort.Strings(onlyEn)
	sort.Strings(onlyZh)
	var p []string
	if len(onlyEn) > 0 {
		p = append(p, fmt.Sprintf("%s only in en: %s", kind, strings.Join(onlyEn, ", ")))
	}
	if len(onlyZh) > 0 {
		p = append(p, fmt.Sprintf("%s only in zh: %s", kind, strings.Join(onlyZh, ", ")))
	}
	return p
}

func counter(in []string) map[string]int {
	m := make(map[string]int, len(in))
	for _, s := range in {
		m[s]++
	}
	return m
}

func sameCounter(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, n := range a {
		if b[k] != n {
			return false
		}
	}
	return true
}
