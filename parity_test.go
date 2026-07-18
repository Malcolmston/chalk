package chalk

// This file encodes known-answer vectors taken directly from Node chalk's own
// AVA test suite (github.com/chalk/chalk, test/chalk.js, test/visible.js,
// test/instance.js and test/level.js). Each assertion reproduces an exact
// expected string the upstream library asserts, mapped onto this port's Go API
// (chalk.red('foo') -> chalk.New().Red().Sprint("foo")). The color level is
// pinned per test so output is deterministic regardless of the environment.
//
// Escape sequences use the literal ESC byte "\x1b" (upstream writes it as
// ""); the two are identical.

import "testing"

// withLevel pins the global color level for the duration of fn, restoring the
// previous state afterwards so parity tests do not leak level state.
func withLevel(l Level, fn func()) {
	SetLevel(l)
	defer ResetDetection()
	fn()
}

// TestParityStyleString mirrors chalk.js "style string": the basic modifier and
// color wrappers emit the documented open/close SGR pairs.
func TestParityStyleString(t *testing.T) {
	withLevel(LevelBasic, func() {
		cases := []struct {
			name string
			got  string
			want string
		}{
			{"underline", New().Underline().Sprint("foo"), "\x1b[4mfoo\x1b[24m"},
			{"red", New().Red().Sprint("foo"), "\x1b[31mfoo\x1b[39m"},
			{"bgRed", New().BgRed().Sprint("foo"), "\x1b[41mfoo\x1b[49m"},
		}
		for _, c := range cases {
			if c.got != c.want {
				t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
			}
		}
	})
}

// TestParityMultipleStyles mirrors chalk.js "support applying multiple styles at
// once": chained styles nest open codes outermost-first and close codes in the
// reverse order.
func TestParityMultipleStyles(t *testing.T) {
	withLevel(LevelBasic, func() {
		if got, want := New().Red().BgGreen().Underline().Sprint("foo"),
			"\x1b[31m\x1b[42m\x1b[4mfoo\x1b[24m\x1b[49m\x1b[39m"; got != want {
			t.Errorf("red.bgGreen.underline = %q, want %q", got, want)
		}
		if got, want := New().Underline().Red().BgGreen().Sprint("foo"),
			"\x1b[4m\x1b[31m\x1b[42mfoo\x1b[49m\x1b[39m\x1b[24m"; got != want {
			t.Errorf("underline.red.bgGreen = %q, want %q", got, want)
		}
	})
}

// TestParityNesting mirrors chalk.js "support nesting styles".
func TestParityNesting(t *testing.T) {
	withLevel(LevelBasic, func() {
		inner := New().Underline().BgBlue().Sprint("bar")
		got := New().Red().Sprint("foo" + inner + "!")
		want := "\x1b[31mfoo\x1b[4m\x1b[44mbar\x1b[49m\x1b[24m!\x1b[39m"
		if got != want {
			t.Errorf("nested = %q, want %q", got, want)
		}
	})
}

// TestParityNestingSameType mirrors chalk.js "support nesting styles of the same
// type (color, underline, bg)": the style-bleed fix re-opens an outer color
// after an inner reset of the same close code.
func TestParityNestingSameType(t *testing.T) {
	withLevel(LevelBasic, func() {
		got := New().Red().Sprint("a" + New().Yellow().Sprint("b"+New().Green().Sprint("c")+"b") + "c")
		want := "\x1b[31ma\x1b[33mb\x1b[32mc\x1b[39m\x1b[31m\x1b[33mb\x1b[39m\x1b[31mc\x1b[39m"
		if got != want {
			t.Errorf("same-type nesting = %q, want %q", got, want)
		}
	})
}

// TestParityReset mirrors chalk.js "reset all styles with `.reset()`".
func TestParityReset(t *testing.T) {
	withLevel(LevelBasic, func() {
		inner := New().Red().BgGreen().Underline().Sprint("foo")
		got := New().Reset().Sprint(inner + "foo")
		want := "\x1b[0m\x1b[31m\x1b[42m\x1b[4mfoo\x1b[24m\x1b[49m\x1b[39mfoo\x1b[0m"
		if got != want {
			t.Errorf("reset = %q, want %q", got, want)
		}
	})
}

// TestParityGrayAlias mirrors chalk.js "alias gray to grey" and "supports
// blackBright color": grey, gray and BrightBlack all emit code 90.
func TestParityGrayAlias(t *testing.T) {
	withLevel(LevelBasic, func() {
		want := "\x1b[90mfoo\x1b[39m"
		if got := New().Grey().Sprint("foo"); got != want {
			t.Errorf("grey = %q, want %q", got, want)
		}
		if got := New().Gray().Sprint("foo"); got != want {
			t.Errorf("gray = %q, want %q", got, want)
		}
		if got := New().BrightBlack().Sprint("foo"); got != want {
			t.Errorf("brightBlack = %q, want %q", got, want)
		}
	})
}

// TestParityCasting mirrors chalk.js "support automatic casting to string" for
// single scalar arguments and "support falsy values" (chalk.red(0)).
func TestParityCasting(t *testing.T) {
	withLevel(LevelBasic, func() {
		if got, want := New().Green().Sprint(98765), "\x1b[32m98765\x1b[39m"; got != want {
			t.Errorf("green(98765) = %q, want %q", got, want)
		}
		if got, want := New().Red().Sprint(0), "\x1b[31m0\x1b[39m"; got != want {
			t.Errorf("red(0) = %q, want %q", got, want)
		}
	})
}

