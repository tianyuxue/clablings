# Clablings

[![test](https://github.com/tianyuxue/clablings/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tianyuxue/clablings/actions/workflows/test.yml)
[![Integration](https://github.com/tianyuxue/clablings/actions/workflows/integration.yml/badge.svg?branch=main)](https://github.com/tianyuxue/clablings/actions/workflows/integration.yml)
[![Release](https://github.com/tianyuxue/clablings/actions/workflows/release.yml/badge.svg)](https://github.com/tianyuxue/clablings/actions/workflows/release.yml)
![Go](https://img.shields.io/badge/go-1.25-00ADD8)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

> A terminal-first network practice runner: guided exercises, quizzes, real
> containerlab topologies, declarative checks, and headless verification.

**English** | [简体中文](README.zh-CN.md)

## Features

- **Guided watch mode**: one exercise at a time, with hints, concept/RFC reading,
  fast/hard reset, and a progress bar.
- **Real network labs**: deploys containerlab topologies and drives Docker.
- **Multiple exercise types**: quiz, runtime `config-fix`, file-driven
  `config-fix`, and C `code-implement`.
- **Declarative assertions**: exec, file, interface, and link checks with
  optional polling (`wait`).
- **Headless test mode**: `clablings test` for CI, with text or JSON output.
- **Bilingual content**: English and Simplified Chinese, with Chinese as the
  default content language.
- **clab-tui integration**: `clablings debug` opens the current chapter's shared
  topology in [clab-tui](https://github.com/tianyuxue/clab-tui).

## Requirements

- **Linux** (the labs use Linux namespaces and the Docker CLI).
- **[containerlab](https://containerlab.dev/install/)** on `PATH`.
- **Docker** with a running daemon.
- Go **1.25** when building from source.
- Optional: **clab-tui** on `PATH` for `clablings debug`.

Running labs usually needs root, or containerlab installed with SUID. The exact
privilege model depends on your containerlab/Docker setup. If you installed
`clablings` under your home directory, use `clablings watch --sudo` (see Notes)
instead of `sudo clablings`.

## Install

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/tianyuxue/clablings/main/scripts/install.sh | sh
```

The script downloads the latest Linux release, verifies `checksums.txt`, and
installs to `~/.local/bin/clablings`. Use `--version`, `--bindir`, or
`--uninstall` if needed:

```bash
./scripts/install.sh --version v0.1.0
./scripts/install.sh --bindir "$HOME/bin"
./scripts/install.sh --uninstall
```

### Manual install

```bash
curl -fsSLO "https://github.com/tianyuxue/clablings/releases/latest/download/clablings_linux_amd64.tar.gz"
curl -fsSLO "https://github.com/tianyuxue/clablings/releases/latest/download/checksums.txt"
sha256sum -c checksums.txt --ignore-missing
tar -xzf clablings_linux_amd64.tar.gz
sudo install -m 0755 clablings_linux_amd64/clablings /usr/local/bin/clablings
```

Use `clablings_linux_arm64.tar.gz` on arm64.

### Build from source

```bash
git clone https://github.com/tianyuxue/clablings
cd clablings
make build
make install          # installs to ~/.local/bin; override PREFIX/DESTDIR
make install-system   # installs to /usr/local/bin; do not prefix with sudo
```

Or:

```bash
go install github.com/tianyuxue/clablings/cmd/clablings@latest
```

## Quick start

The exercises live in a separate repository:

```bash
git clone https://github.com/tianyuxue/clab-exercises
clablings list --root ./clab-exercises
clablings watch --root ./clab-exercises
```

Set `CLABLINGS_ROOT` to avoid passing `--root` every time:

```bash
export CLABLINGS_ROOT="$PWD/clab-exercises"
clablings watch
```

Without `--root` or `CLABLINGS_ROOT`, clablings searches upward from the current
directory for `clablings.yaml`.

## Commands

```bash
clablings watch [--root DIR]        # guided exercise loop
clablings run <exercise_id>         # jump to an exercise
clablings reset [--hard]            # fast / hard reset
clablings list [--json]             # progress
clablings test [<id>|--all] [--answers FILE] [--format text|json]
clablings debug [--sudo] [--root DIR]
clablings read [<chapter_id>] [--lang en|zh]
clablings lint [--root DIR]
```

`clablings test` is the headless verification path for CI. Quiz exercises need
`--answers FILE`; option indices are **0-based**.

## Watch keys

Watch prints a context block once per exercise: title, brief, key legend, and a
progress bar at the end. Actions do not reprint it.

- `Enter` verify the current exercise (on a quiz: start the quiz)
- `h` show hints
- `c` show the chapter's concept and RFC reading slices
- `p` / `n` previous / next exercise
- `Esc` cancel an in-flight verification
- `r` / `f` hard / fast reset
- `l` open the exercise picker
- `q` quit and keep containers
- `x` quit and destroy containers
- `Ctrl+C` quit

During a quiz, `h`/`c` show help and `p`/`n`/`l`/`r`/`f`/`q`/`x` keep their
watch meanings. `l`, `r`, and `f` leave the quiz and reprint the context block;
press `Enter` to start it again.

## Configuration

Optional config file at `~/.config/clablings/config.yaml`:

```yaml
language: zh             # content language: en or zh (default: zh)
containerlab:
  bin_path: ""          # empty = resolve from PATH
docker:
  bin_path: ""          # empty = resolve from PATH
watch:
  poll_interval: 250ms
  debounce: 300ms
build:
  default_image: gcc:13
```

On a read failure or missing field, a warning is printed and the default value is
used. Language precedence is `--lang` > `CLABLINGS_LANG` > `language` > `zh`.
CLI labels stay English.

## Notes

- Labs are isolated per library instance: the lab name is
  `<rootbasename>-<sha1(abs root)[:8]>-<topology name>` (override the instance id
  with `CLABLINGS_INSTANCE`). Container names are `clab-<lab>-<node>`.
- Each deploy generates a derived topology `clablings.<lab>.clab.yml` beside the
  original, so the library directory must be writable. The derived file is
  gitignored in `clab-exercises`.
- `clablings debug` execs `clab-tui` as the current user. A trailing `--sudo`
  launches it via `sudo <clab-tui> --file <derived>` and does not elevate
  `clablings` itself; put `--sudo` before the subcommand
  (`clablings --sudo debug`) to run clablings as root.
- `sudo` resets `PATH`, so `sudo clablings` cannot find a binary installed under
  `~/.local/bin`. Either install to `/usr/local/bin` (`make install-system` or
  the manual install above), or use the global `--sudo` flag:
  `clablings watch --sudo` re-executes the current binary through `sudo` using
  its absolute path. The global flag works before or after every command except
  `debug` and preserves `CLABLINGS_ROOT`, `CLABLINGS_INSTANCE`, and
  `CLABLINGS_LANG`.
- Elevated runs read root's config (`/root/.config/clablings/config.yaml`). The
  content language is forwarded automatically, but
  `containerlab.bin_path`/`docker.bin_path` and `build.default_image` must be
  configured for root or available on root's `PATH`.
- Running clablings with `sudo` can leave derived files and git checkouts
  root-owned; run `sudo chown -R "$USER" <library>` afterwards if needed.
- Content language defaults to Chinese. Use `--lang en`, `CLABLINGS_LANG=en`, or
  `language: en` for English.

## Content

The exercise library is maintained at
[tianyuxue/clab-exercises](https://github.com/tianyuxue/clab-exercises). See its
README for chapter coverage, images, and content authoring rules.

## Development

```bash
make test              # go test ./... -count=1
make lint              # go vet + staticcheck when installed
make build             # static pure-Go binary
make package           # linux/amd64 + linux/arm64 release archives
make test-integration  # root + Docker + containerlab
```

`make test-integration` is privileged and requires a working Docker daemon,
containerlab, and the images listed in `clab-exercises/README.md`. The GitHub
Integration workflow is disabled by default; set the repository variable
`ENABLE_PRIVILEGED_CI=true` once a self-hosted runner is available.

## Contributing

Issues and pull requests are welcome. Before opening a PR, run `go test ./...`
and `go vet ./...`. See [SECURITY.md](SECURITY.md) for private vulnerability
reporting.

## License

[Apache-2.0](LICENSE).
