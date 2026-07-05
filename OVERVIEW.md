# chalk — Overview

`chalk` is a single Go module that covers three jobs the Node ecosystem splits
across three packages: expressive terminal **styling** (Node's *chalk*),
ASCII-art **banners** (*figlet*), and interactive **prompts** (*inquirer*). It
depends only on the standard library plus `golang.org/x/term`, and compiles into
one static binary.

- `github.com/malcolmston/chalk` — chainable ANSI colors and styles.
- `github.com/malcolmston/chalk/figlet` — FIGfont banner rendering, 1000+ fonts,
  gradients and rainbow coloring.
- `github.com/malcolmston/chalk/prompts` — Input, Password, Confirm, Number,
  Select, MultiSelect.

---

## How it works

### ANSI color and style model

A `Style` is an **immutable, chainable** value. `New()` returns an empty style,
and each method (`Red()`, `Bold()`, `BgBlue()`, `Hex(...)`, …) returns a *copy*
with one more SGR (Select Graphic Rendition) pair appended — an `open` code and
its matching `close` code. Because every method returns a fresh `*Style`, a base
style can be shared and extended without any method mutating a value used
elsewhere:

```go
base := chalk.New().Bold()
fmt.Println(base.Red().Sprint("err"))   // bold + red
fmt.Println(base.Green().Sprint("ok"))  // bold + green; base is unchanged
```

At render time (`Sprint`/`Sprintf`/`Sprintln` and the `Print*` writers), the
style wraps the text in its escape sequences, applying the innermost pair first
so that outer styles re-assert themselves. Nested styling is handled correctly:
if the text already contains a given pair's close sequence, the open sequence is
re-inserted after it so an inner reset does not prematurely end an outer style.

### Terminal capability detection

Color is emitted only when it makes sense, and it **degrades gracefully** to
whatever the terminal supports. The capability is expressed as a `Level`:

| Level             | Meaning              |
|-------------------|----------------------|
| `LevelNone`       | no color             |
| `LevelBasic`      | 16 ANSI colors       |
| `Level256`        | 256-color palette    |
| `LevelTrueColor`  | 24-bit color         |

Detection runs once (guarded by a mutex, cached) against the environment and the
stdout file, following the same conventions Node tools use:

1. `NO_COLOR` set → `LevelNone`.
2. `FORCE_COLOR` (`0`/`false`, `1`/`true`, `2`, `3`) → an explicit level.
3. stdout is not a terminal → `LevelNone`.
4. `TERM=dumb` → `LevelNone`.
5. `COLORTERM=truecolor`/`24bit` → `LevelTrueColor`; `TERM` containing `256` →
   `Level256`; otherwise `LevelBasic`.

`SetLevel`, `SetEnabled`, `GetLevel`, `Enabled`, and `ResetDetection` let you
override or re-run detection (a style can also be pinned to a level with
`.Level(...)`, which is handy in tests). High-fidelity colors are converted down
to the detected level automatically: `RGB`/`Hex` collapse to the nearest
256-palette index (`rgbTo256`) or the nearest basic 16-color code (`rgbTo16`),
and `Ansi256` collapses to 16 colors when needed — so the same code prints
truecolor on a modern terminal and readable output over SSH or in a pipe.

### FIGfont banner rendering

The `figlet` package is a Go port of the classic *figlet* engine. `ParseFont`
reads a standard `.flf` FIGfont: it validates the `flf2a` signature, reads the
header (hardblank, height, baseline, layout bits, comment count), skips the
comment block, then reads the fixed-height glyph rows for ASCII 32–126 plus any
optional code-tagged characters, stripping each row's end-mark characters.

Rendering lays glyphs out left to right using one of four horizontal layout
modes:

- **Full width** — glyphs are concatenated with no overlap.
- **Kerning** — glyphs slide together until their non-blank columns touch.
- **Smushing** — overlapping columns are *smushed* into a single character using
  the standard FIGfont rules (equal-character, underscore, hierarchy, opposite
  pair, big-X, and hardblank handling).

The effective mode comes from the font's own layout settings unless an explicit
`Options.Layout` overrides it. Hardblanks (which hold space open during layout)
are replaced with real spaces only at the end, and lowercase input transparently
falls back to an uppercase glyph so capitals-only fonts still render mixed case.

**1000+ importable fonts.** Beyond `LoadFont`/`LoadFontFile`/`LoadFontDir` for
real `.flf` files, the package ships a font registry. A handful of fonts are
hand-authored (a 5-row block font, a 3-row `small` font, a 5-row `banner`
outline font, plus fill variants), and `fonts_generated.go` combines three
independent axes — base shape × ink/fill character × decoration (bold, shadow,
box, outline, mirror, …) — into ~1000 deterministically named, visually distinct
fonts. `Fonts()` lists every registered name (over 1000 total); `GetFont` and
`RenderFont` render by name.

**Gradients and rainbow.** `Gradient` tints a rendered banner column by column,
interpolating between two hex colors; `Rainbow` sweeps the full hue spectrum
across the banner's width. Both drive `chalk`'s truecolor `RGB`, so they degrade
with the terminal like any other color. `RenderGradient` and `RenderRainbow` are
one-call conveniences over the built-in font.

### inquirer-style prompts

The `prompts` package builds interactive prompts on top of `chalk` styling and
`golang.org/x/term`. When the input is a real terminal it is switched to raw
mode so keystrokes arrive unbuffered; a small `keyReader` decodes runes, Enter,
Backspace, Space, Tab, Ctrl-C/Esc, and the arrow-key CSI escape sequences.

Every prompt takes a config struct and reads from `In` / writes to `Out`
(defaulting to `os.Stdin` / `os.Stdout`). Because those are plain
`io.Reader`/`io.Writer`, prompts are **fully testable without a TTY**: feed a
scripted byte stream as `In` and the raw-mode switch becomes a no-op.
`Select`/`MultiSelect` repaint in place by moving the cursor up and clearing to
the end of the screen each frame. Canceling with Ctrl-C or Esc returns
`ErrCanceled`.

---

## How to use it

All three examples below compile against this module.

### Styled output

```go
package main

import (
	"fmt"

	"github.com/malcolmston/chalk"
)

func main() {
	fmt.Println(chalk.New().Red().Bold().Sprint("error!"))
	fmt.Println(chalk.Green("ok")) // package-level shortcut
	fmt.Println(chalk.New().Hex("#ff8800").Underline().Sprint("orange"))

	warn := chalk.New().Bold().BgYellow().Black()
	fmt.Println(warn.Sprintf("  %d warnings  ", 3))
}
```

### A figlet banner (with a gradient)

```go
package main

import (
	"fmt"

	"github.com/malcolmston/chalk/figlet"
)

func main() {
	fmt.Println(figlet.Render("Hello"))                       // built-in font

	out, _ := figlet.RenderFont("banner", "Go")               // a registered font
	fmt.Println(out)

	fmt.Println(figlet.RenderGradient("chalk", "#ff0080", "#00d7ff"))
	fmt.Println(figlet.RenderRainbow("rainbow"))
}
```

### A prompt

```go
package main

import (
	"errors"
	"fmt"

	"github.com/malcolmston/chalk/prompts"
)

func main() {
	name, err := prompts.Input(prompts.InputConfig{
		Message: "What is your name?",
		Default: "friend",
	})
	if errors.Is(err, prompts.ErrCanceled) {
		fmt.Println("canceled")
		return
	}

	i, choice, _ := prompts.Select(prompts.SelectConfig{
		Message: "Pick a color",
		Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
	})

	fmt.Printf("Hi %s — you chose %s (index %d)\n", name, choice.Name, i)
}
```

---

## Why it's better than its predecessor

The predecessor here is really three separate Node libraries — **chalk**,
**figlet**, and **inquirer** — that a CLI typically pulls in together. Compared
with that stack, this project offers:

- **One library, three concerns.** Styling, banners, and prompts share one
  module and one design vocabulary (prompts are themed with `chalk`; banners are
  colored with `chalk`). In Node these are three unrelated dependencies with
  three release cadences.
- **Minimal dependencies, one binary.** Everything is standard-library except
  `golang.org/x/term` (needed for raw-mode input). A Go build produces a single
  static executable — no `node_modules`, no runtime, no transitive supply-chain
  sprawl.
- **1000+ built-in fonts, no font files to ship.** The figlet registry exposes
  over a thousand fonts compiled into the binary, and can still load real `.flf`
  files at runtime. The Node figlet package bundles a smaller set and reads font
  files from disk.
- **Type safety and immutability.** Styles are compile-time-checked method
  chains on immutable values, so a shared base style can be extended without
  spooky action at a distance. Levels, layouts, and prompt configs are typed
  rather than string- or options-bag-driven.
- **Graceful degradation built in.** Truecolor automatically collapses to 256 or
  16 colors, and `NO_COLOR`/`FORCE_COLOR`/`COLORTERM`/`TERM`/TTY detection is
  handled for you.
- **Testable prompts.** Because `In`/`Out` are `io.Reader`/`io.Writer`, prompt
  flows can be driven by a scripted byte stream in unit tests with no terminal.

**Honest tradeoffs.** The bundled fonts are systematically generated variants of
a few hand-authored base shapes, so they do not match the artistic variety of
the full historical FIGfont collection — load real `.flf` files when you need a
specific classic font. The styling API mirrors Node chalk's SGR model rather
than reproducing every helper (e.g. template-literal tagging), and italic /
overline and similar codes render only where the terminal supports them. The
goal is broad, dependable coverage of the common cases in one place, not exact
feature parity with each upstream library.
