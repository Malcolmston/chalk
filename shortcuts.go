package chalk

import "regexp"

// Package-level shortcuts for one-off styling, e.g. chalk.Red("error").

// Bold styles text bold.
func Bold(a ...any) string { return New().Bold().Sprint(a...) }

// Dim styles text dim.
func Dim(a ...any) string { return New().Dim().Sprint(a...) }

// Italic styles text italic.
func Italic(a ...any) string { return New().Italic().Sprint(a...) }

// Underline styles text underlined.
func Underline(a ...any) string { return New().Underline().Sprint(a...) }

// Inverse styles text inverted.
func Inverse(a ...any) string { return New().Inverse().Sprint(a...) }

// Strikethrough styles text struck through.
func Strikethrough(a ...any) string { return New().Strikethrough().Sprint(a...) }

// Black colors text black.
func Black(a ...any) string { return New().Black().Sprint(a...) }

// Red colors text red.
func Red(a ...any) string { return New().Red().Sprint(a...) }

// Green colors text green.
func Green(a ...any) string { return New().Green().Sprint(a...) }

// Yellow colors text yellow.
func Yellow(a ...any) string { return New().Yellow().Sprint(a...) }

// Blue colors text blue.
func Blue(a ...any) string { return New().Blue().Sprint(a...) }

// Magenta colors text magenta.
func Magenta(a ...any) string { return New().Magenta().Sprint(a...) }

// Cyan colors text cyan.
func Cyan(a ...any) string { return New().Cyan().Sprint(a...) }

// White colors text white.
func White(a ...any) string { return New().White().Sprint(a...) }

// Gray colors text gray (bright black).
func Gray(a ...any) string { return New().Gray().Sprint(a...) }

// RGB colors text with a 24-bit color.
func RGB(r, g, b int, a ...any) string { return New().RGB(r, g, b).Sprint(a...) }

// Hex colors text from a hex string like "#ff8800".
func Hex(hex string, a ...any) string { return New().Hex(hex).Sprint(a...) }

// Ansi256 colors text with a 256-palette index.
func Ansi256(n int, a ...any) string { return New().Ansi256(n).Sprint(a...) }

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Strip removes all ANSI SGR escape sequences from s.
func Strip(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// VisibleLength returns the number of visible characters in s, ignoring ANSI
// escape codes (counts runes, not bytes).
func VisibleLength(s string) int { return len([]rune(Strip(s))) }
