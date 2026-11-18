# 只保留中文：移除双语机制设计

> 日期：2026-11-18
> 状态：待确认
> 范围：`clablings`（代码、文档）与 `clab-exercises`（内容、文档）

## 背景

`clab-exercises` 目前是“中文主稿 + 英文摘要”的双语结构，`clablings` 提供
`--lang`、`CLABLINGS_LANG`、`config.language` 三级语言选择，并在 `internal/i18n`
里做中英技术一致性校验。此前的
`clab-exercises/docs/superpowers/specs/2026-10-31-chinese-first-content-design.md`
和 `clablings/docs/superpowers/specs/2026-10-31-default-language-zh-design.md`
都明确保留了英文摘要与英文 CLI 文案。

用户现在决定：两个项目只做中文，不再维护英文内容，代码里不再有语言切换，
已有文档用正常中文重写，已有英文文档去掉。CLI 自身的界面文案（`--help`、报错、
进度提示、选择器标签）保持现状，不做翻译；Go 源码和注释保持英文。

本设计取代上述两份文档中关于“保留英文摘要 / 保留语言选择”的部分。

## 目标

1. 内容文件只有一份中文，不再有语言后缀和 `{en, zh}` 映射。
2. `clablings` 不再有任何内容语言选择：没有 `--lang`、没有 `CLABLINGS_LANG`、
   没有 `config.language`、没有 locale 归一化和回退。
3. `clablings lint` 不再检查翻译缺失或中英对齐，改为检查“只允许中文结构”与
   残留的旧双语文件。
4. 两个仓库的交付文档全部为通顺中文；纯英文文档翻译为中文。
5. 实验事实不变：命令、代码块、链接、节点名、VLAN/AS/接口编号、断言逻辑
   全部保持原样。

## 非目标

- 不改 `topology.clab.yml`、节点脚本、`Makefile`、断言语义、实验流程。
- 不翻译 `clablings` 的 CLI 界面文案、Go 源码注释、测试里的技术字符串。
- 不重写 `docs/superpowers/` 下中英混排的历史计划与评审（历史记录保留原貌）；
  只翻译其中纯英文的文档，并给已失效的设计加上标注。
- 不改 `clab-tui`。
- 不删除 `README.en.md` 之外的任何实验资产（脚本、参考实现、拓扑）。

## 反目标

- 迁移后仍出现 `concept.zh.md`、`reading.en.yaml`、`README.zh.md` 之类后缀。
- 保留 `DefaultLocale`、`localizedString`、locale 参数等空壳抽象。
- 内容迁移时顺手改写命令、编号或断言，导致实验结果变化。

## 目标结构

### 内容文件

```text
chapters/<chapter-id>/
  chapter.yaml          # title 为字符串，objectives 为字符串列表
  concept.md            # 中文概念正文
  reading.yaml          # 阅读切片，intro/question 为字符串
  topology.clab.yml
  exercises/<exercise-id>/
    exercise.yaml       # title/brief/hints/断言/quiz 文案均为普通字符串
    README.md           # 非 quiz 题的实验说明
```

`README` 仍是可选文件（11 道 quiz 题没有 README）；`concept.md` 与
`reading.yaml` 仍是可选文件，保持现有行为。

### schema 变化

`chapter.yaml`：

```yaml
title: VLAN 与 Trunk
objectives:
  - 搞清楚交换机为什么非得知道每一帧是哪个 VLAN 的
```

`exercise.yaml`：

```yaml
title: 为什么需要 VLAN？
brief: 先回答两个问题再往下走……
hints:
  - 想想看：交换机收到一个目的 MAC 不认识的帧，会怎么处理？
questions:
  - prompt: VLAN 主要隔离的是什么？
    multiple: false
    options:
      - text: 广播域
        correct: true
        explanation: VLAN 管的是广播和未知单播能泛洪到多远。
assertions:
  - name: sw1 的 PVID 已修改
    hint: 用 `bridge vlan show` 查看
```

`reading.yaml`：

```yaml
- title: Linux bridge — Introduction
  section: "Introduction"
  kind: required
  url: https://docs.kernel.org/networking/bridge.html#introduction
  intro: 什么是网桥、它如何把网段并成一个逻辑以太网……
  question: 网桥连接的是什么？它在哪一层转发？
```

