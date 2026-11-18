# 只保留中文（移除双语机制）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `clablings` 与 `clab-exercises` 改成纯中文单份内容，删除所有语言选择、双语校验和英文内容/文档。

**Architecture:** 先让 `content` 包只解析普通字符串，再删掉 CLI 的 locale 解析、配置项和 `internal/i18n`，`lint` 改为扫描残留语言后缀文件；随后用一次性迁移程序把 46 个 YAML 和 43 个内容文件转成单中文结构；最后重写/翻译文档。

**Tech Stack:** Go 1.25、YAML（`gopkg.in/yaml.v3`）、Cobra、glamour、Python 3（仅用于迁移后的 YAML 解析校验）。

**Spec:** `clablings/docs/superpowers/specs/2026-11-18-chinese-only-design.md`

## Global Constraints

- 内容只保留中文；不再出现 `concept.<lang>.md`、`README.<lang>.md`、`reading.<lang>.yaml` 或 `{en, zh}` 映射。
- 命令、代码块、链接、节点名、VLAN/AS/接口编号、断言语义与实验行为必须原样保留。
- `clablings` 的 CLI 界面文案（`--help`、报错、进度提示、选择器标签）保持英文，不翻译。
- Go 源码与注释保持英文；YAML 字段键名保持英文（`title`、`brief`、`kind` 等）。
- 交付文档（README、CHANGELOG、SECURITY、内容说明、GitHub 模板）使用通顺中文；历史计划保持原貌，纯英文文档翻译。
- 本次不 bump 版本、不打 tag；不运行需要 root/Docker 的集成测试。
- 每个任务结束时运行该任务列出的验证命令；验证通过才提交。
- 两个仓库各自提交：`clablings` 从 `main` 建 `refactor/chinese-only`；`clab-exercises` 从当前 `feat/l2-switch-code` HEAD 建 `refactor/chinese-only`（保留已有 5 个提交）。

## Review Focus

- **迁移程序误改内容**：`cmd:` 多行块、引号、中文标点、`|` 块必须保持语义不变；Task 3 用“迁移后重新解析 + lint + 人工 diff”钉住。
- **把 `concept.zh.md` 误判为合法文件**：`LegacyLocalizedFiles` 必须命中 `zh`/`en` 等任意语言后缀；Task 2 的测试覆盖三种后缀。
- **只存在 `concept.md` 时 `read` 是否正常**：不得再回退英文；Task 2 的 `read` 测试覆盖。
- **`reading.yaml` 缺失时崩溃**：必须返回 `(nil, nil)`；Task 1 的测试覆盖。
- **旧配置 `language:` 导致启动失败**：必须被忽略；Task 2 的配置测试覆盖。
- **迁移后有内容字段为空**：`title`/`brief` 必须非空；Task 1 的测试覆盖。

---

### Task 1: content 包单语化

**Files:**
- Create: `clablings/internal/content/helpers_test.go`（承载从 `localized_test.go` 移出的 `writeLib`）
- Delete: `clablings/internal/content/localized.go`
- Delete: `clablings/internal/content/localized_test.go`
- Modify: `clablings/internal/content/load.go`
- Modify: `clablings/internal/content/reading.go`
- Modify: `clablings/internal/content/load_test.go`
- Modify: `clablings/internal/content/reading_test.go`
- Modify: `clablings/internal/content/quiz_test.go`
- Modify: `clablings/internal/cli/{debug,list,read,reset,test,watch,lint}.go`（只改 `Load*` 调用）
- Modify: `clablings/internal/cli/read_test.go`（fixture 文件名）

**Interfaces:**
- Produces: `content.Load(root string) (*Library, error)`
- Produces: `content.LoadReading(chapterDir string) ([]Reading, error)`
- Produces DTO 字段：`chapterYAML{Title string; Objectives []string}`、`exerciseYAML{Title, Brief string; Hints []string}`、`optionYAML{Text, Explanation string}`、`questionYAML{Prompt string}`、`assertionYAML{Name, Hint string}`、`readingYAML{Intro, Question string}`。

- [ ] **Step 0: 建工作分支**

```bash
cd clablings
git switch -c refactor/chinese-only
```

- [ ] **Step 1: 写失败测试**

在 `reading_test.go` 中用下面三个测试替换原有两个测试：

