package progress

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// clock is a manually advanced clock. Every test that touches rate, ETA or the
// redraw throttle uses one, so no test sleeps and no test depends on the machine
// it runs on. It is mutex-guarded because animating bars read it from their own
// goroutine.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *clock {
	return &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func TestRenderTemplates(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		elapsed time.Duration
		set     int64
		want    string
	}{
		{
			name:    "half done with percent counts rate and eta",
			cfg:     Config{Total: 100, Width: 10},
			elapsed: 10 * time.Second,
			set:     50,
			want:    "[█████░░░░░]  50% 50/100 5.0/s eta 10s",
		},
		{
			name:    "empty bar has no rate and no eta",
			cfg:     Config{Total: 100, Width: 10},
			elapsed: 10 * time.Second,
			set:     0,
			want:    "[░░░░░░░░░░]   0% 0/100 0/s eta --",
		},
		{
			name:    "complete bar reports zero eta",
			cfg:     Config{Total: 100, Width: 10},
			elapsed: time.Minute,
			set:     100,
			want:    "[██████████] 100% 100/100 1.7/s eta 0s",
		},
		{
			name:    "bytes mode uses IEC units",
			cfg:     Config{Total: 4096, Width: 8, Bytes: true, Template: "{current}/{total} {rate}"},
			elapsed: 2 * time.Second,
			set:     1536,
			want:    "1.5 KiB/4.0 KiB 768 B/s",
		},
		{
			name: "custom charset with a head",
			cfg: Config{Total: 10, Width: 10, Template: "{bar}",
				Charset: Charset{Filled: "=", Empty: "-", Head: ">", Left: "(", Right: ")"}},
			set:  4,
			want: "(===>------)",
		},
		{
			name:    "description elapsed and spinner",
			cfg:     Config{Total: 10, Width: 4, Description: "copy", Template: "{desc}{spinner} {elapsed} {bar}"},
			elapsed: 90 * time.Second,
			set:     5,
			want:    "copy | 1m30s [██░░]",
		},
		{
			name:    "indeterminate hides percent total and eta",
			cfg:     Config{Width: 10, Template: "{bar} {percent} {current}/{total} eta {eta}"},
			elapsed: 0,
			set:     7,
			want:    "[███░░░░░░░] --% 7/? eta --",
		},
		{
			name:    "indeterminate block bounces off the right edge",
			cfg:     Config{Width: 10, Template: "{bar}"},
			elapsed: 7 * pulseInterval,
			set:     0,
			want:    "[░░░░░░░███]",
		},
		{
			name:    "indeterminate block returns from the right edge",
			cfg:     Config{Width: 10, Template: "{bar}"},
			elapsed: 8 * pulseInterval,
			set:     0,
			want:    "[░░░░░░███░]",
		},
		{
			name: "unknown placeholders and literals pass through",
			cfg:  Config{Total: 10, Template: "n={current} {nope} {unclosed"},
			set:  3,
			want: "n=3 {nope} {unclosed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newClock()
			cfg := tc.cfg
			cfg.Now = c.Now
			cfg.Out = &bytes.Buffer{}
			b := New(cfg)
			c.Advance(tc.elapsed)
			b.Set(tc.set)
			if got := b.Render(); got != tc.want {
				t.Errorf("Render()\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestClampAndAccessors(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		do    func(*Bar)
		want  int64
	}{
		{"set below zero clamps", 100, func(b *Bar) { b.Set(-5) }, 0},
		{"set above total clamps", 100, func(b *Bar) { b.Set(500) }, 100},
		{"add above total clamps", 100, func(b *Bar) { b.Add(60); b.Add(60) }, 100},
		{"add is unbounded when total is unknown", 0, func(b *Bar) { b.Add(1 << 40) }, 1 << 40},
		{"write counts bytes", 100, func(b *Bar) {
			if n, err := b.Write([]byte("hello")); n != 5 || err != nil {
				t.Fatalf("Write = %d, %v", n, err)
			}
		}, 5},
		{"set total shrinks the current value", 100, func(b *Bar) { b.Set(80); b.SetTotal(50) }, 50},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := New(Config{Total: tc.total, Out: &bytes.Buffer{}})
			tc.do(b)
			if got := b.Current(); got != tc.want {
				t.Errorf("Current() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDescribeAndTotalAccessor(t *testing.T) {
	b := New(Config{Total: 4, Out: &bytes.Buffer{}, Template: "{desc}{total}"})
	b.Describe("job")
	if got, want := b.Render(), "job 4"; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
	if got := b.Total(); got != 4 {
		t.Errorf("Total() = %d, want 4", got)
	}
	b.SetTotal(0)
	if got := b.Total(); got != 0 {
		t.Errorf("Total() after SetTotal(0) = %d, want 0", got)
	}
	if got, want := b.Render(), "job ?"; got != want {
		t.Errorf("indeterminate Render() = %q, want %q", got, want)
	}
}

// TestTerminalModeRedrawsOneLine pins the in-place contract: every frame starts
// with a carriage return and ends with erase-to-end-of-line, and the only
// newline in the whole session is the one Finish writes.
func TestTerminalModeRedrawsOneLine(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	b := New(Config{Total: 4, Width: 4, Out: &out, Mode: ModeTerminal, Now: c.Now,
		RefreshInterval: -1, Template: "{percent}"})
	for i := 0; i < 4; i++ {
		b.Add(1)
	}
	if n := strings.Count(out.String(), "\n"); n != 0 {
		t.Fatalf("in-place frames contained %d newlines, want 0: %q", n, out.String())
	}
	b.Finish()
	got := out.String()
	if n := strings.Count(got, "\n"); n != 1 {
		t.Errorf("after Finish: %d newlines, want 1: %q", n, got)
	}
	if want := "\r 25%\x1b[K"; !strings.Contains(got, want) {
		t.Errorf("missing frame %q in %q", want, got)
	}
	if !strings.HasSuffix(got, "\r100%\x1b[K\r\n") {
		t.Errorf("unexpected final frame: %q", got)
	}
	if n := strings.Count(got, "\x1b[K"); n != 5 {
		t.Errorf("got %d frames, want 5 (4 updates + Finish): %q", n, got)
	}
}

// TestNonTerminalDegradesToPlainLines is the degraded path: a bytes.Buffer is not
// a terminal, so nothing escape-like may be written and progress must arrive as
// whole lines at a bounded rate.
func TestNonTerminalDegradesToPlainLines(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	b := New(Config{Total: 100, Width: 4, Out: &out, Now: c.Now, Template: "{percent}"})
	for i := 0; i < 100; i++ {
		c.Advance(time.Second) // 100 seconds of work, PlainInterval is 5s
		b.Add(1)
	}
	b.Finish()
	got := out.String()
	if strings.ContainsAny(got, "\x1b\r") {
		t.Fatalf("plain output contained an escape or carriage return: %q", got)
	}
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("plain output does not end in a newline: %q", got)
	}
	// 100 seconds at one line per 5 seconds, plus the first frame and the final
	// one: bounded, and nowhere near 100 lines.
	if len(lines) < 5 || len(lines) > 25 {
		t.Errorf("got %d plain lines, want a bounded handful: %q", len(lines), got)
	}
	if lines[len(lines)-1] != "100%" {
		t.Errorf("last plain line = %q, want %q", lines[len(lines)-1], "100%")
	}
}

// TestPlainModeSkipsUnchangedLines: a frozen clock means the throttle never
// opens, so only the first frame and Finish's final frame reach the log.
func TestPlainModeSkipsUnchangedLines(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	b := New(Config{Total: 10, Out: &out, Mode: ModePlain, Now: c.Now, Template: "{current}"})
	for i := 0; i < 10; i++ {
		b.Add(1)
	}
	b.Finish()
	if got, want := out.String(), "1\n10\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}

	// With the throttle disabled, an unchanged frame is still not repeated.
	out.Reset()
	b2 := New(Config{Total: 10, Out: &out, Mode: ModePlain, Now: c.Now, PlainInterval: -1,
		Template: "{current}"})
	b2.Set(3)
	b2.Set(3)
	b2.Set(4)
	if got, want := out.String(), "3\n4\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestFinishIsIdempotentAndStopsDrawing(t *testing.T) {
	var out bytes.Buffer
	b := New(Config{Total: 10, Out: &out, Mode: ModePlain, PlainInterval: -1, Template: "{current}"})
	b.Finish()
	after := out.Len()
	b.Finish()
	b.Add(1)
	b.Set(2)
	b.Describe("x")
	if out.Len() != after {
		t.Errorf("bar drew after Finish: %q", out.String())
	}
	if got, want := out.String(), "10\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

// TestTerminalModeAutoDetection proves ModeAuto never picks the escape-emitting
// path for something that is not a terminal, including a real *os.File that
// happens to be a pipe (the shape a CI log has).
func TestTerminalModeAutoDetection(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()

	tests := []struct {
		name string
		mode Mode
		out  io.Writer
		want bool
	}{
		{"auto with a buffer is plain", ModeAuto, &bytes.Buffer{}, false},
		{"auto with a pipe file is plain", ModeAuto, w, false},
		{"auto with a discard writer is plain", ModeAuto, io.Discard, false},
		{"forced terminal", ModeTerminal, &bytes.Buffer{}, true},
		{"forced plain on a file", ModePlain, w, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := terminalMode(tc.mode, tc.out); got != tc.want {
				t.Errorf("terminalMode = %v, want %v", got, tc.want)
			}
		})
	}
}

// blockingWriter fails the test if it is written to after seal is called. It is
// how the goroutine tests prove Stop really joined: after Stop returns, no
// further write may happen, which is a leak check that needs no sleep.
type blockingWriter struct {
	mu     sync.Mutex
	t      *testing.T
	sealed bool
	n      int
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sealed {
		w.t.Errorf("write after Stop: %q", p)
	}
	w.n += len(p)
	return len(p), nil
}

func (w *blockingWriter) seal() {
	w.mu.Lock()
	w.sealed = true
	w.mu.Unlock()
}

func TestStartStopNoWritesAfterStop(t *testing.T) {
	w := &blockingWriter{t: t}
	c := newClock()
	b := New(Config{Out: w, Mode: ModeTerminal, Now: c.Now, RefreshInterval: time.Millisecond})
	b.Start()
	b.Start() // second Start is a no-op, not a second goroutine
	for i := 0; i < 200; i++ {
		c.Advance(time.Millisecond)
		b.Add(1)
	}
	b.Stop()
	b.Stop() // idempotent
	w.seal()
	if w.n == 0 {
		t.Error("animated bar wrote nothing")
	}
}

// TestConcurrentUpdatesAreRaceClean drives one bar from many goroutines while it
// animates. Run with -race.
func TestConcurrentUpdatesAreRaceClean(t *testing.T) {
	w := &blockingWriter{t: t}
	c := newClock()
	b := New(Config{Total: 1000, Out: w, Mode: ModeTerminal, Now: c.Now,
		RefreshInterval: time.Millisecond})
	b.Start()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				b.Add(1)
				c.Advance(time.Millisecond)
				_ = b.Render()
			}
		}()
	}
	// Stop from a different goroutine than Start, concurrently with the writers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Stop()
		b.Start()
	}()
	wg.Wait()

	if got := b.Current(); got != 800 {
		t.Errorf("Current() = %d, want 800 (no lost updates)", got)
	}
	b.Finish()
	w.seal()
}

func TestStartAfterFinishDoesNothing(t *testing.T) {
	w := &blockingWriter{t: t}
	b := New(Config{Total: 1, Out: w, Mode: ModeTerminal, RefreshInterval: time.Millisecond})
	b.Finish()
	b.Start()
	w.seal()
	b.Stop()
}
