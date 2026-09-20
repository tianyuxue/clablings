package i18n

import "testing"

func TestMarkdownParityClean(t *testing.T) {
	en := "# Title\n\n## Terms\n\nUse `bridge vlan show` and read [RFC 7348](https://example.com/rfc).\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "# 标题\n\n## 术语\n\n用 `bridge vlan show`，阅读 [RFC 7348](https://example.com/rfc)。\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	if p := MarkdownParity(en, zh); len(p) != 0 {
		t.Fatalf("problems = %v, want none", p)
	}
}

func TestMarkdownParityHeadingCount(t *testing.T) {
	en := "# T\n\n## A\n\n## B\n"
	zh := "# 标题\n\n## A\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a heading count problem")
	}
}

func TestMarkdownParityHeadingLevel(t *testing.T) {
	en := "# T\n\n## A\n\n### B\n"
	zh := "# 标题\n\n## A\n\n## B\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a heading level problem")
	}
}

func TestMarkdownParityCodeBody(t *testing.T) {
	en := "## C\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "## C\n\n```sh\nbridge vlan add dev eth1 vid 20\n```\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a code body problem")
	}
}

func TestMarkdownParityCodeLanguage(t *testing.T) {
	en := "## C\n\n```sh\nrun\n```\n"
	zh := "## C\n\n```bash\nrun\n```\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a code language problem")
	}
}

func TestMarkdownParityInlineCode(t *testing.T) {
	en := "Use `bridge vlan show` and `802.1Q`.\n"
	zh := "用 `bridge vlan show`。\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected an inline code problem")
	}
}

func TestMarkdownParityLinks(t *testing.T) {
	en := "See [RFC 7348](https://example.com/rfc7348).\n"
	zh := "见 [RFC 7348](https://example.com/other)。\n"
	p := MarkdownParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a link problem")
	}
}

func TestMarkdownParityAllowsLocalizedCodeComments(t *testing.T) {
	en := "## C\n\n```sh\n# access ports\nbridge vlan add dev eth1 vid 10\nbridge vlan show   # inspect\n```\n"
	zh := "## C\n\n```sh\n# access 端口\nbridge vlan add dev eth1 vid 10\nbridge vlan show   # 查看\n```\n"
	if p := MarkdownParity(en, zh); len(p) != 0 {
		t.Fatalf("localized comments must not report drift: %v", p)
	}
}

func TestMarkdownParityStillChecksCommandsWithComments(t *testing.T) {
	en := "## C\n\n```sh\n# note\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "## C\n\n```sh\n# 注释\nbridge vlan add dev eth1 vid 20\n```\n"
	if p := MarkdownParity(en, zh); len(p) == 0 {
		t.Fatal("command drift must still be reported")
	}
}