```go
func TestLoadReading(t *testing.T) {
	dir := t.TempDir()
	body := "- rfc: 2328\n  title: OSPF Version 2\n  section: \"7.3\"\n  kind: required\n  url: https://www.rfc-editor.org/rfc/rfc2328\n  intro: 数据库如何同步\n  question: 为什么 DBD 里带 MTU？\n"
	if err := os.WriteFile(filepath.Join(dir, "reading.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReading(dir)
	if err != nil {
		t.Fatalf("LoadReading: %v", err)
	}
	if len(got) != 1 || got[0].RFC != 2328 || got[0].Section != "7.3" {
		t.Fatalf("reading = %+v", got)
	}
	if got[0].Intro != "数据库如何同步" || got[0].Question != "为什么 DBD 里带 MTU？" {
		t.Fatalf("text = %+v", got[0])
	}
}

func TestLoadReadingMissing(t *testing.T) {
	got, err := LoadReading(t.TempDir())
	if err != nil || got != nil {
		t.Fatalf("missing reading: %v %v", got, err)
	}
}

func TestLoadReadingIgnoresLegacyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reading.en.yaml"), []byte("- rfc: 1\n  title: EN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReading(dir)
	if err != nil || got != nil {
		t.Fatalf("legacy file must be ignored: %v %v", got, err)
	}
}
```

在 `load_test.go` 增加：

```go
func TestLoadPlainHumanText(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: 交换\nobjectives: [甲, 乙]\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes: {}\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: config-fix\ntitle: 修 VLAN\nbrief: 把 PVID 改对\nhints: [看 bridge vlan show]\nassertions:\n  - name: PVID 正确\n    type: exec\n    node: n1\n    cmd: \"true\"\n    hint: 检查 PVID\n",
	})
	lib, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ch := lib.Chapters[0]
	if ch.Title != "交换" || len(ch.Objectives) != 2 || ch.Objectives[0] != "甲" {
		t.Fatalf("chapter = %+v", ch)
	}
	ex := ch.Exercises[0]
	if ex.Title != "修 VLAN" || ex.Brief != "把 PVID 改对" || ex.Hints[0] != "看 bridge vlan show" {
		t.Fatalf("exercise = %+v", ex)
	}
	if ex.Assertions[0].Name != "PVID 正确" || ex.Assertions[0].Hint != "检查 PVID" {
		t.Fatalf("assertion = %+v", ex.Assertions[0])
	}
}

func TestLoadRejectsLocalizedMap(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":           "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml": "id: c1\ntitle: {en: Switching, zh: 交换}\n",
	})
	if _, err := Load(root); err == nil {
		t.Fatal("localized map must not parse into a plain string")
	}
}

func TestLoadRejectsEmptyTitleAndBrief(t *testing.T) {
	base := func(exercise string) map[string]string {
		return map[string]string{
			"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
			"chapters/c1/chapter.yaml":               "id: c1\ntitle: t\n",
			"chapters/c1/exercises/e1/exercise.yaml": exercise,
		}
	}
	for _, tt := range []struct{ name, body, want string }{
		{"title", "id: e1\ntype: config-fix\nbrief: b\n", "missing title"},
		{"brief", "id: e1\ntype: config-fix\ntitle: t\n", "missing brief"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeLib(t, base(tt.body)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}
```

把 `TestLoadExerciseNilAssertions` 和 `TestLoadWithLocaleMetadata` 从被删除的
`localized_test.go` 移到 `load_test.go`，并改为普通字符串字段、调用 `Load(root)`；
`TestLoadExerciseNilAssertions` 的 exercise.yaml 补上 `title: t` 和 `brief: b`。

