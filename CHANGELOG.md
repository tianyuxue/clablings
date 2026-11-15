# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-11-15

### Fixed

- `clablings watch` reprints the hot-key legend after showing hints, so the
  legend is visible again after long hint text. The concept view continues to
  reprint the full exercise context.

## [0.2.0] - 2026-11-11

### Added

- Global `--sudo` flag that re-executes clablings through sudo using its
  absolute path, preserving `CLABLINGS_ROOT`, `CLABLINGS_INSTANCE`, and
  `CLABLINGS_LANG`.
- `make install-system` target for installing to `/usr/local/bin` without
  running the build as root.

### Changed

- `clablings debug` resolves the chapter from the enclosing directory
  (including exercise subdirectories) or from the current exercise in
  `.clablings_state.json`.
- `clablings watch` reprints the full exercise context after showing a concept,
  so the objective and hot-key legend stay visible.

### Fixed

- `l` jump failures and same-exercise selections now produce visible,
  actionable messages; containerlab permission errors suggest
  `clablings watch --sudo`.
- Elevated runs hand `.clablings_state.json` back to the invoking user before
  the atomic rename, so non-root commands keep working after `--sudo`.
- `clablings --sudo debug` elevates clablings itself while `debug --sudo` keeps
  its local clab-tui meaning.
- `make install-system` refuses to run under `sudo`, avoiding root-owned build
  artifacts.

## [0.1.0] - 2026-11-01

### Added

- Interactive `watch` mode with per-exercise title, brief, hints, concept/RFC
  reading, progress, and fast/hard reset.
- Command set: `watch`, `run`, `reset`, `list`, `test`, `debug`, `read`, `lint`,
  and `version`.
- containerlab lifecycle management with per-library instance isolation.
- Declarative `exec`, `file`, `interface`, and `link` assertions with optional
  `wait` polling.
- Interactive quizzes with single/multi-select, explanations, and retry.
- Headless verification for CI with text or JSON output.
- C `code-implement` exercises with build and run support.
- Bilingual English/Chinese content; Chinese is the default content language.
- `clablings debug` integration with clab-tui.
- GitHub release automation for linux/amd64 and linux/arm64, with checksums and
  an install script.
