package figlet

import (
	"fmt"
	"strings"
	"testing"
)

// buildFont assembles a minimal but complete font: a header, one comment line,
// and a glyph for every character the format requires, at the given height.
func buildFont(header string, height int) string {
	return buildFontWith(header, height, nil)
}

// TestHeaderRejectsNonNumericFields covers the strict-parsing fix. The header
// numbers used to be read with fmt.Sscanf("%d") and the error discarded, so a
// non-numeric field silently became 0 and "5abc" silently became 5 — producing a
// font that renders wrongly instead of an error the caller can act on.
func TestHeaderRejectsNonNumericFields(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"non-numeric height", "flf2a$ x 1 5 0 1"},
		{"trailing garbage in height", "flf2a$ 1abc 1 5 0 1"},
		{"trailing garbage in comment count", "flf2a$ 1 1 5 0 1x"},
		{"float height", "flf2a$ 1.5 1 5 0 1"},
		{"empty field list", "flf2a$"},
		{"negative comment count", "flf2a$ 1 1 5 0 -1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseFont(strings.NewReader(c.header + "\ncomment\n"))
			if err == nil {
				t.Fatalf("ParseFont(%q) = nil error, want a parse error", c.header)
			}
		})
	}
}

// TestHeaderHeightIsBounded is the resource-exhaustion guard: the declared
// height is attacker-controlled data that sizes allocations, so a font claiming
// to be two billion rows tall must be rejected outright rather than parsed.
func TestHeaderHeightIsBounded(t *testing.T) {
	for _, h := range []string{"2147483647", "1000000", "1001"} {
		header := "flf2a$ " + h + " 1 5 0 1"
		_, err := ParseFont(strings.NewReader(header + "\ncomment\n"))
		if err == nil {
			t.Fatalf("height %s accepted, want rejection", h)
		}
		if !strings.Contains(err.Error(), "maximum") {
			t.Errorf("height %s error = %v, want the height-limit error", h, err)
		}
	}
	// The boundary value itself is still accepted.
	if _, err := ParseFont(strings.NewReader(buildFont(fmt.Sprintf("flf2a$ %d 1 5 0 1", MaxFontHeight), MaxFontHeight))); err != nil {
		t.Errorf("height %d rejected: %v", MaxFontHeight, err)
	}
}

// TestMultiByteHardblank checks that a non-ASCII hardblank is decoded as one
// rune. Reading a single byte split the character and left its tail in the
// numeric field list, which then failed to parse.
func TestMultiByteHardblank(t *testing.T) {
	f, err := ParseFont(strings.NewReader(buildFont("flf2a§ 1 1 5 0 1", 1)))
	if err != nil {
		t.Fatalf("ParseFont with § hardblank: %v", err)
	}
	if f.hardblank != '§' {
		t.Errorf("hardblank = %q, want '§'", f.hardblank)
	}
}

// TestParseCharCodeStrict pins the strict code-tag parse. fmt.Sscanf stopped at
// the first unusable character, so "0x41zzz" was read as 0x41 and the following
// rows were consumed as a glyph; a garbage line in the middle of a font was
// silently treated as a code tag.
func TestParseCharCodeStrict(t *testing.T) {
	ok := []struct {
		in   string
		want rune
	}{
		{"65 A", 'A'},
		{"0x41 A", 'A'},
		{"0X41 A", 'A'},
		{"0101 A", 'A'}, // octal, as C notation and the FIGfont spec allow
		{"+65 A", 'A'},
		{"-2 negative", -2}, // negative codes are legal and simply never match
		{"1114111 max", 0x10ffff},
	}
	for _, c := range ok {
		got, err := parseCharCode(c.in)
		if err != nil {
			t.Errorf("parseCharCode(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseCharCode(%q) = %d, want %d", c.in, got, c.want)
		}
	}

	bad := []string{
		"", "  ", "notacode", "12abc", "0x41zzz", "0xzz", "0x", "-", "+",
		"1114112 past max rune", "99999999999999999999",
		// The FIGfont spec singles out -1 as forbidden.
		"-1 deleted", "-0x1",
	}
	for _, in := range bad {
		if got, err := parseCharCode(in); err == nil {
			t.Errorf("parseCharCode(%q) = %d, want an error", in, got)
		}
	}
}