在 `quiz_test.go` 中把 `LoadWithLocale(root, "zh")` 改为 `Load(root)`，并把
`prompt/text/explanation` 的 `{en: ..., zh: ...}` 改成中文字符串。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd clablings && go test ./internal/content`
Expected: 编译失败或测试失败（`LoadReading` 参数数量不符、`{en, zh}` 无法解析为字符串）。

- [ ] **Step 3: 实现**

- 先把 `localized_test.go` 里的 `writeLib` 辅助函数原样移到新建的
  `helpers_test.go`（`package content`），再删除 `localized.go`。
- 删除 `localized.go`：移除 `DefaultLocale`、`NormalizeLocale`、`localizedString`、`resolve`、`resolveStrings`、`resolveLocalizedList`。
- `load.go`：
  - `Load(root)` 直接解析；`loadChapter(dir)`、`loadExercise(dir)` 去掉 locale。
  - DTO 字段改为普通 `string` / `[]string`，直接把值拷进运行时结构。
  - 在 `loadExercise` 校验 `strings.TrimSpace(ex.Title) == ""` 返回 `"exercise %s: missing title"`，`Brief` 为空返回 `"exercise %s: missing brief"`。
- `reading.go`：只读 `reading.yaml`；不存在返回 `(nil, nil)`；`readingYAML` 的 `Intro`/`Question` 为 `string`。
- 更新 7 个 CLI 文件的调用：`content.LoadWithLocale(root, loc)` → `content.Load(root)`；`content.LoadReading(ch.Dir, loc)` → `content.LoadReading(ch.Dir)`。本任务先保留 `locale()`/`labelsFor()`，Task 2 再删。
- 把 `read_test.go` 的 fixture 从 `reading.en.yaml` 改为 `reading.yaml`，`intro`/`question` 写成普通字符串。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd clablings && go test ./internal/content ./internal/cli -count=1 && go build ./...`
Expected: PASS，且 `go build` 无错误。

- [ ] **Step 5: 提交**

```bash
cd clablings
git add internal/content internal/cli
git commit -m "refactor(content): parse plain single-language fields"
```

---

### Task 2: 移除语言选择、i18n 与旧翻译校验

**Files:**
- Delete: `clablings/internal/i18n/`（`markdown.go`、`markdown_test.go`）
- Delete: `clablings/internal/content/parity_test.go`
- Modify: `clablings/internal/content/lint.go`
- Modify: `clablings/internal/content/lint_test.go`
- Modify: `clablings/internal/cli/root.go`
- Modify: `clablings/internal/cli/elevate.go`
- Modify: `clablings/internal/cli/read.go`
- Modify: `clablings/internal/cli/lint.go`
- Modify: `clablings/internal/config/config.go`
- Modify: `clablings/internal/config/config_test.go`
- Modify: `clablings/internal/cli/root_test.go`
- Modify: `clablings/internal/cli/elevate_test.go`
- Modify: `clablings/internal/cli/read_test.go`
- Modify: `clablings/internal/cli/lint_test.go`（如有 locale fixture）

**Interfaces:**
- Produces: `content.LegacyLocalizedFiles(root string) ([]string, error)`，返回形如 `chapters/c1/concept.zh.md: legacy localized file; use concept.md` 的问题字符串。
- Removes: `content.MissingTranslations`、`content.TranslationParity`、`cli.locale`、`cli.resolveLocale`、`config.Config.Language`。

- [ ] **Step 1: 写失败测试**

删除 `parity_test.go`。在 `content/lint_test.go` 中用下面测试替换 `TestMissingTranslations`：

```go
func TestLegacyLocalizedFiles(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                              "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":                    "id: c1\ntitle: 交换\n",
		"chapters/c1/concept.md":                      "# 交换\n",
		"chapters/c1/reading.yaml":                    "- title: RFC\n",
		"chapters/c1/exercises/e1/exercise.yaml":      "id: e1\ntype: quiz\ntitle: t\nbrief: b\n",
		"chapters/c1/exercises/e1/README.md":          "# 题\n",
		"chapters/c1/exercises/e2/exercise.yaml":      "id: e2\ntype: quiz\ntitle: t\nbrief: b\n",
		"chapters/c1/exercises/e2/README.en.md":       "# Quiz\n",
		"chapters/c1/exercises/e2/README.zh.md":       "# 题目\n",
		"chapters/c1/concept.en.md":                   "# Switching\n",
		"chapters/c1/concept.zh.md":                   "# 交换\n",
		"chapters/c1/reading.en.yaml":                 "- title: RFC\n",
	})
	problems, err := LegacyLocalizedFiles(root)
	if err != nil {
		t.Fatalf("LegacyLocalizedFiles: %v", err)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"concept.en.md", "concept.zh.md", "README.en.md", "README.zh.md", "reading.en.yaml"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	for _, clean := range []string{"concept.md", "README.md", "reading.yaml"} {
		if strings.Contains(joined, clean+":") {
			t.Fatalf("false positive for %q:\n%s", clean, joined)
		}
	}
}
```

