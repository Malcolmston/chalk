# prompts — provides interactive terminal prompts

[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/chalk/prompts.svg)](https://pkg.go.dev/github.com/malcolmston/chalk/prompts)
[![Parity](https://img.shields.io/badge/parity-100%25-brightgreen)](https://github.com/malcolmston/go/tree/main/parity/chalk/nested/prompts)

Package prompts provides interactive terminal prompts, a Go port of Node's
prompts (terkelg/prompts) styled with the sibling chalk package. It offers
text input, password, confirm, number, single-select, and multi-select
prompts, plus a paged `Slides` viewer.

Use this package to ask a user questions from a command-line program: collect
a value (`Input`), read a secret without echoing it (`Password`), confirm a
yes/no decision (`Confirm`), read a bounded number (`Number`), or let the user
pick from a list with the arrow keys (`Select` and `MultiSelect`), or page
through a sequence of styled text pages with `Slides`. Each prompt is driven
by a small config struct — `InputConfig`, `ConfirmConfig` and the rest —
that carries the message, defaults, validation and the input/output streams,
so the API stays flat and there is no shared prompt object to construct first.

## Install

```sh
go get github.com/malcolmston/chalk@v0.5.0
```

```go
import "github.com/malcolmston/chalk/prompts"
```

## Usage

This is the package's own `ExampleConfirm`, so it compiles and its output is
asserted on every `go test ./prompts/`.

```go
ok, err := prompts.Confirm(prompts.ConfirmConfig{
		Message: "Continue?",
		Default: false,
		In:      strings.NewReader("y\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(ok)
```

```
true
```

## Exported surface

### Functions

| Function | What it does |
| --- | --- |
| `func Confirm(cfg ConfirmConfig) (bool, error)` | Confirm prompts for a yes/no answer, returning the boolean result. |
| `func Input(cfg InputConfig) (string, error)` | Input prompts for a line of text. |
| `func Number(cfg NumberConfig) (float64, error)` | Number prompts for a numeric answer with optional bounds. |
| `func Password(cfg PasswordConfig) (string, error)` | Password prompts for a hidden (or masked) secret. |
| `func Slides(cfg SlidesConfig) (int, error)` | Slides presents a paged, in-place viewer for a sequence of text pages. |

### Types

| Type | What it is |
| --- | --- |
| `Choice` | Choice is a selectable option for Select / MultiSelect. |
| `ConfirmConfig` | ConfirmConfig configures Confirm. |
| `InputConfig` | InputConfig configures Input. |
| `MultiSelectConfig` | MultiSelectConfig configures MultiSelect. |
| `NumberConfig` | NumberConfig configures Number. |
| `PasswordConfig` | PasswordConfig configures Password. |
| `SelectConfig` | SelectConfig configures Select. |
| `SlidesConfig` | SlidesConfig configures Slides. |

<details>
<summary><code>Choice</code> — constructors and methods</summary>

| Signature | What it does |
| --- | --- |
| `func MultiSelect(cfg MultiSelectConfig) ([]int, []Choice, error)` | MultiSelect presents a checkbox list: arrows to move, space to toggle, enter to confirm. |
| `func Select(cfg SelectConfig) (int, Choice, error)` | Select presents a single-choice list navigated with the arrow keys. |
| `func (c Choice) ResolvedValue() any` | ResolvedValue returns the choice's Value, falling back to its Name when no Value was set. |

</details>

### Variables

`ErrCanceled`

Full signatures, doc comments and every runnable example are on
[pkg.go.dev](https://pkg.go.dev/github.com/malcolmston/chalk/prompts).

## Measured parity

Compared case-for-case against **`prompts@2.4.2`** by the harness in
[`parity/chalk/nested/prompts`](https://github.com/malcolmston/go/tree/main/parity/chalk/nested/prompts):

| | |
| --- | --- |
| Parity | **100%** |
| Cases | 98 |
| Matching | 91 |
| Mismatching | 0 |
| Declared deviations | 7 |

Regenerate from the aggregator repo with `go test ./parity/chalk/nested/prompts/`.
Declared deviations are documented differences, excluded from the denominator
and listed in the harness report.

## Deviations from upstream

Deliberate differences for this package, where any exist, are recorded in the
module-wide [`API-DEVIATIONS.md`](../API-DEVIATIONS.md).

## License

MIT, as part of [`github.com/malcolmston/chalk`](..).
An independent re-implementation, not affiliated with or endorsed by the
original project.
