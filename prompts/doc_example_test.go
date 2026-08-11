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

// ExampleSelect demonstrates a scripted single-choice list, including the
// MaxVisible option that pages a long list. The scripted input sends two
// "cursor down" escape sequences and then Enter, so the selection lands on the
// third choice; MaxVisible limits the drawn list to three entries at a time and
// scrolls it around the cursor, which is the behavior of upstream prompts'
// "limit" option. Choices marked Disabled are skipped by the arrow keys, and a
// list in which every choice is disabled is rejected with an error rather than
// returning an unselectable answer. As with the other prompts, In and Out make
// the whole interaction testable without a terminal.
func ExampleSelect() {
	idx, choice, err := prompts.Select(prompts.SelectConfig{
		Message:    "Pick a color",
		MaxVisible: 3,
		Choices: []prompts.Choice{
			{Name: "Red"},
			{Name: "Green", Disabled: true},
			{Name: "Blue"},
			{Name: "Violet"},
		},
		In:  strings.NewReader("\x1b[B\x1b[B\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%d %s\n", idx, choice.Name)
	// Output: 3 Violet
}

// ExampleSlides demonstrates the paged slide viewer driven non-interactively.
// The scripted input sends two "cursor right" escape sequences to advance from
// the first page to the third, then "q" to quit. Slides returns the index of the
// page that was on screen when the viewer closed. As with the other prompts, In
// and Out make the whole interaction testable without a terminal; a piped or
// redirected stdin that simply ends is treated as a quit, so the viewer never
// hangs. The takeaway is that Slides is an in-place, arrow-navigable reader for a
// sequence of styled text pages that degrades gracefully outside a TTY.
func ExampleSlides() {
	page, err := prompts.Slides(prompts.SlidesConfig{
		Title: "Intro",
		Pages: []string{"Welcome", "Chapter 1", "The End"},
		In:    strings.NewReader("\x1b[C\x1b[Cq"),
		Out:   &bytes.Buffer{},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("closed on page %d\n", page)
	// Output: closed on page 2
}
