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
- **`chalk/figlet`** — render text as ASCII-art banners with a built-in font or
  any `.flf` FIGfont.

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
