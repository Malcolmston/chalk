// Package spinner provides terminal loaders — an animated spinner on one line
// that resolves into a single final status line — in the spirit of Node's ora,
// styled with the sibling chalk package.
//
//	s := spinner.New(spinner.Config{Text: "Building"}).Start()
//	// ... work ...
//	s.UpdateText("Linking")
//	// ... work ...
//	s.Succeed("Built in 1.2s")
//
// A [Spinner] owns exactly one line of output. While it runs, a goroutine
// repaints that line on every tick of a [Ticker], cycling through a frame set
// ([Dots], [Line], [Bars], or any caller-supplied [FrameSet]). [Spinner.Stop]
// erases the line and leaves the cursor at column zero so whatever the program
// prints next is not glued onto a half-drawn frame; [Spinner.Succeed],
// [Spinner.Fail], [Spinner.Warn] and [Spinner.Info] do the same and then print
// one final line with the symbol and color configured for that [State].
//
// Two behaviors matter more than the API surface.
//
// First, non-terminal output. When Out is not a terminal — a pipe, a file, a CI
// log — the spinner does not animate at all and no goroutine is started: it
// prints the label once at Start and, for the resolving methods, one final status
// line. Cursor-movement and line-erase escapes are never written in that mode,
// because a spinner that "animates" into a pipe emits thousands of useless
// frames and an unreadable log. [Config.Mode] overrides the detection when a
// caller (or a test) wants to force one path.
//
// Second, concurrency. Start, Stop, the resolving methods and [Spinner.UpdateText]
// are safe to call from any goroutine. Start is a no-op while the spinner is
// already running, so a double Start cannot launch a second animation goroutine;
// Stop is idempotent, and it *joins* the goroutine rather than merely signalling
// it, so once Stop returns nothing else will ever write to Out. That join is what
// makes it safe to print from the calling goroutine immediately afterwards.
// Because the tick is injectable, tests drive the animation frame by frame with
// [ManualTicker] and never sleep.
//
// There is no upstream ora API to match here: this package is an addition rather
// than a port, so nothing about it is measured for parity.
package spinner

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/malcolmston/chalk"
	"golang.org/x/term"
)

// FrameSet is a spinner animation: the frames to cycle through and the interval
// they were designed for. Interval is a suggestion — [Config.Interval] wins when
// it is set.
type FrameSet struct {
	Frames   []string
	Interval time.Duration
}

// The bundled frame sets. Dots is the braille-dot spinner most terminals render
// well; Line is pure ASCII, for a terminal or font that cannot draw braille;
// Bars uses block-element characters. All three are ASCII/Unicode graphics, not
// emoji.
var (
	Dots = FrameSet{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond,
	}
	Line = FrameSet{
		Frames:   []string{"-", "\\", "|", "/"},
		Interval: 130 * time.Millisecond,
	}
	Bars = FrameSet{
		Frames:   []string{"▁", "▃", "▄", "▅", "▆", "▇", "▆", "▅", "▄", "▃"},
		Interval: 80 * time.Millisecond,
	}
)

// DefaultInterval is used when neither the config nor the frame set names one.
const DefaultInterval = 80 * time.Millisecond

// State is how a spinner ended. It selects the final symbol and color.
type State int

const (
	// StateStopped is a plain Stop: the line is erased and nothing replaces it.
	StateStopped State = iota
	// StateSucceeded is Succeed.
	StateSucceeded
	// StateFailed is Fail.
	StateFailed
	// StateWarned is Warn.
	StateWarned
	// StateInformed is Info.
	StateInformed
)

// String implements fmt.Stringer, for test failure messages.
func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateSucceeded:
		return "succeeded"
	case StateFailed:
		return "failed"
	case StateWarned:
		return "warned"
	case StateInformed:
		return "informed"
	}
	return "unknown"
}

