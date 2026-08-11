package chalk

import (
	"fmt"
	"strings"
	"testing"
)

func TestStripSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"sgr", "\x1b[31mred\x1b[39m", "red"},
		{"truecolor sgr", "\x1b[38;2;255;136;0mx\x1b[39m", "x"},
		{"cursor movement", "a\x1b[2Ab", "ab"},
		{"erase display", "\x1b[Jclean", "clean"},
		{"osc bel", "\x1b]0;title\x07body", "body"},
		{"osc st", "\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\", "link"},
		{"two-char escape", "\x1bcreset", "reset"},
		{"lone esc preserved", "a\x1b", "a\x1b"},
		{"incomplete csi preserved", "a\x1b[31", "a\x1b[31"},
		{"unicode survives", "\x1b[32m日本語\x1b[39m", "日本語"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Strip(c.in); got != c.want {
				t.Errorf("Strip(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestStripIsIdempotent checks that stripping already-plain text is a no-op and
// that a second pass changes nothing.
func TestStripIsIdempotent(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)
	styled := New().Red().Bold().Sprint("hello ") + New().Hex("#00ff00").Sprint("world")
	once := Strip(styled)
	if once != "hello world" {
		t.Fatalf("Strip = %q", once)
	}
	if twice := Strip(once); twice != once {
		t.Errorf("Strip not idempotent: %q -> %q", once, twice)
	}
}

func TestRuneWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{' ', 1},
		{'\n', 0},
		{'\t', 0},
		{'é', 1},
		{'∑', 1},
		{'日', 2},
		{'한', 2},
		{'ｆ', 2},    // fullwidth latin small f
		{'　', 2},    // ideographic space
		{0x0301, 0}, // combining acute accent
		{0x200b, 0}, // zero width space
		{0xfe0f, 0}, // variation selector-16
		{0x1f600, 2},
		{0x1f1fa, 1}, // regional indicator U (half of a flag)
	}
	for _, c := range cases {
		if got := RuneWidth(c.r); got != c.want {
			t.Errorf("RuneWidth(%U) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestVisibleWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "hello", 5},
		{"styled ascii", "\x1b[31mhello\x1b[39m", 5},
		{"cjk", "日本語", 6},
		{"mixed", "a日b", 4},
		{"combining mark", "é", 1},
		{"precomposed", "é", 1},
		{"emoji", "🙂", 2},
		{"text symbol", "☂", 1},
		{"emoji presentation selector widens", "☂️", 2},
		{"keycap sequence", "1️⃣", 2},
		{"flag", "🇺🇸", 2},
		{"family zwj sequence", "👨‍👩‍👧", 2},
		{"styled cjk", "\x1b[38;5;200m日本\x1b[39m", 4},
		{"hyperlink", "\x1b]8;;https://example.com\x1b\\go\x1b]8;;\x1b\\", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VisibleWidth(c.in); got != c.want {
				t.Errorf("VisibleWidth(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

// TestVisibleLengthVsWidth documents the difference between the two: rune count
// versus screen cells.
func TestVisibleLengthVsWidth(t *testing.T) {
	const s = "日本"
	if got := VisibleLength(s); got != 2 {
		t.Errorf("VisibleLength(%q) = %d, want 2 runes", s, got)
	}
	if got := VisibleWidth(s); got != 4 {
		t.Errorf("VisibleWidth(%q) = %d, want 4 cells", s, got)
	}
}

// TestVisibleWidthAlignsColumns is the practical reason VisibleWidth exists:
// padding styled cells to an equal width must produce equal on-screen columns.
func TestVisibleWidthAlignsColumns(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	cells := []string{New().Red().Sprint("ok"), New().Green().Sprint("日本"), "x"}
	const target = 8
	for _, c := range cells {
		padded := c + strings.Repeat(" ", target-VisibleWidth(c))
		if got := VisibleWidth(padded); got != target {
			t.Errorf("padded %q width = %d, want %d", Strip(c), got, target)
		}
	}
}

// ExampleVisibleWidth shows why ANSI-aware measurement is needed: the raw string
// length counts escape bytes, VisibleLength counts runes, and VisibleWidth
// counts the cells the terminal actually uses.
func ExampleVisibleWidth() {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	s := New().Red().Sprint("日本")
	fmt.Println(len(s), VisibleLength(s), VisibleWidth(s))
	// Output: 16 2 4
}
