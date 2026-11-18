# 开源发布准备设计

> 日期：2026-11-01
> 状态：已实现
> 范围：`clablings` 与 `clab-exercises` 的 GitHub 展示、CI、发布和文档

## 仓库

- `https://github.com/tianyuxue/clablings`：Go CLI 与引擎，Apache-2.0。
- `https://github.com/tianyuxue/clab-exercises`：实验题库，CC BY-SA 4.0。

## clablings

提供中文 README、Apache-2.0 许可证、安全策略、更新日志、issue/PR 模板和 Linux 安装脚本。
安装脚本支持 amd64/arm64、校验和、`--version`、`--bindir`、`--uninstall` 及
`CLABLINGS_BASE_URL`。CI 包含单元测试/构建、可选特权集成测试和 v* 标签发布，发布资产包括
`clablings_linux_amd64.tar.gz`、`clablings_linux_arm64.tar.gz` 与 `checksums.txt`。

题库是独立仓库；README 使用 `--root` 或 `CLABLINGS_ROOT`，不依赖固定的兄弟目录。

## clab-exercises

提供中文 README、安装与快速开始、环境要求、目录结构、章节表、路线、贡献和许可证说明；
题库内容使用 CC BY-SA 4.0，代码仍遵循 clablings 仓库的 Apache-2.0。题库有内容 issue 模板，
不单独配置 CI。

## 首次发布备注

两个仓库应设置 `containerlab`、`network-lab`、`network-engineering`、`go`、`cli`、`learning`
等主题。只有准备好 self-hosted Linux 集成 runner 后，再创建 `ENABLE_PRIVILEGED_CI` 变量。
首个版本标签为 `v0.1.0`，发布工作流会先校验 `clablings version`。
