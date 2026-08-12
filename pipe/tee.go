package pipe

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/malcolmston/chalk"
)

// TeeConfig configures [NewTee].
type TeeConfig struct {
	// Screen receives the styled copy. Defaults to os.Stdout.
	Screen io.Writer
	// Log receives the copy with every escape sequence removed. Nil means no
	// second copy, so a program can always write through a Tee and decide later
	// whether a log file exists.
	Log io.Writer
	// Style is applied to text written through the Tee's Print helpers. Nil
	// means no styling of its own; text that already carries escape codes still
	// reaches Screen styled and Log clean.
	Style *chalk.Style
	// Level pins the color level of the screen copy. Nil means detect it from
	// Screen with [ColorLevel], which yields [chalk.LevelNone] for anything that
	// is not a terminal *os.File — including the bytes.Buffer a test injects, so
	// set this to exercise the styled path without a TTY.
	Level *chalk.Level
}

// Tee writes one message twice: styled to the screen and unstyled to a log or a
// pipe.
//
// This is the reason the package exists. The usual alternative is to build the
// message twice, or to log the styled bytes and leave escape sequences in the
// file for whoever greps it later. A Tee removes the choice: it renders once,
// writes the styled form to Screen when Screen can render it, and always writes
// the stripped form to Log.
//
// A Tee is an [io.Writer], so it can be handed to log.New or fmt.Fprintf. It is
// safe for concurrent use: every write takes a mutex, so a message never
// interleaves with another and the two copies stay in the same order. Values
// derived with [Tee.WithStyle] share that mutex and those writers, so styling a
// sub-logger does not create a second, unsynchronized path to the same file.
type Tee struct {
	shared *teeShared
	style  *chalk.Style
}

// teeShared is the state every derived Tee holds in common: the destinations,
// the color decision, and the lock that keeps their writes ordered.
type teeShared struct {
	mu     sync.Mutex
	screen io.Writer
	log    io.Writer
	level  chalk.Level
}

// NewTee returns a Tee writing to cfg.Screen and cfg.Log.
func NewTee(cfg TeeConfig) *Tee {
	screen := cfg.Screen
	if screen == nil {
		screen = os.Stdout
	}
	level := chalk.LevelNone
	switch {
	case cfg.Level != nil:
		level = *cfg.Level
	default:
		if f, ok := screen.(*os.File); ok {
			level = ColorLevel(f)
		}
	}
	return &Tee{
		shared: &teeShared{screen: screen, log: cfg.Log, level: level},
		style:  cfg.Style,
	}
}

// WithStyle returns a Tee that writes to the same destinations with a different
// default style. The receiver is unchanged, matching the parent package's
// immutable-style convention.
func (t *Tee) WithStyle(s *chalk.Style) *Tee {
	return &Tee{shared: t.shared, style: s}
}

// Level reports the color level of the screen copy. It is [chalk.LevelNone]
// whenever the screen cannot render color, which is also exactly when the two
// copies are byte-identical.
func (t *Tee) Level() chalk.Level { return t.shared.level }

// ColorEnabled reports whether the screen copy will carry escape sequences.
func (t *Tee) ColorEnabled() bool { return t.shared.level > chalk.LevelNone }

// styleText applies the Tee's style to s, or strips s back to plain text when
// the screen cannot render color. Stripping rather than passing through matters:
// callers hand a Tee text that may already be styled by chalk's shortcut
// functions, and forwarding those bytes to a non-terminal screen is the exact bug
// this package is here to prevent.
func (t *Tee) styleText(s string, applyStyle bool) string {
	if t.shared.level == chalk.LevelNone {
		return chalk.Strip(s)
	}
	if !applyStyle || t.style == nil {
		return s
	}
	return t.style.Level(t.shared.level).Sprint(s)
}

// emit is the single funnel every public write goes through. suffix is appended
// to both copies outside the style span, and the whole message plus suffix is
// written under one lock so a concurrent writer cannot land between a line and
// its newline.
func (t *Tee) emit(s string, applyStyle bool, suffix string) error {
	screenText := t.styleText(s, applyStyle) + suffix
	plain := chalk.Strip(s) + suffix

	t.shared.mu.Lock()
	defer t.shared.mu.Unlock()
	if _, err := io.WriteString(t.shared.screen, screenText); err != nil {
		return err
	}
	if t.shared.log != nil {
		if _, err := io.WriteString(t.shared.log, plain); err != nil {
			return err
		}
	}
	return nil
}

// Write implements [io.Writer]. p is treated as text that may already be
// styled: the screen gets it as-is (or stripped, when the screen cannot render
// color) and the log always gets it stripped. The Tee's own Style is not applied
// here - an io.Writer is a byte sink, and log.Logger's prefixes and newlines
// should not be swept into a color span; use [Tee.Print] and friends for that.
//
// On success it reports len(p) written, not the number of bytes that reached
// either destination, because the two differ by the escape sequences and an
// io.Writer must not claim a short write for text it fully accepted.
func (t *Tee) Write(p []byte) (int, error) {
	if err := t.emit(string(p), false, ""); err != nil {
		return 0, err
	}
	return len(p), nil
}

// WriteString writes s the way [Tee.Write] does.
func (t *Tee) WriteString(s string) (int, error) {
	if err := t.emit(s, false, ""); err != nil {
		return 0, err
	}
	return len(s), nil
}

// Print formats its operands like fmt.Sprint, applies the Tee's style, and
// writes both copies.
func (t *Tee) Print(a ...any) error { return t.emit(fmt.Sprint(a...), true, "") }

// Println formats its operands like fmt.Sprintln (spaces between all operands, a
// trailing newline) and writes both copies. The newline is written outside the
// style span, so the style never bleeds past the end of the line.
func (t *Tee) Println(a ...any) error {
	return t.emit(strings.TrimSuffix(fmt.Sprintln(a...), "\n"), true, "\n")
}

// Printf formats according to a format specifier and writes both copies.
func (t *Tee) Printf(format string, a ...any) error {
	return t.emit(fmt.Sprintf(format, a...), true, "")
}
