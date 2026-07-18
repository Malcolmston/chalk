package chalk

// Additional package-level one-off shortcuts, completing the surface that Node
// chalk exposes at the top level (chalk.bgRed, chalk.redBright, chalk.reset, …).
// Each is a thin wrapper over the equivalent [Style] method.

// ---- remaining modifiers ----------------------------------------------------

// Reset styles text with all attributes reset.
func Reset(a ...any) string { return New().Reset().Sprint(a...) }

// Hidden styles text hidden (not displayed).
func Hidden(a ...any) string { return New().Hidden().Sprint(a...) }

// Overline styles text with an overline.
func Overline(a ...any) string { return New().Overline().Sprint(a...) }

// ---- bright foreground colors -----------------------------------------------

// BrightBlack colors text bright black (gray).
func BrightBlack(a ...any) string { return New().BrightBlack().Sprint(a...) }

// BrightRed colors text bright red.
func BrightRed(a ...any) string { return New().BrightRed().Sprint(a...) }

// BrightGreen colors text bright green.
func BrightGreen(a ...any) string { return New().BrightGreen().Sprint(a...) }

// BrightYellow colors text bright yellow.
func BrightYellow(a ...any) string { return New().BrightYellow().Sprint(a...) }

// BrightBlue colors text bright blue.
func BrightBlue(a ...any) string { return New().BrightBlue().Sprint(a...) }

// BrightMagenta colors text bright magenta.
func BrightMagenta(a ...any) string { return New().BrightMagenta().Sprint(a...) }

// BrightCyan colors text bright cyan.
func BrightCyan(a ...any) string { return New().BrightCyan().Sprint(a...) }

// BrightWhite colors text bright white.
func BrightWhite(a ...any) string { return New().BrightWhite().Sprint(a...) }

// ---- background colors ------------------------------------------------------

// BgBlack styles text on a black background.
func BgBlack(a ...any) string { return New().BgBlack().Sprint(a...) }

// BgRed styles text on a red background.
func BgRed(a ...any) string { return New().BgRed().Sprint(a...) }

// BgGreen styles text on a green background.
func BgGreen(a ...any) string { return New().BgGreen().Sprint(a...) }

// BgYellow styles text on a yellow background.
func BgYellow(a ...any) string { return New().BgYellow().Sprint(a...) }

// BgBlue styles text on a blue background.
func BgBlue(a ...any) string { return New().BgBlue().Sprint(a...) }

// BgMagenta styles text on a magenta background.
func BgMagenta(a ...any) string { return New().BgMagenta().Sprint(a...) }

// BgCyan styles text on a cyan background.
func BgCyan(a ...any) string { return New().BgCyan().Sprint(a...) }

// BgWhite styles text on a white background.
func BgWhite(a ...any) string { return New().BgWhite().Sprint(a...) }

// BgGray styles text on a gray (bright black) background.
func BgGray(a ...any) string { return New().BgGray().Sprint(a...) }

// ---- background truecolor / palette -----------------------------------------

// BgRGB styles text on a 24-bit background color.
func BgRGB(r, g, b int, a ...any) string { return New().BgRGB(r, g, b).Sprint(a...) }

// BgHex styles text on a background color from a hex string like "#ff8800".
func BgHex(hex string, a ...any) string { return New().BgHex(hex).Sprint(a...) }

// BgAnsi256 styles text on a 256-palette background color.
func BgAnsi256(n int, a ...any) string { return New().BgAnsi256(n).Sprint(a...) }
