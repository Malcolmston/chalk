package prompts_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/malcolmston/chalk/prompts"
)

// ExampleInput demonstrates driving a text prompt non-interactively. Instead of
// reading from a real terminal, it points the config's In field at a
// strings.Reader that supplies the scripted keystrokes "Ada" followed by a
// carriage return, which stands in for the user typing a name and pressing
// Enter. The rendered prompt is sent to a discarded bytes.Buffer via Out so the
// example does not depend on terminal styling. Because the input is not a TTY,
// the prompt skips raw mode and simply consumes the scripted bytes, returning
// the typed value and a nil error. The takeaway is that any prompt can be tested
// or scripted by supplying In and Out, exercising the exact code path a live
// keyboard would.
func ExampleInput() {
	answer, err := prompts.Input(prompts.InputConfig{
		Message: "Your name?",
		In:      strings.NewReader("Ada\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("got %q\n", answer)
	// Output: got "Ada"
}

// ExampleConfirm demonstrates a scripted yes/no prompt. The config sets a
// Default of false, and the scripted input "y\r" answers yes and submits, so
// Confirm returns true. As with the other prompts, In is a strings.Reader and
// Out is a throwaway buffer, so no real terminal is involved and the result is
// fully deterministic. Confirm interprets "y" and "yes" (case-insensitively) as
// true, "n" and "no" as false, and an empty line as the configured Default. The
// takeaway is that Confirm reduces a line of input to a boolean while still
// honoring a default for empty submissions.
func ExampleConfirm() {
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
	// Output: true
}
