// Package progress draws terminal progress bars, in the house style of the
// sibling chalk/prompts package: one config struct per entry point, an
// injectable Out (and clock), and a graceful degradation when there is no
// terminal on the other end.
//
//	bar := progress.New(progress.Config{Total: 100})
//	for i := 0; i < 100; i++ {
//		bar.Add(1)
//	}
//	bar.Finish()
//
// A [Bar] is a counter plus a layout. [Config] carries the Total, the bar Width,
// the output writer, and a Template that decides where the bar, the percentage,
// the counts, the rate and the ETA appear, so the caller owns the layout instead
// of the package imposing one. Progress is reported with [Bar.Add] and
// [Bar.Set], the bar is closed with [Bar.Finish], and [Bar.Render] returns the
// line that would be drawn as a plain string — which is what makes a bar
// testable with no terminal at all.
//
// # Two output modes
//
// On a terminal a bar rewrites a single line: carriage return, the frame, then
// erase-to-end-of-line. Nothing scrolls, and no cursor-movement escape ever
// reaches a non-terminal. When Out is not a terminal — a pipe, a file, a CI log
// — the bar switches to plain mode: it prints whole lines, terminated by "\n",
// no escapes at all, and at a bounded rate (see Config.PlainInterval, five
// seconds by default) so a long job leaves a handful of log lines instead of
// thousands. A plain line is only emitted when the rendered text actually
// changed, and the final frame from Finish is always emitted. Detection is
// automatic (Out must be an *os.File attached to a terminal), and [Config.Mode]
// overrides it in either direction, which is how tests exercise the in-place
// path against a bytes.Buffer.
//
// # The clock is injectable
//
// ETA and rate are derived from elapsed time, so they are only testable if the
// clock is a parameter: Config.Now defaults to time.Now, and a test supplies a
// counter or a fixed instant instead. Nothing in this package sleeps or reads
// the wall clock directly. Note that the redraw throttle uses the same clock, so
// a frozen clock emits the first frame and the final one and throttles
// everything between; set RefreshInterval (or PlainInterval) to a negative value
// to disable throttling entirely when a test wants every frame.
//
// # Indeterminate work
//
// With Total <= 0 the total is unknown: percentage and ETA render as "--" and
// the bar becomes a block that bounces inside its track, positioned from elapsed
// time (so it, too, is deterministic under an injected clock) rather than from a
// counter of redraws.
//
// # Several bars at once
//
// Concurrent bars sharing one Out would fight over the cursor, so a [Bar] never
// shares: use [NewMulti] to create a [MultiBar], which owns the cursor and
// repaints every one of its bars as a block of lines. Bars created with
// [MultiBar.New] inherit the group's writer, mode and clock and never write on
// their own. In plain mode the group has no cursor to own and simply emits the
// changed lines, so give each bar a {desc} so the log lines can be told apart.
// Animating bars ([Bar.Start], [MultiBar.Start]) run one goroutine that ticks
// the repaint; Start and Stop are safe from any goroutine, Stop joins the
// goroutine before returning, and Finish stops it.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Mode selects how a bar writes to its Out.
type Mode int

const (
	// ModeAuto picks ModeTerminal when Out is a terminal and ModePlain
	// otherwise. This is the zero value, and the right choice for a program.
	ModeAuto Mode = iota
	// ModeTerminal redraws one line in place with ANSI escapes. Force it only
	// when you know the destination understands them (a test buffer, a pty).
	ModeTerminal
	// ModePlain emits whole plain-text lines at a bounded rate and never emits
	// an escape sequence.
	ModePlain
)

// Charset holds the characters a determinate bar is drawn with. Every field is
// expected to be one terminal cell wide; the zero value falls back to
// [DefaultCharset].
type Charset struct {
	// Filled is the completed portion.
	Filled string
	// Empty is the remaining portion.
	Empty string
	// Head is drawn at the leading edge of the completed portion when set, and
	// consumes one cell of it. Empty means no head.
	Head string
	// Left and Right bracket the track. They are outside Width.
	Left, Right string
}

// DefaultCharset is the character set used when Config.Charset is left zero.
var DefaultCharset = Charset{Filled: "█", Empty: "░", Left: "[", Right: "]"}

// Default template strings. A template is expanded by [Bar.Render]: any
// {placeholder} it knows is substituted and anything else — including an
// unknown {placeholder} — is copied through verbatim, so a template is also a
// place to put literal text.
//
// The placeholders are {bar}, {percent}, {current}, {total}, {rate}, {eta},
// {elapsed}, {spinner} and {desc}.
const (
	// DefaultTemplate is used when Total is known.
	DefaultTemplate = "{desc}{bar} {percent} {current}/{total} {rate} eta {eta}"
	// DefaultIndeterminateTemplate is used when Total is unknown, where a
	// percentage and an ETA would be meaningless.
	DefaultIndeterminateTemplate = "{desc}{bar} {current} {rate} {elapsed}"
)