// Symbol is the marker printed on the final line for a [State], with the style
// it is drawn in. A nil Style renders the text unstyled, which is how a caller
// (or a test) asks for plain output without touching chalk's process-global
// level.
type Symbol struct {
	Text  string
	Style *chalk.Style
}

func (sy Symbol) render() string {
	if sy.Style == nil {
		return sy.Text
	}
	return sy.Style.Sprint(sy.Text)
}

// DefaultSymbols returns the final symbol and color per state: a check for
// success, a cross for failure, a triangle for a warning, a disc for info.
// StateStopped deliberately has no entry — a plain Stop leaves no line behind.
// The map is freshly built on each call so a caller can mutate it safely.
func DefaultSymbols() map[State]Symbol {
	return map[State]Symbol{
		StateSucceeded: {Text: "✔", Style: chalk.New().Green()},
		StateFailed:    {Text: "✖", Style: chalk.New().Red()},
		StateWarned:    {Text: "▲", Style: chalk.New().Yellow()},
		StateInformed:  {Text: "●", Style: chalk.New().Cyan()},
	}
}

// Mode selects the animated or the plain rendering path.
type Mode int

const (
	// Auto animates only when Out is a terminal. It is the zero value.
	Auto Mode = iota
	// Animate forces the animated path, escapes and goroutine included. Tests
	// use it to assert on frames written to a buffer.
	Animate
	// Plain forces the non-animated path: label once, final line once, no
	// escapes and no goroutine.
	Plain
)

// Config configures a [Spinner]. The zero value is usable: it animates the
// braille-dot frames on os.Stdout when stdout is a terminal.
type Config struct {
	// Text is the label drawn after the frame. It is also the text of the final
	// line when a resolving method is called without its own text.
	Text string
	// Prefix is written before the frame, verbatim (add your own trailing space
	// if you want one).
	Prefix string
	// Suffix is written after the label, verbatim, on animated frames only.
	Suffix string
	// Frames is the animation. An empty FrameSet means [Dots].
	Frames FrameSet
	// Interval overrides Frames.Interval.
	Interval time.Duration
	// FrameStyle styles the animated frame. Nil means cyan; chalk.New() means
	// unstyled.
	FrameStyle *chalk.Style
	// TextStyle styles the label on animated frames. Nil means unstyled.
	TextStyle *chalk.Style
	// Symbols overrides entries of [DefaultSymbols]; unset states keep their
	// default. A state mapped to the zero Symbol prints no marker.
	Symbols map[State]Symbol
	// Out is where frames go, defaulting to os.Stdout. Terminal detection —
	// and therefore whether anything animates — is done on this writer.
	Out io.Writer
	// Mode overrides that detection. See [Auto], [Animate], [Plain].
	Mode Mode
	// ShowCursor keeps the hardware cursor visible while animating. By default
	// the cursor is hidden for the duration and restored on stop, because a
	// cursor parked on a repainting frame flickers.
	ShowCursor bool
	// NewTicker builds the clock the animation runs on, defaulting to
	// time.Ticker. Tests inject [ManualTicker.New] here.
	NewTicker func(time.Duration) Ticker
}

