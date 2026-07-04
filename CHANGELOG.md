# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-04
### Added
- Initial public release — a terminal color, style and ASCII-art toolkit for Go.
- `chalk` ANSI color/style library with automatic color-level detection.
- `chalk/prompts` — inquirer-style interactive prompts.
- `chalk/figlet` — FIGfont ASCII-art rendering with **1,027 importable fonts**,
  plus gradient and rainbow helpers.
- Automated releases (VERSION-driven tags + GitHub Releases, moving `stable` tag).
- CI: build/test matrix (Go 1.23 & 1.24), `-race` + coverage, golangci-lint v2,
  govulncheck, CodeQL, benchmarks, dependency review, and a stale bot.

[Unreleased]: https://github.com/malcolmston/chalk/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/malcolmston/chalk/releases/tag/v0.1.0
