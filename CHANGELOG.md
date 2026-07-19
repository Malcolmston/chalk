# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-07-19
### Added
- **Upstream-parity tests** for the `figlet` and `prompts` sub-packages, verified
  against patorjk/figlet.js and terkelg/prompts (real `.flf` fonts + expected art
  bundled under `figlet/testdata`); `parity.json` published.
### Changed
- 100% exported-symbol API-doc coverage across the module.

## [0.3.0] - 2026-07-18
### Added
- Color-space conversions mirroring Node chalk's ansi-styles / color-convert:
  `HexToRGB`, `RGBToHex`, `RGBToAnsi256`, `Ansi256ToRGB`, `RGBToAnsi16`,
  `Ansi256ToAnsi16`, `RGBToHSL`, `HSLToRGB`, `RGBToHSV`, `HSVToRGB`, `RGBToHWB`
  and `HWBToRGB`.
- Additional color-model style methods and shortcuts for parity with
  `chalk.hsl` / `chalk.hsv` / `chalk.hwb`: `Style.HSL`, `Style.BgHSL`,
  `Style.HSV`, `Style.BgHSV`, `Style.HWB`, `Style.BgHWB`, and package-level
  `HSL`, `HSV`, `HWB`.
- `Style.Visible` and package-level `Visible` — the chalk `.visible` modifier
  that emits text only when color output is enabled.
- `supportsColor`-style capability predicates: `SupportsColor`, `HasBasic`,
  `Has256`, `HasTrueColor`.
- Completed the package-level shortcut surface with the previously missing
  background and bright colors and remaining modifiers: `BgBlack`…`BgWhite`,
  `BgGray`, `BgRGB`, `BgHex`, `BgAnsi256`, `BrightBlack`…`BrightWhite`,
  `Reset`, `Hidden`, and `Overline`.

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

[Unreleased]: https://github.com/malcolmston/chalk/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/malcolmston/chalk/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/malcolmston/chalk/releases/tag/v0.1.0
