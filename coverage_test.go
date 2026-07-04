package chalk

import (
	"os"
	"strings"
	"testing"
)

// seq builds the SGR open sequence for a raw code, e.g. seq("31") == "\x1b[31m".
func seq(code string) string { return esc + code + "m" }

// --- text modifiers & colors -------------------------------------------------

// TestModifierAndColorCodes exercises every fluent modifier and color method,
// asserting the derived Style emits the expected open SGR code under basic
// color. Codes are taken directly from colors.go (with(open, close)).
func TestModifierAndColorCodes(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	cases := []struct {
		name string
		fn   func(*Style) *Style
		open string
	}{
		{"Reset", (*Style).Reset, "0"},
		{"Bold", (*Style).Bold, "1"},
		{"Dim", (*Style).Dim, "2"},
		{"Italic", (*Style).Italic, "3"},
		{"Underline", (*Style).Underline, "4"},
		{"Inverse", (*Style).Inverse, "7"},
		{"Hidden", (*Style).Hidden, "8"},
		{"Strikethrough", (*Style).Strikethrough, "9"},
		{"Overline", (*Style).Overline, "53"},

		{"Black", (*Style).Black, "30"},
		{"Red", (*Style).Red, "31"},
		{"Green", (*Style).Green, "32"},
		{"Yellow", (*Style).Yellow, "33"},
		{"Blue", (*Style).Blue, "34"},
		{"Magenta", (*Style).Magenta, "35"},
		{"Cyan", (*Style).Cyan, "36"},
		{"White", (*Style).White, "37"},
		{"Gray", (*Style).Gray, "90"},
		{"Grey", (*Style).Grey, "90"},

		{"BrightBlack", (*Style).BrightBlack, "90"},
		{"BrightRed", (*Style).BrightRed, "91"},
		{"BrightGreen", (*Style).BrightGreen, "92"},
		{"BrightYellow", (*Style).BrightYellow, "93"},
		{"BrightBlue", (*Style).BrightBlue, "94"},
		{"BrightMagenta", (*Style).BrightMagenta, "95"},
		{"BrightCyan", (*Style).BrightCyan, "96"},
		{"BrightWhite", (*Style).BrightWhite, "97"},

		{"BgBlack", (*Style).BgBlack, "40"},
		{"BgRed", (*Style).BgRed, "41"},
		{"BgGreen", (*Style).BgGreen, "42"},
		{"BgYellow", (*Style).BgYellow, "43"},
		{"BgBlue", (*Style).BgBlue, "44"},
		{"BgMagenta", (*Style).BgMagenta, "45"},
		{"BgCyan", (*Style).BgCyan, "46"},
		{"BgWhite", (*Style).BgWhite, "47"},
		{"BgGray", (*Style).BgGray, "100"},

		{"BgBrightBlack", (*Style).BgBrightBlack, "100"},
		{"BgBrightRed", (*Style).BgBrightRed, "101"},
		{"BgBrightGreen", (*Style).BgBrightGreen, "102"},
		{"BgBrightYellow", (*Style).BgBrightYellow, "103"},
		{"BgBrightBlue", (*Style).BgBrightBlue, "104"},
		{"BgBrightMagenta", (*Style).BgBrightMagenta, "105"},
		{"BgBrightCyan", (*Style).BgBrightCyan, "106"},
		{"BgBrightWhite", (*Style).BgBrightWhite, "107"},
	}

	for _, c := range cases {
		got := c.fn(New()).Sprint("x")
		if !strings.HasPrefix(got, seq(c.open)) {
			t.Errorf("%s: got %q, want prefix %q", c.name, got, seq(c.open))
		}
		if Strip(got) != "x" {
			t.Errorf("%s: strip = %q, want \"x\"", c.name, Strip(got))
		}
	}
}

// --- 256 / truecolor ---------------------------------------------------------

func TestAnsi256Codes(t *testing.T) {
	SetLevel(Level256)
	defer SetLevel(LevelNone)

	if got := New().Ansi256(5).Sprint("x"); got != seq("38;5;5")+"x"+seq("39") {
		t.Errorf("Ansi256(5) = %q", got)
	}
	if got := New().BgAnsi256(5).Sprint("x"); got != seq("48;5;5")+"x"+seq("49") {
		t.Errorf("BgAnsi256(5) = %q", got)
	}
	// The palette index is masked to a byte.
	if got := New().Ansi256(256 + 7).Sprint("x"); got != seq("38;5;7")+"x"+seq("39") {
		t.Errorf("Ansi256 mask = %q", got)
	}
}

