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

The rendering engine is a line-for-line port of the reference implementation
(patorjk/figlet.js, npm `figlet`) and is measured against it by the
cross-language harness in the aggregator repo, `parity/chalk/nested/figlet/`:
the `.flf` format, all five layout modes (`default`, `full`, `fitted`,
`controlled smushing`, `universal smushing`) horizontally *and* vertically, the
six horizontal and five vertical smushing rules, width wrapping with and without
`WhitespaceBreak`, `ShowHardBlanks` and right-to-left print direction all match
exactly.

Not implemented: control files (`.flc`) and character-code remapping.

Deliberate differences, each covered by a case in that harness:

- **The bundled fonts are this port's own art.** `figlet.RenderFont("Standard",
  …)` renders the hand-authored block font this package ships, not upstream's
  `Standard.flf`; the same goes for `Small`, `Mini`, `Banner` and `Block`, and
  for the ~1000 programmatically generated variants. The point of the registry is
  useful output with no external font files, which means the names collide with
  upstream's without the art matching. Load a real FIGfont with
  `LoadFontFile` / `LoadFont` / `ParseFont` to get upstream-identical output.
  Harness cases: `registry-*`.
- **Header numbers must be whole numbers.** Upstream reads them with `parseInt`,
  which truncates `"1.5"` to 1 and `"5abc"` to 5; a corrupt header therefore
  renders from a silently-corrected height. Header fields are parsed with
  `strconv.Atoi`, so a non-numeric field or trailing garbage is an error.
  Harness cases: `dev-parse-float-height`, `dev-parse-trailing-garbage-height`.
- **The `flf2a` signature is checked.** Upstream only looks at the sixth
  character of the first header field, so it will "parse" an arbitrary text file
  as a font.
- **The declared height is capped at `MaxFontHeight` (1000) and the file at
  `MaxFontBytes` (8 MiB).** Both numbers come out of attacker-controlled data and
  size allocations, so an absurd value would otherwise be an out-of-memory bug
  triggered by a font file.
- **Character codes** accept decimal, `0x` hex and leading-zero octal, and reject
  trailing garbage; upstream's regexes accept the same forms. Negative codes
  other than `-1` remain legal (they simply never match a rune), `-1` is rejected
  as the spec requires, and codes above `unicode.MaxRune` are rejected because
  `rune()` would silently turn them into a different code point.

## prompts

Configuration is a struct per prompt rather than an options object, and prompts
are ordinary functions returning `(value, error)` rather than promises.
Separators, filtered lists, editors and the plugin architecture are not ported;
styling is fixed rather than themeable. The `toggle`, `list`, `date`,
`autocomplete` and `autocompleteMultiselect` types are not ported.

The returned values are measured against upstream by the cross-language harness
in the aggregator repo, `parity/chalk/nested/prompts/`, which drives both
libraries with the same scripted keystrokes. `text`, `password`, `select` and
`multiselect` match exactly, including defaults, initial values, validation
retries, choice values and cancellation.

- `In` / `Out` are `io.Reader` / `io.Writer`, defaulting to stdin/stdout (these
  are upstream's `stdin` / `stdout` question options). Raw mode is entered only
  for a real terminal and is always restored through a deferred call, including
  on Ctrl-C, Esc and validation errors.
- End of input behaves like pressing Enter, *including* running the validator: a
  piped answer cannot skip validation by ending without a newline. Upstream never
  resolves at end of input.
- **An empty submission resolves to the default before the validator runs**, so a
  prompt with both a `Default` and a validator that rejects the empty string is
  answerable by pressing Enter. Harness case:
  `validate-default-satisfies-validator`.
- **After a validation failure the line is cleared** and the prompt is re-asked
  from empty. Upstream leaves the rejected text in the buffer with the cursor at
  its end, so the user has to delete it. Harness case:
  `dev-number-retry-after-validation`.

`Confirm` is line-oriented:

- **`Confirm` reads a whole line** and accepts `y`/`yes`/`n`/`no`
  (case-insensitively), falling back to `Default` for anything else. Upstream
  submits on the *first* `y` or `n` keypress, so answering `maybe` means yes
  there. Harness case: `dev-confirm-yes-inside-a-word`.

`Number` validates rather than coerces, which is where it diverges most:

- **`Min`/`Max` are validation rules, not clamps.** A number outside the bounds is
  rejected and re-prompted; upstream silently rewrites it to the nearest bound, so
  typing 99 into a `max: 10` prompt answers 10. Harness cases:
  `dev-number-above-max`, `dev-number-below-min`.
- **A number prompt always returns a number.** With no `Default`, an empty
  submission is rejected; upstream returns the empty *string* from a number
  prompt. Harness case: `dev-number-empty-no-initial`.
- **`Integer` (the inverse of upstream's `float`) rejects a fractional answer**
  rather than truncating it, and the whole line is read before it is checked.
  Upstream filters keystrokes as they arrive, so `.` is simply dropped in integer
  mode and `3.7` becomes 37. Harness case: `dev-number-decimal-in-integer-mode`.
- **No rounding.** Upstream rounds a float answer to 2 decimals by default (its
  `round` option); the port returns the number as typed. Harness case:
  `dev-number-float-rounding`.
- `Number` rejects `NaN` and infinities. `strconv.ParseFloat` accepts them, and
  every bound comparison against `NaN` is false, so they used to pass validation.
  Upstream's `parseInt`/`parseFloat` produce `NaN` for the same input and render
  it as an empty field.
- Arrow keys do not increment or decrement the value.

Lists:

- `MaxVisible` is upstream's `limit`: it pages a long list around the cursor and
  marks the hidden ends with a dim arrow.
- A list whose choices are all `Disabled` is an error rather than a prompt with
  no valid answer, and a choice that is both `Checked` and `Disabled` does not
  start checked. Upstream lets the cursor rest on a disabled entry and beeps on
  submit.
- `Select` and `MultiSelect` return the index *and* the `Choice`;
  `Choice.ResolvedValue` gives the value upstream would return, falling back to
  the label when no `Value` was set.
- `MultiSelect` has no toggle-all key and no maximum-selection limit, and maps
  left/right to cursor movement rather than to deselect/select.

Additions:

- `Slides` is a paged text viewer with no upstream counterpart.