在 `cli/root_test.go` 删除 `TestResolveLocale`，把 `TestRootHasLangFlag` 改为：

```go
func TestRootHasNoLangFlag(t *testing.T) {
	if NewRootCmd().PersistentFlags().Lookup("lang") != nil {
		t.Fatal("--lang must be removed")
	}
}
```

在 `config_test.go` 删除 `TestLoadFromLanguage`，新增：

```go
func TestLoadFromIgnoresRemovedLanguageKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("language: zh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Build.DefaultImage != defaultBuildImage {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}
```

在 `read_test.go`：fixture 改为 `concept.md`、`reading.yaml`，YAML 用普通字符串；
删除 `TestReadFallsBackToEnglishConcept` 与 `t.Setenv("CLABLINGS_LANG", ...)`；
`TestReadShowsLocalizedConceptAndReading` 增加断言 `必读` 与 `## 目标`。

在 `elevate_test.go`：`TestSudoArgv` 的输入与期望中删除 `CLABLINGS_LANG`。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd clablings && go test ./internal/content ./internal/cli ./internal/config`
Expected: FAIL（`LegacyLocalizedFiles` 未定义、`--lang` 仍存在、`Language` 字段仍存在）。

- [ ] **Step 3: 实现**

- `root.go`：删除 `langFlag`、`defaultLanguage`、`locale`、`resolveLocale` 和
  `PersistentFlags().StringVar(... "lang" ...)`；所有命令改为 `content.Load(resolved)`。
- `elevate.go`：`sudoEnvVars` 去掉 `CLABLINGS_LANG`；删除 `effectiveLocale` 和
  `env["CLABLINGS_LANG"]` 赋值；删除不再使用的 `config` import。
- `config.go`：删除 `Language` 字段。
- `read.go`：删除 `readLabels`/`labelsFor`，在 `printChapterRead` 内使用
  `目标 / 阅读 / 导读 / 问题`；`readConceptFile(dir)` 只读 `concept.md`；
  新增 `readingKindLabel(kind string) string`：`required` → `必读`，
  `optional` → `选读`，其他原样返回，并在输出中使用它。
- `content/lint.go`：删除 `MissingTranslations`、`TranslationParity`、
  `missingText`、`hasLocaleList`、`readIfExists`、`has`、`i18n` import；新增
  `LegacyLocalizedFiles`：仅遍历清单里的章节目录（`filepath.WalkDir`），当文件名
  匹配 `concept.<后缀>.md`、`README.<后缀>.md`、`reading.<后缀>.yaml`
  （且后缀非空）时追加
  `fmt.Sprintf("%s: legacy localized file; use %s", rel, base)`；结果排序。
- `cli/lint.go`：删除 `MissingTranslations` / `TranslationParity` 调用与 `"zh"`，
  把 `LegacyLocalizedFiles` 的结果并入 `problems`；`lintLibrary` 用 `content.Load`。
- 删除 `internal/i18n/` 与 `parity_test.go`。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd clablings && gofmt -l . && go vet ./... && go test ./... -count=1`
Expected: `gofmt -l .` 无输出；vet 与测试全部通过。

- [ ] **Step 5: 提交**

```bash
cd clablings
git add -A
git commit -m "refactor: remove content language selection and i18n parity"
```

---

### Task 3: 迁移 `clab-exercises` 内容为纯中文结构

**Files:**
- Create & delete: `clablings/internal/tmpmigrate/main.go`（一次性迁移程序，任务结束前删除）
- Modify: `clab-exercises/chapters/*/chapter.yaml`（9 个）
- Modify: `clab-exercises/chapters/*/exercises/*/exercise.yaml`（28 个）
- Rename: `concept.zh.md` → `concept.md`（9 个）；删除 `concept.en.md`（9 个）
- Rename: `README.zh.md` → `README.md`（17 个）；删除 `README.en.md`（17 个）
- Rename+modify: `reading.en.yaml` → `reading.yaml`（9 个，展开 `intro`/`question`）

