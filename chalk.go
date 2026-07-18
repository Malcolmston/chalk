// Package chalk is a Go port of Node's chalk — expressive terminal string
// styling. Like the original JavaScript library, it wraps text in ANSI SGR
// (Select Graphic Rendition) escape sequences so that terminals render it with
// color, weight and other attributes. Styles are chainable and compose
// naturally:
//
//	fmt.Println(chalk.New().Red().Bold().Sprint("error!"))
//	fmt.Println(chalk.Green("ok"))                 // package shortcut
//	fmt.Println(chalk.New().Hex("#ff8800").Sprint("orange"))
//
// Reach for chalk whenever a command-line program wants to highlight output —
// errors in red, success in green, hints in a dim gray — without hand-writing
// escape codes or pulling in a dependency. The package is standard-library only.
// Every entry point comes in two flavors: a fluent [Style] built with [New] and
// chained methods for reuse, and package-level shortcuts such as [Red] or [Hex]
// for one-off styling. A [Style] is immutable — each method returns a new value
// — so a configured style is safe to store and share across goroutines.
//
// Internally a [Style] records a list of open/close SGR code pairs. At render
// time each pair is emitted as an escape sequence of the form ESC[<codes>m
// around the text, applying the innermost style first so that outer styles
// re-assert themselves. When a piece of text already contains a matching close
// code (from a nested style), chalk re-opens the outer style immediately after
// it, mirroring the "style bleed" fix in Node chalk so nested colors survive.
// Foreground and background colors, the 16 basic and 16 bright ANSI colors, the
// 256-color palette ([Style.Ansi256]), 24-bit truecolor ([Style.RGB]) and hex
// strings ([Style.Hex]) are all supported.
//
// Color output is enabled automatically when stdout is a terminal and NO_COLOR
// is unset. Detection follows the same conventions as the Node ecosystem: the
// [Level] is derived from NO_COLOR, FORCE_COLOR, COLORTERM and TERM (see
// [GetLevel] and [SetLevel]). Truecolor and 256-color requests degrade
// gracefully down to the nearest color the terminal actually supports, so an
// [Style.RGB] call still produces reasonable output on a 16-color terminal and
// emits nothing at all when the level is [LevelNone]. Detection happens once and
// is cached; [ResetDetection] forces a re-detect and [SetLevel] pins a fixed
// level, which is the recommended way to get deterministic output in tests.
//
// A handful of edge cases are worth knowing. When color is disabled the render
// methods return the input text unchanged, so it is always safe to wrap output
// in a style. [Strip] removes SGR sequences from a string and [VisibleLength]
// reports the on-screen width (counting runes, not bytes, and ignoring escape
// codes), which is useful for laying out tables or padding colored columns.
// Because detection keys off os.Stdout, redirecting output to a file or pipe
// disables color automatically unless FORCE_COLOR overrides it.
//
// Parity with Node chalk is close but not exact. The chainable API, automatic
// level detection, the NO_COLOR/FORCE_COLOR conventions and the truecolor→256→16
// downgrade all match. The differences are idiomatic: styling is applied with
// Sprint/Sprintf/Println methods (and package shortcuts) rather than by calling
// the style as a function, template literal tagging is not provided, and the
// global level is process-wide rather than per-instance. Rarely supported
// attributes such as [Style.Italic] and [Style.Overline] are included for
// completeness even though not every terminal honors them.
package chalk

import (
	"fmt"
	"strings"
)

const esc = "\x1b["

// style is one open/close SGR pair applied to text.
type sgr struct {
	open  string
	close string
}

// Style is an immutable, chainable set of terminal styles. Build one with New()
// and the fluent methods, then render with Sprint/Sprintf/Print/Println.
type Style struct {
	parts []sgr
	// level overrides the global color level for this style when non-nil.
	level *Level
	// visibleOnly, when set, suppresses all output while color is disabled
	// (the chalk ".visible" modifier).
	visibleOnly bool
}

