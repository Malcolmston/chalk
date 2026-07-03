package chalk

import (
	"math"
	"strconv"
)

// ---- text modifiers ---------------------------------------------------------

// Reset clears all styles.
func (s *Style) Reset() *Style { return s.with("0", "0") }

// Bold makes text bold.
func (s *Style) Bold() *Style { return s.with("1", "22") }

// Dim makes text dim/faint.
func (s *Style) Dim() *Style { return s.with("2", "22") }

// Italic makes text italic (not widely supported).
func (s *Style) Italic() *Style { return s.with("3", "23") }

// Underline underlines text.
func (s *Style) Underline() *Style { return s.with("4", "24") }

// Inverse swaps foreground and background.
func (s *Style) Inverse() *Style { return s.with("7", "27") }

// Hidden hides text.
func (s *Style) Hidden() *Style { return s.with("8", "28") }

// Strikethrough draws a line through text.
func (s *Style) Strikethrough() *Style { return s.with("9", "29") }

// Overline draws a line above text.
func (s *Style) Overline() *Style { return s.with("53", "55") }

// ---- foreground colors ------------------------------------------------------

// Black colors text black.
func (s *Style) Black() *Style { return s.with("30", "39") }

// Red colors text red.
func (s *Style) Red() *Style { return s.with("31", "39") }

// Green colors text green.
func (s *Style) Green() *Style { return s.with("32", "39") }

// Yellow colors text yellow.
func (s *Style) Yellow() *Style { return s.with("33", "39") }

// Blue colors text blue.
func (s *Style) Blue() *Style { return s.with("34", "39") }

// Magenta colors text magenta.
func (s *Style) Magenta() *Style { return s.with("35", "39") }

// Cyan colors text cyan.
func (s *Style) Cyan() *Style { return s.with("36", "39") }

// White colors text white.
func (s *Style) White() *Style { return s.with("37", "39") }

// Gray is an alias for BrightBlack.
func (s *Style) Gray() *Style { return s.BrightBlack() }

// Grey is an alias for BrightBlack.
func (s *Style) Grey() *Style { return s.BrightBlack() }

// BrightBlack colors text bright black (gray).
func (s *Style) BrightBlack() *Style { return s.with("90", "39") }

// BrightRed colors text bright red.
func (s *Style) BrightRed() *Style { return s.with("91", "39") }

// BrightGreen colors text bright green.
func (s *Style) BrightGreen() *Style { return s.with("92", "39") }

// BrightYellow colors text bright yellow.
func (s *Style) BrightYellow() *Style { return s.with("93", "39") }

// BrightBlue colors text bright blue.
func (s *Style) BrightBlue() *Style { return s.with("94", "39") }

// BrightMagenta colors text bright magenta.
func (s *Style) BrightMagenta() *Style { return s.with("95", "39") }

// BrightCyan colors text bright cyan.
func (s *Style) BrightCyan() *Style { return s.with("96", "39") }

// BrightWhite colors text bright white.
func (s *Style) BrightWhite() *Style { return s.with("97", "39") }

// ---- background colors ------------------------------------------------------

// BgBlack sets a black background.
func (s *Style) BgBlack() *Style { return s.with("40", "49") }

// BgRed sets a red background.
func (s *Style) BgRed() *Style { return s.with("41", "49") }

// BgGreen sets a green background.
func (s *Style) BgGreen() *Style { return s.with("42", "49") }

// BgYellow sets a yellow background.
func (s *Style) BgYellow() *Style { return s.with("43", "49") }

// BgBlue sets a blue background.
func (s *Style) BgBlue() *Style { return s.with("44", "49") }

// BgMagenta sets a magenta background.
func (s *Style) BgMagenta() *Style { return s.with("45", "49") }

// BgCyan sets a cyan background.
func (s *Style) BgCyan() *Style { return s.with("46", "49") }

// BgWhite sets a white background.
func (s *Style) BgWhite() *Style { return s.with("47", "49") }

// BgGray is an alias for BgBrightBlack.
func (s *Style) BgGray() *Style { return s.BgBrightBlack() }

// BgBrightBlack sets a bright black background.
func (s *Style) BgBrightBlack() *Style { return s.with("100", "49") }

// BgBrightRed sets a bright red background.
func (s *Style) BgBrightRed() *Style { return s.with("101", "49") }

// BgBrightGreen sets a bright green background.
func (s *Style) BgBrightGreen() *Style { return s.with("102", "49") }

// BgBrightYellow sets a bright yellow background.
func (s *Style) BgBrightYellow() *Style { return s.with("103", "49") }

// BgBrightBlue sets a bright blue background.
func (s *Style) BgBrightBlue() *Style { return s.with("104", "49") }

// BgBrightMagenta sets a bright magenta background.
func (s *Style) BgBrightMagenta() *Style { return s.with("105", "49") }

// BgBrightCyan sets a bright cyan background.
func (s *Style) BgBrightCyan() *Style { return s.with("106", "49") }

// BgBrightWhite sets a bright white background.
func (s *Style) BgBrightWhite() *Style { return s.with("107", "49") }