// Defaults for the numbers a caller usually does not care about.
const (
	// DefaultWidth is the track width in cells.
	DefaultWidth = 30
	// DefaultRefreshInterval bounds in-place redraws on a terminal.
	DefaultRefreshInterval = 100 * time.Millisecond
	// DefaultPlainInterval bounds plain-text lines, keeping a CI log short.
	DefaultPlainInterval = 5 * time.Second
	// pulseInterval is how long one step of the indeterminate animation (and
	// one spinner frame) lasts.
	pulseInterval = 100 * time.Millisecond
)

// Config configures a [Bar].
type Config struct {
	// Total is the amount of work the bar counts up to. Zero or negative means
	// the total is unknown and the bar is indeterminate.
	Total int64
	// Width is the track width in cells (default DefaultWidth). Brackets from
	// Charset are drawn outside it.
	Width int
	// Out is the destination (defaults to os.Stdout).
	Out io.Writer
	// Template is the layout (default DefaultTemplate, or
	// DefaultIndeterminateTemplate when Total is unknown).
	Template string
	// Description fills {desc}. A trailing space is added when it is non-empty
	// so a template can place it flush against the bar.
	Description string
	// Bytes formats {current}, {total} and {rate} as IEC byte sizes, so 1536
	// renders as "1.5 KiB".
	Bytes bool
	// Charset overrides the bar characters field by field; unset fields fall
	// back to DefaultCharset.
	Charset Charset
	// Now is the clock. It defaults to time.Now, and exists so ETA and rate can
	// be tested without sleeping.
	Now func() time.Time
	// Mode forces the output style (default ModeAuto).
	Mode Mode
	// RefreshInterval bounds in-place redraws (default
	// DefaultRefreshInterval). A negative value disables the throttle.
	RefreshInterval time.Duration
	// PlainInterval bounds plain-text lines (default DefaultPlainInterval). A
	// negative value disables the throttle.
	PlainInterval time.Duration
}

// Bar is a single progress bar. It is safe for concurrent use.
type Bar struct {
	mu      sync.Mutex
	total   int64
	current int64
	width   int
	tmpl    string
	desc    string
	bytes   bool
	chars   Charset
	now     func() time.Time
	out     io.Writer
	plain   bool // resolved Mode: plain text rather than in-place
	refresh time.Duration
	start   time.Time

	lastDraw  time.Time
	drawn     bool
	lastPlain string
	finished  bool

	// group is set for a bar owned by a MultiBar; such a bar never writes to
	// out itself, it asks the group to repaint the whole block.
	group *MultiBar

	// animation state
	started bool
	stopped bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// New creates a bar from cfg. It does not draw anything yet: the first frame is
// written by the first [Bar.Add], [Bar.Set], [Bar.Start] or [Bar.Finish]. The
// elapsed time that feeds rate and ETA starts here.
func New(cfg Config) *Bar {
	b := &Bar{}
	b.init(cfg, nil)
	return b
}

func (b *Bar) init(cfg Config, group *MultiBar) {
	b.total = cfg.Total
	b.width = cfg.Width
	if b.width <= 0 {
		b.width = DefaultWidth
	}
	b.tmpl = cfg.Template
	if b.tmpl == "" {
		if cfg.Total > 0 {
			b.tmpl = DefaultTemplate
		} else {
			b.tmpl = DefaultIndeterminateTemplate
		}
	}
	b.desc = cfg.Description
	b.bytes = cfg.Bytes
	b.chars = mergeCharset(cfg.Charset)
	b.now = cfg.Now
	if b.now == nil {
		b.now = time.Now
	}
	b.out = cfg.Out
	if b.out == nil {
		b.out = os.Stdout
	}
	b.plain = !terminalMode(cfg.Mode, b.out)
	if b.plain {
		b.refresh = cfg.PlainInterval
		if b.refresh == 0 {
			b.refresh = DefaultPlainInterval
		}
	} else {
		b.refresh = cfg.RefreshInterval
		if b.refresh == 0 {
			b.refresh = DefaultRefreshInterval
		}
	}
	b.group = group
	b.start = b.now()
}

// mergeCharset fills unset fields from DefaultCharset. Left and Right are
// treated as a pair: a caller who wants no brackets sets one of them to a
// single space rather than fighting the zero value.
func mergeCharset(c Charset) Charset {
	if c.Filled == "" {
		c.Filled = DefaultCharset.Filled
	}
	if c.Empty == "" {
		c.Empty = DefaultCharset.Empty
	}
	if c.Left == "" && c.Right == "" {
		c.Left, c.Right = DefaultCharset.Left, DefaultCharset.Right
	}
	return c
}

// terminalMode resolves Mode against out. Auto asks golang.org/x/term whether
// out is a real terminal; anything that is not an *os.File (a bytes.Buffer, a
// pipe, a log wrapper) is not, so the safe plain path is chosen by default.
func terminalMode(m Mode, out io.Writer) bool {
	switch m {
	case ModeTerminal:
		return true
	case ModePlain:
		return false
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Add advances the bar by delta and redraws (subject to the refresh throttle).
// The increment happens under the bar's lock rather than as a read-modify-write
// through Set, so concurrent producers cannot lose an update.
func (b *Bar) Add(delta int64) {
	b.mu.Lock()
	b.current = clamp(b.current+delta, b.total)
	b.mu.Unlock()
	b.draw(false)
}

// Set moves the bar to an absolute value and redraws (subject to the refresh
// throttle). Values are clamped at zero and, when the total is known, at the
// total.
func (b *Bar) Set(n int64) {
	b.mu.Lock()
	b.current = clamp(n, b.total)
	b.mu.Unlock()
	b.draw(false)
}

// SetTotal declares (or revises) the total. Passing zero or a negative value
// turns the bar indeterminate; it does not change the template chosen at
// construction.
func (b *Bar) SetTotal(total int64) {
	b.mu.Lock()
	b.total = total
	if total > 0 && b.current > total {
		b.current = total
	}
	b.mu.Unlock()
	b.draw(false)
}

// Describe replaces the {desc} text.
func (b *Bar) Describe(desc string) {
	b.mu.Lock()
	b.desc = desc
	b.mu.Unlock()
	b.draw(false)
}

// Current returns the current value.
func (b *Bar) Current() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current
}

// Total returns the total, or a non-positive value when it is unknown.
func (b *Bar) Total() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total
}

// Write makes a Bar an io.Writer that counts the bytes passing through it, so a
// download can be measured with io.Copy(io.MultiWriter(dst, bar), src). It
// always reports the whole slice as written.
func (b *Bar) Write(p []byte) (int, error) {
	b.Add(int64(len(p)))
	return len(p), nil
}

// Render returns the bar as a plain string, with no escape sequences and no
// trailing newline, exactly as the current mode would draw it. It is the
// testing seam: assert on this instead of on terminal output.
func (b *Bar) Render() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.line()
}

