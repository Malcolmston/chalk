package progress

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMultiBarRender(t *testing.T) {
	c := newClock()
	m := NewMulti(MultiConfig{Out: &bytes.Buffer{}, Now: c.Now})
	a := m.New(Config{Total: 10, Width: 5, Description: "a", Template: "{desc}{bar} {percent}"})
	b := m.New(Config{Total: 10, Width: 5, Description: "b", Template: "{desc}{bar} {percent}"})
	a.Set(5)
	b.Set(10)

	want := "a [██░░░]  50%\nb [█████] 100%"
	if got := m.Render(); got != want {
		t.Errorf("Render()\n got %q\nwant %q", got, want)
	}
}

// TestMultiBarTerminalOwnsTheCursor checks the block repaint: one cursor-up of
// exactly the block height, then one erased line per bar. Two independent bars
// could not do this without overwriting each other.
func TestMultiBarTerminalOwnsTheCursor(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	m := NewMulti(MultiConfig{Out: &out, Mode: ModeTerminal, Now: c.Now, RefreshInterval: -1})
	a := m.New(Config{Total: 4, Width: 2, Template: "a{current}"})
	b := m.New(Config{Total: 4, Width: 2, Template: "b{current}"})

	// Creating the bars painted the block once; from here on every repaint is a
	// cursor-up of the block height followed by the block.
	out.Reset()
	a.Add(1)
	if got, want := out.String(), "\r\x1b[2A\ra1\x1b[K\n\rb0\x1b[K\n"; got != want {
		t.Fatalf("block repaint = %q, want %q", got, want)
	}
	out.Reset()
	b.Add(2)
	if got, want := out.String(), "\r\x1b[2A\ra1\x1b[K\n\rb2\x1b[K\n"; got != want {
		t.Errorf("second repaint = %q, want %q", got, want)
	}
	// Every repaint moves up exactly as many lines as it wrote, so nothing
	// scrolls and no line is ever orphaned.
	m.Stop()
	full := out.String()
	if strings.Count(full, "\x1b[2A") != strings.Count(full, "\n")/2 {
		t.Errorf("cursor moves do not match written lines: %q", full)
	}
}

// TestMultiBarPlainModeEmitsChangedLinesOnly is the degraded group path: no
// cursor escapes at all, and one line per change instead of a repainted block.
func TestMultiBarPlainModeEmitsChangedLinesOnly(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	m := NewMulti(MultiConfig{Out: &out, Now: c.Now, PlainInterval: -1})
	a := m.New(Config{Total: 4, Description: "a", Template: "{desc}{current}"})
	b := m.New(Config{Total: 4, Description: "b", Template: "{desc}{current}"})
	a.Add(1)
	b.Add(2)
	a.Finish()
	b.Finish()

	got := out.String()
	if strings.ContainsAny(got, "\x1b\r") {
		t.Fatalf("plain group output contained an escape or carriage return: %q", got)
	}
	want := strings.Join([]string{"a 0", "b 0", "a 1", "b 2", "a 4", "b 4", ""}, "\n")
	if got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

// TestMultiBarConcurrentBarsAreRaceClean advances several grouped bars from
// several goroutines while the group animates. Run with -race. It also proves the
// writes do not interleave: every frame the group emits is a whole line.
func TestMultiBarConcurrentBarsAreRaceClean(t *testing.T) {
	w := &lineWriter{}
	c := newClock()
	m := NewMulti(MultiConfig{Out: w, Mode: ModeTerminal, Now: c.Now,
		RefreshInterval: time.Millisecond})
	bars := make([]*Bar, 4)
	for i := range bars {
		bars[i] = m.New(Config{Total: 100, Width: 8, Description: fmt.Sprintf("job%d", i),
			Template: "{desc}{bar} {percent}"})
	}
	m.Start()

	var wg sync.WaitGroup
	for _, b := range bars {
		wg.Add(1)
		go func(b *Bar) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				b.Add(1)
				c.Advance(time.Millisecond)
			}
		}(b)
	}
	wg.Wait()
	m.Stop()

	for _, b := range bars {
		if got := b.Current(); got != 100 {
			t.Errorf("%s: Current() = %d, want 100", b.Render(), got)
		}
	}
	if got := len(m.Render()); got == 0 {
		t.Error("Render() is empty after the run")
	}
	if !strings.HasSuffix(m.Render(), "100%") {
		t.Errorf("last line not complete: %q", m.Render())
	}
	w.check(t)
}

// lineWriter records writes and verifies each one is a self-contained repaint
// (starts with a carriage return, ends with a newline), which is what "no
// interleaving garbage" means in practice.
type lineWriter struct {
	mu     sync.Mutex
	writes []string
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.writes = append(w.writes, string(p))
	w.mu.Unlock()
	return len(p), nil
}

func (w *lineWriter) check(t *testing.T) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.writes) == 0 {
		t.Fatal("group wrote nothing")
	}
	for i, s := range w.writes {
		if !strings.HasPrefix(s, "\r") || !strings.HasSuffix(s, "\n") {
			t.Fatalf("write %d is not a whole repaint: %q", i, s)
		}
	}
}

func TestMultiBarStopIsIdempotentAndGroupOverridesIO(t *testing.T) {
	var out bytes.Buffer
	c := newClock()
	m := NewMulti(MultiConfig{Out: &out, Mode: ModeTerminal, Now: c.Now, RefreshInterval: -1})
	// A bar's own Out/Mode/Now must be ignored: the group owns the cursor.
	other := &bytes.Buffer{}
	b := m.New(Config{Total: 2, Width: 2, Out: other, Mode: ModePlain, Template: "x{current}"})
	b.Add(1)
	m.Start()
	m.Stop()
	m.Stop()
	if other.Len() != 0 {
		t.Errorf("grouped bar wrote to its own Out: %q", other.String())
	}
	if !strings.Contains(out.String(), "\rx1\x1b[K") {
		t.Errorf("group output missing the bar frame: %q", out.String())
	}
}
