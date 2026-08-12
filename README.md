# chalk

[![Go Test](https://github.com/Malcolmston/chalk/actions/workflows/go-test.yml/badge.svg)](https://github.com/Malcolmston/chalk/actions/workflows/go-test.yml)
[![Go Lint](https://github.com/Malcolmston/chalk/actions/workflows/go-lint.yml/badge.svg)](https://github.com/Malcolmston/chalk/actions/workflows/go-lint.yml)
[![Go Vuln](https://github.com/Malcolmston/chalk/actions/workflows/go-vuln.yml/badge.svg)](https://github.com/Malcolmston/chalk/actions/workflows/go-vuln.yml)
[![Web Unit](https://github.com/Malcolmston/chalk/actions/workflows/web-unit.yml/badge.svg)](https://github.com/Malcolmston/chalk/actions/workflows/web-unit.yml)
[![Web E2E](https://github.com/Malcolmston/chalk/actions/workflows/web-e2e.yml/badge.svg)](https://github.com/Malcolmston/chalk/actions/workflows/web-e2e.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/chalk.svg)](https://pkg.go.dev/github.com/malcolmston/chalk)
[![Go Report Card](https://goreportcard.com/badge/github.com/malcolmston/chalk)](https://goreportcard.com/report/github.com/malcolmston/chalk)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Malcolmston/chalk)](go.mod)
[![Release](https://img.shields.io/github/v/release/Malcolmston/chalk?sort=semver)](https://github.com/Malcolmston/chalk/releases)
[![Last Commit](https://img.shields.io/github/last-commit/Malcolmston/chalk)](https://github.com/Malcolmston/chalk/commits)
[![Code Size](https://img.shields.io/github/languages/code-size/Malcolmston/chalk)](https://github.com/Malcolmston/chalk)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Docs](https://img.shields.io/badge/docs-vercel-2f9bff)](https://go-malcolms-projects-18e573c3.vercel.app/lib/chalk)

**Node's chalk, inquirer, and figlet — for Go.**

`chalk` brings expressive terminal styling to Go, plus interactive prompts
(inquirer-style) and ASCII-art banners (figlet):

- **`chalk`** — chainable ANSI colors & styles, 16 / 256 / truecolor, hex & RGB,
  automatic capability detection (`NO_COLOR` / `FORCE_COLOR` / TTY).
- **`chalk/prompts`** — Input, Password, Confirm, Number, Select, MultiSelect, Slides.
- **`chalk/progress`** — progress bars: determinate or indeterminate, ETA and
  rate, byte-size formatting, and a group for several bars at once.
- **`chalk/figlet`** — render text as ASCII-art banners with a built-in font or
  any `.flf` FIGfont.
- **`chalk/spinner`** — one-line loaders that resolve into a final status line,
  and that quietly stop animating when stdout is not a terminal.

```go
import "github.com/malcolmston/chalk"

fmt.Println(chalk.New().Red().Bold().Sprint("error!"))
fmt.Println(chalk.Green("ok"))
fmt.Println(chalk.New().Hex("#ff8800").Underline().Sprint("orange"))
```

## Install

```sh
go get github.com/malcolmston/chalk
```

## Colors & styles

Styles are immutable and chainable; render with `Sprint`/`Sprintf`/`Print`/
`Println`:

```go
c := chalk.New().Bold().BgBlue().White()
fmt.Println(c.Sprint("  hello  "))
```

- **Modifiers**: `Bold`, `Dim`, `Italic`, `Underline`, `Inverse`, `Hidden`,
  `Strikethrough`, `Overline`.
- **Colors**: `Black`…`White`, `Gray`, `BrightRed`…`BrightWhite`, and the `Bg*`
  equivalents.
- **True/256 color**: `RGB(r,g,b)`, `Hex("#rrggbb")`, `Ansi256(n)`, and `Bg*`
  variants. Colors degrade automatically to the terminal's capability.
- **Shortcuts**: `chalk.Red("x")`, `chalk.Bold("y")`, `chalk.Hex("#f80", "z")`, …
- **Utilities**: `chalk.Strip(s)` removes ANSI escape sequences;
  `chalk.VisibleLength(s)` counts visible runes; `chalk.VisibleWidth(s)` counts
  terminal cells; `chalk.RuneWidth(r)` measures a single rune.

Colors are resolved when a style *renders*, not when it is built, so a style
stored in a package variable follows a later `SetLevel`, and `Level` can pin one
style without touching the global setting:

```go
warn := chalk.New().Hex("#ff8800")             // built once, reused
warn.Level(chalk.LevelTrueColor).Sprint("hot") // "\x1b[38;2;255;136;0mhot\x1b[39m"
warn.Level(chalk.Level256).Sprint("hot")       // "\x1b[38;5;214mhot\x1b[39m"
```

### Measuring styled text

Escape sequences occupy bytes but no screen cells, and CJK characters and emoji
occupy two cells each — so neither `len` nor a rune count aligns a colored
column correctly:

```go
cell := chalk.Green("图表")
len(cell)                  // 16 — bytes, including escape codes
chalk.VisibleLength(cell)  // 2  — runes
chalk.VisibleWidth(cell)   // 4  — terminal cells; this is the one to pad with
```

`chalk.Strip` removes whole escape sequences, not just colors: CSI cursor and
erase commands and OSC hyperlinks are stripped too.

### Color detection

Detection follows the precedence of Node's `supports-color`:

1. `NO_COLOR` (set at all) disables color outright.
2. `FORCE_COLOR=0`/`false` disables it; any other value (`1`/`2`/`3`, `true`, or
   empty) sets a *floor* and lifts the TTY requirement. It is a floor, not a cap:
   `FORCE_COLOR=1` in a truecolor terminal still gives truecolor. Numbers are
   clamped to 0–3.
3. Azure Pipelines (`TF_BUILD` + `AGENT_NAME`) gets color even when redirected.
4. Otherwise, output that is not a terminal gets no color.
5. `TERM=dumb` gets no more than the floor.
6. Then CI providers (`GITHUB_ACTIONS`, `TRAVIS`, `CIRCLECI`, `GITLAB_CI`,
   `BUILDKITE`, `DRONE`, codeship, …), `TEAMCITY_VERSION`, `COLORTERM`, known
   truecolor terminals (`xterm-kitty`, `xterm-ghostty`, `wezterm`, Windows
   Terminal), `TERM_PROGRAM` (iTerm, Apple Terminal), and finally the `TERM`
   name itself.

Override it explicitly:

```go
chalk.SetLevel(chalk.LevelTrueColor) // force truecolor
chalk.SetEnabled(false)              // disable all color
chalk.ResetDetection()               // re-detect from the environment
```

## Prompts

Interactive prompts styled with chalk, in the spirit of inquirer:

```go
import "github.com/malcolmston/chalk/prompts"

name, _ := prompts.Input(prompts.InputConfig{Message: "Name?", Default: "friend"})
pw, _   := prompts.Password(prompts.PasswordConfig{Message: "Password:"})
ok, _   := prompts.Confirm(prompts.ConfirmConfig{Message: "Continue?", Default: true})
age, _  := prompts.Number(prompts.NumberConfig{Message: "Age?", Integer: true})

i, choice, _ := prompts.Select(prompts.SelectConfig{
	Message: "Pick one",
	Choices: []prompts.Choice{ {Name: "Red"}, {Name: "Green"}, {Name: "Blue"} },
})

idxs, chosen, _ := prompts.MultiSelect(prompts.MultiSelectConfig{
	Message: "Pick many",
	Choices: []prompts.Choice{ {Name: "a"}, {Name: "b", Checked: true}, {Name: "c"} },
})
```

`Select` / `MultiSelect` use arrow keys (space to toggle, enter to confirm) on a
real terminal, and are fully testable by feeding a scripted key stream to the
`In` field. Canceling (Ctrl-C / Esc) returns `prompts.ErrCanceled`, and the
terminal is always restored on the way out. Set `MaxVisible` to page a long list
around the cursor (upstream prompts' `limit`):

```go
prompts.Select(prompts.SelectConfig{
	Message:    "Pick one",
	MaxVisible: 5,        // show 5 at a time, scrolling with the cursor
	Choices:    choices,
})
```

### Fuzzy finding

`Fuzzy` is an fzf-style finder: a filter line the user types into, a list that
narrows live, and the matched characters highlighted inside each candidate. It
returns the index of the chosen entry. `FuzzyMulti` is the checkbox variant —
Tab toggles (Space belongs to the filter), Enter confirms:

```go
i, _ := prompts.Fuzzy(prompts.FuzzyConfig{
	Message:    "Deploy where?",
	Choices:    []prompts.Choice{ {Name: "docker-compose"}, {Name: "kubernetes-staging"} },
	MaxVisible: 10,
	Match:      prompts.FuzzyOptions{SmartCase: true}, // uppercase query => case-sensitive
})

idxs, chosen, _ := prompts.FuzzyMulti(prompts.FuzzyMultiConfig{Message: "Pick many", Choices: choices})
```

The matcher is exported and pure, so it can be used (and tested) on its own:

```go
for _, m := range prompts.FuzzyRank("mn", []string{"main.go", "domain.txt", "man"}, prompts.FuzzyOptions{}) {
	fmt.Println(m.Candidate, m.Score, m.Positions) // Positions are rune offsets to highlight
}
score, positions, ok := prompts.FuzzyScore("mn", "main.go", prompts.FuzzyOptions{})
```

Matching is subsequence matching with an optimal (not greedy) alignment. Ranking
prefers a match at the start of the candidate or at a word boundary (the first
query rune counts double), then consecutive runs, then shorter gaps and earlier
matches; equal scores break by shorter candidate, then earlier first match, then
input order — a total order, so a redrawn list never jitters. Matching is
case-insensitive by default; `SmartCase` makes an uppercase rune in the query
turn on case sensitivity, and `CaseSensitive` forces it. An empty query returns
everything in input order.

When `Out` is not a terminal — a pipe, a file, a CI log — the prompt does not
repaint: it writes the filter line once and the answer summary at the end, with
LF endings and no cursor-movement escapes. `FuzzyConfig.Repaint` overrides the
detection in either direction, which is how tests assert on rendered frames
without a TTY. End of input resolves like Enter (`ErrNoMatch` when the filter
matches nothing), so a scripted or piped run never hangs.

`Slides` is a paged, in-place viewer for a sequence of styled text pages —
navigate with the arrow keys (or space / `n` / `p`), and quit with `q`, Esc or
Ctrl-C. It returns the index of the page on screen when it closed:

```go
page, _ := prompts.Slides(prompts.SlidesConfig{
	Title: "Intro",
	Pages: []string{"Welcome", "Chapter 1", chalk.Cyan("The End")},
	Loop:  true, // wrap around at the ends
})
```

Like the other prompts, `Slides` reads keys from any `io.Reader`, so it is fully
scriptable via the `In` field; outside a TTY it skips raw mode and treats end of
input as a quit, so a piped stdin never hangs.

## Spinners

A loader owns one line: it animates while work happens and resolves into a single
status line.

```go
import "github.com/malcolmston/chalk/spinner"

s := spinner.New(spinner.Config{Text: "Building"}).Start()
// ... work ...
s.UpdateText("Linking")
// ... work ...
s.Succeed("Built in 1.2s")   // or Fail / Warn / Info / Stop
```

`Stop` erases the spinner line and leaves the cursor at column zero, so the next
thing printed starts clean. `Succeed`, `Fail`, `Warn` and `Info` do the same and
then print the symbol and color configured for that state (`Config.Symbols`
overrides any of them). Frames come from `spinner.Dots` (braille), `spinner.Line`
(pure ASCII), `spinner.Bars` (block elements), or any `FrameSet` you supply:

```go
spinner.New(spinner.Config{
	Text:     "Uploading",
	Frames:   spinner.FrameSet{Frames: []string{"<", "^", ">", "v"}},
	Interval: 60 * time.Millisecond,
	Prefix:   "[dist] ",
})
```

**Outside a terminal it does not animate at all.** When `Out` is a pipe, a file
or a CI log — or `TERM=dumb` — the spinner starts no goroutine, writes no cursor
escapes, and prints the label once at `Start` plus one final line when it
resolves. `Config.Mode` forces either path (`spinner.Animate` / `spinner.Plain`),
and `Config.NewTicker` replaces the clock, so a test can step the animation frame
by frame with `spinner.NewManualTicker()` and assert on the exact bytes without a
terminal or a sleep. `Start`, `Stop`, the resolving methods and `UpdateText` are
safe from any goroutine; a double `Start` never starts a second animation, and
`Stop` joins the goroutine before returning.

## Pipes & stream detection

```go
import "github.com/malcolmston/chalk/pipe"

pipe.IsTerminal(os.Stdout)   // is this stream a TTY?
pipe.IsPiped(os.Stdin)       // was data piped in?
w, h := pipe.Size(os.Stdout) // measured, or COLUMNS/LINES, or 80x24
pipe.ColorLevel(os.Stderr)   // may *this* stream be styled?
```

`chalk/pipe` is the detection layer plus the streaming helpers a program needs to
behave when its input or output is not a terminal.

Detection builds on the parent package rather than repeating it: the colour
*tier* is `chalk.GetLevel()`, and `pipe.ColorLevel` only decides whether colour is
allowed for one specific stream — the question chalk cannot answer, since its
detection is bound to `os.Stdout`. It adds `CLICOLOR` / `CLICOLOR_FORCE` (which
chalk does not implement) alongside `NO_COLOR`, `FORCE_COLOR` and `TERM=dumb`, and
`IsTerminal` asks the OS through `golang.org/x/term` instead of accepting any
character device, so `/dev/null` is not mistaken for a terminal. Nothing is
changed globally unless you ask: `pipe.SyncColor(os.Stdout)` pushes the answer
into `chalk.SetLevel`.

Streaming:

```go
// Reads piped stdin; returns pipe.ErrNoInput immediately instead of hanging
// when stdin is a keyboard. Timeout / Ctx / MaxBytes are optional.
data, err := pipe.ReadAll(pipe.ReadConfig{})

// A filter stage: transform each line, drop it by returning false.
n, err := pipe.Lines(pipe.LinesConfig{Transform: func(l string) (string, bool) {
    return strings.ToUpper(l), l != ""
}})

// Pretty on screen, clean in the log, from one call.
t := pipe.NewTee(pipe.TeeConfig{Screen: os.Stdout, Log: logFile, Style: chalk.New().Red()})
t.Println("build failed")   // styled to stdout, plain text to logFile
```

**Outside a terminal nothing garbles.** No helper here emits a cursor escape at
all; a `Tee` whose screen is a pipe or a file strips every escape sequence from
*both* copies, so the two are byte-identical, and text already styled by
`chalk.Red(...)` is stripped rather than forwarded. A read that would block on an
interactive terminal returns `pipe.ErrNoInput` instead — the hang with no input
and no explanation is the bug this exists to prevent — and `Size` falls back to
`COLUMNS`/`LINES` and then to 80x24. Every entry point takes its `In`/`Out` from
its config struct, so all of it is testable with `strings.Reader` and
`bytes.Buffer` and no terminal; a `Tee` is an `io.Writer` and is safe to write
from many goroutines, and `ReadAll`'s timed read never blocks the caller past its
deadline.

## Progress bars

```go
import "github.com/malcolmston/chalk/progress"

bar := progress.New(progress.Config{Total: int64(len(items))})
for _, it := range items {
	process(it)
	bar.Add(1)
}
bar.Finish()
```

`Config` carries the `Total`, the bar `Width`, the `Out` writer and a `Template`,
so the layout is the caller's: `{bar} {percent} {current}/{total} {rate} {eta}`,
plus `{elapsed}`, `{spinner}` and `{desc}`. Anything else in the template is
copied through, so literal text goes there too.

```go
bar := progress.New(progress.Config{
	Total:    fileSize,
	Bytes:    true,               // 1536 renders as "1.5 KiB", rates as "768 B/s"
	Charset:  progress.Charset{Filled: "=", Empty: "-", Head: ">"},
	Template: "{desc}{bar} {current}/{total} {rate} eta {eta}",
})
io.Copy(io.MultiWriter(dst, bar), src) // a Bar is an io.Writer that counts
bar.Finish()
```

With `Total` left at zero the total is unknown: the percentage and ETA render as
`--` and the bar becomes a block bouncing inside its track. `Start` animates it
from its own goroutine while the caller is busy, and `Stop` (or `Finish`) joins
that goroutine:

```go
bar := progress.New(progress.Config{Template: "{spinner} indexing {bar} {current} {elapsed}"})
bar.Start()
defer bar.Finish()
```

Two things make it behave outside a terminal, which is where progress bars
usually break:

- **On a terminal one line is rewritten in place** (carriage return plus
  erase-to-end-of-line). **When `Out` is not a terminal** — a pipe, a file, a CI
  log — the bar emits plain lines terminated by `\n`, never an escape sequence,
  and only when the text changed *and* the rate limit allows it
  (`PlainInterval`, five seconds by default), so a long job leaves a handful of
  log lines instead of thousands. Detection is automatic; `Mode: ModeTerminal` /
  `ModePlain` forces it either way.
- **The clock is injectable.** `Config.Now` defaults to `time.Now`, and a test
  supplies its own instead, so the ETA and rate are asserted exactly and no test
  sleeps. `Bar.Render()` returns the line as a plain string for those assertions.

Several bars sharing one writer need one owner of the cursor, so use a group
rather than independent bars:

```go
g := progress.NewMulti(progress.MultiConfig{})
a := g.New(progress.Config{Total: 100, Description: "fetch"})
b := g.New(progress.Config{Total: 100, Description: "build"})
g.Start()
// ... advance a and b from any goroutines ...
g.Stop()
```

`MultiBar` repaints all of its bars as one block, so their frames never
interleave; in plain mode it emits only the lines that changed, which is why each
bar should carry a `{desc}`.

## Figlet

```go
import "github.com/malcolmston/chalk/figlet"

fmt.Println(figlet.Render("Hello"))                       // built-in font
fmt.Println(chalk.Cyan(figlet.Render("Colored!")))        // pipe through chalk

f, _ := figlet.LoadFontFile("slant.flf")                  // any .flf FIGfont
fmt.Println(f.Render("Custom", figlet.Options{Layout: figlet.LayoutSmush}))
```

The engine implements FIGfont parsing and the horizontal layout modes
(full-width, kerning, and smushing with the standard rules). `Options.Width`
wraps long text at word boundaries so a banner fits a narrow terminal:

```go
fmt.Println(figlet.Render("go chalk banner", figlet.Options{Width: 40}))
```

Font files are parsed strictly: a malformed header, a non-numeric field or a
character code with trailing garbage is an error rather than a silently wrong
font, and an absurd declared height is rejected instead of being used to size an
allocation (see `figlet.MaxFontHeight`).

### Fonts & color

Several fonts are built in and registered by name; load real `.flf` fonts from a
directory to add more:

```go
figlet.Fonts()                       // ["at","block","dark","dots","light","medium","plus","standard","stars"]
out, _ := figlet.RenderFont("block", "Hi")
figlet.LoadFontDir("./fonts")        // register every .flf in a directory
```

Pipe banners through gradients or a rainbow (uses chalk truecolor):

```go
fmt.Println(figlet.RenderGradient("Hello", "#ff0080", "#00d7ff"))
fmt.Println(figlet.RenderRainbow("Rainbow"))
```

## Images

```go
import chalkimage "github.com/malcolmston/chalk/image"

img, _ := chalkimage.Open("logo.png")            // PNG, JPEG and GIF (stdlib decoders)
_ = chalkimage.Fprint(img, chalkimage.Config{Width: 40})
```

The universal renderer downscales the picture (box-averaged, not nearest
neighbour) to a grid of character cells and paints it with the half-block trick:
`U+2580` with its own foreground and background colour shows two vertical pixels
per cell. It follows chalk's own colour detection rather than a second scheme of
its own, so it degrades from truecolour to 256 colours, to the 16 ANSI colours,
and finally to a greyscale ASCII ramp. Because a cell is about twice as tall as
it is wide, and each holds two stacked pixels, the aspect correction is
`rows = cols*h/(2*w)`; set `Width` to fit a width, `Height` to fit a height, or
both to fit inside a box (`Fit` reports the grid it chose).

Where a terminal supports a graphics protocol, `Protocol` can send the image
itself: `ProtocolITerm2` (an inline image in an OSC 1337 sequence) or
`ProtocolKitty` (chunked Kitty graphics). The default `ProtocolAuto` detects
conservatively — only unambiguous identification counts, and any multiplexer in
the way disqualifies the terminal — and falls back to the ANSI renderer when
unsure, because a wrong guess dumps raw base64 across the screen.

**Nothing is written when `Out` is not a terminal.** A pipe, a file and a CI log
get zero bytes; `Force: true` renders anyway, and even then a native protocol is
never auto-selected. `Grid` returns the sampled cells (`Upper`/`Lower` colours per
cell) so a test can assert on the picture without a terminal at all.

`Player` animates a decoded GIF, compositing frame disposal onto a persistent
canvas and repainting in place from one goroutine:

```go
g, _ := gif.DecodeAll(f)
p, _ := chalkimage.NewPlayer(g, chalkimage.PlayerConfig{Width: 40})
p.Start()
defer p.Stop() // safe from any goroutine, idempotent, and waits for the goroutine
```

Outside a TTY it does not animate at all — repainting needs cursor movement — and
draws a single still frame only when `Force` is set.

## Examples

```sh
go run ./examples/banner "Go"   # print an ASCII banner
go run ./examples/demo          # colors + figlet + interactive prompts
go run ./examples/table         # width-aware aligned table + capability report
```

Deviations from the upstream libraries are listed in
[API-DEVIATIONS.md](API-DEVIATIONS.md).

## License

[MIT](LICENSE)
