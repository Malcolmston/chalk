package figlet

import (
	"strings"
	"testing"
)

func TestBuiltinRender(t *testing.T) {
	out := Render("HI")
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 rows, got %d:\n%s", len(lines), out)
	}
	// Every row should be the same visible width (block art is rectangular).
	w := len([]rune(lines[0]))
	for i, l := range lines {
		if len([]rune(l)) != w {
			t.Fatalf("row %d width %d != %d\n%s", i, len([]rune(l)), w, out)
		}
	}
	// "H" contains the vertical bars; the art should contain '#'.
	if !strings.Contains(out, "#") {
		t.Fatalf("no block characters in output:\n%s", out)
	}
}

func TestLowercaseFallback(t *testing.T) {
	if Render("hi") != Render("HI") {
		t.Fatal("lowercase should fall back to uppercase glyphs")
	}
}

func TestMultiLine(t *testing.T) {
	out := Render("A\nB")
	// Two 5-row blocks joined by a newline => 10 lines, i.e. 9 newlines.
	if n := strings.Count(out, "\n"); n != 9 {
		t.Fatalf("multiline newline count = %d, want 9\n%s", n, out)
	}
}

// A minimal FIGfont (height 2, full width). The parser tolerates a short font;
// glyphs are read in ASCII order starting at 32 (space), so this defines space
// (32) and '!' (33). Endmarks are '@'; '$' is the hardblank.
const miniFont = `flf2a$ 2 2 4 -1 1
mini test font
$$@
$$@@
X@
X@@
`

func TestParseAndRenderCustomFont(t *testing.T) {
	f, err := ParseFont(strings.NewReader(miniFont))
	if err != nil {
		t.Fatal(err)
	}
	if f.Height() != 2 {
		t.Fatalf("height = %d, want 2", f.Height())
	}
	// The '!' glyph is "X" over "X".
	if out := f.Render("!"); out != "X\nX" {
		t.Fatalf("custom render = %q, want \"X\\nX\"", out)
	}
	// Hardblank ($) in the space glyph should render as spaces, not '$'.
	if strings.Contains(f.Render(" "), "$") {
		t.Fatalf("hardblank leaked into output: %q", f.Render(" "))
	}
}

func TestSmushLayoutResolution(t *testing.T) {
	// oldLayout 1 = smushing with the equal-character rule (bit 1).
	font := `flf2a$ 1 1 3 1 0
|$@@
`
	f, err := ParseFont(strings.NewReader(font))
	if err != nil {
		t.Fatal(err)
	}
	if layout, rules := f.resolveLayout(LayoutDefault); layout != LayoutSmush || rules&1 == 0 {
		t.Fatalf("expected smush layout with equal rule, got %v rules=%d", layout, rules)
	}
}

func TestEqualCharSmushing(t *testing.T) {
	// Build a font whose '!' glyph is a single "|" column, height 1, with
	// equal-character smushing. Two '!' side by side smush into one "|".
	font := `flf2a$ 1 1 3 1 0
$@@
|@@
`
	f, err := ParseFont(strings.NewReader(font))
	if err != nil {
		t.Fatal(err)
	}
	// space (32) = "$" (hardblank), '!' (33) = "|".
	if out := f.Render("!!"); out != "|" {
		t.Fatalf("equal-char smush = %q, want \"|\"", out)
	}
}