// Spinner is a one-line terminal loader. Create one with [New]; every method is
// safe to call from any goroutine.
type Spinner struct {
	out        io.Writer
	frames     []string
	interval   time.Duration
	frameStyle *chalk.Style
	textStyle  *chalk.Style
	symbols    map[State]Symbol
	newTicker  func(time.Duration) Ticker
	animate    bool
	hideCursor bool

	// mu guards the mutable state *and* serializes writes to out, so the
	// animation goroutine and a caller can never interleave halves of a line.
	mu      sync.Mutex
	prefix  string
	suffix  string
	text    string
	frame   int
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// New builds a Spinner from cfg. It writes nothing until [Spinner.Start].
func New(cfg Config) *Spinner {
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}

	frames := cfg.Frames.Frames
	if len(frames) == 0 {
		frames = Dots.Frames
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = cfg.Frames.Interval
	}
	if interval <= 0 {
		interval = DefaultInterval
	}

	frameStyle := cfg.FrameStyle
	if frameStyle == nil {
		frameStyle = chalk.New().Cyan()
	}

	symbols := DefaultSymbols()
	for st, sy := range cfg.Symbols {
		symbols[st] = sy
	}

	newTicker := cfg.NewTicker
	if newTicker == nil {
		newTicker = newRealTicker
	}

	animate := false
	switch cfg.Mode {
	case Animate:
		animate = true
	case Plain:
		animate = false
	default:
		animate = isTerminal(out)
	}

	return &Spinner{
		out:        out,
		frames:     frames,
		interval:   interval,
		frameStyle: frameStyle,
		textStyle:  cfg.TextStyle,
		symbols:    symbols,
		newTicker:  newTicker,
		animate:    animate,
		hideCursor: !cfg.ShowCursor,
		prefix:     sanitize(cfg.Prefix),
		suffix:     sanitize(cfg.Suffix),
		text:       sanitize(cfg.Text),
	}
}

// isTerminal reports whether w is a real terminal. Only an *os.File can be one;
// anything else (a bytes.Buffer, a pipe wrapper, an io.MultiWriter) is treated as
// a log. TERM=dumb is honored too: such a terminal cannot be trusted with cursor
// movement even though it is a tty.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	if !term.IsTerminal(int(f.Fd())) {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}

// sanitize flattens the line breaks out of caller text. A newline in the label
// would scroll the frame off its own line, and the erase escape only clears the
// row the cursor is on, so the leftovers would pile up on screen.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}

// Animating reports whether this spinner animates, i.e. whether Out was judged
// (or forced) to be a terminal. Useful to a caller that wants to print extra
// progress detail only in the non-animated case.
func (s *Spinner) Animating() bool { return s.animate }

// Running reports whether the spinner is currently started.
func (s *Spinner) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Text returns the current label.
func (s *Spinner) Text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.text
}

// Interval returns the resolved tick interval.
func (s *Spinner) Interval() time.Duration { return s.interval }

// Start begins the spinner and returns it, so it can be chained onto [New]. It
// is a no-op if the spinner is already running: a second Start must not leave a
// second goroutine painting the same line.
//
// When animating, the first frame is painted synchronously before Start returns,
// so a caller (and a test) always sees a frame without waiting for a tick. When
// not animating, the label is printed once as an ordinary line and no goroutine
// is created.
func (s *Spinner) Start() *Spinner {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return s
	}
	s.running = true
	s.frame = 0

	if !s.animate {
		label := strings.TrimSpace(s.prefix + s.text + s.suffix)
		writeString(s.out, label+"\n")
		s.mu.Unlock()
		return s
	}

	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	stop, done := s.stopCh, s.doneCh
	if s.hideCursor {
		writeString(s.out, hideCursorSeq)
	}
	s.paintLocked()
	s.mu.Unlock()

	go s.loop(stop, done)
	return s
}

// loop is the animation goroutine. It exits on stop or on a closed tick channel,
// and closes done last of all so that a Stop waiting on done knows both that no
// further write can happen and that the ticker has been released.
func (s *Spinner) loop(stop <-chan struct{}, done chan struct{}) {
	t := s.newTicker(s.interval)
	defer close(done)
	defer t.Stop()

	for {
		select {
		case <-stop:
			return
		case _, ok := <-t.C():
			if !ok {
				return
			}
			s.mu.Lock()
			if s.running {
				s.frame++
				s.paintLocked()
			}
			s.mu.Unlock()
			if a, ok := t.(Acker); ok {
				a.Acked()
			}
		}
	}
}

const (
	hideCursorSeq = "\x1b[?25l"
	showCursorSeq = "\x1b[?25h"
	// clearLineSeq returns to column zero and erases to end of line. Erasing
	// rather than overwriting with spaces means a shorter label cannot leave
	// the tail of a longer one on screen, and it needs no width bookkeeping.
	clearLineSeq = "\r\x1b[K"
)

