// Package prompts provides interactive terminal prompts in the spirit of Node's
// inquirer / @inquirer/prompts, styled with the sibling chalk package. It offers
// text input, password, confirm, number, single-select, and multi-select
// prompts.
//
//	name, _ := prompts.Input(prompts.InputConfig{Message: "Your name?"})
//	ok, _ := prompts.Confirm(prompts.ConfirmConfig{Message: "Continue?", Default: true})
//	i, choice, _ := prompts.Select(prompts.SelectConfig{
//		Message: "Pick one",
//		Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
//	})
//
// Use this package to ask a user questions from a command-line program: collect
// a value ([Input]), read a secret without echoing it ([Password]), confirm a
// yes/no decision ([Confirm]), read a bounded number ([Number]), or let the user
// pick from a list with the arrow keys ([Select] and [MultiSelect]). Each prompt
// is driven by a small config struct — [InputConfig], [ConfirmConfig] and the
// rest — that carries the message, defaults, validation and the input/output
// streams, so the API stays flat and there is no shared prompt object to
// construct first.
//
// Internally every prompt runs the same loop. It puts the terminal into raw mode
// (via golang.org/x/term) so keystrokes arrive one at a time, decodes each
// keypress — including arrow-key escape sequences — with an internal key reader,
// updates its in-memory state, and repaints. The line prompts echo characters as
// they are typed (or a mask rune, or nothing for a hidden password) and
// re-prompt when a Validate function rejects the input. The list prompts redraw
// their whole frame in place using ANSI cursor-movement codes, moving a pointer
// and, for [MultiSelect], toggling checkboxes. On acceptance the prompt clears
// its frame and prints a one-line summary of the answer.
//
// Two semantics matter most. First, cancellation: pressing Ctrl-C or Esc makes a
// prompt return [ErrCanceled], which callers should treat distinctly from a
// normal answer. Second, input handling in non-interactive contexts: because the
// key reader works on any io.Reader, raw mode is skipped automatically when the
// input is not a real terminal. Setting the config's In field to something like
// strings.NewReader lets you script a prompt for tests or piped input, where the
// scripted bytes drive the same code path a live keyboard would, and reaching
// end of input behaves like pressing Enter on the current buffer. An empty
// submission falls back to the configured default, and [Number] additionally
// enforces optional Min/Max bounds and integer-only mode.
//
// Parity with inquirer is by feel rather than by API. The prompt types, the
// green "?" prefix, the pointer and checkbox styling, default values, validation
// with inline error messages and the answer summary all echo the Node
// experience. The differences are that configuration is a Go struct instead of
// an options object, prompts are called as ordinary functions returning
// (value, error) instead of returning promises, and inquirer features such as
// separators, paged/filtered lists, editors and a plugin architecture are not
// included. Styling is fixed by this package rather than themeable.
package prompts

import (
	"errors"
	"io"
	"os"

	"github.com/malcolmston/chalk"
	"golang.org/x/term"
)

// ErrCanceled is returned when the user cancels a prompt (Ctrl-C or Esc).
var ErrCanceled = errors.New("prompts: canceled")

// Choice is a selectable option for Select / MultiSelect.
type Choice struct {
	// Name is the label shown to the user.
	Name string
	// Value is an arbitrary value associated with the choice (defaults to Name
	// when nil).
	Value any
	// Disabled marks the choice as unselectable.
	Disabled bool
	// Checked pre-selects the choice in a MultiSelect.
	Checked bool
}

// resolveIO returns the effective input/output, defaulting to os.Stdin/os.Stdout.
func resolveIO(in io.Reader, out io.Writer) (io.Reader, io.Writer) {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return in, out
}

// enterRaw puts a terminal input into raw mode so keystrokes arrive
// unbuffered. It returns a restore func (a no-op when in is not a terminal, e.g.
// a scripted reader in tests).
func enterRaw(in io.Reader) (restore func()) {
	f, ok := in.(*os.File)
	if !ok {
		return func() {}
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return func() {}
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return func() {}
	}
	return func() { _ = term.Restore(fd, old) }
}

// theme holds the styles used to render prompts.
var (
	stylePrefix   = chalk.New().Green()      // "?" prefix
	styleMessage  = chalk.New().Bold()       // the question
	stylePointer  = chalk.New().Cyan()       // selection pointer
	styleSelected = chalk.New().Cyan()       // highlighted choice
	styleDim      = chalk.New().Gray()       // hints / disabled
	styleAnswer   = chalk.New().Cyan()       // the chosen answer echoed back
	styleError    = chalk.New().Red()        // validation errors
	styleCheckOn  = chalk.New().Green()      // checked box
	styleHelp     = chalk.New().Gray().Dim() // key hints
)

// writeString is a small helper.
func writeString(w io.Writer, s string) { _, _ = io.WriteString(w, s) }