// TestAnsi256Downgrade forces basic color so Ansi256 must degrade through
// ansi256ToRGB -> rgbTo16.
func TestAnsi256Downgrade(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	cases := []int{9, 15, 240, 200, 0}
	for _, n := range cases {
		got := New().Ansi256(n).Sprint("x")
		if !strings.HasPrefix(got, esc) || !strings.HasSuffix(got, seq("39")) {
			t.Errorf("Ansi256(%d) basic = %q", n, got)
		}
		if Strip(got) != "x" {
			t.Errorf("Ansi256(%d) strip = %q", n, Strip(got))
		}
	}
	// Background variant too.
	if got := New().BgAnsi256(200).Sprint("x"); Strip(got) != "x" {
		t.Errorf("BgAnsi256 basic strip = %q", got)
	}
}

func TestRGBTrueColor(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)

	if got := New().RGB(1, 2, 3).Sprint("x"); got != seq("38;2;1;2;3")+"x"+seq("39") {
		t.Errorf("RGB truecolor = %q", got)
	}
	if got := New().BgRGB(1, 2, 3).Sprint("x"); got != seq("48;2;1;2;3")+"x"+seq("49") {
		t.Errorf("BgRGB truecolor = %q", got)
	}
	// clamp: negative floors to 0, >255 ceils to 255.
	if got := New().RGB(-5, 300, 128).Sprint("x"); got != seq("38;2;0;255;128")+"x"+seq("39") {
		t.Errorf("RGB clamp = %q", got)
	}
}

func TestBgRGBDowngrade(t *testing.T) {
	SetLevel(Level256)
	if got := New().BgRGB(255, 0, 0).Sprint("x"); got != seq("48;5;196")+"x"+seq("49") {
		t.Errorf("BgRGB 256 = %q", got)
	}
	SetLevel(LevelBasic)
	// 255,0,0 -> bright red fg 91, bg = 91+10 = 101.
	if got := New().BgRGB(255, 0, 0).Sprint("x"); got != seq("101")+"x"+seq("49") {
		t.Errorf("BgRGB basic = %q", got)
	}
	SetLevel(LevelNone)
}

// TestRGBTo256Grayscale covers the grayscale ramp branches of rgbTo256.
func TestRGBTo256Grayscale(t *testing.T) {
	SetLevel(Level256)
	defer SetLevel(LevelNone)

	cases := []struct {
		v    int
		want int
	}{
		{0, 16},    // r < 8  -> pure black cube corner
		{255, 231}, // r > 248 -> white cube corner
		{100, 241}, // mid gray -> ramp: round((100-8)/247*24)+232
	}
	for _, c := range cases {
		got := New().RGB(c.v, c.v, c.v).Sprint("x")
		want := seq("38;5;"+itoa(c.want)) + "x" + seq("39")
		if got != want {
			t.Errorf("RGB(%d) gray = %q, want %q", c.v, got, want)
		}
	}
}

// TestRGBTo16Value0 hits the value==0 (near-black) branch of rgbTo16.
func TestRGBTo16Value0(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	if got := New().RGB(0, 0, 0).Sprint("x"); got != seq("30")+"x"+seq("39") {
		t.Errorf("RGB(0,0,0) basic = %q, want black 30", got)
	}
	// Mid-intensity red (value==1, no +60 bump).
	if got := New().RGB(128, 0, 0).Sprint("x"); got != seq("31")+"x"+seq("39") {
		t.Errorf("RGB(128,0,0) basic = %q, want dim red 31", got)
	}
}

// --- parseHex edge cases -----------------------------------------------------

func TestParseHexEdges(t *testing.T) {
	cases := []struct {
		in      string
		r, g, b int
	}{
		{"#ffffff", 255, 255, 255},
		{"000000", 0, 0, 0},
		{"#abc", 0xaa, 0xbb, 0xcc}, // shorthand with leading '#'
		{"12", 0, 0, 0},            // wrong length -> zero
		{"gggggg", 0, 0, 0},        // invalid hex digits -> zero
		{"#12345", 0, 0, 0},        // 5 chars -> zero
	}
	for _, c := range cases {
		r, g, b := parseHex(c.in)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("parseHex(%q) = %d,%d,%d want %d,%d,%d", c.in, r, g, b, c.r, c.g, c.b)
		}
	}
}

