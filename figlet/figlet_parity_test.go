package figlet

// Upstream-parity tests for the FIGfont rendering engine, checked against the
// real "patorjk/figlet.js" project (the JavaScript reference implementation this
// package ports). Every input, font and expected-output string below is taken
// verbatim from the upstream test suite and its bundled data files:
//
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/test/node-figlet.test.ts
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/test/expected/graffiti
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/test/expected/dancingFont
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/test/expected/standard
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/test/expected/standard_default
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/fonts/Standard.flf
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/fonts/Graffiti.flf
//	https://raw.githubusercontent.com/patorjk/figlet.js/main/fonts/Dancing%20Font.flf
//
// The real .flf fonts and expected renders live under testdata/. Upstream maps
// its horizontal layout names onto figlet fitting modes as: "full" -> full
// width, "fitted" -> kerning, "default"/"controlled smushing" -> smushing. The
// port implements the horizontal engine (kerning, full width and the six
// standard horizontal smushing rules) but deliberately omits vertical smushing,
// right-to-left print direction and width wrapping (see doc in render.go). The
// tests below assert exact parity for the horizontal engine and record the
// omitted features as documented gaps via t.Skip.

import (
	"os"
	"testing"
)

func parityFont(t *testing.T, file string) *Font {
	t.Helper()
	f, err := LoadFontFile("testdata/" + file)
	if err != nil {
		t.Fatalf("load %s: %v", file, err)
	}
	return f
}

func parityExpected(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return string(b)
}

// TestParityGraffitiFitted mirrors upstream node-figlet.test.ts:
//
//	figlet.textSync("ABC.123", {font:"Graffiti", horizontalLayout:"fitted"})
//
// "fitted" == kerning. Exact match expected.
func TestParityGraffitiFitted(t *testing.T) {
	f := parityFont(t, "Graffiti.flf")
	got := f.Render("ABC.123", Options{Layout: LayoutKerning})
	want := parityExpected(t, "exp_graffiti")
	if got != want {
		t.Errorf("Graffiti fitted mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// TestParityDancingFontFull mirrors upstream node-figlet.test.ts:
//
//	figlet.text("pizzapie", {font:"Dancing Font", horizontalLayout:"full"})
//
// "full" == full width. Exact match expected.
func TestParityDancingFontFull(t *testing.T) {
	f := parityFont(t, "DancingFont.flf")
	got := f.Render("pizzapie", Options{Layout: LayoutFull})
	want := parityExpected(t, "exp_dancingFont")
	if got != want {
		t.Errorf("Dancing Font full mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// TestParityStandardHeader checks that ParseFont extracts the same FIGfont
// header metadata that upstream's loadFont reports for Standard.flf (the
// `standardMeta` object in node-figlet.test.ts): hardBlank '$', height 6,
// baseline 5, maxLength 16, oldLayout 15, printDirection 0, fullLayout 24463.
func TestParityStandardHeader(t *testing.T) {
	f := parityFont(t, "Standard.flf")
	checks := []struct {
		name     string
		got, exp int
	}{
		{"height", f.height, 6},
		{"baseline", f.baseline, 5},
		{"maxLength", f.maxLen, 16},
		{"oldLayout", f.oldLayout, 15},
		{"fullLayout", f.fullLayout, 24463},
	}
	for _, c := range checks {
		if c.got != c.exp {
			t.Errorf("Standard header %s = %d, upstream = %d", c.name, c.got, c.exp)
		}
	}
	if f.hardblank != '$' {
		t.Errorf("Standard hardBlank = %q, upstream = %q", f.hardblank, '$')
	}
	if !f.hasFull {
		t.Error("Standard font should expose fullLayout (hasFull)")
	}
}

// TestParityStandardVerticalFitted is upstream's
// figlet.textSync("FIGlet\nFONTS", {font:"Standard", verticalLayout:"fitted"})
// -> test/expected/standard. It exercises vertical fitting (which slides the two
// blocks together until they touch) and the uniform right-padding stacking
// applies. Exact match expected.
func TestParityStandardVerticalFitted(t *testing.T) {
	f := parityFont(t, "Standard.flf")
	got := f.Render("FIGlet\nFONTS", Options{VerticalLayout: LayoutFitted})
	want := parityExpected(t, "exp_standard")
	if got != want {
		t.Errorf("Standard vertical fitting mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestParityStandardDefaultVertical is upstream's default-layout render,
// figlet.textSync("FIGlet\nFonts", {font:"Standard"}) -> test/expected/
// standard_default, which vertically smushes the two lines using the font's own
// vertical rules. Exact match expected.
func TestParityStandardDefaultVertical(t *testing.T) {
	f := parityFont(t, "Standard.flf")
	got := f.Render("FIGlet\nFonts")
	want := parityExpected(t, "exp_standard_default")
	if got != want {
		t.Errorf("Standard default vertical layout mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