// ---- 256-color and truecolor ------------------------------------------------

// Ansi256 sets the foreground to a 256-palette color (0–255), degrading to 16
// colors on limited terminals.
func (s *Style) Ansi256(n int) *Style {
	return s.with(fg256(n, s.effectiveLevel()), "39")
}

// BgAnsi256 sets the background to a 256-palette color (0–255).
func (s *Style) BgAnsi256(n int) *Style {
	return s.with(bg256(n, s.effectiveLevel()), "49")
}

// RGB sets the foreground to a 24-bit color, degrading to 256/16 colors as
// needed.
func (s *Style) RGB(r, g, b int) *Style {
	return s.with(fgRGB(r, g, b, s.effectiveLevel()), "39")
}

// BgRGB sets the background to a 24-bit color.
func (s *Style) BgRGB(r, g, b int) *Style {
	return s.with(bgRGB(r, g, b, s.effectiveLevel()), "49")
}

// Hex sets the foreground from a hex color like "#ff8800" or "f80".
func (s *Style) Hex(hex string) *Style {
	r, g, b := parseHex(hex)
	return s.RGB(r, g, b)
}

// BgHex sets the background from a hex color.
func (s *Style) BgHex(hex string) *Style {
	r, g, b := parseHex(hex)
	return s.BgRGB(r, g, b)
}

// ---- color-space conversions ------------------------------------------------

func fgRGB(r, g, b int, level Level) string {
	switch {
	case level >= LevelTrueColor:
		return "38;2;" + itoa(clamp(r)) + ";" + itoa(clamp(g)) + ";" + itoa(clamp(b))
	case level == Level256:
		return "38;5;" + itoa(rgbTo256(r, g, b))
	default:
		return itoa(rgbTo16(r, g, b))
	}
}

func bgRGB(r, g, b int, level Level) string {
	switch {
	case level >= LevelTrueColor:
		return "48;2;" + itoa(clamp(r)) + ";" + itoa(clamp(g)) + ";" + itoa(clamp(b))
	case level == Level256:
		return "48;5;" + itoa(rgbTo256(r, g, b))
	default:
		return itoa(rgbTo16(r, g, b) + 10) // fg code + 10 = bg code
	}
}

func fg256(n int, level Level) string {
	if level >= Level256 {
		return "38;5;" + itoa(n&0xff)
	}
	r, g, b := ansi256ToRGB(n)
	return itoa(rgbTo16(r, g, b))
}

func bg256(n int, level Level) string {
	if level >= Level256 {
		return "48;5;" + itoa(n&0xff)
	}
	r, g, b := ansi256ToRGB(n)
	return itoa(rgbTo16(r, g, b) + 10)
}

// rgbTo256 maps a 24-bit color to the nearest xterm-256 palette index.
func rgbTo256(r, g, b int) int {
	r, g, b = clamp(r), clamp(g), clamp(b)
	if r == g && g == b {
		switch {
		case r < 8:
			return 16
		case r > 248:
			return 231
		default:
			return int(math.Round((float64(r)-8)/247*24)) + 232
		}
	}
	return 16 +
		36*int(math.Round(float64(r)/255*5)) +
		6*int(math.Round(float64(g)/255*5)) +
		int(math.Round(float64(b)/255*5))
}

// rgbTo16 maps a 24-bit color to the nearest basic 16-color foreground code
// (30–37 or 90–97).
func rgbTo16(r, g, b int) int {
	r, g, b = clamp(r), clamp(g), clamp(b)
	value := int(math.Round(float64(max3(r, g, b)) / 255 * 2))
	if value == 0 {
		return 30
	}
	ansi := 30 + (bit(b)<<2 | bit(g)<<1 | bit(r))
	if value == 2 {
		ansi += 60
	}
	return ansi
}

// ansi256ToRGB converts an xterm-256 index back to an approximate RGB triple.
func ansi256ToRGB(n int) (int, int, int) {
	n &= 0xff
	switch {
	case n < 16:
		// Standard 16 colors — approximate.
		v := 128
		if n >= 8 {
			v = 255
			n -= 8
		}
		r := 0
		g := 0
		b := 0
		if n&1 != 0 {
			r = v
		}
		if n&2 != 0 {
			g = v
		}
		if n&4 != 0 {
			b = v
		}
		return r, g, b
	case n >= 232:
		v := (n-232)*10 + 8
		return v, v, v
	default:
		n -= 16
		r := n / 36
		g := (n % 36) / 6
		b := n % 6
		conv := func(c int) int {
			if c == 0 {
				return 0
			}
			return c*40 + 55
		}
		return conv(r), conv(g), conv(b)
	}
}

func parseHex(hex string) (int, int, int) {
	h := hex
	if len(h) > 0 && h[0] == '#' {
		h = h[1:]
	}
	if len(h) == 3 { // shorthand like "f80"
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 0, 0, 0
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func bit(c int) int {
	if c > 127 {
		return 1
	}
	return 0
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

func itoa(n int) string { return strconv.Itoa(n) }