// --- Sprint family -----------------------------------------------------------

func TestSprintFamily(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	if got := New().Red().Sprintf("%d-%s", 7, "z"); got != seq("31")+"7-z"+seq("39") {
		t.Errorf("Sprintf = %q", got)
	}

	// Sprintln: newline lives OUTSIDE the closing code.
	got := New().Red().Sprintln("a", "b")
	want := seq("31") + "a b" + seq("39") + "\n"
	if got != want {
		t.Errorf("Sprintln = %q, want %q", got, want)
	}
}

func TestPrintFamily(t *testing.T) {
	SetLevel(LevelNone) // plain output for a predictable byte count
	defer SetLevel(LevelNone)

	// Redirect stdout to a pipe to capture and verify output.
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	n1, e1 := New().Red().Print("hi")
	n2, e2 := New().Red().Printf("%d", 42)
	n3, e3 := New().Red().Println("bye")

	w.Close()
	os.Stdout = orig

	buf := make([]byte, 1024)
	m, _ := r.Read(buf)
	out := string(buf[:m])

	if e1 != nil || e2 != nil || e3 != nil {
		t.Fatalf("print errors: %v %v %v", e1, e2, e3)
	}
	if n1 != 2 || n2 != 2 || n3 != 4 {
		t.Errorf("print byte counts = %d %d %d, want 2 2 4", n1, n2, n3)
	}
	if out != "hi42bye\n" {
		t.Errorf("captured stdout = %q, want %q", out, "hi42bye\n")
	}
}

// --- immutability / chaining -------------------------------------------------

func TestStyleImmutability(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	base := New().Red()
	derived := base.Bold().Underline()

	if len(base.parts) != 1 {
		t.Fatalf("base mutated: %d parts, want 1", len(base.parts))
	}
	// The base still renders as plain red.
	if got := base.Sprint("x"); got != seq("31")+"x"+seq("39") {
		t.Errorf("base after derivation = %q", got)
	}
	// The derived style carries all three layers.
	d := derived.Sprint("x")
	for _, code := range []string{"31", "1", "4"} {
		if !strings.Contains(d, seq(code)) {
			t.Errorf("derived missing %q: %q", seq(code), d)
		}
	}

	// Level() also returns a copy, leaving the receiver's level untouched.
	pinned := base.Level(LevelTrueColor)
	if base.level != nil {
		t.Error("Level() mutated the receiver")
	}
	if pinned.effectiveLevel() != LevelTrueColor {
		t.Error("pinned level not applied")
	}
}

// --- VisibleLength with runes ------------------------------------------------

func TestVisibleLengthRunes(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	styled := New().Green().Sprint("héllo") // 5 runes, one multibyte
	if VisibleLength(styled) != 5 {
		t.Errorf("VisibleLength = %d, want 5", VisibleLength(styled))
	}
	if Strip(styled) != "héllo" {
		t.Errorf("Strip = %q", Strip(styled))
	}
}

// --- level state accessors ---------------------------------------------------

func TestLevelAccessors(t *testing.T) {
	defer SetLevel(LevelNone)

	SetLevel(Level256)
	if GetLevel() != Level256 {
		t.Errorf("GetLevel = %v, want Level256", GetLevel())
	}
	if !Enabled() {
		t.Error("Enabled() = false at Level256")
	}

	SetLevel(LevelNone)
	if Enabled() {
		t.Error("Enabled() = true at LevelNone")
	}

	// SetEnabled(false) forces none.
	SetEnabled(false)
	if Enabled() {
		t.Error("SetEnabled(false) left color enabled")
	}
	// SetEnabled(true) yields at least basic color regardless of terminal.
	SetEnabled(true)
	if GetLevel() < LevelBasic {
		t.Errorf("SetEnabled(true) level = %v, want >= LevelBasic", GetLevel())
	}
}

// TestResetDetection forces re-detection from the environment. With NO_COLOR set
// the re-detected level must be none.
func TestResetDetection(t *testing.T) {
	defer SetLevel(LevelNone)
	t.Setenv("NO_COLOR", "1")
	ResetDetection()
	if GetLevel() != LevelNone {
		t.Errorf("re-detected level = %v, want LevelNone (NO_COLOR set)", GetLevel())
	}
}

// --- detectLevel / isTerminal ------------------------------------------------

