package spinner

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malcolmston/chalk"
)

// syncBuf is a Writer safe to read while a spinner goroutine writes to it. The
// spinner serializes its own writes, but a test that inspects the output from
// another goroutine still needs the read side locked.
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
	n   int
}

func (b *syncBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.n++
	return b.buf.Write(p)
}

func (b *syncBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *syncBuf) Writes() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.n
}

// plainSymbols are the default symbols with the colors removed, so assertions
// read as literal text and do not depend on chalk's process-global level.
func plainSymbols() map[State]Symbol {
	return map[State]Symbol{
		StateSucceeded: {Text: "OK"},
		StateFailed:    {Text: "ERR"},
		StateWarned:    {Text: "WARN"},
		StateInformed:  {Text: "INFO"},
	}
}

const clr = "\r\x1b[K"

// TestAnimatedFrames drives the animated path against a buffer with a manual
// clock: every expectation below is exact, with no sleeping anywhere.
func TestAnimatedFrames(t *testing.T) {
	tests := []struct {
		name  string
		cfg   Config
		steps func(s *Spinner, mt *ManualTicker)
		want  string
	}{
		{
			name: "start paints the first frame immediately",
			cfg:  Config{Text: "working"},
			want: clr + "⠋ working",
		},
		{
			name:  "each tick advances one frame",
			cfg:   Config{Text: "working"},
			steps: func(s *Spinner, mt *ManualTicker) { mt.Tick(); mt.Tick() },
			want:  clr + "⠋ working" + clr + "⠙ working" + clr + "⠹ working",
		},
		{
			name: "frames wrap around the set",
			cfg:  Config{Text: "x", Frames: Line},
			steps: func(s *Spinner, mt *ManualTicker) {
				for i := 0; i < 4; i++ {
					mt.Tick()
				}
			},
			want: clr + "- x" + clr + "\\ x" + clr + "| x" + clr + "/ x" + clr + "- x",
		},
		{
			name: "caller supplied frame set",
			cfg:  Config{Text: "x", Frames: FrameSet{Frames: []string{"a", "b"}}},
			steps: func(s *Spinner, mt *ManualTicker) {
				mt.Tick()
				mt.Tick()
			},
			want: clr + "a x" + clr + "b x" + clr + "a x",
		},
		{
			name: "prefix and suffix are written verbatim",
			cfg:  Config{Text: "load", Prefix: "[build] ", Suffix: " ...", Frames: Bars},
			want: clr + "[build] ▁ load ...",
		},
		{
			name: "update text repaints without waiting for a tick",
			cfg:  Config{Text: "first"},
			steps: func(s *Spinner, mt *ManualTicker) {
				s.UpdateText("second")
				mt.Tick()
			},
			want: clr + "⠋ first" + clr + "⠋ second" + clr + "⠙ second",
		},
		{
			name: "update prefix and suffix repaint",
			cfg:  Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) {
				s.UpdatePrefix("> ")
				s.UpdateSuffix(" <")
			},
			want: clr + "⠋ t" + clr + "> ⠋ t" + clr + "> ⠋ t <",
		},
		{
			name: "newlines in the label are flattened",
			cfg:  Config{Text: "a\nb\r\nc"},
			want: clr + "⠋ a b c",
		},
		{
			name:  "stop erases the line and prints nothing",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Stop() },
			want:  clr + "⠋ t" + clr,
		},
		{
			name:  "succeed resolves to its symbol and label",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Succeed("") },
			want:  clr + "⠋ t" + clr + "OK t\n",
		},
		{
			name:  "succeed with its own text replaces the label",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Succeed("done") },
			want:  clr + "⠋ t" + clr + "OK done\n",
		},
		{
			name:  "fail resolves to the failure symbol",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Fail("broke") },
			want:  clr + "⠋ t" + clr + "ERR broke\n",
		},
		{
			name:  "warn resolves to the warning symbol",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Warn("hmm") },
			want:  clr + "⠋ t" + clr + "WARN hmm\n",
		},
		{
			name:  "info resolves to the info symbol",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Info("fyi") },
			want:  clr + "⠋ t" + clr + "INFO fyi\n",
		},
		{
			name:  "a tick after stop paints nothing more",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Stop(); mt.Tick() },
			want:  clr + "⠋ t" + clr,
		},
		{
			name:  "stop is idempotent",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Succeed("done"); s.Succeed("again"); s.Stop() },
			want:  clr + "⠋ t" + clr + "OK done\n",
		},
		{
			name:  "double start does not repaint or restart",
			cfg:   Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) { s.Start(); mt.Tick() },
			want:  clr + "⠋ t" + clr + "⠙ t",
		},
		{
			name: "restart after stop begins from the first frame",
			cfg:  Config{Text: "t"},
			steps: func(s *Spinner, mt *ManualTicker) {
				mt.Tick()
				s.Stop()
				mt.Reset()
				s.Start()
			},
			want: clr + "⠋ t" + clr + "⠙ t" + clr + clr + "⠋ t",
		},
		{
			name: "custom styles are applied to frame and label",
			cfg: Config{
				Text:       "t",
				FrameStyle: chalk.New().Level(chalk.LevelBasic).Red(),
				TextStyle:  chalk.New().Level(chalk.LevelBasic).Bold(),
			},
			want: clr + "\x1b[31m⠋\x1b[39m \x1b[1mt\x1b[22m",
		},
		{
			name: "a state mapped to an empty symbol prints no final line",
			cfg: Config{
				Text:    "t",
				Symbols: map[State]Symbol{StateSucceeded: {}},
			},
			steps: func(s *Spinner, mt *ManualTicker) { s.Succeed("done") },
			want:  clr + "⠋ t" + clr,
		},
		{
			name:  "an empty label draws the frame alone",
			cfg:   Config{},
			steps: func(s *Spinner, mt *ManualTicker) { s.Succeed("") },
			want:  clr + "⠋" + clr + "OK\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf syncBuf
			mt := NewManualTicker()
			cfg := tc.cfg
			cfg.Out = &buf
			cfg.Mode = Animate
			cfg.ShowCursor = true
			cfg.NewTicker = mt.New
			if cfg.Symbols == nil {
				cfg.Symbols = plainSymbols()
			}
			if cfg.FrameStyle == nil {
				cfg.FrameStyle = chalk.New() // unstyled, so assertions are literal
			}

			s := New(cfg).Start()
			if tc.steps != nil {
				tc.steps(s, mt)
			}
			// Read the output before the housekeeping Stop below, so a case
			// that deliberately leaves the spinner running is not charged for
			// the erase that tearing it down writes.
			got := buf.String()
			s.Stop() // always leave the goroutine joined
			mt.Stop()

			if got != tc.want {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestPlainOutput is the degraded path: no animation, no escapes, one line at
// start and one at the end.
func TestPlainOutput(t *testing.T) {
	tests := []struct {
		name  string
		cfg   Config
		steps func(s *Spinner)
		want  string
	}{
		{
			name: "start prints the label once",
			cfg:  Config{Text: "working"},
			want: "working\n",
		},
		{
			name: "prefix and suffix join the label",
			cfg:  Config{Text: "working", Prefix: "[a] ", Suffix: " ..."},
			want: "[a] working ...\n",
		},
		{
			name:  "plain stop adds no line",
			cfg:   Config{Text: "working"},
			steps: func(s *Spinner) { s.Stop() },
			want:  "working\n",
		},
		{
			name:  "succeed adds exactly one final line",
			cfg:   Config{Text: "working"},
			steps: func(s *Spinner) { s.Succeed("done") },
			want:  "working\nOK done\n",
		},
		{
			name:  "fail adds exactly one final line",
			cfg:   Config{Text: "working"},
			steps: func(s *Spinner) { s.Fail("nope") },
			want:  "working\nERR nope\n",
		},
		{
			name: "updates print nothing but reach the final line",
			cfg:  Config{Text: "one"},
			steps: func(s *Spinner) {
				s.UpdateText("two")
				s.UpdateText("three")
				s.UpdatePrefix("x")
				s.UpdateSuffix("y")
				s.Info("")
			},
			want: "one\nINFO three\n",
		},
		{
			name: "a double start still prints one label",
			cfg:  Config{Text: "one"},
			steps: func(s *Spinner) {
				s.Start()
				s.Start()
				s.Succeed("")
			},
			want: "one\nOK one\n",
		},
		{
			// A resolving text replaces the label, so the restarted spinner
			// announces the text it was resolved with, not the original.
			name: "restart after stop prints the label again",
			cfg:  Config{Text: "one"},
			steps: func(s *Spinner) {
				s.Succeed("first")
				s.Start()
				s.Succeed("second")
			},
			want: "one\nOK first\nfirst\nOK second\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf syncBuf
			cfg := tc.cfg
			cfg.Out = &buf
			cfg.Mode = Plain
			cfg.Symbols = plainSymbols()

			s := New(cfg).Start()
			if tc.steps != nil {
				tc.steps(s)
			}

			got := buf.String()
			if got != tc.want {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, tc.want)
			}
			if strings.Contains(got, "\x1b") {
				t.Errorf("plain output contains an escape sequence: %q", got)
			}
		})
	}
}

