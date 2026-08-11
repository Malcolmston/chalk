# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-08-11
### Added
- **Interactive console input** (`chalk/prompts`): a paged `Slides` viewer with
  arrow/space/`q` navigation, optional looping and a clamped start page,
  alongside the existing `Input`, `Password`, `Confirm`, `Number`, `Select` and
  `MultiSelect`. Non-TTY input degrades cleanly and never hangs.

### Changed
- `chalk/figlet` is now a line-for-line port of figlet.js: vertical layout and
  its five smushing rules, upstream width wrapping, `ShowHardBlanks`,
  `PrintDirection`, controlled/universal smushing, the incremental
  horizontal-overlap scan, positional hierarchy rules, the seven mandated
  post-ASCII glyphs and the correct end-mark rule. Measured parity against
  `figlet@1.11.4` rose from 50.8% to 100% of compared cases.
- `chalk/prompts` measured against `prompts@2.4.2` rose from 74.7% to 100%: a
  line prompt now substitutes its default before validating (a prompt with a
  default plus a validator was previously unanswerable by Enter), and
  `Choice.ResolvedValue` implements the documented value-defaults-to-name rule.
### Added
- `VisibleWidth` and `RuneWidth`: ANSI-aware terminal-cell measurement that
  counts CJK characters and emoji as two columns and combining marks as zero,
  for aligning colored output (upstream's `string-width`).
- Color detection now recognizes CI providers (`GITHUB_ACTIONS`, `GITEA_ACTIONS`,
  Travis, CircleCI, AppVeyor, GitLab, Buildkite, Drone, codeship, Azure
  Pipelines), `TEAMCITY_VERSION`, `TERM_PROGRAM` (iTerm, Apple Terminal),
  `xterm-kitty`/`xterm-ghostty`/`wezterm` and Windows Terminal, following
  `supports-color`'s order of precedence.
- `figlet.Options.Width` is implemented: banners wrap to the given number of
  columns.
- `prompts.SelectConfig.MaxVisible` / `MultiSelectConfig.MaxVisible` page long
  lists around the cursor (upstream prompts' `limit`).
- `examples/table` demonstrating width-aware alignment and capability reporting,
  plus `API-DEVIATIONS.md`.
- `figlet` now implements the *whole* reference rendering engine: vertical
  layout and the five vertical smushing rules (`Options.VerticalLayout`),
  upstream's width wrapping including `Options.WhitespaceBreak`,
  `Options.ShowHardBlanks`, right-to-left `Options.PrintDirection`, and the
  `controlled smushing` / `universal smushing` layouts as `LayoutControlledSmush`
  / `LayoutUniversalSmush` (`LayoutFitted` is the new name for `LayoutKerning`).
- `figlet.Font.Metadata` and `figlet.Font.FittingRules` expose the FIGfont header
  fields and the layout rules they resolve to, matching `figlet.metadata`.
- `prompts.Choice.ResolvedValue` returns a choice's `Value`, falling back to its
  `Name` — the fallback `Choice` already documented but nothing implemented.

### Changed
- `figlet` renders byte-for-byte identically to patorjk/figlet.js: multi-line
  input is now stacked with the font's vertical layout (which can fuse the
  blocks' touching rows) and padded to a uniform row width, glyph rows are laid
  out ragged rather than pre-padded, and the horizontal overlap is computed with
  upstream's incremental algorithm rather than a blank-counting approximation.
  Output for a multi-line render, or for any font whose header enables vertical
  rules, therefore changes.
- `figlet.ParseFont` reads the seven code-page glyphs the format mandates after
  ASCII (Ä Ö Ü ä ö ü ß), which are stored positionally with no code tag; skipping
  them lost those glyphs *and* mis-read every code-tagged glyph after them.
- `figlet.ParseFont` rejects an incomplete glyph table instead of half-loading
  it, and reports a malformed code tag as an error rather than silently treating
  it as the end of the font. Whole-file size is bounded by
  `figlet.MaxFontBytes`.
- A FIGfont parsed from a file no longer falls back to the uppercase or space
  glyph for an undefined character; it contributes nothing, as upstream does. The
  capitals-only fonts bundled with this package keep the fallback.

### Fixed
- The `figlet` end-mark stripper removed *every* trailing repeat of a glyph row's
  last character, so a row of `@@@@` vanished and any font whose art ends in the
  end-mark character was mis-parsed. It now removes one mark (two on a glyph's
  final row) plus trailing whitespace, as upstream does.
- `figlet`'s hierarchy and opposite-pair smushing rules used a hand-written rank
  table that disagreed with upstream's positional one.
- Line prompts validated the raw input before substituting the default, so a
  prompt with both a `Default` and a validator that rejects the empty string was
  unanswerable by pressing Enter.
- 256-color and truecolor styles resolved their escape codes when the style was
  *built* rather than when it rendered, so a stored style ignored a later
  `SetLevel` and `Style.Level` could not override a color chained before it.
- `Strip` removed only SGR color codes; it now removes complete CSI, OSC, DCS and
  two-character escape sequences.
- FIGfont headers were parsed with `fmt.Sscanf` and the error discarded, so a
  non-numeric field became 0 and trailing garbage was ignored; character codes
  had the same problem. Both are now strict, and the declared font height is
  bounded (`figlet.MaxFontHeight`) so it cannot be used to force a huge
  allocation.
- `figlet`'s full-width merge mutated the caller's rows in place, corrupting a
  block that was still being held.
- `prompts.Number` accepted `NaN` and infinities (every bound check against NaN
  is false) and tested "whole number" with an `int64` conversion that is
  undefined for large floats.
- End of input skipped validation in the line prompts.
- `Select`/`MultiSelect` returned a disabled choice when every choice was
  disabled, and a `Checked` + `Disabled` choice started checked.
- Out-of-range 256-color indices were masked to a byte (turning `-1` into white);
  they are now clamped.

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
