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
	// Smush bit (128) set plus rule bits -> LayoutSmush under default.
	font := "flf2a$ 1 1 5 0 1 0 191 0\ncomment\n$@\n|@\n"
	f, err := ParseFont(strings.NewReader(font))
	if err != nil {
		t.Fatal(err)
	}
	if !f.hasFull {
		t.Fatal("hasFull not set for 8-field header")
	}
	if layout, rules := f.resolveLayout(LayoutDefault); layout != LayoutSmush {
		t.Fatalf("layout = %v, want smush; rules=%d", layout, rules)
	}
}

// TestParseFontCodeTags builds a complete 95-glyph font followed by a
// code-tagged glyph so the optional code-tag parsing loop runs.
func TestParseFontCodeTags(t *testing.T) {
	var b strings.Builder
	b.WriteString("flf2a$ 1 1 5 0 1\n") // height 1, 1 comment line
	b.WriteString("a comment\n")
	for c := 32; c <= 126; c++ { // required glyphs 32..126
		b.WriteString("X@\n")
	}
	b.WriteString("\n")      // blank line -> exercises the continue branch
	b.WriteString("0xC4\n")  // code tag (196) in hex
	b.WriteString("Y@\n")    // its glyph
	b.WriteString("junk!\n") // not a code tag -> loop breaks

	f, err := ParseFont(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	if g, ok := f.chars[rune(0xC4)]; !ok {
		t.Fatalf("code-tagged glyph 0xC4 not parsed")
	} else if g[0] != "Y" {
		t.Fatalf("code-tagged glyph = %q, want \"Y\"", g[0])
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
		{"-0x1 neg", -1},
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

	bad := []string{"", "notacode", "12abc"}
	for _, in := range bad {
		if _, err := parseCharCode(in); err == nil {
			t.Errorf("parseCharCode(%q): expected error", in)
		}
	}
}

// --- stripEndmark ------------------------------------------------------------

func TestStripEndmark(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"X@", "X"},
		{"XX@@", "XX"}, // double end-mark
		{"@@", ""},     // all end-mark
		{"abc#", "abc"},
	}
	for _, c := range cases {
		if got := stripEndmark(c.in); got != c.want {
			t.Errorf("stripEndmark(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- resolveLayout -----------------------------------------------------------

func TestResolveLayout(t *testing.T) {
	cases := []struct {
		name       string
		f          *Font
		requested  Layout
		wantLayout Layout
	}{
		{"explicit full", &Font{oldLayout: 5}, LayoutFull, LayoutFull},
		{"explicit smush hasFull", &Font{hasFull: true, fullLayout: 0xFF}, LayoutSmush, LayoutSmush},
		{"explicit kerning negative rules", &Font{oldLayout: -1}, LayoutKerning, LayoutKerning},
		{"default hasFull smush", &Font{hasFull: true, fullLayout: 128 | 1}, LayoutDefault, LayoutSmush},
		{"default hasFull kerning", &Font{hasFull: true, fullLayout: 64}, LayoutDefault, LayoutKerning},
		{"default hasFull full", &Font{hasFull: true, fullLayout: 0}, LayoutDefault, LayoutFull},
		{"default old full", &Font{oldLayout: -1}, LayoutDefault, LayoutFull},
		{"default old kerning", &Font{oldLayout: 0}, LayoutDefault, LayoutKerning},
		{"default old smush", &Font{oldLayout: 1}, LayoutDefault, LayoutSmush},
	}
	for _, c := range cases {
		layout, _ := c.f.resolveLayout(c.requested)
		if layout != c.wantLayout {
			t.Errorf("%s: layout = %v, want %v", c.name, layout, c.wantLayout)
		}
	}
}

// --- smushem rules -----------------------------------------------------------

func TestSmushem(t *testing.T) {
	f := &Font{hardblank: '$'}

	cases := []struct {
		name  string
		a, b  rune
		smush bool
		rules int
		want  rune
	}{
		{"space-left", ' ', 'X', true, 0, 'X'},
		{"space-right", 'X', ' ', true, 0, 'X'},
		{"both-hardblank-rule32", '$', '$', true, 32, '$'},
		{"hardblank-no-rule", '$', 'X', true, 1, 0},
		{"no-smush", 'A', 'B', false, 1, 0},
		{"universal", 'A', 'B', true, 0, 'B'},
		{"rule1-equal", '|', '|', true, 1, '|'},
		{"rule2-underscore-left", '_', '|', true, 2, '|'},
		{"rule2-underscore-right", '|', '_', true, 2, '|'},
		{"rule4-hierarchy", '|', '/', true, 4, '/'}, // '/' outranks '|'
		{"rule4-hierarchy-rev", '/', '|', true, 4, '/'},
		{"rule8-opposite", '[', ']', true, 8, '|'},
		{"rule16-slash", '/', '\\', true, 16, '|'},
		{"rule16-backslash", '\\', '/', true, 16, 'Y'},
		{"rule16-gtlt", '>', '<', true, 16, 'X'},
		{"no-match", 'A', 'B', true, 1, 0}, // rule1 only, unequal -> 0
	}
	for _, c := range cases {
		if got := f.smushem(c.a, c.b, c.smush, c.rules); got != c.want {
			t.Errorf("%s: smushem(%q,%q,%v,%d) = %q, want %q",
				c.name, c.a, c.b, c.smush, c.rules, got, c.want)
		}
	}
}

func TestRankAndOpposite(t *testing.T) {
	if rank('|') != 1 || rank('/') != 2 || rank('[') != 3 || rank('{') != 4 || rank('(') != 5 || rank('<') != 6 {
		t.Error("rank returned unexpected values")
	}
	if rank('A') != 0 {
		t.Error("rank of non-bracket should be 0")
	}
	if !isOppositePair('[', ']') || !isOppositePair('(', ')') || !isOppositePair('{', '}') {
		t.Error("isOppositePair missed a real pair")
	}
	if isOppositePair('[', '}') {
		t.Error("isOppositePair matched a mismatched pair")
	}
}

// --- glyphFor fallbacks ------------------------------------------------------

func TestGlyphFor(t *testing.T) {
	f := &Font{
		height: 1,
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
	noSpace := &Font{height: 1, chars: map[rune][]string{'A': {"AA"}}}
	if g := noSpace.glyphFor('Z'); g != nil {
		t.Errorf("expected nil for unknown char with no space glyph, got %v", g)
	}
	// renderLine skips nil glyphs without panicking.
	if out := noSpace.renderLine("ZZ", LayoutFull, 0); out != "" {
		// no defined glyphs -> empty row
		if strings.TrimSpace(out) != "" {
			t.Errorf("renderLine with only-skipped glyphs = %q", out)
		}
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

// TestRenderLayouts drives kerning and full-width smushing merges through the
// built-in font, covering merge/overlap/rowOverlap paths.
func TestRenderLayouts(t *testing.T) {
	for _, layout := range []Layout{LayoutFull, LayoutKerning, LayoutSmush} {
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