字段键名保持英文（`title`、`objectives`、`kind` 等），因为它们是机器接口；
只有人类可读的值改为中文。

## 代码改动

### content 包

- 删除 `internal/content/localized.go`：`DefaultLocale`、`NormalizeLocale`、
  `localizedString`、`resolve`、`resolveStrings`、`resolveLocalizedList` 全部移除。
- `chapterYAML.Title` 改为 `string`，`Objectives` 改为 `[]string`。
- `exerciseYAML` 的 `Title`、`Brief`、`Hints`、断言 `Name`/`Hint`、quiz
  `Prompt`/`Text`/`Explanation` 全部改为普通 `string` / `[]string`。
- `readingYAML.Intro`、`Question` 改为 `string`。
- `Load(root)` 成为唯一入口，删除 `LoadWithLocale`；`loadChapter`、`loadExercise`
  去掉 locale 参数。
- `LoadReading(chapterDir)` 只读 `reading.yaml`，文件不存在返回 `(nil, nil)`。
- 保留现有校验：必填 `id`、题型合法、quiz 约束、代码题 build/run。
- 新增轻量校验：`title`、`brief` 去掉首尾空白后不得为空（quiz 的 prompt /
  option text 已有校验）。

### 删除 i18n

- 删除整个 `internal/i18n`（`markdown.go` 及其测试）。它的唯一用途是中英技术对齐。
- 删除 `content.MissingTranslations`、`content.TranslationParity`、
  `hasLocaleList`、`readIfExists` 及相关测试（`parity_test.go`、
  `localized_test.go`）。

### CLI

- `internal/cli/root.go`：删除 `langFlag`、`defaultLanguage`、`locale`、
  `resolveLocale` 和 `--lang` 注册；各命令直接调用 `content.Load(resolved)`。
- `internal/cli/elevate.go`：删除 `effectiveLocale` 与 `CLABLINGS_LANG` 转发；
  `CLABLINGS_ROOT`、`CLABLINGS_INSTANCE` 的转发保持不变。
- `internal/config/config.go`：删除 `Language` 字段及对应测试。旧配置里的
  `language:` 键会被 YAML 忽略，不需要迁移提示。
- `internal/cli/read.go`：删除 `labelsFor` 的语言分支，固定使用
  `目标 / 阅读 / 导读 / 问题`；`readConceptFile` 只读 `concept.md`（不再回退
  `concept.en.md`）；`kind` 显示为 `必读` / `选读`，未知值原样显示。
- `debug`、`list`、`read`、`reset`、`test`、`watch` 全部改为 `content.Load`。

### lint

- `lintLibrary` 调用 `content.Load(root)`。
- 删除 `MissingTranslations` / `TranslationParity` 调用与 `"zh"` 字面量。
- 新增 `LegacyLocalizedFiles(root)`：扫描清单中的章节目录，发现
  `concept.<lang>.md`、`README.<lang>.md`、`reading.<lang>.yaml` 时报告错误，
  提示“只支持无语言后缀的 concept.md / README.md / reading.yaml”。
- 保留现有拓扑节点、断言、文件、reload/run 引用检查。

## 文档改动

### `clablings`

- `README.md`：以现有中文 `README.zh-CN.md` 为基础重写，删除 `--lang`、
  `CLABLINGS_LANG`、双语内容说明；然后删除 `README.zh-CN.md`。
- `CHANGELOG.md`：全部历史条目翻译为中文，保留 Keep a Changelog 的版本结构；
  新增 `[Unreleased]` 条目，记录本次不兼容变更。本次不 bump 版本、不打 tag。
- `SECURITY.md`：改为中文。
- `.github/PULL_REQUEST_TEMPLATE.md`、
  `.github/ISSUE_TEMPLATE/{bug_report,feature_request,config}.yml`：界面文字翻译
  为中文，保留 GitHub 表单字段与标签的机器键名。工作流的 `name:` 等技术字符串
  不翻译。
- `docs/superpowers/specs/2026-09-19-clablings-content-style-guide.md`：删除英文
  摘要规范、中英对齐规则，保留并整理中文写作规范（术语、语气、章节骨架、
  命令一致性）。