// Finish completes the bar: it stops any animation goroutine, snaps a
// determinate bar to its total, writes the final frame unthrottled, and (on a
// terminal, when the bar is not in a group) moves to the next line so later
// output does not land on top of the bar. It is idempotent, and a finished bar
// never draws again — later Add/Set calls still update the counter (Current keeps
// working) but nothing more is written, so the final frame stays on screen.
func (b *Bar) Finish() {
	b.Stop()
	b.mu.Lock()
	if b.finished {
		b.mu.Unlock()
		return
	}
	b.finished = true
	if b.total > 0 {
		b.current = b.total
	}
	if b.group != nil {
		b.mu.Unlock()
		b.group.repaint(true)
		return
	}
	line := b.line()
	b.lastDraw, b.drawn, b.lastPlain = b.now(), true, line
	if b.plain {
		writeString(b.out, line+"\n")
	} else {
		writeString(b.out, "\r"+line+"\x1b[K\r\n")
	}
	b.mu.Unlock()
}

// Start begins animating the bar from its own goroutine, repainting on every
// refresh interval. It is what makes an indeterminate bar (or an ETA) move while
// the caller is blocked in a slow operation. Calling it twice without a Stop is
// a no-op, and it does nothing once the bar is finished.
func (b *Bar) Start() {
	b.mu.Lock()
	if b.started || b.finished {
		b.mu.Unlock()
		return
	}
	iv := b.refresh
	if iv <= 0 {
		iv = pulseInterval
	}
	b.started, b.stopped = true, false
	b.stopCh = make(chan struct{})
	stop := b.stopCh
	b.mu.Unlock()

	b.draw(true)
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		t := time.NewTicker(iv)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				b.draw(false)
			}
		}
	}()
}

// Stop halts the animation goroutine and waits for it to exit, so a stopped bar
// never writes again. It is safe to call from a different goroutine than Start,
// safe to call more than once, and safe to call on a bar that was never
// started. Unlike Finish it leaves the bar's value and cursor alone.
func (b *Bar) Stop() {
	b.mu.Lock()
	if b.started && !b.stopped {
		b.stopped = true
		close(b.stopCh)
	}
	b.started = false
	b.mu.Unlock()
	b.wg.Wait()
}

