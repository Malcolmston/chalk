# API deviations

This port mirrors the APIs of [chalk](https://github.com/chalk/chalk),
[supports-color](https://github.com/chalk/supports-color),
[figlet](https://github.com/patorjk/figlet.js) and
[prompts](https://github.com/terkelg/prompts) on purpose. Where behavior differs,
it is listed here with the reason.

## chalk

| Upstream | Here | Why |
| --- | --- | --- |
| `chalk.red('x')` — the style is callable | `chalk.New().Red().Sprint("x")`, or the shortcut `chalk.Red("x")` | Go has no callable values with methods. |
| Tagged template literals | — | No Go equivalent. |
| `chalk.level` is per-instance | `SetLevel` / `GetLevel` are process-global; `Style.Level` pins one style | A package-level default plus an explicit per-style override covers the same ground. |
| `strip-ansi`, `string-width` are separate packages | `Strip`, `VisibleLength`, `VisibleWidth`, `RuneWidth` | Standard-library-only: no dependency to add. |
| `chalk.ansi256(263)` passes the index through | Clamped to 0–255 | Emitting an out-of-range SGR parameter is meaningless; clamping keeps `-1` from becoming bright white. |

Notes on semantics that do match, and are easy to get wrong:

- **Colors resolve at render time.** `New().RGB(...)` records the color, not the
  escape code. The truecolor → 256 → 16 downgrade is chosen when the style
  renders, so a stored style follows a later `SetLevel` and `.Level()` overrides
  a color chained before it.
- **Empty input yields no escape codes**, matching `chalk.red('') === ''`.
- **Line breaks close and re-open the style**, so color never bleeds across
  lines (upstream's `stringEncaseCRLFWithFirstIndex`). Both LF and CRLF.
- **Nested styles re-open the outer style** after an inner close code of the same
  type — upstream's "style bleed" fix.

### Color detection

Precedence follows `supports-color` (see the README for the full ladder). Two
deliberate differences:

- `NO_COLOR` is honored when the variable is *present*, whatever its value, and
  is checked before `FORCE_COLOR`.
- `COLORTERM=24bit` is accepted as truecolor in addition to `truecolor`. It is a
  common spelling that upstream does not recognize.

Command-line flags (`--color`, `--no-color`, `--color=256`) are not sniffed:
`os.Args` parsing belongs to the program, not to a styling library. Upstream's
Windows-release-number check is also not implemented; Windows Terminal is
detected through `WT_SESSION` instead.

## figlet

Implemented: the `.flf` format, the four layout modes, and the standard
horizontal smushing rules (equal character, underscore, hierarchy, opposite
pair, big X, hardblank).

Not implemented: vertical smushing, right-to-left print direction, control files
(`.flc`) and character-code remapping.

Additions and stricter behavior:

- `Options.Width` wraps at word boundaries. A word wider than the limit is
  emitted whole on its own line rather than split.
- Header fields are parsed with `strconv`, so a non-numeric field or trailing
  garbage is an error. Previously `fmt.Sscanf` accepted `"5abc"` as 5 and, with
  its error ignored, turned a non-numeric field into 0.
- Character codes accept decimal, `0x` hex and leading-zero octal, and reject
  trailing garbage. Negative codes remain legal (they simply never match a rune);
  codes above `unicode.MaxRune` are rejected.
- The declared height is capped at `MaxFontHeight` (1000). It is read from the
  file and used to size allocations, so an absurd value would otherwise be an
  out-of-memory bug triggered by a font file.
- A truncated glyph table is still tolerated: whatever characters were read are
  usable.

## prompts

Configuration is a struct per prompt rather than an options object, and prompts
are ordinary functions returning `(value, error)` rather than promises.
Separators, filtered lists, editors and the plugin architecture are not ported;
styling is fixed rather than themeable.

- `In` / `Out` are `io.Reader` / `io.Writer`, defaulting to stdin/stdout. Raw
  mode is entered only for a real terminal and is always restored through a
  deferred call, including on Ctrl-C, Esc and validation errors.
- End of input behaves like pressing Enter, *including* running the validator: a
  piped answer cannot skip validation by ending without a newline.
- `Number` rejects `NaN` and infinities. `strconv.ParseFloat` accepts them, and
  every bound comparison against `NaN` is false, so they used to pass validation.
- `MaxVisible` is upstream's `limit`: it pages a long list around the cursor and
  marks the hidden ends with a dim arrow.
- A list whose choices are all `Disabled` is an error rather than a prompt with
  no valid answer, and a choice that is both `Checked` and `Disabled` does not
  start checked.