// paintLocked writes the current frame. The caller must hold s.mu.
func (s *Spinner) paintLocked() {
	writeString(s.out, clearLineSeq+s.lineLocked())
}

// lineLocked renders the current frame line. The caller must hold s.mu.
func (s *Spinner) lineLocked() string {
	frame := s.frames[s.frame%len(s.frames)]
	text := s.text
	if s.textStyle != nil {
		text = s.textStyle.Sprint(text)
	}
	var b strings.Builder
	b.WriteString(s.prefix)
	b.WriteString(s.frameStyle.Sprint(frame))
	if text != "" {
		b.WriteString(" ")
		b.WriteString(text)
	}
	b.WriteString(s.suffix)
	return b.String()
}

// UpdateText replaces the label. While animating it repaints immediately, so the
// new text shows without waiting for the next tick. In the non-animated mode it
// only records the text — printing a line per update is what turns a piped build
// log into noise; the text still shows up on the final status line.
func (s *Spinner) UpdateText(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text = sanitize(text)
	if s.running && s.animate {
		s.paintLocked()
	}
}

// UpdatePrefix and UpdateSuffix replace the strings drawn around the frame,
// repainting immediately when animating.
func (s *Spinner) UpdatePrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefix = sanitize(prefix)
	if s.running && s.animate {
		s.paintLocked()
	}
}

// UpdateSuffix replaces the suffix. See [Spinner.UpdatePrefix].
func (s *Spinner) UpdateSuffix(suffix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suffix = sanitize(suffix)
	if s.running && s.animate {
		s.paintLocked()
	}
}

// Stop ends the animation, erases the spinner line and leaves the cursor at
// column zero, so the next thing the program prints starts on a clean line. It
// prints no final status; use [Spinner.Succeed] and friends for that.
//
// Stop is idempotent and safe from any goroutine, and it does not return until
// the animation goroutine has exited.
func (s *Spinner) Stop() { s.StopWith(StateStopped, "") }

// Succeed stops the spinner and prints the success line. An empty text keeps the
// current label. Fail, Warn and Info differ only in state.
func (s *Spinner) Succeed(text string) { s.StopWith(StateSucceeded, text) }

// Fail stops the spinner and prints the failure line.
func (s *Spinner) Fail(text string) { s.StopWith(StateFailed, text) }

// Warn stops the spinner and prints the warning line.
func (s *Spinner) Warn(text string) { s.StopWith(StateWarned, text) }

// Info stops the spinner and prints the info line.
func (s *Spinner) Info(text string) { s.StopWith(StateInformed, text) }

// StopWith is the primitive the other stop methods use: it ends the animation
// and resolves into state's symbol and color. A state with no symbol (by default
// [StateStopped]) leaves no line behind. Passing an empty text keeps the current
// label.
//
// The sequence is deliberate. The goroutine is signalled and joined *before*
// anything is written, so the final line cannot be interleaved with, or
// overwritten by, one last frame.
func (s *Spinner) StopWith(state State, text string) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	if text != "" {
		s.text = sanitize(text)
	}
	stop, done := s.stopCh, s.doneCh
	s.stopCh, s.doneCh = nil, nil
	animating := s.animate
	s.mu.Unlock()

	if animating {
		close(stop)
		<-done // join, do not merely signal
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if animating {
		writeString(s.out, clearLineSeq)
		if s.hideCursor {
			writeString(s.out, showCursorSeq)
		}
	}
	if sy, ok := s.symbols[state]; ok && sy.Text != "" {
		writeString(s.out, strings.TrimSpace(sy.render()+" "+s.text)+"\n")
	}
}

// writeString ignores the write error, as the rest of chalk does: a spinner is
// decoration, and a program should not have to check an error to draw one.
func writeString(w io.Writer, s string) { _, _ = io.WriteString(w, s) }
