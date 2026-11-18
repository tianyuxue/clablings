# 更新日志

本文件记录项目的重要变更，格式遵循
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，版本号遵循
[语义化版本](https://semver.org/spec/v2.0.0.html)。

## [未发布]

### 变更

- 题库内容改为纯中文单份文件：`concept.md`、`reading.yaml`、`README.md`。
- `chapter.yaml` / `exercise.yaml` 的人类文案改为普通字符串，不再支持 `{en, zh}` 映射。

### 移除

- 移除 `--lang`、`CLABLINGS_LANG` 和 `config.language`。
- 移除中英内容一致性校验（`internal/i18n`）。
- 删除英文内容文件与英文摘要。

## [0.2.1] - 2026-11-15

### 修复

- `clablings watch` 显示提示后会重新打印快捷键图例；较长的提示不会再把图例推出屏幕。
  查看概念后仍会重新打印完整的练习上下文。

## [0.2.0] - 2026-11-11

### 新增

- 全局 `--sudo` 参数：通过绝对路径重新以 root 身份运行 clablings，并保留
  `CLABLINGS_ROOT` 和 `CLABLINGS_INSTANCE`。
- `make install-system` 安装目标：安装到 `/usr/local/bin`，无需以 root 执行构建。

### 变更

- `clablings debug` 会从当前目录（包括练习子目录）所在章节，或
  `.clablings_state.json` 中的当前练习解析章节。
- `clablings watch` 显示概念后会重新打印完整的练习上下文，保留目标和快捷键图例。

### 修复

- 跳转失败和选择当前练习现在会显示明确的可操作提示；containerlab 权限错误会建议
  使用 `clablings watch --sudo`。
- 提权运行会在原子重命名前把 `.clablings_state.json` 交还给调用用户。
- `clablings --sudo debug` 提权 clablings 本身；`debug --sudo` 保留其启动 clab-tui 的含义。
- `make install-system` 拒绝在 sudo 环境下运行，避免产生 root 所有的构建产物。

## [0.1.0] - 2026-11-01

### 新增

- 交互式 `watch` 模式：显示练习标题、简介、提示、概念/RFC 阅读、进度以及快速/硬重置。
- 命令集：`watch`、`run`、`reset`、`list`、`test`、`debug`、`read`、`lint`、`version`。
- 按题库实例隔离的 containerlab 生命周期管理。
- 声明式 `exec`、`file`、`interface`、`link` 断言，支持可选的 `wait` 轮询。
- 交互式 quiz，支持单选、多选、解释和重试。
- 面向 CI 的无头验证，支持文本或 JSON 输出。
- 支持构建和运行 C 代码题。
- 中文单一内容结构。
- `clablings debug` 与 clab-tui 集成。
- GitHub 发布自动化，支持 linux/amd64、linux/arm64、校验和及安装脚本。
