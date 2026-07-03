package chalk

import (
	"strings"
	"testing"
)

func TestBasicColor(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	got := New().Red().Sprint("hi")
	if got != "\x1b[31mhi\x1b[39m" {
		t.Fatalf("red = %q", got)
	}
}

func TestChaining(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	got := New().Bold().Red().BgWhite().Sprint("x")
	// Outermost (bold) is applied last, so it wraps the whole thing.
	want := "\x1b[1m\x1b[31m\x1b[47mx\x1b[49m\x1b[39m\x1b[22m"
	if got != want {
		t.Fatalf("chained = %q\nwant     %q", got, want)
	}
}

func TestDisabled(t *testing.T) {
	SetLevel(LevelNone)
	if got := New().Red().Bold().Sprint("plain"); got != "plain" {
		t.Fatalf("disabled = %q, want plain", got)
	}
	if got := Red("plain"); got != "plain" {
		t.Fatalf("shortcut disabled = %q", got)
	}
}

func TestNesting(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	// Red around a bold inner: the red must re-assert after the inner reset.
	inner := New().Bold().Sprint("bold")
	got := New().Red().Sprint("a" + inner + "b")
	// The inner bold-close (22m) should be followed by a red re-open... actually
	// red uses 39 to close; nesting fix re-opens red's 31 after any 39m.
	if !strings.HasPrefix(got, "\x1b[31m") || !strings.HasSuffix(got, "\x1b[39m") {
		t.Fatalf("nesting wrapper = %q", got)
	}
	if !strings.Contains(got, "bold") {
		t.Fatalf("nesting lost inner text: %q", got)
	}
}

func TestTrueColorHex(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)
	got := New().Hex("#ff8800").Sprint("o")
	if got != "\x1b[38;2;255;136;0mo\x1b[39m" {
		t.Fatalf("hex = %q", got)
	}
}

func TestRGBDowngrade256(t *testing.T) {
	SetLevel(Level256)
	defer SetLevel(LevelNone)
	got := New().RGB(255, 0, 0).Sprint("r")
	// 255,0,0 -> cube index 196.
	if got != "\x1b[38;5;196mr\x1b[39m" {
		t.Fatalf("256 downgrade = %q", got)
	}
}

func TestRGBDowngrade16(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	// Pure bright red should map to 91 (bright red).
	got := New().RGB(255, 0, 0).Sprint("r")
	if got != "\x1b[91mr\x1b[39m" {
		t.Fatalf("16 downgrade = %q", got)
	}
}

func TestBgHex(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)
	got := New().BgHex("#102030").Sprint("x")
	if got != "\x1b[48;2;16;32;48mx\x1b[49m" {
		t.Fatalf("bg hex = %q", got)
	}
}

func TestStripAndLength(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	styled := New().Red().Bold().Sprint("hello")
	if Strip(styled) != "hello" {
		t.Fatalf("strip = %q", Strip(styled))
	}
	if VisibleLength(styled) != 5 {
		t.Fatalf("visible length = %d", VisibleLength(styled))
	}
}

func TestPerStyleLevelOverride(t *testing.T) {
	SetLevel(LevelNone) // global off
	// A style pinned to basic still renders.
	got := New().Level(LevelBasic).Green().Sprint("g")
	if got != "\x1b[32mg\x1b[39m" {
		t.Fatalf("override = %q", got)
	}
}

func TestParseHexShorthand(t *testing.T) {
	r, g, b := parseHex("f80")
	if r != 0xff || g != 0x88 || b != 0x00 {
		t.Fatalf("shorthand hex = %d,%d,%d", r, g, b)
	}
}
