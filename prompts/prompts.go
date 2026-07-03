// Package prompts provides interactive terminal prompts in the spirit of Node's
// inquirer / @inquirer/prompts, styled with chalk. It offers text input,
// password, confirm, number, single-select, and multi-select prompts.
//
//	name, _ := prompts.Input(prompts.InputConfig{Message: "Your name?"})
//	ok, _ := prompts.Confirm(prompts.ConfirmConfig{Message: "Continue?", Default: true})
//	i, choice, _ := prompts.Select(prompts.SelectConfig{
//		Message: "Pick one",
//		Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
//	})
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