// clearColorEnv unsets every color-relevant env var for the duration of the test.
func clearColorEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"NO_COLOR", "FORCE_COLOR", "COLORTERM", "TERM"} {
		if v, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			t.Cleanup(func(kk, vv string) func() {
				return func() { os.Setenv(kk, vv) }
			}(k, v))
		} else {
			t.Cleanup(func(kk string) func() {
				return func() { os.Unsetenv(kk) }
			}(k))
		}
	}
}

func TestDetectLevelEnv(t *testing.T) {
	// NO_COLOR wins outright.
	t.Run("NO_COLOR", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("NO_COLOR", "")
		if got := detectLevel(os.Stdout); got != LevelNone {
			t.Errorf("NO_COLOR level = %v", got)
		}
	})

	forceCases := []struct {
		val  string
		want Level
	}{
		{"0", LevelNone},
		{"false", LevelNone},
		{"1", LevelBasic},
		{"true", LevelBasic},
		{"", LevelBasic},
		{"2", Level256},
		{"3", LevelTrueColor},
		{"weird", LevelBasic},
	}
	for _, c := range forceCases {
		t.Run("FORCE_COLOR="+c.val, func(t *testing.T) {
			clearColorEnv(t)
			t.Setenv("FORCE_COLOR", c.val)
			if got := detectLevel(os.Stdout); got != c.want {
				t.Errorf("FORCE_COLOR=%q level = %v, want %v", c.val, got, c.want)
			}
		})
	}
}

// TestDetectLevelTerminal drives the terminal-capability branches using
// /dev/null, which is a character device so isTerminal reports true.
func TestDetectLevelTerminal(t *testing.T) {
	dev, err := os.Open("/dev/null")
	if err != nil {
		t.Skipf("cannot open /dev/null: %v", err)
	}
	defer dev.Close()

	if !isTerminal(dev) {
		t.Skip("/dev/null is not reported as a char device here")
	}

	cases := []struct {
		name      string
		term      string
		colorterm string
		want      Level
	}{
		{"dumb", "dumb", "", LevelNone},
		{"truecolor", "xterm", "truecolor", LevelTrueColor},
		{"24bit", "xterm", "24bit", LevelTrueColor},
		{"256", "xterm-256color", "", Level256},
		{"basic", "xterm", "", LevelBasic},
		{"empty-term", "", "", LevelBasic},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearColorEnv(t)
			t.Setenv("TERM", c.term)
			t.Setenv("COLORTERM", c.colorterm)
			if got := detectLevel(dev); got != c.want {
				t.Errorf("term=%q colorterm=%q level = %v, want %v", c.term, c.colorterm, got, c.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(nil) {
		t.Error("isTerminal(nil) = true")
	}
	// A regular file is not a terminal.
	f, err := os.CreateTemp(t.TempDir(), "notatty")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("isTerminal(regular file) = true")
	}
}

// --- package-level shortcuts -------------------------------------------------

func TestShortcuts(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	cases := []struct {
		name string
		fn   func(...any) string
		open string
	}{
		{"Bold", Bold, "1"},
		{"Dim", Dim, "2"},
		{"Italic", Italic, "3"},
		{"Underline", Underline, "4"},
		{"Inverse", Inverse, "7"},
		{"Strikethrough", Strikethrough, "9"},
		{"Black", Black, "30"},
		{"Red", Red, "31"},
		{"Green", Green, "32"},
		{"Yellow", Yellow, "33"},
		{"Blue", Blue, "34"},
		{"Magenta", Magenta, "35"},
		{"Cyan", Cyan, "36"},
		{"White", White, "37"},
		{"Gray", Gray, "90"},
	}
	for _, c := range cases {
		got := c.fn("x")
		if !strings.HasPrefix(got, seq(c.open)) || Strip(got) != "x" {
			t.Errorf("%s(...) = %q, want prefix %q", c.name, got, seq(c.open))
		}
	}

	if got := RGB(1, 2, 3, "x"); Strip(got) != "x" || !strings.Contains(got, esc) {
		t.Errorf("RGB shortcut = %q", got)
	}
	if got := Hex("#010203", "x"); Strip(got) != "x" || !strings.Contains(got, esc) {
		t.Errorf("Hex shortcut = %q", got)
	}
	if got := Ansi256(5, "x"); Strip(got) != "x" || !strings.Contains(got, esc) {
		t.Errorf("Ansi256 shortcut = %q", got)
	}
}