**Interfaces:**
- Consumes: Task 1 的 `content.Load`、`content.LoadReading`；Task 2 的
  `content.LegacyLocalizedFiles`。

- [ ] **Step 0: 建内容工作分支**

```bash
cd clab-exercises
git switch -c refactor/chinese-only
```

该分支从当前 `feat/l2-switch-code` HEAD 出发，保留已有 5 个提交。

- [ ] **Step 1: 写迁移程序**

创建 `clablings/internal/tmpmigrate/main.go`，行为固定如下：

1. 参数为题库根目录；遍历 `chapters/*/chapter.yaml`、
   `chapters/*/exercises/*/exercise.yaml`、`chapters/*/reading.en.yaml`。
2. 用 `yaml.Node` 解析；后序遍历每个节点：若映射节点含有键 `zh`，把该映射节点
   原地替换为 `zh` 的值节点（`*n = *zhValue`），保留其标量风格。
3. 用 `yaml.NewEncoder` + `SetIndent(2)` 写回；写回前先 `yaml.Unmarshal` 一次
   输出，确认仍是合法 YAML，否则退出非零。
4. `reading.en.yaml` 写回为同级 `reading.yaml`，随后删除 `reading.en.yaml`。
5. 对每个章节：`concept.zh.md` 改名为 `concept.md`、删除 `concept.en.md`；
   每个练习目录：`README.zh.md` 改名为 `README.md`、删除 `README.en.md`
   （文件不存在时跳过）。
6. 任一章节/练习含 `en` 或 `zh` 语言映射却未被替换时（例如只有 `en`），退出非零。

- [ ] **Step 2: 运行迁移并检查**

```bash
cd clablings
go run ./internal/tmpmigrate ../clab-exercises
cd ../clab-exercises
git status --short
grep -rn "^\s*zh:\|^\s*en:" chapters --include=*.yaml || true
find chapters -name 'concept.*.md' -o -name 'README.*.md' -o -name 'reading.*.yaml'
```

Expected: `zh:`/`en:` 无输出；`find` 无输出；`git status` 显示 9 个 concept
改名、17 个 README 改名、9 个 reading 改名、28+9 个 YAML 修改。

- [ ] **Step 3: 校验所有 YAML 可解析**

```bash
cd clab-exercises
python3 - <<'PY'
import glob, yaml, sys
bad = 0
for p in glob.glob("chapters/**/*.yaml", recursive=True):
    try:
        yaml.safe_load(open(p, encoding="utf-8"))
    except Exception as e:
        bad += 1
        print(p, e)
sys.exit(1 if bad else 0)
PY
```

Expected: 退出码 0，无输出。

- [ ] **Step 4: 检查 diff 只做结构变化**

```bash
cd clab-exercises
git diff --stat
git diff -M -- chapters/01-vlan-trunk/chapter.yaml chapters/01-vlan-trunk/reading.yaml \
  chapters/01-vlan-trunk/exercises/05_l2_switch/exercise.yaml
```

Expected: `cmd: |` / `start: |` 块内容保持不变；中文值不变；只删除 `en` 行、
去掉 `zh:` 层级和语言映射。抽查上述三个文件，确认命令、编号、链接未变。

- [ ] **Step 5: 删除迁移程序**

```bash
cd clablings
rm -rf internal/tmpmigrate
```

- [ ] **Step 6: 运行 lint 与冒烟验证**

```bash
cd clablings
go run ./cmd/clablings lint --root ../clab-exercises
go run ./cmd/clablings list --root ../clab-exercises | head -20
go run ./cmd/clablings read 01-vlan-trunk --root ../clab-exercises | head -40
```

Expected: `lint` 输出 `ok: 0 warning(s)` 且退出码 0；`list` 显示中文标题；
`read` 显示 `## 目标`、`## 阅读`、`必读`，不出现英文标签。

- [ ] **Step 7: 提交**

```bash
cd ../clab-exercises
git add -A
git commit -m "content: migrate exercises to single Chinese structure"
```

---

### Task 4: `clablings` 文档中文化

