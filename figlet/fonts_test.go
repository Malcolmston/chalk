package figlet

import (
	"strings"
	"testing"
)

// TestBundledDistinctFonts checks the hand-authored, genuinely-distinct bundled
// fonts (as opposed to the fill-character variants of the block font) are
// registered and render at their declared heights.
func TestBundledDistinctFonts(t *testing.T) {
	cases := []struct {
		name   string
		height int
	}{
		{"small", 3},
		{"mini", 3},
		{"banner", 5},
		{"outline", 5},
	}
	for _, c := range cases {
		f, ok := GetFont(c.name)
		if !ok {
			t.Errorf("font %q not registered", c.name)
			continue
		}
		if f.Height() != c.height {
			t.Errorf("font %q height = %d, want %d", c.name, f.Height(), c.height)
		}
		out, err := RenderFont(c.name, "ABC XYZ 123")
		if err != nil {
			t.Errorf("RenderFont(%q): %v", c.name, err)
			continue
		}
		lines := strings.Split(out, "\n")
		if len(lines) != c.height {
			t.Errorf("font %q rendered %d lines, want %d:\n%s", c.name, len(lines), c.height, out)
		}
	}
}

// TestBundledFontsCoverAlphanumerics verifies each distinct font defines a glyph
// for every uppercase letter and digit (so nothing renders as blank).
func TestBundledFontsCoverAlphanumerics(t *testing.T) {
	for _, name := range []string{"small", "banner"} {
		f, _ := GetFont(name)
		for ch := 'A'; ch <= 'Z'; ch++ {
			if _, ok := f.chars[ch]; !ok {
				t.Errorf("font %q missing glyph %q", name, string(ch))
			}
		}
		for ch := '0'; ch <= '9'; ch++ {
			if _, ok := f.chars[ch]; !ok {
				t.Errorf("font %q missing digit %q", name, string(ch))
			}
		}
	}
}

// TestSmallFontWordSpacing confirms the hardblank-based gutter keeps a space in
// the output (words don't fuse under kerning).
func TestSmallFontWordSpacing(t *testing.T) {
	out, err := RenderFont("small", "A B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "  ") {
		t.Errorf("expected a visible word gap in small-font output:\n%s", out)
	}
}
