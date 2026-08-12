package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// MultiConfig configures a [MultiBar]. Its writer, mode, clock and intervals are
// imposed on every bar the group creates: sharing one cursor only works if one
// object owns it.
type MultiConfig struct {
	// Out is the destination (defaults to os.Stdout).
	Out io.Writer
	// Mode forces the output style (default ModeAuto).
	Mode Mode
	// Now is the clock (defaults to time.Now).
	Now func() time.Time
	// RefreshInterval bounds in-place repaints (default
	// DefaultRefreshInterval); negative disables the throttle.
	RefreshInterval time.Duration
	// PlainInterval bounds plain-text lines (default DefaultPlainInterval);
	// negative disables the throttle.
	PlainInterval time.Duration
}

// MultiBar draws several bars as one block of lines. It exists because two bars
// writing carriage returns to the same terminal would overwrite each other's
// line: the group owns the cursor, repaints every bar together, and serialises
// the writes, so bars may be advanced from any number of goroutines. In plain
// mode there is no cursor to own, and the group simply emits the lines that
// changed, rate-limited like a single bar — include {desc} in each bar's template
// so the log lines can be told apart.
type MultiBar struct {
	mu    sync.Mutex
	out   io.Writer
	mode  Mode
	now   func() time.Time
	plain bool
	// refresh is the throttle for the resolved mode.
	refresh time.Duration
	// intervals are kept so bars created later inherit them.
	tickIv, plainIv time.Duration

	bars      []*Bar
	prevLines int
	lastDraw  time.Time
	drawn     bool
	lastPlain map[*Bar]string

	started bool
	stopped bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewMulti creates a bar group.
func NewMulti(cfg MultiConfig) *MultiBar {
	m := &MultiBar{
		out:       cfg.Out,
		mode:      cfg.Mode,
		now:       cfg.Now,
		tickIv:    cfg.RefreshInterval,
		plainIv:   cfg.PlainInterval,
		lastPlain: map[*Bar]string{},
	}
	if m.out == nil {
		m.out = os.Stdout
	}
	if m.now == nil {
		m.now = time.Now
	}
	m.plain = !terminalMode(cfg.Mode, m.out)
	if m.plain {
		m.refresh = cfg.PlainInterval
		if m.refresh == 0 {
			m.refresh = DefaultPlainInterval
		}
	} else {
		m.refresh = cfg.RefreshInterval
		if m.refresh == 0 {
			m.refresh = DefaultRefreshInterval
		}
	}
	return m
}

// New adds a bar to the group and returns it. The group's Out, Mode, Now and
// intervals replace whatever cfg carried, because the group — not the bar — owns
// the cursor and the redraw budget. Everything else in cfg (Total, Width,
// Template, Description, Bytes, Charset) is the bar's own.
func (m *MultiBar) New(cfg Config) *Bar {
	cfg.Out, cfg.Mode, cfg.Now = m.out, m.mode, m.now
	cfg.RefreshInterval, cfg.PlainInterval = m.tickIv, m.plainIv
	b := &Bar{}
	b.init(cfg, m)
	m.mu.Lock()
	m.bars = append(m.bars, b)
	m.mu.Unlock()
	m.repaint(true)
	return b
}

// Render returns the group's frame as plain text: one line per bar, joined by
// newlines, with no escape sequences. This is the group's testing seam.
func (m *MultiBar) Render() string { return strings.Join(m.lines(), "\n") }

// lines snapshots each bar's rendered line. It takes the bars' locks one at a
// time and never while holding one of them, which together with Bar.draw's
// discipline (never call into the group holding a bar lock) keeps the two lock
// levels ordered.
func (m *MultiBar) lines() []string {
	m.mu.Lock()
	bars := make([]*Bar, len(m.bars))
	copy(bars, m.bars)
	m.mu.Unlock()
	out := make([]string, len(bars))
	for i, b := range bars {
		out[i] = b.Render()
	}
	return out
}

// repaint redraws the whole block if the throttle allows it (or force is set).
func (m *MultiBar) repaint(force bool) {
	lines := m.lines()

	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if !force && !dueAt(now, m.lastDraw, m.drawn, m.refresh) {
		return
	}
	if m.plain {
		// No cursor to move: emit only the bars whose text changed, so a CI log
		// grows by lines that carry information.
		bars := m.bars
		for i, b := range bars {
			if i >= len(lines) {
				break
			}
			if m.lastPlain[b] == lines[i] {
				continue
			}
			m.lastPlain[b] = lines[i]
			writeString(m.out, lines[i]+"\n")
		}
	} else {
		var sb strings.Builder
		if m.prevLines > 0 {
			// Return to the top of the block. The previous repaint left the
			// cursor on the line after it.
			sb.WriteString(fmt.Sprintf("\r\x1b[%dA", m.prevLines))
		}
		for _, l := range lines {
			sb.WriteString("\r" + l + "\x1b[K\n")
		}
		writeString(m.out, sb.String())
		m.prevLines = len(lines)
	}
	m.lastDraw, m.drawn = now, true
}

// Start animates the group from one goroutine, repainting every refresh
// interval. Like [Bar.Start] it is a no-op when already started, and Stop joins
// the goroutine.
func (m *MultiBar) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	iv := m.refresh
	if iv <= 0 {
		iv = pulseInterval
	}
	m.started, m.stopped = true, false
	m.stopCh = make(chan struct{})
	stop := m.stopCh
	m.mu.Unlock()

	m.repaint(true)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		t := time.NewTicker(iv)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				m.repaint(false)
			}
		}
	}()
}

// Stop halts the group's goroutine, waits for it to exit, and paints one final
// frame so the block shows the finished state. It is safe from any goroutine and
// safe to call repeatedly. It does not call Finish on the member bars: a caller
// may want to leave a partially-completed bar visible as-is.
func (m *MultiBar) Stop() {
	m.mu.Lock()
	if m.started && !m.stopped {
		m.stopped = true
		close(m.stopCh)
	}
	m.started = false
	m.mu.Unlock()
	m.wg.Wait()
	m.repaint(true)
}
