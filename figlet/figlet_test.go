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

// buildFontWith assembles a *complete* FIGfont: a header, one comment line, and
// a glyph for every character the format requires (see requiredChars). Entries
// in art override the default filler glyph, which is how a test gives one
// specific character real sub-lines.
//
// Completeness matters: the parser rejects a font whose glyph table is short,
// so a two-glyph fixture is no longer a valid font.
func buildFontWith(header string, height int, art map[rune][]string) string {
	var b strings.Builder
	b.WriteString(header + "\n")
	b.WriteString("comment\n")
	for _, c := range requiredChars {
		g := art[c]
		for r := 0; r < height; r++ {
			cell := "#"
			if r < len(g) {
				cell = g[r]
			}
			mark := "@"
			if r == height-1 {
				mark = "@@"
			}
			b.WriteString(cell + mark + "\n")
		}
	}
	return b.String()
}

// miniFont is a height-2, full-width font whose '!' glyph is a column of "X"
// and whose space glyph is two hardblanks.
var miniFont = buildFontWith("flf2a$ 2 2 4 -1 1", 2, map[rune][]string{
	' ': {"$$", "$$"},
	'!': {"X", "X"},
})

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
	// oldLayout 1 = smushing with the equal-character rule (bit 1), which
	// resolves to controlled smushing because a rule is enabled.
	f, err := ParseFont(strings.NewReader(buildFontWith("flf2a$ 1 1 3 1 1", 1, nil)))
	if err != nil {
		t.Fatal(err)
	}
	r := f.FittingRules()
	if r.HorizontalLayout != layoutControlledSmushing || !r.HorizontalRules[0] {
		t.Fatalf("expected controlled smushing with the equal-character rule, got %+v", r)
	}
}

func TestEqualCharSmushing(t *testing.T) {
	// A font whose '!' glyph is a single "|" column, height 1, with
	// equal-character smushing. Two '!' side by side smush into one "|".
	f, err := ParseFont(strings.NewReader(buildFontWith("flf2a$ 1 1 3 1 1", 1, map[rune][]string{
		' ': {"$"},
		'!': {"|"},
	})))
	if err != nil {
		t.Fatal(err)
	}
	if out := f.Render("!!"); out != "|" {
		t.Fatalf("equal-char smush = %q, want \"|\"", out)
	}
}