// TestMalformedFontsNeverPanic feeds a pile of broken inputs through the whole
// parse-and-render path. Any of them may return an error; none may panic.
func TestMalformedFontsNeverPanic(t *testing.T) {
	inputs := []string{
		"",
		"flf2a",
		"flf2a$",
		"flf2a$ ",
		"flf2a$ 1",
		"flf2a$ 1 1 5 0 0",
		"flf2a$ 1 1 5 0 0\n",
		"flf2a$ 2 1 5 0 0\nonly one row\n",
		"flf2a$ 1 1 5 0 0\n@\n@\n@\n",
		"flf2a$ 1 1 5 -1 0\n@@@\n",
		"flf2a$ 1 1 5 0 0 0 -1 0\n@\n",
		"flf2a$ 1 1 5 0 0 0 999999999 0\n@\n",
		"flf2a\xff 1 1 5 0 0\n@\n",
		"flf2a$ 1 1 5 0 0\n\x00\x00\x00\n",
		buildFont("flf2a$ 1 1 5 0 1", 1) + "0x110000\nX@\n",
		buildFont("flf2a$ 1 1 5 0 1", 1) + "-0x1\nX@\n",
		buildFont("flf2a$ 3 2 5 15 1", 3),
		strings.Repeat("flf2a$ 1 1 5 0 0\n", 3),
	}
	for i, in := range inputs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			f, err := ParseFont(strings.NewReader(in))
			if err != nil {
				return
			}
			if f == nil {
				t.Fatal("ParseFont returned nil font and nil error")
			}
			// Rendering a parsed-but-odd font must also be panic-free.
			for _, layout := range []Layout{LayoutDefault, LayoutFull, LayoutKerning, LayoutSmush} {
				_ = f.Render("Hello, World! 123", Options{Layout: layout})
				_ = f.Render("wrap me please", Options{Layout: layout, Width: 10})
			}
		})
	}
}

// TestRenderWidthWraps covers Options.Width and Options.WhitespaceBreak.
func TestRenderWidthWraps(t *testing.T) {
	plain := Render("one two")
	natural := widestRow(plain)

	// A generous width leaves the line alone.
	if got := Render("one two", Options{Width: natural + 10}); got != plain {
		t.Errorf("a width wider than the text changed the output")
	}
	// Zero and negative widths mean "no wrapping".
	for _, w := range []int{0, -5} {
		if got := Render("one two", Options{Width: w}); got != plain {
			t.Errorf("Width=%d changed the output", w)
		}
	}
	// A width below the natural width must split the text over more rows.
	for _, wsb := range []bool{false, true} {
		wrapped := Render("one two", Options{Width: natural - 1, WhitespaceBreak: wsb})
		if rows(wrapped) <= rows(plain) {
			t.Errorf("WhitespaceBreak=%v: wrapping produced %d rows, want more than %d",
				wsb, rows(wrapped), rows(plain))
		}
		if got := widestRow(wrapped); got >= natural {
			t.Errorf("WhitespaceBreak=%v: wrapped width = %d, want < %d", wsb, got, natural)
		}
		// A width no glyph can fit must still terminate and still emit ink.
		if tiny := Render("one two", Options{Width: 1, WhitespaceBreak: wsb}); !strings.Contains(tiny, "#") {
			t.Errorf("WhitespaceBreak=%v: Width=1 produced no output", wsb)
		}
	}
}

// widestRow measures the widest row of rendered output.
func widestRow(s string) int {
	w := 0
	for _, line := range strings.Split(s, "\n") {
		if n := len([]rune(line)); n > w {
			w = n
		}
	}
	return w
}

// rows counts the rows of rendered output.
func rows(s string) int { return len(strings.Split(s, "\n")) }

// TestRenderIsIdempotent is a regression test for block aliasing: the assembly
// keeps earlier blocks alive while it builds candidates (see the wrapping path),
// so a merge that wrote into its input corrupted them and, with a font whose
// glyphs are shared with the registry, corrupted the font itself.
func TestRenderIsIdempotent(t *testing.T) {
	f := BuiltinFont()
	for _, opts := range []Options{{}, {Layout: LayoutFull}, {Width: 12}, {Width: 12, WhitespaceBreak: true}} {
		first := f.Render("one two three", opts)
		if second := f.Render("one two three", opts); second != first {
			t.Errorf("Render(%+v) is not idempotent:\n%q\nbecame\n%q", opts, first, second)
		}
	}
}
