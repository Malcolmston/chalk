// Package chalk is a Go port of Node's chalk — expressive terminal string
// styling. Styles are chainable and compose naturally:
//
//	fmt.Println(chalk.New().Red().Bold().Sprint("error!"))
//	fmt.Println(chalk.Green("ok"))                 // package shortcut
//	fmt.Println(chalk.New().Hex("#ff8800").Sprint("orange"))
//
// Color output is enabled automatically when stdout is a terminal and NO_COLOR
// is unset; it degrades gracefully from truecolor to 256 to 16 colors based on
// the detected terminal capability.
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
}

// New returns an empty Style.
func New() *Style { return &Style{} }

// with returns a copy of the style with an additional SGR pair.
func (s *Style) with(open, close string) *Style {
	cp := &Style{parts: make([]sgr, len(s.parts), len(s.parts)+1), level: s.level}
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
// handling nested styles by re-opening after any inner close code.
func (s *Style) render(text string) string {
	if s.effectiveLevel() == LevelNone || len(s.parts) == 0 {
		return text
	}
	// Apply innermost first so outer styles re-assert after inner resets.
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
