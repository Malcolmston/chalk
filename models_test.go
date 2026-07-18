package chalk

import "testing"

func TestColorModelMethodsTrueColor(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"HSL red", New().HSL(0, 100, 50).Sprint("x"), "\x1b[38;2;255;0;0mx\x1b[39m"},
		{"BgHSL green", New().BgHSL(120, 100, 50).Sprint("x"), "\x1b[48;2;0;255;0mx\x1b[49m"},
		{"HSV blue", New().HSV(240, 100, 100).Sprint("x"), "\x1b[38;2;0;0;255mx\x1b[39m"},
		{"BgHSV red", New().BgHSV(0, 100, 100).Sprint("x"), "\x1b[48;2;255;0;0mx\x1b[49m"},
		{"HWB red", New().HWB(0, 0, 0).Sprint("x"), "\x1b[38;2;255;0;0mx\x1b[39m"},
		{"BgHWB white", New().BgHWB(0, 100, 0).Sprint("x"), "\x1b[48;2;255;255;255mx\x1b[49m"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q; want %q", c.name, c.got, c.want)
		}
	}
}

func TestColorModelShortcuts(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)

	if got := HSL(0, 100, 50, "x"); got != "\x1b[38;2;255;0;0mx\x1b[39m" {
		t.Errorf("HSL shortcut = %q", got)
	}
	if got := HSV(120, 100, 100, "x"); got != "\x1b[38;2;0;255;0mx\x1b[39m" {
		t.Errorf("HSV shortcut = %q", got)
	}
	if got := HWB(240, 0, 0, "x"); got != "\x1b[38;2;0;0;255mx\x1b[39m" {
		t.Errorf("HWB shortcut = %q", got)
	}
}

func TestVisibleModifier(t *testing.T) {
	SetLevel(LevelNone)
	if got := New().Visible().Red().Sprint("hi"); got != "" {
		t.Errorf("Visible with color off = %q; want empty", got)
	}
	if got := Visible("hi"); got != "" {
		t.Errorf("Visible shortcut with color off = %q; want empty", got)
	}

	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)
	if got := New().Visible().Red().Sprint("hi"); got != "\x1b[31mhi\x1b[39m" {
		t.Errorf("Visible with color on = %q; want red hi", got)
	}
	if got := Visible("hi"); got != "hi" {
		t.Errorf("Visible shortcut with color on = %q; want hi", got)
	}
}

func TestVisibleImmutable(t *testing.T) {
	SetLevel(LevelNone)
	base := New().Red()
	_ = base.Visible()
	// The original style must be unaffected by deriving a Visible copy.
	if got := base.Sprint("z"); got != "z" {
		t.Errorf("base style mutated by Visible: %q", got)
	}
}
