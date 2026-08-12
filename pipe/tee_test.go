package pipe

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/malcolmston/chalk"
)

// levelPtr is a helper for TeeConfig.Level, whose nil-versus-set distinction is
// the difference between detection and an explicit pin.
func levelPtr(l chalk.Level) *chalk.Level { return &l }

func TestTee(t *testing.T) {
	red := chalk.New().Red()

	tests := []struct {
		name       string
		cfg        TeeConfig
		do         func(*Tee) error
		wantScreen string
		wantLog    string
	}{
		{
			name:       "no level pinned means no color for a buffer",
			cfg:        TeeConfig{Style: red},
			do:         func(tee *Tee) error { return tee.Print("hi") },
			wantScreen: "hi",
			wantLog:    "hi",
		},
		{
			name:       "pinned level styles the screen only",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.Print("hi") },
			wantScreen: "\x1b[31mhi\x1b[39m",
			wantLog:    "hi",
		},
		{
			name:       "Println puts the newline outside the style",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.Println("a", "b") },
			wantScreen: "\x1b[31ma b\x1b[39m\n",
			wantLog:    "a b\n",
		},
		{
			name:       "Printf formats",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.Printf("%d%%", 42) },
			wantScreen: "\x1b[31m42%\x1b[39m",
			wantLog:    "42%",
		},
		{
			name:       "already styled text reaches the screen and is stripped for the log",
			cfg:        TeeConfig{Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.Print(chalk.New().Level(chalk.LevelBasic).Green().Sprint("ok")) },
			wantScreen: "\x1b[32mok\x1b[39m",
			wantLog:    "ok",
		},
		{
			name: "already styled text is stripped for a non-terminal screen",
			cfg:  TeeConfig{},
			do: func(tee *Tee) error {
				return tee.Print(chalk.New().Level(chalk.LevelTrueColor).Hex("#ff8800").Sprint("warn"))
			},
			wantScreen: "warn",
			wantLog:    "warn",
		},
		{
			name:       "Write does not apply the style",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { _, err := tee.Write([]byte("raw")); return err },
			wantScreen: "raw",
			wantLog:    "raw",
		},
		{
			name:       "WriteString does not apply the style",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { _, err := tee.WriteString("raw"); return err },
			wantScreen: "raw",
			wantLog:    "raw",
		},
		{
			name:       "WithStyle overrides",
			cfg:        TeeConfig{Style: red, Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.WithStyle(chalk.New().Bold()).Print("b") },
			wantScreen: "\x1b[1mb\x1b[22m",
			wantLog:    "b",
		},
		{
			name:       "no style at all passes text through",
			cfg:        TeeConfig{Level: levelPtr(chalk.LevelBasic)},
			do:         func(tee *Tee) error { return tee.Print("plain") },
			wantScreen: "plain",
			wantLog:    "plain",
		},
		{
			name:       "truecolor style downgrades with the pinned level",
			cfg:        TeeConfig{Style: chalk.New().Hex("#ff8800"), Level: levelPtr(chalk.Level256)},
			do:         func(tee *Tee) error { return tee.Print("hot") },
			wantScreen: "\x1b[38;5;214mhot\x1b[39m",
			wantLog:    "hot",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var screen, logBuf bytes.Buffer
			cfg := tc.cfg
			cfg.Screen = &screen
			cfg.Log = &logBuf
			tee := NewTee(cfg)
			if err := tc.do(tee); err != nil {
				t.Fatalf("write: %v", err)
			}
			if screen.String() != tc.wantScreen {
				t.Errorf("screen = %q, want %q", screen.String(), tc.wantScreen)
			}
			if logBuf.String() != tc.wantLog {
				t.Errorf("log = %q, want %q", logBuf.String(), tc.wantLog)
			}
		})
	}
}

// TestTeeNonTerminalScreenNeverEmitsEscapes is the degraded-path guarantee: with
// no Level pinned and a screen that is not a terminal, nothing written through a
// Tee contains an escape byte, no matter what the caller handed it.
func TestTeeNonTerminalScreenNeverEmitsEscapes(t *testing.T) {
	clearEnv(t)
	pinLevel(t, chalk.LevelTrueColor)

	var screen, logBuf bytes.Buffer
	tee := NewTee(TeeConfig{Screen: &screen, Log: &logBuf, Style: chalk.New().Red().Bold()})
	if tee.ColorEnabled() {
		t.Fatal("ColorEnabled = true for a bytes.Buffer screen")
	}
	if tee.Level() != chalk.LevelNone {
		t.Fatalf("Level = %v, want LevelNone", tee.Level())
	}
	if err := tee.Println(chalk.New().Level(chalk.LevelTrueColor).Cyan().Sprint("styled"), 1, true); err != nil {
		t.Fatalf("Println: %v", err)
	}
	if err := tee.Printf("%s\n", chalk.New().Level(chalk.Level256).Ansi256(200).Sprint("more")); err != nil {
		t.Fatalf("Printf: %v", err)
	}
	for name, got := range map[string]string{"screen": screen.String(), "log": logBuf.String()} {
		if strings.ContainsRune(got, 0x1b) {
			t.Errorf("%s contains an escape byte: %q", name, got)
		}
	}
	if screen.String() != logBuf.String() {
		t.Errorf("screen %q and log %q should be identical without color", screen.String(), logBuf.String())
	}
	if want := "styled 1 true\nmore\n"; screen.String() != want {
		t.Errorf("screen = %q, want %q", screen.String(), want)
	}
}

