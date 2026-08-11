package figlet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malcolmston/chalk"
)

// --- ParseFont error paths ---------------------------------------------------

func TestParseFontErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"bad signature", "notaflf 1 1 1 0 0\n"},
		{"malformed header", "flf2a\n"},
		{"incomplete header", "flf2a$ 1 1\n"},
		{"invalid height", "flf2a$ 0 0 1 0 0\ncomment\n"},
		{"truncated comment", "flf2a$ 1 1 1 0 3\nonly one comment line\n"},
	}
	for _, c := range cases {
		if _, err := ParseFont(strings.NewReader(c.in)); err == nil {
			t.Errorf("%s: expected error, got nil", c.name)
		}
	}
}

// TestParseFontFullLayout exercises the >=7 field header branch that sets the
// fullLayout / hasFull fields.
func TestParseFontFullLayout(t *testing.T) {
	// Smush bit (128) set plus rule bits -> controlled smushing under default.
	f, err := ParseFont(strings.NewReader(buildFontWith("flf2a$ 1 1 5 0 1 0 191 0", 1, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if !f.hasFull {
		t.Fatal("hasFull not set for 8-field header")
	}
	if r := f.FittingRules(); r.HorizontalLayout != layoutControlledSmushing {
		t.Fatalf("horizontal layout = %d, want controlled smushing; rules = %+v", r.HorizontalLayout, r)
	}
}

// TestParseFontCodeTags appends a code-tagged glyph to a complete font so the
// optional code-tag parsing loop runs.
func TestParseFontCodeTags(t *testing.T) {
	base := buildFontWith("flf2a$ 1 1 5 0 1", 1, nil)

	f, err := ParseFont(strings.NewReader(base + "0x2603 SNOWMAN\n\u2603@@\n"))
	if err != nil {
		t.Fatal(err)
	}
	if g, ok := f.chars['\u2603']; !ok {
		t.Fatal("code-tagged glyph 0x2603 not parsed")
	} else if g[0] != "\u2603" {
		t.Fatalf("code-tagged glyph = %q, want a snowman", g[0])
	}

	// A blank line ends the glyph table; anything after it is ignored.
	f, err = ParseFont(strings.NewReader(base + "\n0x2603 SNOWMAN\n\u2603@@\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.chars['\u2603']; ok {
		t.Error("a glyph after the terminating blank line should not be read")
	}

	// A line that is neither blank nor a code tag is a parse error, not a
	// silent stop: treating it as the end of the font quietly discarded every
	// glyph after it.
	if _, err := ParseFont(strings.NewReader(base + "junk!\nX@@\n")); err == nil {
		t.Error("a non-numeric code tag should be an error")
	}
}

// --- parseCharCode -----------------------------------------------------------

func TestParseCharCode(t *testing.T) {
	ok := []struct {
		in   string
		want rune
	}{
		{"196 LATIN", 196},
		{"0xC4 x", 196},
		{"-0x2 neg", -2},
		{"32", 32},
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

	// -1 is explicitly not a permitted character code.
	bad := []string{"", "notacode", "12abc", "-1", "-0x1"}
	for _, in := range bad {
		if _, err := parseCharCode(in); err == nil {
			t.Errorf("parseCharCode(%q): expected error", in)
		}
	}
}

// --- removeEndChar -----------------------------------------------------------

func TestRemoveEndChar(t *testing.T) {
	cases := []struct {
		in            string
		lineNum, hght int
		want          string
	}{
		{"", 0, 1, ""},
		{"X@", 0, 1, "X"},
		{"XX@@", 1, 2, "XX"},  // final row: one or two end marks
		{"XX@@", 0, 2, "XX@"}, // other rows: exactly one
		{"@@", 0, 1, ""},      // a row that is nothing but end marks
		{"X@@@", 0, 1, "X@"},  // only the trailing pair is the mark
		{"abc#", 0, 1, "abc"}, // the end mark is whatever the row ends with
		{"X@   ", 0, 1, "X"},  // TOIlet fonts pad after the end mark
		{"   ", 0, 1, "   "},  // a blank row has no end mark to remove
	}
	for _, c := range cases {
		if got := removeEndChar(c.in, c.lineNum, c.hght); got != c.want {
			t.Errorf("removeEndChar(%q, %d, %d) = %q, want %q", c.in, c.lineNum, c.hght, got, c.want)
		}
	}
}

// --- getSmushingRules --------------------------------------------------------

func TestGetSmushingRules(t *testing.T) {
	full := func(n int) *int { return &n }

	cases := []struct {
		name      string
		old       int
		fl        *int
		wantH     int
		wantV     int
		wantHRule [6]bool
		wantVRule [5]bool
	}{
		{"old -1 is full width", -1, nil, layoutFullWidth, layoutFullWidth, [6]bool{}, [5]bool{}},
		{"old 0 is fitting", 0, nil, layoutFitting, layoutFullWidth, [6]bool{}, [5]bool{}},
		{"old 1 is controlled smushing", 1, nil, layoutControlledSmushing, layoutFullWidth,
			[6]bool{true}, [5]bool{}},
		{"old 15 enables rules 1-4", 15, nil, layoutControlledSmushing, layoutFullWidth,
			[6]bool{true, true, true, true}, [5]bool{}},
		{"full 64 is fitting", 0, full(64), layoutFitting, layoutFullWidth, [6]bool{}, [5]bool{}},
		{"full 128 is universal smushing", 0, full(128), layoutSmushing, layoutFullWidth,
			[6]bool{}, [5]bool{}},
		{"full 129 is controlled smushing", 0, full(129), layoutControlledSmushing, layoutFullWidth,
			[6]bool{true}, [5]bool{}},
		// 24463 is the Standard font: vertical smushing plus every vertical rule
		// and horizontal rules 1-4.
		{"full 24463 (Standard)", 15, full(24463), layoutControlledSmushing, layoutControlledSmushing,
			[6]bool{true, true, true, true}, [5]bool{true, true, true, true, true}},
	}
	for _, c := range cases {
		r := getSmushingRules(c.old, c.fl)
		if r.hLayout != c.wantH || r.vLayout != c.wantV {
			t.Errorf("%s: layouts = h%d v%d, want h%d v%d", c.name, r.hLayout, r.vLayout, c.wantH, c.wantV)
		}
		if r.hRule != c.wantHRule || r.vRule != c.wantVRule {
			t.Errorf("%s: rules = h%v v%v, want h%v v%v", c.name, r.hRule, r.vRule, c.wantHRule, c.wantVRule)
		}
	}
}

// --- smushing rules ----------------------------------------------------------

func TestHorizontalSmushRules(t *testing.T) {
	all := fittingRules{hLayout: layoutControlledSmushing, hRule: [6]bool{true, true, true, true, true, true}}
	only := func(i int) fittingRules {
		r := fittingRules{hLayout: layoutControlledSmushing}
		r.hRule[i] = true
		return r
	}

	cases := []struct {
		name string
		a, b rune
		r    fittingRules
		want rune // 0 means "no rule applies"
	}{
		{"rule1 equal", '|', '|', only(0), '|'},
		{"rule1 skips hardblanks", '$', '$', only(0), 0},
		{"rule2 underscore left", '_', '|', only(1), '|'},
		{"rule2 underscore right", '|', '_', only(1), '|'},
		{"rule3 hierarchy", '|', '/', only(2), '/'},
		{"rule3 hierarchy reversed", '/', '|', only(2), '/'},
		{"rule3 needs a gap of two", '/', '\\', only(2), 0},
		{"rule4 opposite pair", '[', ']', only(3), '|'},
		{"rule4 same bracket", '[', '[', only(3), '|'},
		{"rule4 unrelated", '[', '(', only(3), 0},
		{"rule5 slash", '/', '\\', only(4), '|'},
		{"rule5 backslash", '\\', '/', only(4), 'Y'},
		{"rule5 gt lt", '>', '<', only(4), 'X'},
		{"rule5 lt gt is not smushed", '<', '>', only(4), 0},
		{"rule6 both hardblanks", '$', '$', only(5), '$'},
		{"rule6 one hardblank", '$', 'X', only(5), 0},
		{"no rule matches", 'A', 'B', all, 0},
	}
	for _, c := range cases {
		got, ok := hRuleSmush(c.a, c.b, '$', c.r)
		if !ok {
			got = 0
		}
		if got != c.want {
			t.Errorf("%s: hRuleSmush(%q,%q) = %q, want %q", c.name, c.a, c.b, got, c.want)
		}
	}
}

func TestUniversalSmush(t *testing.T) {
	cases := []struct {
		a, b, want rune
	}{
		{'A', ' ', 'A'}, // a space never overwrites ink
		{' ', 'B', 'B'}, // ink overwrites a space
		{'A', 'B', 'B'}, // otherwise the later character wins
		{'A', '$', 'A'}, // a hardblank never overwrites ink
		{' ', '$', '$'}, // but it does overwrite a blank
	}
	for _, c := range cases {
		if got := uniSmush(c.a, c.b, '$'); got != c.want {
			t.Errorf("uniSmush(%q,%q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestVerticalSmushRules(t *testing.T) {
	if got, ok := vRule4Smush('-', '_'); !ok || got != '=' {
		t.Errorf("vRule4Smush('-','_') = %q,%v, want '=',true", got, ok)
	}
	if got, ok := vRule4Smush('_', '-'); !ok || got != '=' {
		t.Errorf("vRule4Smush('_','-') = %q,%v, want '=',true", got, ok)
	}
	if _, ok := vRule4Smush('-', '-'); ok {
		t.Error("vRule4Smush smushes identical lines, which is rule 1's job")
	}
	if got, ok := vRule5Smush('|', '|'); !ok || got != '|' {
		t.Errorf("vRule5Smush('|','|') = %q,%v, want '|',true", got, ok)
	}
	if _, ok := vRule1Smush('a', 'b'); ok {
		t.Error("vRule1Smush matched unequal characters")
	}
}

// --- glyphFor fallbacks ------------------------------------------------------

func TestGlyphFor(t *testing.T) {
	f := &Font{
		height:   1,
		fallback: true,
		chars: map[rune][]string{
			'A': {"AA"},
			' ': {"  "},
		},
	}
	// Direct hit.
	if g := f.glyphFor('A'); g == nil || g[0] != "AA" {
		t.Error("direct glyph lookup failed")
	}
	// Lowercase falls back to uppercase.
	if g := f.glyphFor('a'); g == nil || g[0] != "AA" {
		t.Error("lowercase should fall back to uppercase")
	}
	// Unknown char falls back to the space glyph.
	if g := f.glyphFor('Z'); g == nil || g[0] != "  " {
		t.Error("unknown char should fall back to space glyph")
	}

	// A font with no space glyph returns nil for unknown chars.
	noSpace := &Font{height: 1, fallback: true, chars: map[rune][]string{'A': {"AA"}}}
	if g := noSpace.glyphFor('Z'); g != nil {
		t.Errorf("expected nil for unknown char with no space glyph, got %v", g)
	}
	// Render skips nil glyphs without panicking, and produces no ink at all.
	if out := noSpace.Render("ZZ"); strings.TrimSpace(out) != "" {
		t.Errorf("Render with only-skipped glyphs = %q", out)
	}

	// A font parsed from a .flf has no fallback: an undefined character is
	// skipped outright, exactly as the reference implementation does.
	strict := &Font{height: 1, chars: map[rune][]string{'A': {"AA"}, ' ': {"  "}}}
	if g := strict.glyphFor('a'); g != nil {
		t.Error("a parsed font must not fall back to the uppercase glyph")
	}
	if g := strict.glyphFor('Z'); g != nil {
		t.Error("a parsed font must not fall back to the space glyph")
	}
}

// --- LoadFont / LoadFontFile / LoadFontDir -----------------------------------

func TestLoadFontReader(t *testing.T) {
	f, err := LoadFont(strings.NewReader(miniFont))
	if err != nil {
		t.Fatal(err)
	}
	if f.Height() != 2 {
		t.Fatalf("height = %d, want 2", f.Height())
	}
}

func TestLoadFontFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mini.flf")
	if err := os.WriteFile(path, []byte(miniFont), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFontFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Height() != 2 {
		t.Fatalf("height = %d, want 2", f.Height())
	}
	// Missing file errors.
	if _, err := LoadFontFile(filepath.Join(dir, "nope.flf")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFontDir(t *testing.T) {
	dir := t.TempDir()
	// A valid font.
	if err := os.WriteFile(filepath.Join(dir, "CovMini.flf"), []byte(miniFont), 0o644); err != nil {
		t.Fatal(err)
	}
	// A non-font file (skipped by extension).
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An invalid .flf (skipped because it fails to parse).
	if err := os.WriteFile(filepath.Join(dir, "broken.flf"), []byte("not a font"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A subdirectory (skipped).
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFontDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0] != "CovMini" {
		t.Fatalf("loaded = %v, want [CovMini]", loaded)
	}
	// The font registered under its base name (case-insensitive lookup).
	if _, ok := GetFont("covmini"); !ok {
		t.Fatal("loaded font not registered")
	}

	// A missing directory errors.
	if _, err := LoadFontDir(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Fatal("expected error for missing dir")
	}
}

// --- unknownFontError.Error --------------------------------------------------

func TestUnknownFontError(t *testing.T) {
	_, err := RenderFont("definitely-not-a-font", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "definitely-not-a-font") {
		t.Errorf("error message = %q, want it to name the font", err.Error())
	}
}

// --- color helpers -----------------------------------------------------------

func TestHexRGB(t *testing.T) {
	cases := []struct {
		in      string
		r, g, b int
	}{
		{"#ff8800", 255, 136, 0},
		{"ff8800", 255, 136, 0},
		{"#f80", 255, 136, 0}, // shorthand
		{"#FFFFFF", 255, 255, 255},
		{"zz", 255, 255, 255},     // invalid length -> white default
		{"#12345", 255, 255, 255}, // 5 chars -> white default
	}
	for _, c := range cases {
		r, g, b := hexRGB(c.in)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("hexRGB(%q) = %d,%d,%d want %d,%d,%d", c.in, r, g, b, c.r, c.g, c.b)
		}
	}
}

func TestGradientAndRainbowSingleColumn(t *testing.T) {
	chalk.SetLevel(chalk.LevelTrueColor)
	defer chalk.SetLevel(chalk.LevelNone)

	// A single narrow line still renders without dividing by zero.
	if got := Gradient("A", "#ff0000", "#0000ff"); chalk.Strip(got) != "A" {
		t.Errorf("gradient single = %q", chalk.Strip(got))
	}
	if got := Rainbow(""); got != "" {
		t.Errorf("rainbow empty = %q", got)
	}
	// Multi-line banner keeps its line structure after coloring.
	banner := "AB\nCD"
	if chalk.Strip(Rainbow(banner)) != banner {
		t.Errorf("rainbow multiline strip mismatch")
	}
	if chalk.Strip(Gradient(banner, "#000000", "#ffffff")) != banner {
		t.Errorf("gradient multiline strip mismatch")
	}
}

func TestHSVToRGBSpectrum(t *testing.T) {
	// Sample each 60-degree sector so every branch of hsvToRGB runs.
	for _, h := range []float64{0, 30, 90, 150, 210, 270, 330} {
		r, g, b := hsvToRGB(h, 1, 1)
		if r < 0 || r > 255 || g < 0 || g > 255 || b < 0 || b > 255 {
			t.Errorf("hsvToRGB(%v) out of range: %d,%d,%d", h, r, g, b)
		}
	}
}

// --- layout rendering paths --------------------------------------------------

// TestRenderLayouts drives every horizontal layout through the built-in font.
func TestRenderLayouts(t *testing.T) {
	for _, layout := range []Layout{LayoutFull, LayoutFitted, LayoutControlledSmush, LayoutUniversalSmush} {
		out := Render("AB", Options{Layout: layout})
		lines := strings.Split(out, "\n")
		if len(lines) != 5 {
			t.Errorf("layout %v: got %d rows, want 5", layout, len(lines))
		}
		w := len([]rune(lines[0]))
		for i, l := range lines {
			if len([]rune(l)) != w {
				t.Errorf("layout %v: row %d width %d != %d", layout, i, len([]rune(l)), w)
			}
		}
	}
}
