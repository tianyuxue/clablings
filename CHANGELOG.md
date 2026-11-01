# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
