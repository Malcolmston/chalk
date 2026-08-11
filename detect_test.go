package chalk

import (
	"os"
	"testing"
)

// The tests in this file pin the precedence rules of detectLevel one by one.
// Every case clears the whole color environment first (see clearColorEnv) and
// uses t.Setenv, which restores the previous value when the test ends — these
// are process-global reads, so leaking one variable would silently change the
// answer for every later test.

// charDevice returns a file that isTerminal reports as a terminal, skipping the
// test when the platform does not provide one.
func charDevice(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Skipf("cannot open /dev/null: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if !isTerminal(f) {
		t.Skip("/dev/null is not reported as a char device here")
	}
	return f
}

// notATerminal returns a regular file, which isTerminal reports as not a
// terminal.
func notATerminal(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "notatty")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestNoColorBeatsForceColor pins the top of the precedence chain: NO_COLOR is
// checked before anything else, so it wins even against an explicit
// FORCE_COLOR=3.
func TestNoColorBeatsForceColor(t *testing.T) {
	clearColorEnv(t)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "3")
	t.Setenv("COLORTERM", "truecolor")
	if got := detectLevel(charDevice(t)); got != LevelNone {
		t.Errorf("NO_COLOR with FORCE_COLOR=3 = %v, want LevelNone", got)
	}
}

// TestForceColorClamped checks that a numeric FORCE_COLOR is clamped into the
// 0-3 range rather than falling into the "unrecognized" bucket. Before the fix
// FORCE_COLOR=4 produced LevelBasic; supports-color computes min(parseInt, 3).
func TestForceColorClamped(t *testing.T) {
	cases := []struct {
		val  string
		want Level
	}{
		{"4", LevelTrueColor},
		{"99", LevelTrueColor},
		{"-1", LevelNone},
		{"03", LevelTrueColor},
	}
	for _, c := range cases {
		t.Run("FORCE_COLOR="+c.val, func(t *testing.T) {
			clearColorEnv(t)
			t.Setenv("FORCE_COLOR", c.val)
			if got := detectLevel(notATerminal(t)); got != c.want {
				t.Errorf("FORCE_COLOR=%q = %v, want %v", c.val, got, c.want)
			}
		})
	}
}

// TestForceColorIsAFloorNotACap mirrors supports-color: FORCE_COLOR raises the
// minimum level and suppresses the TTY requirement, but detection may still
// report something better.
func TestForceColorIsAFloorNotACap(t *testing.T) {
	t.Run("truecolor terminal beats FORCE_COLOR=1", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("FORCE_COLOR", "1")
		t.Setenv("COLORTERM", "truecolor")
		if got := detectLevel(notATerminal(t)); got != LevelTrueColor {
			t.Errorf("FORCE_COLOR=1 + COLORTERM=truecolor = %v, want LevelTrueColor", got)
		}
	})
	t.Run("floor survives an unhelpful TERM", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("FORCE_COLOR", "2")
		t.Setenv("TERM", "dumb")
		if got := detectLevel(notATerminal(t)); got != Level256 {
			t.Errorf("FORCE_COLOR=2 + TERM=dumb = %v, want Level256", got)
		}
	})
	t.Run("no color without a terminal or FORCE_COLOR", func(t *testing.T) {
		clearColorEnv(t)
		t.Setenv("COLORTERM", "truecolor")
		if got := detectLevel(notATerminal(t)); got != LevelNone {
			t.Errorf("redirected output = %v, want LevelNone", got)
		}
	})
}

// TestDetectAzurePipelines pins the one rule that fires before the TTY check:
// Azure Pipelines renders color in its log view even though the build's stdout
// is a pipe.
func TestDetectAzurePipelines(t *testing.T) {
	clearColorEnv(t)
	t.Setenv("TF_BUILD", "True")
	t.Setenv("AGENT_NAME", "Hosted Agent")
	if got := detectLevel(notATerminal(t)); got != LevelBasic {
		t.Errorf("Azure Pipelines = %v, want LevelBasic", got)
	}
	// TF_BUILD on its own is not enough.
	clearColorEnv(t)
	t.Setenv("TF_BUILD", "True")
	if got := detectLevel(notATerminal(t)); got != LevelNone {
		t.Errorf("TF_BUILD alone = %v, want LevelNone", got)
	}
}

