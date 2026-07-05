package prompts

import (
	"bytes"
	"strings"
	"testing"
)

func TestKeyReaderDecoding(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     keyType
		wantRune rune
	}{
		{name: "enter cr", in: "\r", want: keyEnter},
		{name: "enter lf", in: "\n", want: keyEnter},
		{name: "backspace del", in: "\x7f", want: keyBackspace},
		{name: "backspace bs", in: "\x08", want: keyBackspace},
		{name: "ctrl-c", in: "\x03", want: keyCtrlC},
		{name: "tab", in: "\t", want: keyTab},
		{name: "space", in: " ", want: keySpace, wantRune: ' '},
		{name: "rune", in: "x", want: keyRune, wantRune: 'x'},
		{name: "eof", in: "", want: keyEOF},
		{name: "arrow up csi", in: "\x1b[A", want: keyUp},
		{name: "arrow down csi", in: "\x1b[B", want: keyDown},
		{name: "arrow right csi", in: "\x1b[C", want: keyRight},
		{name: "arrow left csi", in: "\x1b[D", want: keyLeft},
		{name: "arrow up ss3", in: "\x1bOA", want: keyUp},
		{name: "arrow down ss3", in: "\x1bOB", want: keyDown},
		{name: "bare esc", in: "\x1b", want: keyEsc},
		{name: "esc non-csi", in: "\x1bx", want: keyEsc},
		{name: "esc csi truncated", in: "\x1b[", want: keyEsc},
		{name: "esc csi unknown dir", in: "\x1b[Z", want: keyEsc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kr := newKeyReader(strings.NewReader(tt.in))
			k := kr.read()
			if k.typ != tt.want {
				t.Fatalf("read(%q).typ = %d, want %d", tt.in, k.typ, tt.want)
			}
			if tt.wantRune != 0 && k.r != tt.wantRune {
				t.Fatalf("read(%q).r = %q, want %q", tt.in, k.r, tt.wantRune)
			}
		})
	}
}

func TestStep(t *testing.T) {
	ch := []Choice{{Name: "a"}, {Name: "b", Disabled: true}, {Name: "c"}}
	// Forward from 0 skips the disabled index 1 and lands on 2.
	if got := step(ch, 0, 1); got != 2 {
		t.Fatalf("step forward = %d, want 2", got)
	}
	// Backward from 0 wraps to 2 (skipping nothing disabled at the end).
	if got := step(ch, 0, -1); got != 2 {
		t.Fatalf("step backward wrap = %d, want 2", got)
	}
	// Empty choices returns the cursor unchanged.
	if got := step(nil, 5, 1); got != 5 {
		t.Fatalf("step on empty = %d, want 5", got)
	}
	// All choices disabled: the loop exhausts and returns the last cursor value.
	allDisabled := []Choice{{Disabled: true}, {Disabled: true}}
	if got := step(allDisabled, 0, 1); got < 0 || got > 1 {
		t.Fatalf("step all-disabled = %d, out of range", got)
	}
}

func TestFirstSelectable(t *testing.T) {
	ch := []Choice{{Name: "a", Disabled: true}, {Name: "b"}, {Name: "c"}}
	// Start out of range is clamped to 0; 0 is disabled so it advances to 1.
	if got := firstSelectable(ch, 99); got != 1 {
		t.Fatalf("firstSelectable(99) = %d, want 1", got)
	}
	// A negative start is clamped to 0 as well.
	if got := firstSelectable(ch, -1); got != 1 {
		t.Fatalf("firstSelectable(-1) = %d, want 1", got)
	}
	// A non-disabled start is returned as-is.
	if got := firstSelectable(ch, 2); got != 2 {
		t.Fatalf("firstSelectable(2) = %d, want 2", got)
	}
	// Empty choices returns the clamped start.
	if got := firstSelectable(nil, 3); got != 0 {
		t.Fatalf("firstSelectable(empty) = %d, want 0", got)
	}
}

func f64(v float64) *float64 { return &v }

func TestNumberBounds(t *testing.T) {
	tests := []struct {
		name string
		cfg  NumberConfig
		in   string
		want float64
	}{
		{
			name: "min rejects then accepts",
			cfg:  NumberConfig{Message: "n", Min: f64(10)},
			in:   "5\r15\r",
			want: 15,
		},
		{
			name: "max rejects then accepts",
			cfg:  NumberConfig{Message: "n", Max: f64(10)},
			in:   "20\r7\r",
			want: 7,
		},
		{
			name: "integer rejects fraction",
			cfg:  NumberConfig{Message: "n", Integer: true},
			in:   "2.5\r4\r",
			want: 4,
		},
		{
			name: "not a number retries",
			cfg:  NumberConfig{Message: "n"},
			in:   "abc\r3\r",
			want: 3,
		},
		{
			name: "default on empty",
			cfg:  NumberConfig{Message: "n", Default: f64(99)},
			in:   "\r",
			want: 99,
		},
		{
			name: "custom validate",
			cfg: NumberConfig{Message: "n", Validate: func(v float64) error {
				if v == 0 {
					return errNotANumber
				}
				return nil
			}},
			in:   "0\r8\r",
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.In = strings.NewReader(tt.in)
			cfg.Out = &bytes.Buffer{}
			got, err := Number(cfg)
			if err != nil {
				t.Fatalf("Number: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Number = %v, want %v", got, tt.want)
			}
		})
	}
}
