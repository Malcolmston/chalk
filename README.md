# chalk

**Node's chalk, inquirer, and figlet — for Go.**

`chalk` brings expressive terminal styling to Go, plus interactive prompts
(inquirer-style) and ASCII-art banners (figlet):

- **`chalk`** — chainable ANSI colors & styles, 16 / 256 / truecolor, hex & RGB,
  automatic capability detection (`NO_COLOR` / `FORCE_COLOR` / TTY).
- **`chalk/prompts`** — Input, Password, Confirm, Number, Select, MultiSelect.
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
- **Utilities**: `chalk.Strip(s)` removes ANSI codes; `chalk.VisibleLength(s)`
  counts visible runes.

### Color detection

Output is enabled when stdout is a TTY and `NO_COLOR` is unset, honoring
`FORCE_COLOR` and `COLORTERM`/`TERM`. Override it:

```go
chalk.SetLevel(chalk.LevelTrueColor) // force truecolor
chalk.SetEnabled(false)              // disable all color
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
	Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
})

idxs, chosen, _ := prompts.MultiSelect(prompts.MultiSelectConfig{
	Message: "Pick many",
	Choices: []prompts.Choice{{Name: "a"}, {Name: "b", Checked: true}, {Name: "c"}},
})
```

`Select` / `MultiSelect` use arrow keys (space to toggle, enter to confirm) on a
real terminal, and are fully testable by feeding a scripted key stream to the
`In` field. Canceling (Ctrl-C / Esc) returns `prompts.ErrCanceled`.

## Figlet

```go
import "github.com/malcolmston/chalk/figlet"

fmt.Println(figlet.Render("Hello"))                       // built-in font
fmt.Println(chalk.Cyan(figlet.Render("Colored!")))        // pipe through chalk

f, _ := figlet.LoadFontFile("slant.flf")                  // any .flf FIGfont
fmt.Println(f.Render("Custom", figlet.Options{Layout: figlet.LayoutSmush}))
```

The engine implements FIGfont parsing and the horizontal layout modes
(full-width, kerning, and smushing with the standard rules). A compact block
font is built in; load real `.flf` fonts for more styles.

## Examples

```sh
go run ./examples/banner "Go"   # print an ASCII banner
go run ./examples/demo          # colors + figlet + interactive prompts
```

## License

[MIT](LICENSE)