// TestDetectCIProviders covers the CI block. All of these require a terminal,
// matching supports-color, where the CI checks come after the TTY test.
func TestDetectCIProviders(t *testing.T) {
	dev := charDevice(t)
	cases := []struct {
		name string
		env  map[string]string
		want Level
	}{
		{"github actions", map[string]string{"CI": "true", "GITHUB_ACTIONS": "true"}, LevelTrueColor},
		{"gitea actions", map[string]string{"CI": "true", "GITEA_ACTIONS": "true"}, LevelTrueColor},
		{"codeship", map[string]string{"CI": "true", "CI_NAME": "codeship"}, LevelBasic},
		{"travis", map[string]string{"CI": "true", "TRAVIS": "1"}, LevelBasic},
		{"gitlab", map[string]string{"CI": "true", "GITLAB_CI": "1"}, LevelBasic},
		{"unknown ci", map[string]string{"CI": "true"}, LevelNone},
		// The CI block runs before COLORTERM, so an unknown CI reports no color
		// even in a terminal that advertises truecolor.
		{"unknown ci ignores COLORTERM", map[string]string{"CI": "true", "COLORTERM": "truecolor"}, LevelNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearColorEnv(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			if got := detectLevel(dev); got != c.want {
				t.Errorf("%s = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

// TestDetectCIRespectsForceColorFloor checks that an unknown CI still honors an
// explicit FORCE_COLOR, which is how most projects get color out of CI logs.
func TestDetectCIRespectsForceColorFloor(t *testing.T) {
	clearColorEnv(t)
	t.Setenv("CI", "true")
	t.Setenv("FORCE_COLOR", "3")
	if got := detectLevel(notATerminal(t)); got != LevelTrueColor {
		t.Errorf("CI + FORCE_COLOR=3 = %v, want LevelTrueColor", got)
	}
}

// TestTeamCityVersion pins the TEAMCITY_VERSION rule: TeamCity only learned to
// render ANSI color in 9.1.
func TestTeamCityVersion(t *testing.T) {
	cases := []struct {
		v    string
		want bool
	}{
		{"9.0.5", false},
		{"9.1.5", true},
		{"9.01.0", true}, // upstream's 0*[1-9] allows a padded nonzero minor
		{"10.0.0", true},
		{"23.11.1", true},
		{"8.9.9", false},
		{"9", false},
		{"not.a.version", false},
		{"", false},
	}
	for _, c := range cases {
		if got := teamCitySupportsColor(c.v); got != c.want {
			t.Errorf("teamCitySupportsColor(%q) = %v, want %v", c.v, got, c.want)
		}
	}

	dev := charDevice(t)
	clearColorEnv(t)
	t.Setenv("TEAMCITY_VERSION", "9.1.5")
	if got := detectLevel(dev); got != LevelBasic {
		t.Errorf("TeamCity 9.1.5 = %v, want LevelBasic", got)
	}
	clearColorEnv(t)
	t.Setenv("TEAMCITY_VERSION", "9.0.5")
	t.Setenv("COLORTERM", "truecolor")
	if got := detectLevel(dev); got != LevelNone {
		t.Errorf("TeamCity 9.0.5 = %v, want LevelNone (checked before COLORTERM)", got)
	}
}

// TestDetectTerminalPrograms covers the truecolor terminals recognized by name
// and the TERM_PROGRAM branch.
func TestDetectTerminalPrograms(t *testing.T) {
	dev := charDevice(t)
	cases := []struct {
		name string
		env  map[string]string
		want Level
	}{
		{"kitty", map[string]string{"TERM": "xterm-kitty"}, LevelTrueColor},
		{"ghostty", map[string]string{"TERM": "xterm-ghostty"}, LevelTrueColor},
		{"wezterm", map[string]string{"TERM": "wezterm"}, LevelTrueColor},
		{"windows terminal", map[string]string{"WT_SESSION": "abc"}, LevelTrueColor},
		{"iterm 3", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.4.19"}, LevelTrueColor},
		{"iterm 2", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "2.9.20150512"}, Level256},
		{"iterm unknown version", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "iTerm.app"}, Level256},
		{"apple terminal", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "Apple_Terminal"}, Level256},
		{"unknown term program", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "Hyper"}, LevelBasic},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearColorEnv(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			if got := detectLevel(dev); got != c.want {
				t.Errorf("%s = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

// TestTermLooksColorful pins the TERM-name heuristic borrowed from
// supports-color's regex.
func TestTermLooksColorful(t *testing.T) {
	yes := []string{"screen", "screen.xterm", "xterm", "vt100", "vt220", "rxvt-unicode", "linux", "cygwin", "konsole-256color", "ansi.sys"}
	no := []string{"", "dumb", "sun", "vt52", "emacs"}
	for _, term := range yes {
		if !termLooksColorful(term) {
			t.Errorf("termLooksColorful(%q) = false, want true", term)
		}
	}
	for _, term := range no {
		if termLooksColorful(term) {
			t.Errorf("termLooksColorful(%q) = true, want false", term)
		}
	}
}

// TestDetect256Suffix checks the "-256color" / "-256" TERM suffixes, which must
// match at the end of the name only: a terminal called "256colorless" is not a
// 256-color terminal.
func TestDetect256Suffix(t *testing.T) {
	dev := charDevice(t)
	cases := []struct {
		term string
		want Level
	}{
		{"xterm-256color", Level256},
		{"screen-256color", Level256},
		{"putty-256", Level256},
		{"xterm", LevelBasic},
	}
	for _, c := range cases {
		t.Run(c.term, func(t *testing.T) {
			clearColorEnv(t)
			t.Setenv("TERM", c.term)
			if got := detectLevel(dev); got != c.want {
				t.Errorf("TERM=%q = %v, want %v", c.term, got, c.want)
			}
		})
	}
}

// TestEnvForceColorReportsPresence checks the second return value, which is what
// lets a set-but-unhelpful FORCE_COLOR skip the TTY requirement.
func TestEnvForceColorReportsPresence(t *testing.T) {
	clearColorEnv(t)
	if _, ok := envForceColor(); ok {
		t.Error("envForceColor reported set with FORCE_COLOR unset")
	}
	t.Setenv("FORCE_COLOR", "")
	lvl, ok := envForceColor()
	if !ok || lvl != LevelBasic {
		t.Errorf("envForceColor() = %v, %v; want LevelBasic, true", lvl, ok)
	}
}