**Files:**
- Modify: `clablings/README.md`
- Delete: `clablings/README.zh-CN.md`
- Modify: `clablings/CHANGELOG.md`
- Modify: `clablings/SECURITY.md`
- Modify: `clablings/docs/superpowers/specs/2026-09-19-clablings-content-style-guide.md`
- Modify: `clablings/docs/superpowers/specs/2026-09-19-clablings-content-design.md`
- Modify: `clablings/docs/superpowers/specs/2026-09-17-clablings-design.md`
- Modify: `clablings/docs/superpowers/specs/2026-09-20-clablings-i18n-parity-design.md`
- Modify: `clablings/docs/superpowers/specs/2026-10-31-default-language-zh-design.md`
- Modify: `clablings/docs/superpowers/specs/2026-10-31-quiz-shortcuts-design.md`
- Modify: `clablings/docs/superpowers/specs/2026-11-01-open-source-release-design.md`
- Modify: `clablings/.github/PULL_REQUEST_TEMPLATE.md`
- Modify: `clablings/.github/ISSUE_TEMPLATE/{bug_report,feature_request,config}.yml`

- [ ] **Step 1: 重写 README 并删除中文副本**

以 `README.zh-CN.md` 为基础写 `README.md`：保留安装、构建、命令、配置、测试、
架构说明；删除 `--lang`、`CLABLINGS_LANG`、`config.language` 和双语内容示例；
内容结构改为 `concept.md` / `reading.yaml` / `README.md`。确认无 `README.zh-CN.md`
引用后删除该文件。

- [ ] **Step 2: 翻译 CHANGELOG 与 SECURITY**

`CHANGELOG.md`：逐条翻译历史版本，保留版本号、日期和 `Added/Changed/Fixed/Removed`
的分组语义（可译为“新增/变更/修复/移除”），在顶部新增：

```markdown
## [Unreleased]

### 变更
- 题库内容改为纯中文单份文件：`concept.md`、`reading.yaml`、`README.md`。
- `chapter.yaml` / `exercise.yaml` 的人类文案改为普通字符串，不再支持 `{en, zh}` 映射。

### 移除
- 移除 `--lang`、`CLABLINGS_LANG` 和 `config.language`。
- 移除中英内容一致性校验（`internal/i18n`）。
- 删除英文内容文件与英文摘要。
```

版本号与 tag 不变。`SECURITY.md` 全文译为中文，链接保持原样。

- [ ] **Step 3: 处理设计文档**

- 在 `2026-09-19-clablings-content-design.md` 与 `2026-09-17-clablings-design.md`
  顶部加：

  ```markdown
  > **已被取代（2026-11-18）**：本文档中关于双语内容、`--lang`、`CLABLINGS_LANG`、
  > `config.language`、`{en, zh}` 映射和语言回退的约定，已由
  > `2026-11-18-chinese-only-design.md` 取代。其余设计仍然有效。
  ```

- `2026-09-20-clablings-i18n-parity-design.md` 与
  `2026-10-31-default-language-zh-design.md`：翻译为中文，顶部加
  `> **已废弃（2026-11-18）**：本文描述的语言机制已被完全移除，仅作历史记录保留。`
- `2026-10-31-quiz-shortcuts-design.md`、`2026-11-01-open-source-release-design.md`
  翻译为中文，技术结论、命令、版本号不变。
- `2026-09-19-clablings-content-style-guide.md`：删除英文摘要规范和中英对齐规则，
  保留中文语气、术语、章节骨架、命令与链接一致性要求。

- [ ] **Step 4: 翻译 GitHub 模板**

翻译 4 个模板的 `name`、`description`、`label`、`description`、`placeholder`、
`value` 等人类可读文字；`id`、`type`、`labels`、`validations` 等机器键名保持不变。

- [ ] **Step 5: 验证并提交**

```bash
cd clablings
test ! -e README.zh-CN.md
grep -rn -- "--lang\|CLABLINGS_LANG\|config.language" README.md CHANGELOG.md SECURITY.md .github || true
grep -rln "concept\.en\.md\|README\.en\.md\|reading\.en\.yaml\|英文摘要" README.md CHANGELOG.md SECURITY.md .github
```

Expected: 前两条无输出；第三条无输出。然后：

```bash
git add -A
git commit -m "docs: rewrite clablings docs in Chinese"
```

---

### Task 5: `clab-exercises` 文档与工作区说明中文化