// draw writes a frame if the throttle allows it (or force is set). A grouped bar
// delegates to its group; note the bar's lock is never held across that call,
// which is what keeps the bar/group locks ordered one way only.
func (b *Bar) draw(force bool) {
	if b.group != nil {
		b.group.repaint(force)
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.finished {
		return
	}
	now := b.now()
	if !force && !dueAt(now, b.lastDraw, b.drawn, b.refresh) {
		return
	}
	line := b.line()
	if b.plain {
		// A CI log gains nothing from an identical line repeated, so only a
		// change is worth a line.
		if line == b.lastPlain {
			return
		}
		b.lastPlain = line
		writeString(b.out, line+"\n")
	} else {
		writeString(b.out, "\r"+line+"\x1b[K")
	}
	b.lastDraw, b.drawn = now, true
}

// dueAt reports whether a frame is due: always for the first one, never before
// the interval has elapsed, always when the interval is negative.
func dueAt(now, last time.Time, drawn bool, interval time.Duration) bool {
	if !drawn || interval < 0 {
		return true
	}
	return now.Sub(last) >= interval
}

// line renders the template. The caller must hold b.mu.
func (b *Bar) line() string {
	now := b.now()
	elapsed := now.Sub(b.start)
	if elapsed < 0 {
		elapsed = 0
	}
	cur, total := b.current, b.total
	desc := b.desc
	if desc != "" {
		desc += " "
	}

	var rate float64
	if s := elapsed.Seconds(); s > 0 {
		rate = float64(cur) / s
	}

	percent, eta, totalStr := "--%", "--", "?"
	if total > 0 {
		p := int(float64(cur) / float64(total) * 100)
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		percent = fmt.Sprintf("%3d%%", p)
		totalStr = formatAmount(total, b.bytes)
		switch {
		case cur >= total:
			eta = formatDuration(0)
		case rate > 0:
			eta = formatDuration(time.Duration(float64(total-cur) / rate * float64(time.Second)))
		}
	}

	return expand(b.tmpl, func(name string) (string, bool) {
		switch name {
		case "bar":
			return b.track(elapsed), true
		case "percent":
			return percent, true
		case "current":
			return formatAmount(cur, b.bytes), true
		case "total":
			return totalStr, true
		case "rate":
			return formatRate(rate, b.bytes), true
		case "eta":
			return eta, true
		case "elapsed":
			return formatDuration(elapsed), true
		case "spinner":
			return spinnerFrame(elapsed), true
		case "desc":
			return desc, true
		}
		return "", false
	})
}

// track draws the bar itself: a filled/empty split when the total is known, and
// a block bouncing inside the track when it is not. The bouncing block is
// positioned from elapsed time rather than from a redraw counter so that it is
// reproducible under an injected clock.
func (b *Bar) track(elapsed time.Duration) string {
	w := b.width
	var body string
	if b.total > 0 {
		filled := int(float64(w) * float64(b.current) / float64(b.total))
		if filled < 0 {
			filled = 0
		}
		if filled > w {
			filled = w
		}
		if b.chars.Head != "" && filled > 0 && filled < w {
			body = strings.Repeat(b.chars.Filled, filled-1) + b.chars.Head +
				strings.Repeat(b.chars.Empty, w-filled)
		} else {
			body = strings.Repeat(b.chars.Filled, filled) + strings.Repeat(b.chars.Empty, w-filled)
		}
	} else {
		block := w / 5
		if block < 3 {
			block = 3
		}
		if block > w {
			block = w
		}
		pos, span := 0, w-block
		if span > 0 {
			p := int(elapsed/pulseInterval) % (2 * span)
			if p > span {
				p = 2*span - p
			}
			pos = p
		}
		body = strings.Repeat(b.chars.Empty, pos) + strings.Repeat(b.chars.Filled, block) +
			strings.Repeat(b.chars.Empty, w-block-pos)
	}
	return b.chars.Left + body + b.chars.Right
}

// expand substitutes {name} using lookup, copying anything it does not
// recognise — including an unclosed brace — through unchanged.
func expand(tmpl string, lookup func(string) (string, bool)) string {
	var out strings.Builder
	for i := 0; i < len(tmpl); {
		if tmpl[i] != '{' {
			out.WriteByte(tmpl[i])
			i++
			continue
		}
		end := strings.IndexByte(tmpl[i:], '}')
		if end < 0 {
			out.WriteString(tmpl[i:])
			break
		}
		name := tmpl[i+1 : i+end]
		if v, ok := lookup(name); ok {
			out.WriteString(v)
		} else {
			out.WriteString(tmpl[i : i+end+1])
		}
		i += end + 1
	}
	return out.String()
}

func writeString(w io.Writer, s string) { _, _ = io.WriteString(w, s) }
