// Package pipe helps a command-line program behave correctly when its input or
// output is a pipe rather than a terminal.
//
// Two halves. The first is detection: [IsTerminal], [IsPiped], [IsRedirected],
// [Size] / [Width] / [Height] with a documented fallback, and [ColorLevel],
// which decides whether color may be emitted to one specific stream. The second
// is streaming: [ReadAll] reads piped stdin without hanging forever when stdin
// is a keyboard, [Lines] applies a caller's transform line by line and writes
// through, and [Tee] writes a styled copy to the screen and an unstyled copy to
// a log or pipe from one call.
//
//	if pipe.IsPiped(os.Stdin) {
//		data, _ := pipe.ReadAll(pipe.ReadConfig{})
//		_ = data
//	}
//	t := pipe.NewTee(pipe.TeeConfig{Screen: os.Stdout, Log: logFile})
//	t.Style(chalk.New().Red()).Println("failed")
//
// Every entry point takes its streams from a config struct or an explicit
// argument, so behaviour is testable with fake readers and writers and no
// terminal at all. Nothing in this package emits cursor-movement escapes, so
// there is no non-TTY path that garbles a log; the only thing that changes when
// a stream is not a terminal is that color is dropped, sizes come from the
// fallback, and reads that would block are refused with [ErrNoInput] instead of
// hanging.
//
// # Relationship to the parent package's color detection
//
// The parent chalk package already implements the supports-color ladder
// (NO_COLOR, FORCE_COLOR, CI providers, COLORTERM, TERM names) against
// os.Stdout, exposed as chalk.GetLevel. This package does not re-implement it and
// does not disagree with it: [ColorLevel] asks chalk for the capability tier and
// only adds the parts chalk cannot answer, namely the per-stream question ("is
// *this* writer a terminal") and the CLICOLOR / CLICOLOR_FORCE convention, which
// chalk does not implement at all.
//
// Two answers can therefore differ from the parent package's, both deliberately
// and both documented on the functions themselves: [IsTerminal] asks the OS
// through golang.org/x/term where chalk's internal check accepts any character
// device (so chalk considers /dev/null a terminal and this package does not), and
// [ColorLevel] honours CLICOLOR / CLICOLOR_FORCE, which chalk ignores. Neither
// changes chalk's global state; a program that wants chalk itself to follow this
// package's answer calls [SyncColor] once at startup.
package pipe

import (
	"os"
	"strconv"
	"strings"

	"github.com/malcolmston/chalk"
	"golang.org/x/term"
)

// Fallback dimensions used when a stream has no terminal to measure: the
// classic 80x24, which is what every tool from less to git falls back to and is
// therefore what a reader of a log file expects the wrapping to match.
const (
	// DefaultWidth is the assumed column count when there is no TTY.
	DefaultWidth = 80
	// DefaultHeight is the assumed row count when there is no TTY.
	DefaultHeight = 24
)

// IsTerminal reports whether f is attached to a terminal.
//
// This asks the operating system through golang.org/x/term rather than looking
// at the file mode. The distinction matters: the parent package's internal check
// treats any character device as a terminal, which makes /dev/null look
// interactive. A program that decides to animate or to wait for a keypress on
// that answer misbehaves, so this package uses the stricter test.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// IsPiped reports whether f is a pipe, i.e. the `a | b` and `<(...)` cases.
//
// It is deliberately narrower than [IsRedirected]: a program that wants to know
// "did someone hand me streamed data" wants this, while a program deciding
// whether it may draw wants IsRedirected, because a plain file redirect is just
// as unable to render escape codes as a pipe is.
func IsPiped(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe != 0
}

// IsRedirected reports whether f is anything other than a terminal: a pipe, a
// regular file, /dev/null, or a closed descriptor. This is the predicate to
// branch on before drawing anything positional.
func IsRedirected(f *os.File) bool { return !IsTerminal(f) }

// Size returns the character cell dimensions of f.
//
// When f is a terminal the size comes from the kernel. Otherwise the COLUMNS and
// LINES environment variables are consulted — they are how a caller tells a
// redirected program how wide to wrap, and honouring them is why this is not
// simply a constant — and finally [DefaultWidth] x [DefaultHeight] is returned.
// The result is always positive, so callers can divide by it without checking.
func Size(f *os.File) (width, height int) {
	w, h, _ := measure(f)
	return w, h
}

// measure is [Size] plus whether the numbers came from a real terminal.
func measure(f *os.File) (width, height int, measured bool) {
	if IsTerminal(f) {
		if w, h, err := term.GetSize(int(f.Fd())); err == nil && w > 0 && h > 0 {
			return w, h, true
		}
	}
	return envDimension("COLUMNS", DefaultWidth), envDimension("LINES", DefaultHeight), false
}

// Width returns the column count of f, falling back as [Size] documents.
func Width(f *os.File) int { w, _ := Size(f); return w }

// Height returns the row count of f, falling back as [Size] documents.
func Height(f *os.File) int { _, h := Size(f); return h }