**Files:**
- Modify: `clab-exercises/README.md`
- Modify: `clab-exercises/docs/superpowers/specs/2026-10-31-chinese-first-content-design.md`
- Modify: `clab-exercises/docs/superpowers/plans/2026-10-31-chinese-first-content.md`
- Modify: `clab-exercises/.github/ISSUE_TEMPLATE/{config,content_issue}.yml`
- Modify: `/home/jty/codes/clab/AGENTS.md`

- [ ] **Step 1: 更新题库 README**

删除双语说明；目录树改为 `concept.md`、`reading.yaml`、`README.md`；删除
“中文主稿，英文是摘要”和“翻译问题”；章节表、路线、贡献、许可证内容保持中文。

- [ ] **Step 2: 标注并翻译历史文档**

在 `2026-10-31-chinese-first-content-design.md` 顶部加：

```markdown
> **已被取代（2026-11-18）**：本文档中“英文保留为摘要”的约定已由
> `clablings/docs/superpowers/specs/2026-11-18-chinese-only-design.md` 取代；
> 内容已改为纯中文。其余写作规范仍然有效。
```

把 `2026-10-31-chinese-first-content.md` 全文翻译为中文，保留任务清单与代码路径，
顶部注明这是历史执行记录。其余中英混排的计划与评审不改。

- [ ] **Step 3: 翻译 Issue 模板**

翻译两个模板的人类可读文字，保留 `name`/`description` 的 YAML 键与 `labels`。

- [ ] **Step 4: 更新工作区 AGENTS.md**

把 `clablings` 与 `clab-exercises` 两节中关于双语的内容改为单中文结构：

- `concept.en.md` + `concept.zh.md` → `concept.md`
- `reading.<lang>.yaml` → `reading.yaml`
- `README.<lang>.md` → `README.md`
- 删除 `{en, zh}` 映射、`CLABLINGS_LANG`、`--lang`、缺失中文翻译检查、
  “elevated runs forward content language”等描述
- `clablings lint` 说明改为：检查必需文件、残留语言后缀文件、节点引用

构建、测试、安装、拓扑和 `clab-tui` 的说明保持不变。

- [ ] **Step 5: 验证并提交**

```bash
cd clab-exercises
grep -rn "concept\.en\.md\|concept\.zh\.md\|README\.en\.md\|README\.zh\.md\|reading\.en\.yaml\|英文摘要\|翻译问题" README.md .github || true
grep -rn -- "--lang\|CLABLINGS_LANG\|{en, zh}" README.md .github || true
git add -A
git commit -m "docs: rewrite exercise library docs in Chinese"
```

Expected: 两条 `grep` 均无输出。

---

### Task 6: 最终验证

**Files:** 无新增；只读验证。

- [ ] **Step 1: 代码检查**

```bash
cd clablings
gofmt -l .
go vet ./...
go test ./... -count=1
```

Expected: `gofmt -l .` 无输出；vet 与测试全部通过。

- [ ] **Step 2: 内容与 lint**

```bash
cd clablings
go run ./cmd/clablings lint --root ../clab-exercises
cd ../clab-exercises
find chapters -name '*.en.md' -o -name '*.zh.md' -o -name 'reading.*.yaml'
grep -rn "^\s*en:\|^\s*zh:" chapters --include=*.yaml || true
```

Expected: `lint` 为 `ok: 0 warning(s)`；`find` 与 `grep` 无输出。

- [ ] **Step 3: 运行态冒烟（不部署容器）**

```bash
cat > /tmp/opencode/answers.yaml <<'YAML'
01-vlan-trunk/00_why_vlan:
  - [0]
  - [0, 1, 3]
YAML
cd clablings
go run ./cmd/clablings test 01-vlan-trunk/00_why_vlan --answers /tmp/opencode/answers.yaml --root ../clab-exercises
go run ./cmd/clablings read 02-stp --root ../clab-exercises | head -30
```

Expected: quiz 退出码 0；`read` 输出中文标签和内容。

- [ ] **Step 4: 确认工作区状态**

```bash
cd /home/jty/codes/clab/clablings && git status --short && git log --oneline -3
cd /home/jty/codes/clab/clab-exercises && git status --short && git log --oneline -3
```

Expected: 两个仓库工作区干净；`clablings` 新增 3 个提交，`clab-exercises` 新增 2 个提交。
