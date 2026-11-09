# Clablings

[![test](https://github.com/tianyuxue/clablings/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tianyuxue/clablings/actions/workflows/test.yml)
[![Integration](https://github.com/tianyuxue/clablings/actions/workflows/integration.yml/badge.svg?branch=main)](https://github.com/tianyuxue/clablings/actions/workflows/integration.yml)
[![Release](https://github.com/tianyuxue/clablings/actions/workflows/release.yml/badge.svg)](https://github.com/tianyuxue/clablings/actions/workflows/release.yml)
![Go](https://img.shields.io/badge/go-1.25-00ADD8)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

> 在终端里做网络实验：按章节推进、答题、跑真实的 containerlab 拓扑，用声明式断言判定结果，也能在 CI 里无头运行。

[English](README.md) | **简体中文**

## 功能

- **watch 模式**：一次只做一道题，有提示、章节概念/RFC 阅读、快速/硬重置和进度条。
- **真实网络实验**：部署 containerlab 拓扑，直接驱动 Docker。
- **多种题型**：quiz、运行态 `config-fix`、文件驱动 `config-fix`、C 语言 `code-implement`。
- **声明式断言**：exec、file、interface、link 四类检查，支持 `wait` 轮询等待收敛。
- **无头验证**：`clablings test` 可在 CI 里跑，支持 text 和 JSON 输出。
- **中英双语内容**：默认中文，可切英文。
- **clab-tui 联动**：`clablings debug` 用 [clab-tui](https://github.com/tianyuxue/clab-tui) 打开当前章节的共享拓扑。

## 前置条件

- **Linux**：实验依赖 Linux namespace 和 Docker CLI。
- **containerlab**：需要在 `PATH` 上，安装方法见 [containerlab 文档](https://containerlab.dev/install/)。
- **Docker**：daemon 正在运行。
- 从源码构建需要 Go **1.25**。
- 可选：**clab-tui** 在 `PATH` 上，供 `clablings debug` 使用。

跑实验通常需要 root，或者给 containerlab 配 SUID。具体权限模型取决于你的 containerlab/Docker 安装方式。如果 clablings 装在用户目录，用 `clablings watch --sudo`（见“说明”），不要直接 `sudo clablings`。

## 安装

### 一行脚本

```bash
curl -fsSL https://raw.githubusercontent.com/tianyuxue/clablings/main/scripts/install.sh | sh
```

脚本会下载最新的 Linux release，校验 `checksums.txt`，然后装到 `~/.local/bin/clablings`。需要指定版本、安装目录或卸载时：

```bash
./scripts/install.sh --version v0.1.0
./scripts/install.sh --bindir "$HOME/bin"
./scripts/install.sh --uninstall
```

### 手动安装

```bash
curl -fsSLO "https://github.com/tianyuxue/clablings/releases/latest/download/clablings_linux_amd64.tar.gz"
curl -fsSLO "https://github.com/tianyuxue/clablings/releases/latest/download/checksums.txt"
sha256sum -c checksums.txt --ignore-missing
tar -xzf clablings_linux_amd64.tar.gz
sudo install -m 0755 clablings_linux_amd64/clablings /usr/local/bin/clablings
```

arm64 机器把文件名换成 `clablings_linux_arm64.tar.gz`。

### 从源码构建

```bash
git clone https://github.com/tianyuxue/clablings
cd clablings
make build
make install          # 默认装到 ~/.local/bin，可用 PREFIX/DESTDIR 覆盖
make install-system   # 装到 /usr/local/bin，不要给 make 加 sudo
```

也可以直接：

```bash
go install github.com/tianyuxue/clablings/cmd/clablings@latest
```

## 快速开始

题库在单独的仓库里：

```bash
git clone https://github.com/tianyuxue/clab-exercises
clablings list --root ./clab-exercises
clablings watch --root ./clab-exercises
```

不想每次都写 `--root`，可以设环境变量：

```bash
export CLABLINGS_ROOT="$PWD/clab-exercises"
clablings watch
```

如果既没传 `--root`，也没设 `CLABLINGS_ROOT`，clablings 会从当前目录往上找 `clablings.yaml`。

## 命令

```bash
clablings watch [--root DIR]        # 交互式刷题
clablings run <exercise_id>         # 跳到指定题目
clablings reset [--hard]            # 快速重置 / 硬重置
clablings list [--json]             # 查看进度
clablings test [<id>|--all] [--answers FILE] [--format text|json]
clablings debug [--sudo] [--root DIR]
clablings read [<chapter_id>] [--lang en|zh]
clablings lint [--root DIR]
```

`clablings test` 是给 CI 用的无头验证入口。quiz 题需要 `--answers FILE`，选项下标从 **0** 开始。

## watch 快捷键

每进入一道题，watch 会打印一次上下文块：标题、brief、快捷键和末尾的进度条。动作不会重复打印。

- `Enter` 验证当前题（quiz 上是开始答题）
- `h` 看提示
- `c` 看本章概念和 RFC 阅读切片
- `p` / `n` 上一题 / 下一题
- `Esc` 取消正在进行的验证
- `r` / `f` 硬重置 / 快速重置
- `l` 打开题目选择器
- `q` 退出，保留容器
- `x` 退出并销毁容器
- `Ctrl+C` 退出

quiz 答题时，`h`/`c` 用作帮助，`p`/`n`/`l`/`r`/`f`/`q`/`x` 保持 watch 里的含义。`l`、`r`、`f` 会离开当前 quiz 并重新打印上下文，按 `Enter` 可以重新开始。

## 配置

可选配置文件：`~/.config/clablings/config.yaml`

```yaml
language: zh             # 内容语言：en 或 zh（默认 zh）
containerlab:
  bin_path: ""          # 留空 = 从 PATH 查找
docker:
  bin_path: ""          # 留空 = 从 PATH 查找
watch:
  poll_interval: 250ms
  debounce: 300ms
build:
  default_image: gcc:13
```

读取失败或字段缺失时会打印警告，并使用默认值。语言优先级是 `--lang` > `CLABLINGS_LANG` > `language` > `zh`。CLI 标签保持英文。

## 说明

- 每个题库实例的 lab 名是 `<rootbasename>-<sha1(abs root)[:8]>-<topology name>`，可以用 `CLABLINGS_INSTANCE` 覆盖实例 ID。容器名是 `clab-<lab>-<node>`。
- 每次部署都会在原始拓扑旁生成 `clablings.<lab>.clab.yml`，所以题库目录必须可写。这个文件在 `clab-exercises` 里已经 gitignore。
- `clablings debug` 会以当前用户启动 `clab-tui`；`--sudo` 放在子命令后面时，会用 `sudo <clab-tui> --file <derived>` 启动，不会提升 clablings 本身。要让 clablings 本身以 root 运行，把 `--sudo` 放在子命令前面：`clablings --sudo debug`。
- `sudo` 会重置 `PATH`，所以装在 `~/.local/bin` 的 `clablings` 用 `sudo clablings` 是找不到的。要么装到 `/usr/local/bin`（`make install-system` 或上面的手动安装），要么用全局 `--sudo`：`clablings watch --sudo` 会用当前二进制的绝对路径通过 sudo 重新执行自己。全局 `--sudo` 放在子命令前后都可以（`debug` 除外），并保留 `CLABLINGS_ROOT`、`CLABLINGS_INSTANCE`、`CLABLINGS_LANG`。
- 提权后的进程读的是 root 的配置（`/root/.config/clablings/config.yaml`）。内容语言会自动转发；`containerlab.bin_path`/`docker.bin_path` 和 `build.default_image` 需要给 root 配好，或者确保它们在 root 的 `PATH` 上。
- 用 `sudo` 跑 clablings 可能让派生拓扑和 git checkout 变成 root 所有；需要的话跑 `sudo chown -R "$USER" <library>` 改回来。
- 内容默认中文。切英文用 `--lang en`、`CLABLINGS_LANG=en`，或配置里写 `language: en`。

## 题库

题库维护在 [tianyuxue/clab-exercises](https://github.com/tianyuxue/clab-exercises)。章节覆盖、镜像要求和内容写作规范看它的 README。

## 开发

```bash
make test              # go test ./... -count=1
make lint              # go vet + 已安装时的 staticcheck
make build             # 纯 Go 静态二进制
make package           # linux/amd64 + linux/arm64 发布包
make test-integration  # 需要 root + Docker + containerlab
```

`make test-integration` 需要 Docker daemon、containerlab 和 `clab-exercises/README.md` 里列出的镜像。GitHub 上的 Integration workflow 默认关闭；准备好 self-hosted runner 后，把仓库变量 `ENABLE_PRIVILEGED_CI` 设为 `true`。

## 参与贡献

欢迎提 issue 和 PR。开 PR 前先跑 `go test ./...` 和 `go vet ./...`。安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。

## 许可证

[Apache-2.0](LICENSE)。