// TestPlainModeStartsNoGoroutine proves the "no animation into a pipe" promise at
// the level that matters: nothing is scheduled at all, so there is no clock to
// tick and no frame to emit.
func TestPlainModeStartsNoGoroutine(t *testing.T) {
	var buf syncBuf
	ticked := false
	s := New(Config{
		Text: "t",
		Out:  &buf,
		Mode: Plain,
		NewTicker: func(time.Duration) Ticker {
			ticked = true
			return newRealTicker(time.Hour)
		},
	})

	before := goroutines()
	s.Start()
	if ticked {
		t.Error("plain mode built a ticker; it must not animate")
	}
	if got := goroutines(); got > before {
		t.Errorf("plain mode leaked a goroutine: %d -> %d", before, got)
	}
	s.Succeed("done")
	if n := strings.Count(buf.String(), "\n"); n != 2 {
		t.Errorf("want 2 lines (label + status), got %d in %q", n, buf.String())
	}
}

// TestAutoDetectionOnFile is the non-TTY test with a real *os.File: a file on
// disk is a terminal to nobody, so Auto must pick the plain path and write no
// escapes even though Out is an *os.File.
func TestAutoDetectionOnFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s := New(Config{Text: "working", Out: f, Symbols: plainSymbols()})
	if s.Animating() {
		t.Fatal("Animating() is true for a regular file")
	}
	s.Start()
	s.Succeed("done")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "working\nOK done\n"; got != want {
		t.Errorf("file contents = %q, want %q", got, want)
	}
}

