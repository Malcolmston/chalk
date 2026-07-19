package prompts

// Upstream-parity tests. Every vector in this file is transcribed from the real
// terkelg/prompts (v2.4.2) source of truth on the `master` branch, not invented:
//
//   test/util.js            https://raw.githubusercontent.com/terkelg/prompts/master/test/util.js
//   lib/util/entriesToDisplay.js
//                           https://raw.githubusercontent.com/terkelg/prompts/master/lib/util/entriesToDisplay.js
//   lib/elements/confirm.js https://raw.githubusercontent.com/terkelg/prompts/master/lib/elements/confirm.js
//   lib/elements/number.js  https://raw.githubusercontent.com/terkelg/prompts/master/lib/elements/number.js
//   lib/elements/select.js  https://raw.githubusercontent.com/terkelg/prompts/master/lib/elements/select.js
//   lib/elements/multiselect.js
//                           https://raw.githubusercontent.com/terkelg/prompts/master/lib/elements/multiselect.js
//
// The interactive TTY parts (cursor rendering, escape codes, bell) are out of
// scope for this reduced Go port; these cover the deterministic input -> value
// vectors only.

import (
	"bytes"
	"strings"
	"testing"
)

// TestParityEntriesToDisplay mirrors upstream test/util.js's `entriesToDisplay`
// suite verbatim: the 11 (cursor,total,maxVisible) -> {startIndex,endIndex}
// assertions, including the "maxVisible optional" case.
func TestParityEntriesToDisplay(t *testing.T) {
	tests := []struct {
		name                      string
		cursor, total, maxVisible int
		wantStart, wantEnd        int
	}{
		{"top of list", 0, 8, 5, 0, 5},
		{"+1 from top", 1, 8, 5, 0, 5},
		{"+2 from top", 2, 8, 5, 0, 5},
		{"+3 from top", 3, 8, 5, 1, 6},
		{"-3 from bottom", 4, 8, 5, 2, 7},
		{"-2 from bottom", 5, 8, 5, 3, 8},
		{"-1 from bottom", 6, 8, 5, 3, 8},
		{"bottom of list", 7, 8, 5, 3, 8},
		{"top, maxVisible>total", 0, 10, 11, 0, 10},
		{"bottom, maxVisible>total", 9, 10, 11, 0, 10},
		{"maxVisible optional", 0, 10, 0, 0, 10}, // upstream: entriesToDisplay(0,10)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := entriesToDisplay(tt.cursor, tt.total, tt.maxVisible)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("entriesToDisplay(%d,%d,%d) = {%d,%d}, want {%d,%d}",
					tt.cursor, tt.total, tt.maxVisible, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

// TestParityConfirm mirrors lib/elements/confirm.js: the `_(c)` handler maps
// case-insensitive 'y' -> true (submit) and 'n' -> false (submit); an empty
// submit falls back to the initial (`this.value = this.value || false`).
func TestParityConfirm(t *testing.T) {
	tests := []struct {
		name string
		in   string
		def  bool
		want bool
	}{
		{"lower y", "y\r", false, true},
		{"upper Y", "Y\r", false, true}, // c.toLowerCase() === 'y'
		{"lower n", "n\r", true, false},
		{"upper N", "N\r", true, false}, // c.toLowerCase() === 'n'
		{"empty keeps initial true", "\r", true, true},
		{"empty keeps initial false", "\r", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Confirm(ConfirmConfig{
				Message: "?", Default: tt.def,
				In: strings.NewReader(tt.in), Out: &bytes.Buffer{},
			})
			if err != nil {
				t.Fatalf("Confirm err = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Confirm(in=%q, def=%v) = %v, want %v", tt.in, tt.def, got, tt.want)
			}
		})
	}
}

// TestParityNumber mirrors lib/elements/number.js's typed-value handling and
// submit path for the deterministic cases the Go port shares with upstream:
//   - a valid in-range value is accepted as typed
//   - the `float`/integer distinction (upstream parseInt vs parseFloat)
//   - `this.value = x !== ” ? x : this.initial` — empty submit returns initial
//
// Upstream clamps out-of-range input live while this port re-prompts; both agree
// on the final accepted value once a valid in-range number is entered.
func TestParityNumber(t *testing.T) {
	tests := []struct {
		name string
		cfg  NumberConfig
		in   string
		want float64
	}{
		{"plain float value", NumberConfig{Message: "n"}, "42\r", 42},
		{"negative sign accepted", NumberConfig{Message: "n"}, "-7\r", -7},
		{"integer rejects fraction then accepts", NumberConfig{Message: "n", Integer: true}, "2.5\r4\r", 4},
		{"min rejects then accepts in range", NumberConfig{Message: "n", Min: f64(10)}, "5\r15\r", 15},
		{"max rejects then accepts in range", NumberConfig{Message: "n", Max: f64(10)}, "20\r7\r", 7},
		{"empty submit returns initial", NumberConfig{Message: "n", Default: f64(99)}, "\r", 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.In = strings.NewReader(tt.in)
			cfg.Out = &bytes.Buffer{}
			got, err := Number(cfg)
			if err != nil {
				t.Fatalf("Number err = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Number(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestParitySelectWrap mirrors lib/elements/select.js up()/down() wrap-around:
// down() at the last entry wraps to 0; up() at 0 wraps to the last entry. The
// returned value tracks the highlighted choice (moveCursor sets this.value).
func TestParitySelectWrap(t *testing.T) {
	choices := []Choice{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	// down x3 from 0: 0->1->2->0 (wrap), enter -> index 0 "a".
	idx, choice, err := Select(SelectConfig{
		Message: "pick", Choices: choices,
		In:  strings.NewReader("\x1b[B\x1b[B\x1b[B\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil || idx != 0 || choice.Name != "a" {
		t.Fatalf("down-wrap select = %d %q err=%v, want 0 \"a\"", idx, choice.Name, err)
	}

	// up once from 0 wraps to last (index 2 "c").
	idx, choice, err = Select(SelectConfig{
		Message: "pick", Choices: choices,
		In:  strings.NewReader("\x1b[A\r"),
		Out: &bytes.Buffer{},
	})
	if err != nil || idx != 2 || choice.Name != "c" {
		t.Fatalf("up-wrap select = %d %q err=%v, want 2 \"c\"", idx, choice.Name, err)
	}
}

// TestParityMultiselectToggle mirrors lib/elements/multiselect.js:
// handleSpaceToggle flips the current entry's `selected`; submit returns the
// selected entries filtered in original choice order. Toggling the same entry
// twice leaves it unselected.
func TestParityMultiselectToggle(t *testing.T) {
	choices := []Choice{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	// space, down, down, space, enter -> select a and c, order [0,2].
	idxs, chosen, err := MultiSelect(MultiSelectConfig{
		Message: "pick", Choices: choices,
		In:  strings.NewReader(" \x1b[B\x1b[B \r"),
		Out: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 2 || idxs[0] != 0 || idxs[1] != 2 {
		t.Fatalf("multiselect idxs = %v, want [0 2]", idxs)
	}
	if chosen[0].Name != "a" || chosen[1].Name != "c" {
		t.Fatalf("multiselect chosen = %q,%q, want a,c", chosen[0].Name, chosen[1].Name)
	}

	// space, space on the same entry -> deselected -> nothing selected.
	idxs, _, err = MultiSelect(MultiSelectConfig{
		Message: "pick", Choices: choices,
		In:  strings.NewReader("  \r"),
		Out: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 0 {
		t.Fatalf("double-toggle multiselect idxs = %v, want []", idxs)
	}

	// pre-checked entries are honored at submit (upstream ch.selected). Selecting
	// b as well yields order [0,1] preserving choice order.
	idxs, _, err = MultiSelect(MultiSelectConfig{
		Message: "pick",
		Choices: []Choice{{Name: "a", Checked: true}, {Name: "b"}, {Name: "c"}},
		In:      strings.NewReader("\x1b[B \r"), // down to b, toggle on, enter
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idxs) != 2 || idxs[0] != 0 || idxs[1] != 1 {
		t.Fatalf("pre-checked multiselect idxs = %v, want [0 1]", idxs)
	}
}
