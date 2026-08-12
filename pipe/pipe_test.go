package pipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/malcolmston/chalk"
)

// colorEnvVars are every variable that can change a color decision. Tests clear
// all of them so a developer's own shell (CLICOLOR is set by default on macOS)
// cannot change the result.
var colorEnvVars = []string{"NO_COLOR", "FORCE_COLOR", "CLICOLOR", "CLICOLOR_FORCE", "TERM", "COLUMNS", "LINES"}

// clearEnv unsets every variable in colorEnvVars for the duration of the test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range colorEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v) // registers the restore
			if err := os.Unsetenv(k); err != nil {
				t.Fatalf("unset %s: %v", k, err)
			}
			continue
		}
		// Setenv then Unsetenv so t.Setenv's cleanup removes it again.
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}
}

// pinLevel forces chalk's global level and restores auto-detection afterwards.
func pinLevel(t *testing.T, l chalk.Level) {
	t.Helper()
	chalk.SetLevel(l)
	t.Cleanup(chalk.ResetDetection)
}

// tempFile returns a regular file open for writing: a redirected stream.
func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// osPipe returns the read and write ends of a real pipe.
func osPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	return r, w
}

func TestStreamKindDetection(t *testing.T) {
	file := tempFile(t)
	pr, pw := osPipe(t)

	tests := []struct {
		name                        string
		f                           *os.File
		terminal, piped, redirected bool
	}{
		{"nil", nil, false, false, true},
		{"regular file", file, false, false, true},
		{"pipe read end", pr, false, true, true},
		{"pipe write end", pw, false, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTerminal(tc.f); got != tc.terminal {
				t.Errorf("IsTerminal = %v, want %v", got, tc.terminal)
			}
			if got := IsPiped(tc.f); got != tc.piped {
				t.Errorf("IsPiped = %v, want %v", got, tc.piped)
			}
			if got := IsRedirected(tc.f); got != tc.redirected {
				t.Errorf("IsRedirected = %v, want %v", got, tc.redirected)
			}
		})
	}
}

func TestSizeFallbackWithoutTerminal(t *testing.T) {
	file := tempFile(t)

	tests := []struct {
		name       string
		columns    string
		lines      string
		wantW      int
		wantH      int
		unsetBoth  bool
		wantFallbk bool
	}{
		{name: "defaults", unsetBoth: true, wantW: DefaultWidth, wantH: DefaultHeight, wantFallbk: true},
		{name: "env honored", columns: "120", lines: "40", wantW: 120, wantH: 40, wantFallbk: true},
		{name: "junk ignored", columns: "wide", lines: "-3", wantW: DefaultWidth, wantH: DefaultHeight, wantFallbk: true},
		{name: "padded value", columns: " 100 ", lines: "\t30", wantW: 100, wantH: 30, wantFallbk: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			if !tc.unsetBoth {
				t.Setenv("COLUMNS", tc.columns)
				t.Setenv("LINES", tc.lines)
			}
			w, h := Size(file)
			if w != tc.wantW || h != tc.wantH {
				t.Errorf("Size = %d x %d, want %d x %d", w, h, tc.wantW, tc.wantH)
			}
			if got := Width(file); got != tc.wantW {
				t.Errorf("Width = %d, want %d", got, tc.wantW)
			}
			if got := Height(file); got != tc.wantH {
				t.Errorf("Height = %d, want %d", got, tc.wantH)
			}
			if info := Inspect(file); info.Fallback != tc.wantFallbk {
				t.Errorf("Inspect().Fallback = %v, want %v", info.Fallback, tc.wantFallbk)
			}
		})
	}
}

func TestColorLevelForRedirectedStream(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		pin   chalk.Level
		want  chalk.Level
		wantB bool
	}{
		{name: "no tty, no env", pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "NO_COLOR wins", env: map[string]string{"NO_COLOR": "", "FORCE_COLOR": "3"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "FORCE_COLOR=0 disables", env: map[string]string{"FORCE_COLOR": "0"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "FORCE_COLOR=false disables", env: map[string]string{"FORCE_COLOR": "false"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "FORCE_COLOR forces tier", env: map[string]string{"FORCE_COLOR": "1"}, pin: chalk.Level256, want: chalk.Level256, wantB: true},
		{name: "FORCE_COLOR floors at basic", env: map[string]string{"FORCE_COLOR": "1"}, pin: chalk.LevelNone, want: chalk.LevelBasic, wantB: true},
		{name: "CLICOLOR_FORCE forces", env: map[string]string{"CLICOLOR_FORCE": "1"}, pin: chalk.LevelTrueColor, want: chalk.LevelTrueColor, wantB: true},
		{name: "CLICOLOR_FORCE=0 is not a force", env: map[string]string{"CLICOLOR_FORCE": "0"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "CLICOLOR=0 disables", env: map[string]string{"CLICOLOR": "0", "CLICOLOR_FORCE": "0"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "CLICOLOR=1 is not a force", env: map[string]string{"CLICOLOR": "1"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "TERM=dumb disables", env: map[string]string{"TERM": "dumb"}, pin: chalk.LevelTrueColor, want: chalk.LevelNone},
		{name: "FORCE_COLOR beats TERM=dumb", env: map[string]string{"TERM": "dumb", "FORCE_COLOR": "2"}, pin: chalk.Level256, want: chalk.Level256, wantB: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			pinLevel(t, tc.pin)

			file := tempFile(t)
			if got := ColorLevel(file); got != tc.want {
				t.Errorf("ColorLevel = %v, want %v", got, tc.want)
			}
			if got := ColorEnabled(file); got != tc.wantB {
				t.Errorf("ColorEnabled = %v, want %v", got, tc.wantB)
			}
			if got := Inspect(file).Color; got != tc.want {
				t.Errorf("Inspect().Color = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestColorLevelNeverStylesAPipe is the degraded-path guarantee stated on its
// own: with no environment override, nothing this package writes to a pipe or a
// file carries color, however capable chalk thinks the process's stdout is.
func TestColorLevelNeverStylesAPipe(t *testing.T) {
	clearEnv(t)
	pinLevel(t, chalk.LevelTrueColor)

	pr, pw := osPipe(t)
	for _, f := range []*os.File{tempFile(t), pr, pw} {
		if got := ColorLevel(f); got != chalk.LevelNone {
			t.Errorf("ColorLevel(%v) = %v, want LevelNone", f.Name(), got)
		}
	}
}

func TestSyncColorPushesDecisionIntoChalk(t *testing.T) {
	clearEnv(t)
	pinLevel(t, chalk.LevelTrueColor)

	got := SyncColor(tempFile(t))
	if got != chalk.LevelNone {
		t.Fatalf("SyncColor = %v, want LevelNone", got)
	}
	if chalk.GetLevel() != chalk.LevelNone {
		t.Fatalf("chalk.GetLevel = %v, want LevelNone", chalk.GetLevel())
	}
	if chalk.Enabled() {
		t.Fatal("chalk.Enabled = true after SyncColor to a redirected stream")
	}
}

func TestInspectNilFile(t *testing.T) {
	clearEnv(t)
	pinLevel(t, chalk.LevelTrueColor)

	info := Inspect(nil)
	want := Info{Terminal: false, Piped: false, Width: DefaultWidth, Height: DefaultHeight, Fallback: true, Color: chalk.LevelNone}
	if info != want {
		t.Errorf("Inspect(nil) = %+v, want %+v", info, want)
	}
}