// TestAutoDetectionOnBuffer covers the common case: an io.Writer that is not a
// file at all.
func TestAutoDetectionOnBuffer(t *testing.T) {
	var buf bytes.Buffer
	if New(Config{Out: &buf}).Animating() {
		t.Error("a bytes.Buffer must not be treated as a terminal")
	}
}

// TestDumbTerminalIsNotAnimated documents the TERM=dumb rule. It cannot use a
// real tty in a test, so it exercises isTerminal directly on a non-tty file plus
// the env var: the point is that the env check is consulted at construction.
func TestDumbTerminalIsNotAnimated(t *testing.T) {
	t.Setenv("TERM", "dumb")
	f, err := os.Create(filepath.Join(t.TempDir(), "x"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("isTerminal true with TERM=dumb")
	}
}

func TestIntervalResolution(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want time.Duration
	}{
		{name: "default", cfg: Config{}, want: DefaultInterval},
		{name: "from frame set", cfg: Config{Frames: Line}, want: Line.Interval},
		{
			name: "config wins over frame set",
			cfg:  Config{Frames: Line, Interval: 5 * time.Millisecond},
			want: 5 * time.Millisecond,
		},
		{
			name: "non positive interval falls back",
			cfg:  Config{Interval: -1},
			want: DefaultInterval,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.Out = &bytes.Buffer{}
			if got := New(cfg).Interval(); got != tc.want {
				t.Errorf("Interval() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestTickerReceivesResolvedInterval checks the interval actually reaches the
// clock, which is the part a caller can observe.
func TestTickerReceivesResolvedInterval(t *testing.T) {
	var buf syncBuf
	mt := NewManualTicker()
	s := New(Config{
		Out: &buf, Mode: Animate, NewTicker: mt.New,
		Frames: Line, Interval: 7 * time.Millisecond,
	}).Start()
	// Tick once so the goroutine has certainly built the ticker.
	mt.Tick()
	s.Stop()
	if got := mt.Interval(); got != 7*time.Millisecond {
		t.Errorf("ticker interval = %v, want 7ms", got)
	}
}

func TestCursorHiddenAndRestored(t *testing.T) {
	var buf syncBuf
	mt := NewManualTicker()
	s := New(Config{Text: "t", Out: &buf, Mode: Animate, NewTicker: mt.New,
		FrameStyle: chalk.New(), Symbols: plainSymbols()}).Start()
	s.Succeed("done")
	want := hideCursorSeq + clr + "⠋ t" + clr + showCursorSeq + "OK done\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}

	var buf2 syncBuf
	mt2 := NewManualTicker()
	s = New(Config{Text: "t", Out: &buf2, Mode: Animate, NewTicker: mt2.New,
		ShowCursor: true, FrameStyle: chalk.New(), Symbols: plainSymbols()}).Start()
	s.Stop()
	if got := buf2.String(); strings.Contains(got, hideCursorSeq) || strings.Contains(got, showCursorSeq) {
		t.Errorf("ShowCursor still emitted a cursor escape: %q", got)
	}
}

func TestRunningAndText(t *testing.T) {
	var buf syncBuf
	mt := NewManualTicker()
	s := New(Config{Text: "a", Out: &buf, Mode: Animate, NewTicker: mt.New})
	if s.Running() {
		t.Error("Running() before Start")
	}
	s.Start()
	if !s.Running() {
		t.Error("!Running() after Start")
	}
	s.UpdateText("b")
	if got := s.Text(); got != "b" {
		t.Errorf("Text() = %q, want %q", got, "b")
	}
	s.Succeed("c")
	if s.Running() {
		t.Error("Running() after Succeed")
	}
	if got := s.Text(); got != "c" {
		t.Errorf("Text() = %q, want %q", got, "c")
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateStopped, "stopped"},
		{StateSucceeded, "succeeded"},
		{StateFailed, "failed"},
		{StateWarned, "warned"},
		{StateInformed, "informed"},
		{State(42), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("State(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestDefaultSymbolsAreIndependentCopies(t *testing.T) {
	a := DefaultSymbols()
	delete(a, StateSucceeded)
	if _, ok := DefaultSymbols()[StateSucceeded]; !ok {
		t.Error("DefaultSymbols shares state between calls")
	}
}

// TestStopJoinsGoroutine proves Stop joins rather than signals: after Stop
// returns, a delivered tick cannot produce another write.
func TestStopJoinsGoroutine(t *testing.T) {
	var buf syncBuf
	mt := NewManualTicker()
	s := New(Config{Text: "t", Out: &buf, Mode: Animate, NewTicker: mt.New}).Start()
	mt.Tick()
	s.Stop()

	writes := buf.Writes()
	for i := 0; i < 50; i++ {
		if mt.Tick() {
			t.Fatal("ticker consumed a tick after Stop; the goroutine is still alive")
		}
	}
	if got := buf.Writes(); got != writes {
		t.Errorf("writes after Stop: %d -> %d", writes, got)
	}
}

// countingTicker counts how many clocks a spinner built and stopped. Because the
// animation goroutine builds its ticker on entry and stops it on exit, those two
// counters are an exact, non-flaky proxy for "how many goroutines did you start"
// and "did they all exit" — better than watching runtime.NumGoroutine.
type countingTicker struct {
	*ManualTicker
	mu     sync.Mutex
	builds int
	stops  int
}

func (c *countingTicker) New(d time.Duration) Ticker {
	c.mu.Lock()
	c.builds++
	c.mu.Unlock()
	c.ManualTicker.New(d)
	return c
}

func (c *countingTicker) Stop() {
	c.mu.Lock()
	c.stops++
	c.mu.Unlock()
	c.ManualTicker.Stop()
}

func (c *countingTicker) counts() (builds, stops int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.builds, c.stops
}

// TestNoGoroutineLeak covers a double Start (one goroutine, not two) and a clean
// exit after Stop.
func TestNoGoroutineLeak(t *testing.T) {
	var buf syncBuf
	ct := &countingTicker{ManualTicker: NewManualTicker()}
	s := New(Config{Text: "t", Out: &buf, Mode: Animate, NewTicker: ct.New})
	s.Start()
	s.Start()
	s.Start()
	ct.Tick()

	if builds, stops := ct.counts(); builds != 1 || stops != 0 {
		t.Fatalf("after three Starts: builds=%d stops=%d, want 1 and 0", builds, stops)
	}

	s.Stop()
	if builds, stops := ct.counts(); builds != 1 || stops != 1 {
		t.Errorf("after Stop: builds=%d stops=%d, want 1 and 1", builds, stops)
	}
	if got := waitForGoroutines(0, 0); got == 0 {
		t.Error("impossible goroutine count")
	}
}

// TestConcurrentUse hammers every exported method from several goroutines. Its
// real assertion is made by the race detector: run with -race.
func TestConcurrentUse(t *testing.T) {
	var buf syncBuf
	base := goroutines()
	// A real, very fast clock here: the point of this test is contention
	// between the animation goroutine and callers, so the animation should keep
	// running while the workers hammer it.
	s := New(Config{Text: "t", Out: &buf, Mode: Animate, Interval: time.Millisecond})

	var wg sync.WaitGroup
	const workers = 8

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				switch (i + j) % 6 {
				case 0:
					s.Start()
				case 1:
					s.UpdateText("text")
				case 2:
					s.UpdatePrefix("p ")
				case 3:
					s.UpdateSuffix(" s")
				case 4:
					_ = s.Running()
					_ = s.Text()
				case 5:
					s.Stop()
				}
			}
		}(i)
	}
	wg.Wait()

	s.Succeed("done")

	if s.Running() {
		t.Error("spinner still running after the final Succeed")
	}
	if got := waitForGoroutines(base, 2*time.Second); got > base {
		t.Errorf("goroutine leaked after the final Succeed: %d, want %d", got, base)
	}
}

// TestRealClockAnimates is the one test that uses wall-clock time, to prove the
// default ticker is wired up at all. It waits for frames rather than sleeping a
// fixed amount.
func TestRealClockAnimates(t *testing.T) {
	var buf syncBuf
	s := New(Config{
		Text: "t", Out: &buf, Mode: Animate, ShowCursor: true,
		Interval: time.Millisecond, FrameStyle: chalk.New(),
		Symbols: plainSymbols(),
	}).Start()

	deadline := time.Now().Add(2 * time.Second)
	for buf.Writes() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	s.Succeed("done")

	if n := buf.Writes(); n < 3 {
		t.Errorf("real ticker produced %d writes, want at least 3", n)
	}
	if got := buf.String(); !strings.HasSuffix(got, clr+"OK done\n") {
		t.Errorf("output does not end with the resolved line: %q", got)
	}
}

func goroutines() int {
	runtime.Gosched()
	return runtime.NumGoroutine()
}

// waitForGoroutines waits until the goroutine count reaches want, or the timeout
// elapses, and returns the last count seen. Goroutine bookkeeping is inherently
// asynchronous, so polling is the only honest way to assert on it.
func waitForGoroutines(want int, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	got := goroutines()
	for got != want && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		got = goroutines()
	}
	return got
}