func TestTeeNilLog(t *testing.T) {
	var screen bytes.Buffer
	tee := NewTee(TeeConfig{Screen: &screen, Level: levelPtr(chalk.LevelBasic), Style: chalk.New().Red()})
	if err := tee.Print("x"); err != nil {
		t.Fatalf("Print: %v", err)
	}
	if screen.String() != "\x1b[31mx\x1b[39m" {
		t.Errorf("screen = %q", screen.String())
	}
}

func TestTeeWriteContract(t *testing.T) {
	var screen bytes.Buffer
	tee := NewTee(TeeConfig{Screen: &screen, Level: levelPtr(chalk.LevelBasic)})

	p := []byte(chalk.New().Level(chalk.LevelBasic).Red().Sprint("hello"))
	n, err := tee.Write(p)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(p) {
		t.Errorf("n = %d, want %d (the input length, not the rendered length)", n, len(p))
	}
	if n, err := tee.WriteString("abc"); err != nil || n != 3 {
		t.Errorf("WriteString = %d, %v; want 3, nil", n, err)
	}
	// A Tee must be usable as the destination of a log.Logger.
	lg := log.New(tee, "", 0)
	lg.Print("through a logger")
	if !strings.Contains(screen.String(), "through a logger\n") {
		t.Errorf("screen = %q", screen.String())
	}
}

func TestTeeWriteErrors(t *testing.T) {
	t.Run("screen error", func(t *testing.T) {
		tee := NewTee(TeeConfig{Screen: errWriter{}, Log: &bytes.Buffer{}})
		if _, err := tee.Write([]byte("x")); !errors.Is(err, errBoom) {
			t.Fatalf("err = %v, want errBoom", err)
		}
		if err := tee.Println("x"); !errors.Is(err, errBoom) {
			t.Fatalf("Println err = %v, want errBoom", err)
		}
	})
	t.Run("log error", func(t *testing.T) {
		var screen bytes.Buffer
		tee := NewTee(TeeConfig{Screen: &screen, Log: errWriter{}})
		if err := tee.Print("x"); !errors.Is(err, errBoom) {
			t.Fatalf("err = %v, want errBoom", err)
		}
		if screen.String() != "x" {
			t.Errorf("screen = %q; the screen copy should still have been written", screen.String())
		}
	})
}

// TestTeeConcurrent proves a Tee is safe to write from many goroutines: run with
// -race, and check that no line was lost or interleaved.
func TestTeeConcurrent(t *testing.T) {
	var screen, logBuf syncBuffer
	tee := NewTee(TeeConfig{Screen: &screen, Log: &logBuf, Level: levelPtr(chalk.LevelBasic), Style: chalk.New().Green()})
	derived := tee.WithStyle(chalk.New().Red())

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if err := tee.Println("green", i); err != nil {
				t.Errorf("Println: %v", err)
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			if err := derived.Printf("red %d\n", i); err != nil {
				t.Errorf("Printf: %v", err)
			}
		}(i)
	}
	wg.Wait()

	lines := SplitLines(logBuf.String())
	if len(lines) != 2*n {
		t.Fatalf("log has %d lines, want %d", len(lines), 2*n)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "green ") && !strings.HasPrefix(l, "red ") {
			t.Fatalf("interleaved line %q", l)
		}
		if strings.ContainsRune(l, 0x1b) {
			t.Fatalf("log line carries an escape sequence: %q", l)
		}
	}
	if !strings.Contains(screen.String(), "\x1b[31m") || !strings.Contains(screen.String(), "\x1b[32m") {
		t.Error("screen copy lost its styling")
	}
}

// syncBuffer is a bytes.Buffer with a lock. The Tee serializes its own writes,
// but the test reads the buffer afterwards and a plain bytes.Buffer would make
// -race flag the test's own access pattern rather than the code under test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Example_tee shows the package's reason for existing: one call, a styled line on
// screen and a clean line in the log. Both destinations are buffers here, which
// is also how it is tested.
func Example_tee() {
	var screen, logFile bytes.Buffer
	level := chalk.LevelBasic // pinned so the example is deterministic
	tee := NewTee(TeeConfig{Screen: &screen, Log: &logFile, Style: chalk.New().Red(), Level: &level})

	_ = tee.Println("build failed")

	fmt.Printf("screen: %q\nlog:    %q\n", screen.String(), logFile.String())
	// Output:
	// screen: "\x1b[31mbuild failed\x1b[39m\n"
	// log:    "build failed\n"
}

// Example_lines filters and rewrites a piped stream.
func Example_lines() {
	in := strings.NewReader("keep me\n\nDROP\nkeep this too\n")
	var out bytes.Buffer

	n, err := Lines(LinesConfig{
		In:  in,
		Out: &out,
		Transform: func(line string) (string, bool) {
			if line == "" || line == "DROP" {
				return "", false
			}
			return "> " + line, true
		},
	})
	fmt.Println(n, err)
	fmt.Print(out.String())
	// Output:
	// 2 <nil>
	// > keep me
	// > keep this too
}

// Example_readAll shows the no-hang contract: reading a reader that is not a
// terminal just works, and a terminal is refused rather than blocking.
func Example_readAll() {
	data, err := ReadAll(ReadConfig{In: strings.NewReader("piped payload")})
	fmt.Printf("%q %v\n", data, err)
	// Output:
	// "piped payload" <nil>
}
