package chalk

import "testing"

func TestExtendedShortcuts(t *testing.T) {
	SetLevel(LevelBasic)
	defer SetLevel(LevelNone)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Reset", Reset("x"), "\x1b[0mx\x1b[0m"},
		{"Hidden", Hidden("x"), "\x1b[8mx\x1b[28m"},
		{"Overline", Overline("x"), "\x1b[53mx\x1b[55m"},
		{"BrightBlack", BrightBlack("x"), "\x1b[90mx\x1b[39m"},
		{"BrightRed", BrightRed("x"), "\x1b[91mx\x1b[39m"},
		{"BrightGreen", BrightGreen("x"), "\x1b[92mx\x1b[39m"},
		{"BrightYellow", BrightYellow("x"), "\x1b[93mx\x1b[39m"},
		{"BrightBlue", BrightBlue("x"), "\x1b[94mx\x1b[39m"},
		{"BrightMagenta", BrightMagenta("x"), "\x1b[95mx\x1b[39m"},
		{"BrightCyan", BrightCyan("x"), "\x1b[96mx\x1b[39m"},
		{"BrightWhite", BrightWhite("x"), "\x1b[97mx\x1b[39m"},
		{"BgBlack", BgBlack("x"), "\x1b[40mx\x1b[49m"},
		{"BgRed", BgRed("x"), "\x1b[41mx\x1b[49m"},
		{"BgGreen", BgGreen("x"), "\x1b[42mx\x1b[49m"},
		{"BgYellow", BgYellow("x"), "\x1b[43mx\x1b[49m"},
		{"BgBlue", BgBlue("x"), "\x1b[44mx\x1b[49m"},
		{"BgMagenta", BgMagenta("x"), "\x1b[45mx\x1b[49m"},
		{"BgCyan", BgCyan("x"), "\x1b[46mx\x1b[49m"},
		{"BgWhite", BgWhite("x"), "\x1b[47mx\x1b[49m"},
		{"BgGray", BgGray("x"), "\x1b[100mx\x1b[49m"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q; want %q", c.name, c.got, c.want)
		}
	}
}

func TestBackgroundColorShortcutsTrueColor(t *testing.T) {
	SetLevel(LevelTrueColor)
	defer SetLevel(LevelNone)

	if got := BgRGB(255, 136, 0, "x"); got != "\x1b[48;2;255;136;0mx\x1b[49m" {
		t.Errorf("BgRGB = %q", got)
	}
	if got := BgHex("#ff8800", "x"); got != "\x1b[48;2;255;136;0mx\x1b[49m" {
		t.Errorf("BgHex = %q", got)
	}

	SetLevel(Level256)
	if got := BgAnsi256(196, "x"); got != "\x1b[48;5;196mx\x1b[49m" {
		t.Errorf("BgAnsi256 = %q", got)
	}
}
