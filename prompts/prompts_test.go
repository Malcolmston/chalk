package prompts

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/malcolmston/chalk"
)

func init() { chalk.SetLevel(chalk.LevelNone) } // deterministic, unstyled output

func TestInput(t *testing.T) {
	out := &bytes.Buffer{}
	got, err := Input(InputConfig{
		Message: "Name?",
		In:      strings.NewReader("Ada\r"),
		Out:     out,
	})
	if err != nil || got != "Ada" {
		t.Fatalf("input = %q err=%v", got, err)
	}
}

func TestInputDefault(t *testing.T) {
	got, _ := Input(InputConfig{
		Message: "Name?",
		Default: "Anon",
		In:      strings.NewReader("\r"), // empty submit
		Out:     &bytes.Buffer{},
	})
	if got != "Anon" {
		t.Fatalf("default = %q", got)
	}
}

func TestInputBackspace(t *testing.T) {
	got, _ := Input(InputConfig{
		Message: "x",
		In:      strings.NewReader("abcd\x7f\x7f\r"), // type abcd, delete twice -> ab
		Out:     &bytes.Buffer{},
	})
	if got != "ab" {
		t.Fatalf("backspace = %q", got)
	}
}

func TestInputValidateRetry(t *testing.T) {
	calls := 0
	got, _ := Input(InputConfig{
		Message: "x",
		In:      strings.NewReader("bad\rgood\r"),
		Out:     &bytes.Buffer{},
		Validate: func(s string) error {
			calls++
			if s != "good" {
				return errors.New("nope")
			}
			return nil
		},
	})
	if got != "good" || calls != 2 {
		t.Fatalf("validate retry = %q calls=%d", got, calls)
	}
}

func TestPasswordHidden(t *testing.T) {
	out := &bytes.Buffer{}
	got, _ := Password(PasswordConfig{
		Message: "pw",
		In:      strings.NewReader("secret\r"),
		Out:     out,
	})
	if got != "secret" {
		t.Fatalf("password = %q", got)
	}
	if strings.Contains(out.String(), "secret") {
		t.Fatal("password should not be echoed")
	}
}

func TestPasswordMask(t *testing.T) {
	out := &bytes.Buffer{}
	Password(PasswordConfig{Message: "pw", Mask: '*', In: strings.NewReader("abc\r"), Out: out})
	if !strings.Contains(out.String(), "***") {
		t.Fatalf("mask output = %q", out.String())
	}
}

func TestConfirm(t *testing.T) {
	for _, c := range []struct {
		in   string
		def  bool
		want bool
	}{
		{"y\r", false, true},
		{"n\r", true, false},
		{"\r", true, true},   // default
		{"\r", false, false}, // default
		{"yes\r", false, true},
	} {
		got, _ := Confirm(ConfirmConfig{Message: "?", Default: c.def, In: strings.NewReader(c.in), Out: &bytes.Buffer{}})
		if got != c.want {
			t.Fatalf("confirm(%q, def=%v) = %v, want %v", c.in, c.def, got, c.want)
		}
	}
}

func TestNumber(t *testing.T) {
	got, err := Number(NumberConfig{Message: "n", In: strings.NewReader("42\r"), Out: &bytes.Buffer{}})
	if err != nil || got != 42 {
		t.Fatalf("number = %v err=%v", got, err)
	}
}

func TestNumberIntegerBound(t *testing.T) {
	min := 0.0
	// "3.5" fails Integer, then "-1" fails Min, then "5" passes.
	got, _ := Number(NumberConfig{
		Message: "n", Integer: true, Min: &min,
		In:  strings.NewReader("3.5\r-1\r5\r"),
		Out: &bytes.Buffer{},
	})
	if got != 5 {
		t.Fatalf("number bound = %v", got)
	}
}

func TestSelectArrowKeys(t *testing.T) {
	// Down, Down, Enter -> index 2.
	idx, choice, err := Select(SelectConfig{
		Message: "pick",
		Choices: []Choice{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		In:      strings.NewReader("\x1b[B\x1b[B\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 2 || choice.Name != "c" {
		t.Fatalf("select = %d %q err=%v", idx, choice.Name, err)
	}
}

func TestSelectWrapAndSkipDisabled(t *testing.T) {
	// Up from index 0 wraps to last; middle is disabled so it's skipped.
	idx, _, _ := Select(SelectConfig{
		Message: "pick",
		Choices: []Choice{{Name: "a"}, {Name: "b", Disabled: true}, {Name: "c"}},
		In:      strings.NewReader("\x1b[A\r"), // up -> wraps to c (index 2)
		Out:     &bytes.Buffer{},
	})
	if idx != 2 {
		t.Fatalf("wrap/skip select = %d, want 2", idx)
	}
}

func TestMultiSelect(t *testing.T) {
	// Toggle index 0 (space), down, down, toggle index 2 (space), enter.
	idxs, chosen, err := MultiSelect(MultiSelectConfig{
		Message: "pick many",
		Choices: []Choice{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		In:      strings.NewReader(" \x1b[B\x1b[B \r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 2 || idxs[0] != 0 || idxs[1] != 2 {
		t.Fatalf("multiselect idxs = %v", idxs)
	}
	if chosen[0].Name != "a" || chosen[1].Name != "c" {
		t.Fatalf("multiselect chosen = %v", chosen)
	}
}

func TestCancel(t *testing.T) {
	_, err := Input(InputConfig{Message: "x", In: strings.NewReader("ab\x03"), Out: &bytes.Buffer{}}) // Ctrl-C
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("expected ErrCanceled, got %v", err)
	}
}
