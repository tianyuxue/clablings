package i18n

import "testing"

func TestTechnicalParityClean(t *testing.T) {
	en := "# Title\n\n## Terms\n\nUse `bridge vlan show` and read [RFC 7348](https://example.com/rfc).\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "# 标题\n\n## 术语\n\n用 `bridge vlan add`，阅读 [RFC 7348](https://example.com/rfc)。\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	if p := TechnicalParity(en, zh); len(p) != 0 {
		t.Fatalf("problems = %v, want none", p)
	}
}

func TestTechnicalParityAllowsDifferentStructure(t *testing.T) {
	en := "# Title\n\n## A\n\nShort prose with `one` term.\n\n## B\n\nMore prose.\n\n```sh\nbridge vlan show\n```\n"
	zh := "# 标题\n\n## 术语\n\n更长的中文说明，可以拆成多个段落，并使用不同的行内代码。\n\n- 列表项也可以不同。\n\n```sh\nbridge vlan show\n```\n"
	if p := TechnicalParity(en, zh); len(p) != 0 {
		t.Fatalf("structure and inline code may differ: %v", p)
	}
}

func TestTechnicalParityCodeBody(t *testing.T) {
	en := "## C\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "## C\n\n```sh\nbridge vlan add dev eth1 vid 20\n```\n"
	p := TechnicalParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a code body problem")
	}
}

func TestTechnicalParityCodeLanguage(t *testing.T) {
	en := "## C\n\n```sh\nrun\n```\n"
	zh := "## C\n\n```bash\nrun\n```\n"
	p := TechnicalParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a code language problem")
	}
}

func TestTechnicalParityCodeOrderIndependent(t *testing.T) {
	en := "```sh\nfirst\n```\n\n```sh\nsecond\n```\n"
	zh := "```sh\nsecond\n```\n\n```sh\nfirst\n```\n"
	if p := TechnicalParity(en, zh); len(p) != 0 {
		t.Fatalf("code block order may differ: %v", p)
	}
}

func TestTechnicalParityLinks(t *testing.T) {
	en := "See [RFC 7348](https://example.com/rfc7348).\n"
	zh := "见 [RFC 7348](https://example.com/other)。\n"
	p := TechnicalParity(en, zh)
	if len(p) == 0 {
		t.Fatal("expected a link problem")
	}
}

func TestTechnicalParityAllowsLocalizedCodeComments(t *testing.T) {
	en := "## C\n\n```sh\n# access ports\nbridge vlan add dev eth1 vid 10\nbridge vlan show   # inspect\n```\n"
	zh := "## C\n\n```sh\n# access 端口\nbridge vlan add dev eth1 vid 10\nbridge vlan show   # 查看\n```\n"
	if p := TechnicalParity(en, zh); len(p) != 0 {
		t.Fatalf("localized comments must not report drift: %v", p)
	}
}

func TestTechnicalParityStillChecksCommandsWithComments(t *testing.T) {
	en := "## C\n\n```sh\n# note\nbridge vlan add dev eth1 vid 10\n```\n"
	zh := "## C\n\n```sh\n# 注释\nbridge vlan add dev eth1 vid 20\n```\n"
	if p := TechnicalParity(en, zh); len(p) == 0 {
		t.Fatal("command drift must still be reported")
	}
}