- `docs/superpowers/specs/2026-09-19-clablings-content-design.md` 与
  `docs/superpowers/specs/2026-09-17-clablings-design.md`：在顶部加“已被取代”
  标注，说明双语、`--lang`、`{en, zh}` 相关约定不再生效；正文作为历史记录
  保留，不逐段改写。
- `docs/superpowers/specs/2026-09-20-clablings-i18n-parity-design.md`：翻译为中文，
  顶部标注“已被 2026-11-18 设计取代”。
- `docs/superpowers/specs/2026-10-31-default-language-zh-design.md`：翻译为中文，
  顶部标注“已被 2026-11-18 设计取代（语言选择整体移除）”。
- `docs/superpowers/specs/2026-10-31-quiz-shortcuts-design.md`、
  `docs/superpowers/specs/2026-11-01-open-source-release-design.md`：翻译为中文，
  技术结论不变。
- 本设计文档即新增的中文设计记录。

### `clab-exercises`

- `README.md`：删除 `concept.en.md` / `README.en.md` / `reading.en.yaml` 和
  “中文主稿、英文摘要”的说明，改为单一中文结构；其余章节、路线、贡献、
  许可证内容保持中文表述。
- `docs/superpowers/specs/2026-10-31-chinese-first-content-design.md`：顶部标注
  已被本设计取代；其“保留英文摘要”的部分不再生效。
- `docs/superpowers/plans/2026-10-31-chinese-first-content.md`：纯英文计划，
  翻译为中文并标注为历史记录。
- `.github/ISSUE_TEMPLATE/{config,content_issue}.yml`：界面文字翻译为中文，
  保留机器键名。
- 其余中英混排的历史计划与评审保持原貌。

### 工作区

- `/home/jty/codes/clab/AGENTS.md` 中关于中文优先、`--lang`、
  `CLABLINGS_LANG`、`reading.<lang>.yaml`、`README.<lang>.md`、
  “缺失中文翻译”的说明，改为单一中文结构；构建与验证命令不变。

## 迁移顺序

1. **clablings 代码**：content 模型去 locale、删除 i18n、CLI 与 config 去语言、
   lint 调整；同步修改或删除测试。
2. **clab-exercises 内容**：9 章 × (`concept.zh.md` → `concept.md`、
   `reading.en.yaml` → `reading.yaml` 并展开 `intro`/`question`)、17 个
   `README.zh.md` → `README.md`、28 个 `exercise.yaml` 与 9 个 `chapter.yaml`
   展开 `{en, zh}`；删除 9 个 `concept.en.md`、17 个 `README.en.md`。
3. **文档**：按上面的清单重写、翻译、删除。
4. **lint 与测试修正**：内容迁移后运行 lint，修正被新校验发现的问题。

## 验证

```bash
cd clablings
gofmt -l .            # 应为空
go vet ./...
go test ./... -count=1
go run ./cmd/clablings lint --root ../clab-exercises

go run ./cmd/clablings list --root ../clab-exercises
go run ./cmd/clablings read 01-vlan-trunk --root ../clab-exercises
```

- `lint` 必须 0 problem；`LegacyLocalizedFiles` 在迁移后不得命中任何文件。
- 全库搜索确认没有 `concept.en.md`、`README.en.md`、`reading.en.yaml`、
  `{en:` / `zh:` 内容字段、`CLABLINGS_LANG`、`--lang` 文档残留。
- 抽查 `git diff -M`：`concept.zh.md → concept.md`、`README.zh.md → README.md`
  只发生文件名与 YAML 结构变化，代码块、命令、链接、编号不改。
- 抽一道 quiz 用 `clablings test --answers` 跑通加载与判分（不部署容器）。
- 不要求 root/Docker：本次不改变实验逻辑，特权集成测试不在验证范围内。

## 留给实现者的决定

- 文档翻译的具体措辞与段落顺序。
- `LegacyLocalizedFiles` 的扫描实现细节（只扫清单内章节，避免误报
  `docs/` 或生成目录）。
- 是否顺带清理 `clab-exercises` 里已被 gitignore 的派生文件；默认不动。