// envDimension reads a positive integer from an environment variable, returning
// def when it is unset, unparsable or non-positive.
func envDimension(name string, def int) int {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// colorGate is the decision the environment forces on us, independent of any
// terminal check.
type colorGate int

const (
	gateAuto colorGate = iota // no opinion: fall through to the TTY test
	gateOff                   // color is forbidden outright
	gateOn                    // color is demanded even for a pipe
)

// envColorGate collapses the environment variables that override per-stream
// detection into a single decision.
//
// Precedence, highest first: NO_COLOR, FORCE_COLOR, CLICOLOR_FORCE, CLICOLOR,
// TERM=dumb. NO_COLOR wins because no-color.org specifies it as unconditional,
// and the parent package already checks it first — agreeing here keeps the two
// answers from diverging. CLICOLOR is only consulted after FORCE_COLOR because
// FORCE_COLOR is the newer and more widely honoured of the two.
func envColorGate() colorGate {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return gateOff
	}
	if v, ok := os.LookupEnv("FORCE_COLOR"); ok {
		if v == "0" || strings.EqualFold(v, "false") {
			return gateOff
		}
		return gateOn
	}
	// CLICOLOR_FORCE (the BSD/ls convention): any value other than 0 means
	// "colorize even when not a tty".
	if v, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && v != "0" {
		return gateOn
	}
	// CLICOLOR=0 disables color; CLICOLOR=1 means "color when a tty", which is
	// already the default, so it is not a force.
	if v, ok := os.LookupEnv("CLICOLOR"); ok && v == "0" {
		return gateOff
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return gateOff
	}
	return gateAuto
}

// ColorLevel reports the color level that may be written to f.
//
// The capability tier — basic, 256 or truecolor — is chalk's answer
// (chalk.GetLevel, and therefore chalk.SetLevel if the program pinned it); this
// function only decides whether color is allowed at all for this particular
// stream, which chalk cannot do because its detection is bound to os.Stdout. The
// order is:
//
//  1. The environment gate (NO_COLOR / FORCE_COLOR / CLICOLOR_FORCE / CLICOLOR /
//     TERM=dumb). Off means [chalk.LevelNone]; on skips the terminal test.
//  2. f not being a terminal means [chalk.LevelNone]. This is the case this
//     package exists for: styling a pipe writes escape bytes into someone's data.
//  3. Otherwise chalk's level, with [chalk.LevelBasic] as a floor.
//
// The floor in step 3 is the one non-obvious part. chalk detected its level
// against os.Stdout, so a program whose stdout is redirected has
// chalk.GetLevel() == LevelNone; without the floor, a genuinely interactive
// os.Stderr would then be treated as colorless. Raising None to Basic for a
// stream that really is a terminal is the smallest correction that fixes it, and
// it never claims a *higher* tier than chalk detected.
func ColorLevel(f *os.File) chalk.Level {
	switch envColorGate() {
	case gateOff:
		return chalk.LevelNone
	case gateOn:
		return atLeastBasic(chalk.GetLevel())
	}
	if !IsTerminal(f) {
		return chalk.LevelNone
	}
	return atLeastBasic(chalk.GetLevel())
}

// atLeastBasic raises LevelNone to LevelBasic and leaves any real tier alone.
func atLeastBasic(l chalk.Level) chalk.Level {
	if l < chalk.LevelBasic {
		return chalk.LevelBasic
	}
	return l
}

// ColorEnabled reports whether any color may be written to f.
func ColorEnabled(f *os.File) bool { return ColorLevel(f) > chalk.LevelNone }

// SyncColor pushes this package's per-stream answer for f into the parent
// package's process-global level and returns what it set.
//
// It is opt-in on purpose. Calling chalk.SetLevel behind a program's back would
// mean two packages fighting over one global, and CLICOLOR support would arrive
// as a surprise; a program that wants chalk's shortcut functions to honour
// CLICOLOR and to follow the stream it actually writes to says so by calling
// this once at startup.
func SyncColor(f *os.File) chalk.Level {
	l := ColorLevel(f)
	chalk.SetLevel(l)
	return l
}

// Info is a snapshot of everything this package can say about one stream. It
// exists so a program can log its own idea of the world in one line when a user
// reports "the colors are wrong in my CI".
type Info struct {
	// Terminal reports whether the stream is a TTY.
	Terminal bool
	// Piped reports whether the stream is a pipe specifically.
	Piped bool
	// Width and Height are the measured or fallback dimensions.
	Width, Height int
	// Fallback reports that Width/Height did not come from a terminal.
	Fallback bool
	// Color is the level [ColorLevel] would allow for the stream.
	Color chalk.Level
}

// Inspect returns an [Info] for f. A nil f is reported as a non-terminal with
// fallback dimensions and no color, which is what a caller wants when the
// descriptor is missing entirely.
func Inspect(f *os.File) Info {
	w, h, measured := measure(f)
	return Info{
		Terminal: IsTerminal(f),
		Piped:    IsPiped(f),
		Width:    w,
		Height:   h,
		Fallback: !measured,
		Color:    ColorLevel(f),
	}
}