// TestParityEmptyInput mirrors chalk.js "don't output escape codes if the input
// is empty": chalk.red() === ” and chalk.red.blue.black() === ”.
func TestParityEmptyInput(t *testing.T) {
	withLevel(LevelBasic, func() {
		if got := New().Red().Sprint(); got != "" {
			t.Errorf("red() = %q, want empty", got)
		}
		if got := New().Red().Blue().Black().Sprint(); got != "" {
			t.Errorf("red.blue.black() = %q, want empty", got)
		}
		if got := New().Red().Sprint(""); got != "" {
			t.Errorf(`red("") = %q, want empty`, got)
		}
	})
}

// TestParityLineBreaks mirrors chalk.js "line breaks should open and close
// colors" and its CRLF variant: the style is closed before each newline and
// re-opened after it.
func TestParityLineBreaks(t *testing.T) {
	withLevel(LevelBasic, func() {
		if got, want := New().Grey().Sprint("hello\nworld"),
			"\x1b[90mhello\x1b[39m\n\x1b[90mworld\x1b[39m"; got != want {
			t.Errorf("LF = %q, want %q", got, want)
		}
		if got, want := New().Grey().Sprint("hello\r\nworld"),
			"\x1b[90mhello\x1b[39m\r\n\x1b[90mworld\x1b[39m"; got != want {
			t.Errorf("CRLF = %q, want %q", got, want)
		}
	})
}

// TestParityRedBold mirrors chalk.js "sets correct level for chalkStderr and
// respects it": chalk.red.bold('foo').
func TestParityRedBold(t *testing.T) {
	withLevel(LevelBasic, func() {
		if got, want := New().Red().Bold().Sprint("foo"),
			"\x1b[31m\x1b[1mfoo\x1b[22m\x1b[39m"; got != want {
			t.Errorf("red.bold = %q, want %q", got, want)
		}
	})
}

// TestParityHexDowngrade mirrors chalk.js "properly convert RGB to 16/256 colors
// on basic color terminals" and "don't emit RGB codes if level is 0". The
// per-style Level override reproduces upstream's new Chalk({level: n}).
func TestParityHexDowngrade(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"level1-hex", New().Level(LevelBasic).Hex("#FF0000").Sprint("hello"), "\x1b[91mhello\x1b[39m"},
		{"level1-bgHex", New().Level(LevelBasic).BgHex("#FF0000").Sprint("hello"), "\x1b[101mhello\x1b[49m"},
		{"level2-hex", New().Level(Level256).Hex("#FF0000").Sprint("hello"), "\x1b[38;5;196mhello\x1b[39m"},
		{"level2-bgHex", New().Level(Level256).BgHex("#FF0000").Sprint("hello"), "\x1b[48;5;196mhello\x1b[49m"},
		{"level3-bgHex", New().Level(LevelTrueColor).BgHex("#FF0000").Sprint("hello"), "\x1b[48;2;255;0;0mhello\x1b[49m"},
		{"level0-hex", New().Level(LevelNone).Hex("#FF0000").Sprint("hello"), "hello"},
		{"level0-bgHex", New().Level(LevelNone).BgHex("#FF0000").Sprint("hello"), "hello"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestParityLevelGating mirrors instance.js and level.js: at level 0 no color is
// emitted; at level >= 1 the style is applied. The per-style Level override is
// this port's equivalent of upstream's isolated Chalk instances.
func TestParityLevelGating(t *testing.T) {
	if got := New().Level(LevelNone).Red().Sprint("foo"); got != "foo" {
		t.Errorf("level0 red = %q, want %q", got, "foo")
	}
	if got, want := New().Level(Level256).Red().Sprint("foo"), "\x1b[31mfoo\x1b[39m"; got != want {
		t.Errorf("level2 red = %q, want %q", got, want)
	}
	withLevel(LevelNone, func() {
		if got := New().Red().Sprint("foo"); got != "foo" {
			t.Errorf("global level0 red = %q, want %q", got, "foo")
		}
	})
}

// TestParityVisible mirrors visible.js: the .visible modifier suppresses output
// entirely when the level is 0 but is transparent when color is enabled.
func TestParityVisible(t *testing.T) {
	// level 3: visible is transparent.
	if got, want := New().Level(LevelTrueColor).Visible().Red().Sprint("foo"), "\x1b[31mfoo\x1b[39m"; got != want {
		t.Errorf("visible.red @3 = %q, want %q", got, want)
	}
	if got, want := New().Level(LevelTrueColor).Red().Visible().Sprint("foo"), "\x1b[31mfoo\x1b[39m"; got != want {
		t.Errorf("red.visible @3 = %q, want %q", got, want)
	}
	if got, want := New().Level(LevelTrueColor).Visible().Sprint("foo"), "foo"; got != want {
		t.Errorf("visible @3 = %q, want %q", got, want)
	}
	// level 0: visible suppresses all output.
	if got := New().Level(LevelNone).Visible().Red().Sprint("foo"); got != "" {
		t.Errorf("visible.red @0 = %q, want empty", got)
	}
	if got := New().Level(LevelNone).Red().Visible().Sprint("foo"); got != "" {
		t.Errorf("red.visible @0 = %q, want empty", got)
	}
	if got := New().Level(LevelNone).Visible().Sprint("foo"); got != "" {
		t.Errorf("visible @0 = %q, want empty", got)
	}
}
