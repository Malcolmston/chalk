// Package prompts provides interactive terminal prompts, a Go port of Node's
// prompts (terkelg/prompts) styled with the sibling chalk package. It offers
// text input, password, confirm, number, single-select, and multi-select
// prompts, plus a paged [Slides] viewer.
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
// pick from a list with the arrow keys ([Select] and [MultiSelect]), or page
// through a sequence of styled text pages with [Slides]. Each prompt
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
// submission falls back to the configured default — before the validator runs,
// so a default is always an answerable one — and [Number] additionally enforces
// optional Min/Max bounds and integer-only mode.
//
// Parity with upstream is measured on the returned values, by a cross-language
// harness that drives both libraries with the same scripted keystrokes: [Input],
// [Password], [Select] and [MultiSelect] match exactly, including defaults,
// initial values, validation retries, choice values and cancellation. What the
// prompts *draw* is this package's own styling and is deliberately not compared.
// Configuration is a Go struct instead of an options object, prompts are ordinary
// functions returning (value, error) rather than promises, and the toggle, list,
// date and autocomplete types, separators, filtered lists and the plugin
// architecture are not ported. [Confirm] is line-oriented where upstream submits
// on the first y/n keypress, and [Number] validates its bounds where upstream
// clamps; both differences, and the rest, are listed in API-DEVIATIONS.md.
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

// ResolvedValue returns the choice's Value, falling back to its Name when no
// Value was set. This is the value upstream prompts hands back for a selected
// choice, and it is what a caller wants when the label *is* the value: reading
// Value directly gives nil for such a choice, and Choice's own documentation
// promised the fallback without anything implementing it.
func (c Choice) ResolvedValue() any {
	if c.Value != nil {
		return c.Value
	}
	return c.Name
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
