# Backlog — missing features & gaps

Curated real work for chalk / prompts / figlet. (See the note at the bottom
about the "10,000" target.)

## chalk (color)

- [ ] `chalk.level` per-instance instances (`new Chalk({level})`) — have global.
- [ ] Template-literal tagged styling (`chalk.tagged` parser) equivalent.
- [ ] `chalk.visible` (only render when color enabled), `chalk.reset` chaining.
- [ ] `supportsColor` details (stdout vs stderr, Windows 10 build detection,
      CI provider detection, `--color`/`--no-color` flag parsing).
- [ ] 16m→256→16 downgrade quality (nearest-color via CIE ΔE, not cube rounding).
- [ ] `chalk.ansi`, `chalk.bgAnsi` (raw code), blink/rapid-blink, double-underline,
      framed/encircled, curly/dotted/dashed underline styles.
- [ ] Underline color (`4:3` + `58;2;r;g;b`), overline color.
- [ ] Hyperlinks (OSC 8) and iTerm/Kitty image escapes.
- [ ] `stripAnsi` for OSC sequences (only CSI SGR handled today).
- [ ] `ansi-styles`-style open/close maps exported for interop.

## prompts (inquirer / @inquirer/prompts)

- [ ] Autocomplete / search prompt (async source, filtering).
- [ ] Editor prompt (`$EDITOR` / `$VISUAL`).
- [ ] Expand prompt, Rawlist prompt, Checkbox validation + min/max choices.
- [ ] Paging for long Select/MultiSelect lists + search-as-you-type.
- [ ] Choice separators, disabled reasons, descriptions/hints per choice.
- [ ] Default value indicators, transformers, and answer formatting.
- [ ] Multi-line input, input history, cursor movement (←/→, Home/End, word-jump).
- [ ] `when`/`skip` conditional prompts + a `Form`/`prompt([...])` sequencer.
- [ ] Validation with async + rendered spinner while validating.
- [ ] Password reveal toggle; masked confirmation.
- [ ] Number prompt increment/decrement with ↑/↓ and step.
- [ ] Progress bars, spinners (ora-style), and a `Confirm` with custom labels.
- [ ] Windows console (no-ANSI) fallback + resize handling (SIGWINCH).
- [ ] Theming API (colors/symbols) beyond the built-in palette.

## figlet

- [ ] Bundle the real FIGfonts (standard, slant, big, block, banner, doom, …)
      via `go:embed` instead of only the compact built-in.
- [ ] Vertical smushing + vertical layout modes.
- [ ] Right-to-left print direction; full/fitted/controlled-smushing per axis.
- [ ] Hard-blank + deep-copy edge cases; full-width vs kerning per font default.
- [ ] `figlet -l` list fonts, load fonts from a directory, `.tlf` (toilet) fonts.
- [ ] Colorized/gradient output helpers (integrate with chalk RGB).
- [ ] Width-aware wrapping to terminal columns.
- [ ] Unicode / combining-character handling in glyph widths.

## Shared / tooling

- [ ] Terminal size + resize events package.
- [ ] `boxen`-style boxes, `cli-table`-style tables, `log-symbols`, `figures`.
- [ ] Benchmarks; fuzz the FIGfont parser and ANSI stripper.
- [ ] `golangci-lint` config.

---

### On the "10,000 items" request

Rather than pad to 10,000 synthetic entries, this lists real, actionable gaps.
The richest real backlog here is the FIGfont library (hundreds of fonts) and the
inquirer prompt catalog — I can expand either into an exhaustive checklist.
