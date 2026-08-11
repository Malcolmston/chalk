package prompts

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
)

// TestNumberRejectsNaNAndInf covers a degenerate input that used to pass every
// check: strconv.ParseFloat accepts "NaN" and "Inf", and every comparison
// against NaN is false, so a NaN answer sailed through Min/Max and was handed
// back to the caller.
func TestNumberRejectsNaNAndInf(t *testing.T) {
	min, max := 0.0, 10.0
	for _, in := range []string{"NaN", "nan", "Inf", "+Inf", "-Infinity"} {
		t.Run(in, func(t *testing.T) {
			// The bad value is rejected and the prompt re-asks; "5" is accepted.
			got, err := Number(NumberConfig{
				Message: "n", Min: &min, Max: &max,
				In:  strings.NewReader(in + "\r5\r"),
				Out: &bytes.Buffer{},
			})
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != 5 {
				t.Errorf("Number(%q then 5) = %v, want 5 (the bad value must be rejected)", in, got)
			}
		})
	}
}

// TestNumberIntegerHugeValue pins the overflow fix. The whole-number test used
// int64(v), and converting a float that far outside the int64 range is undefined
// in Go, so the comparison was meaningless. math.Trunc is exact at every
// magnitude.
func TestNumberIntegerHugeValue(t *testing.T) {
	got, err := Number(NumberConfig{
		Message: "n", Integer: true,
		In:  strings.NewReader("1e300\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != 1e300 {
		t.Errorf("Number(1e300) = %v, want 1e300", got)
	}
	if math.Trunc(got) != got {
		t.Errorf("1e300 should be a whole number")
	}

	// A large but genuinely fractional value is still rejected. 1e15 has a
	// float64 spacing of 0.125, so the .5 survives the parse (unlike, say,
	// 9007199254740993.5, which rounds to a whole number and is correctly
	// accepted).
	got, err = Number(NumberConfig{
		Message: "n", Integer: true,
		In:  strings.NewReader("1000000000000000.5\r7\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil || got != 7 {
		t.Fatalf("fractional value accepted: got %v err %v", got, err)
	}
}

// TestValidationRunsAtEndOfInput checks that a scripted answer that ends without
// a newline still goes through the validator. It used to be returned unchecked,
// so piped input could deliver a value the prompt would have rejected from a
// keyboard.
func TestValidationRunsAtEndOfInput(t *testing.T) {
	_, err := Input(InputConfig{
		Message:  "x",
		In:       strings.NewReader("bad"), // EOF, no Enter
		Out:      &bytes.Buffer{},
		Validate: func(string) error { return errors.New("nope") },
	})
	if err == nil {
		t.Fatal("EOF with a failing validator returned no error")
	}

	// A value that passes is still returned.
	got, err := Input(InputConfig{
		Message:  "x",
		In:       strings.NewReader("good"),
		Out:      &bytes.Buffer{},
		Validate: func(string) error { return nil },
	})
	if err != nil || got != "good" {
		t.Fatalf("EOF = %q, %v; want \"good\", nil", got, err)
	}
}

// TestListsRejectAllDisabled covers the degenerate list: every entry disabled
// means there is no answer to give, but the cursor still sat on entry 0 and
// Enter returned it.
func TestListsRejectAllDisabled(t *testing.T) {
	choices := []Choice{{Name: "a", Disabled: true}, {Name: "b", Disabled: true}}

	idx, _, err := Select(SelectConfig{
		Message: "pick", Choices: choices,
		In: strings.NewReader("\r"), Out: &bytes.Buffer{},
	})
	if err == nil {
		t.Errorf("Select with all choices disabled returned index %d and no error", idx)
	}

	if _, _, err := MultiSelect(MultiSelectConfig{
		Message: "pick", Choices: choices,
		In: strings.NewReader("\r"), Out: &bytes.Buffer{},
	}); err == nil {
		t.Error("MultiSelect with all choices disabled returned no error")
	}
}

// TestMultiSelectIgnoresCheckedDisabled checks that a choice marked both Checked
// and Disabled does not come back as selected: the user cannot untick it, so it
// must not start ticked.
func TestMultiSelectIgnoresCheckedDisabled(t *testing.T) {
	idxs, chosen, err := MultiSelect(MultiSelectConfig{
		Message: "pick",
		Choices: []Choice{
			{Name: "a", Checked: true, Disabled: true},
			{Name: "b", Checked: true},
		},
		In:  strings.NewReader("\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 1 || idxs[0] != 1 || chosen[0].Name != "b" {
		t.Errorf("selected = %v (%v), want only index 1", idxs, chosen)
	}
}

// TestSelectMaxVisible exercises the paging window, which is upstream prompts'
// "limit" option. entriesToDisplay already existed and was tested, but nothing
// in the prompts themselves used it.
func TestSelectMaxVisible(t *testing.T) {
	choices := make([]Choice, 10)
	for i := range choices {
		choices[i] = Choice{Name: string(rune('a' + i))}
	}
	out := &bytes.Buffer{}
	idx, _, err := Select(SelectConfig{
		Message:    "pick",
		Choices:    choices,
		MaxVisible: 3,
		In:         strings.NewReader("\r"),
		Out:        out,
	})
	if err != nil || idx != 0 {
		t.Fatalf("select = %d, %v", idx, err)
	}
	frame := out.String()
	// With a window of 3 around index 0, only a, b and c are drawn.
	for _, name := range []string{"a", "b", "c"} {
		if !strings.Contains(frame, name) {
			t.Errorf("frame missing visible choice %q:\n%s", name, frame)
		}
	}
	if strings.Contains(frame, "j") {
		t.Errorf("frame shows a choice outside the window:\n%s", frame)
	}
	if !strings.Contains(frame, "↓") {
		t.Errorf("frame lacks the more-below hint:\n%s", frame)
	}
}

// TestMultiSelectMaxVisibleScrolls moves the cursor past the bottom of the
// window and checks that the window follows it.
func TestMultiSelectMaxVisibleScrolls(t *testing.T) {
	choices := make([]Choice, 8)
	for i := range choices {
		choices[i] = Choice{Name: "item" + string(rune('0'+i))}
	}
	out := &bytes.Buffer{}
	// Five downs, then toggle and confirm.
	_, chosen, err := MultiSelect(MultiSelectConfig{
		Message:    "pick",
		Choices:    choices,
		MaxVisible: 3,
		In:         strings.NewReader(strings.Repeat("\x1b[B", 5) + " \r"),
		Out:        out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chosen) != 1 || chosen[0].Name != "item5" {
		t.Fatalf("chosen = %v, want item5", chosen)
	}
	if !strings.Contains(out.String(), "↑") {
		t.Errorf("scrolled frame lacks the more-above hint:\n%s", out.String())
	}
}

// TestVisibleWindow pins the window arithmetic directly, including the "show
// everything" default.
func TestVisibleWindow(t *testing.T) {
	cases := []struct {
		cur, total, maxVisible int
		start, end             int
		wantMore, wantLess     bool
	}{
		{0, 10, 0, 0, 10, false, false}, // no limit
		{0, 10, 3, 0, 3, false, true},   // top of a long list
		{5, 10, 3, 4, 7, true, true},    // middle
		{9, 10, 3, 7, 10, true, false},  // bottom
		{0, 2, 5, 0, 2, false, false},   // window larger than the list
		{0, 3, -1, 0, 3, false, false},  // negative limit treated as none
	}
	for _, c := range cases {
		start, end, more, less := visibleWindow(c.cur, c.total, c.maxVisible)
		if start != c.start || end != c.end || more != c.wantMore || less != c.wantLess {
			t.Errorf("visibleWindow(%d,%d,%d) = %d,%d,%v,%v; want %d,%d,%v,%v",
				c.cur, c.total, c.maxVisible, start, end, more, less,
				c.start, c.end, c.wantMore, c.wantLess)
		}
	}
}

// TestEnterRawIsANoOpForNonTerminals checks the restore function is always
// callable. Every prompt defers it, so a nil or panicking restore would leave a
// real terminal in raw mode after an error or a Ctrl-C.
func TestEnterRawIsANoOpForNonTerminals(t *testing.T) {
	restore := enterRaw(strings.NewReader("scripted"))
	if restore == nil {
		t.Fatal("enterRaw returned a nil restore func")
	}
	restore()
	restore() // must be safe to call twice
}

// TestPromptsRestoreTerminalOnCancel is a behavioral check that the deferred
// restore runs on the cancel path as well as the success path: all prompts must
// return through the defer rather than, say, calling os.Exit.
func TestPromptsRestoreTerminalOnCancel(t *testing.T) {
	calls := []struct {
		name string
		run  func() error
	}{
		{"Input", func() error {
			_, err := Input(InputConfig{Message: "x", In: strings.NewReader("\x03"), Out: &bytes.Buffer{}})
			return err
		}},
		{"Password", func() error {
			_, err := Password(PasswordConfig{Message: "x", In: strings.NewReader("\x03"), Out: &bytes.Buffer{}})
			return err
		}},
		{"Confirm", func() error {
			_, err := Confirm(ConfirmConfig{Message: "x", In: strings.NewReader("\x1b"), Out: &bytes.Buffer{}})
			return err
		}},
		{"Number", func() error {
			_, err := Number(NumberConfig{Message: "x", In: strings.NewReader("\x03"), Out: &bytes.Buffer{}})
			return err
		}},
		{"Select", func() error {
			_, _, err := Select(SelectConfig{Message: "x", Choices: []Choice{{Name: "a"}}, In: strings.NewReader("\x03"), Out: &bytes.Buffer{}})
			return err
		}},
		{"MultiSelect", func() error {
			_, _, err := MultiSelect(MultiSelectConfig{Message: "x", Choices: []Choice{{Name: "a"}}, In: strings.NewReader("\x1b"), Out: &bytes.Buffer{}})
			return err
		}},
	}
	for _, c := range calls {
		if err := c.run(); !errors.Is(err, ErrCanceled) {
			t.Errorf("%s on Ctrl-C/Esc = %v, want ErrCanceled", c.name, err)
		}
	}
}