// New returns an empty Style.
func New() *Style { return &Style{} }

// with returns a copy of the style with an additional SGR pair.
func (s *Style) with(open, close string) *Style {
	cp := &Style{parts: make([]sgr, len(s.parts), len(s.parts)+1), level: s.level, visibleOnly: s.visibleOnly}
	copy(cp.parts, s.parts)
	cp.parts = append(cp.parts, sgr{open: open, close: close})
	return cp
}

// Level pins this style to a specific color level regardless of the global
// setting (useful for testing or forcing output).
func (s *Style) Level(l Level) *Style {
	cp := *s
	cp.level = &l
	return &cp
}

// effectiveLevel resolves the color level for rendering.
func (s *Style) effectiveLevel() Level {
	if s.level != nil {
		return *s.level
	}
	return currentLevel()
}

// render wraps text in this style's SGR codes (unless color is disabled),
// handling nested styles by re-opening after any inner close code and closing
// then re-opening the style around every line break.
//
// Two edge cases mirror Node chalk exactly. Empty input yields no escape codes
// at all (chalk.red() === ”), so it is always safe to style a possibly-empty
// value. And line breaks close the active style before each newline and re-open
// it afterwards, so colors do not bleed across lines and each visual line is
// styled independently; both LF ("\n") and CRLF ("\r\n") are handled, with the
// carriage return preserved ahead of the closing code.
func (s *Style) render(text string) string {
	if s.visibleOnly && s.effectiveLevel() == LevelNone {
		return ""
	}
	if s.effectiveLevel() == LevelNone || len(s.parts) == 0 {
		return text
	}
	// Empty input produces no escape codes, matching Node chalk.
	if text == "" {
		return ""
	}
	// Close and re-open the style around each line break so colors never bleed
	// across lines (Node chalk's stringEncaseCRLFWithFirstIndex behavior).
	if strings.IndexByte(text, '\n') >= 0 {
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			cr := ""
			if strings.HasSuffix(line, "\r") {
				line = line[:len(line)-1]
				cr = "\r"
			}
			lines[i] = s.wrap(line) + cr
		}
		return strings.Join(lines, "\n")
	}
	return s.wrap(text)
}

// wrap encloses a single line (no newline) in this style's SGR codes, applying
// the innermost pair first so outer styles re-assert after an inner reset. When
// the line already contains one of this style's close codes (from a nested
// style of the same type) the outer style is re-opened immediately after it so
// nested colors survive — the "style bleed" fix from Node chalk.
func (s *Style) wrap(text string) string {
	for i := len(s.parts) - 1; i >= 0; i-- {
		p := s.parts[i]
		openSeq := esc + p.open + "m"
		closeSeq := esc + p.close + "m"
		if strings.Contains(text, closeSeq) {
			text = strings.ReplaceAll(text, closeSeq, closeSeq+openSeq)
		}
		text = openSeq + text + closeSeq
	}
	return text
}

// Sprint styles the concatenation of its operands (like fmt.Sprint).
func (s *Style) Sprint(a ...any) string { return s.render(fmt.Sprint(a...)) }

// Sprintf styles a formatted string.
func (s *Style) Sprintf(format string, a ...any) string {
	return s.render(fmt.Sprintf(format, a...))
}

// Sprintln styles its operands with a trailing newline (the newline is outside
// the style codes).
func (s *Style) Sprintln(a ...any) string {
	return s.render(strings.TrimSuffix(fmt.Sprintln(a...), "\n")) + "\n"
}

// Print writes the styled operands to stdout.
func (s *Style) Print(a ...any) (int, error) { return fmt.Print(s.Sprint(a...)) }

// Printf writes a styled formatted string to stdout.
func (s *Style) Printf(format string, a ...any) (int, error) {
	return fmt.Print(s.Sprintf(format, a...))
}

// Println writes the styled operands and a newline to stdout.
func (s *Style) Println(a ...any) (int, error) { return fmt.Print(s.Sprintln(a...)) }
