package figlet

import (
	"math"
	"strings"

	"github.com/malcolmston/chalk"
)

// Gradient colors a rendered banner with a left-to-right color gradient from
// startHex to endHex (e.g. "#ff0080" → "#00d7ff"). Each column is tinted by its
// horizontal position.
func Gradient(banner, startHex, endHex string) string {
	sr, sg, sb := hexRGB(startHex)
	er, eg, eb := hexRGB(endHex)

	lines := strings.Split(banner, "\n")
	width := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > width {
			width = n
		}
	}
	if width < 2 {
		width = 2
	}

	var out strings.Builder
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		for col, ch := range []rune(line) {
			t := float64(col) / float64(width-1)
			r := int(math.Round(lerp(float64(sr), float64(er), t)))
			g := int(math.Round(lerp(float64(sg), float64(eg), t)))
			b := int(math.Round(lerp(float64(sb), float64(eb), t)))
			out.WriteString(chalk.New().RGB(r, g, b).Sprint(string(ch)))
		}
	}
	return out.String()
}

// Rainbow colors a rendered banner across the full hue spectrum by column.
func Rainbow(banner string) string {
	lines := strings.Split(banner, "\n")
	width := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > width {
			width = n
		}
	}
	if width < 1 {
		width = 1
	}

	var out strings.Builder
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		for col, ch := range []rune(line) {
			hue := float64(col) / float64(width) * 360
			r, g, b := hsvToRGB(hue, 1, 1)
			out.WriteString(chalk.New().RGB(r, g, b).Sprint(string(ch)))
		}
	}
	return out.String()
}

// RenderGradient renders text with the built-in font and applies a gradient.
func RenderGradient(text, startHex, endHex string) string {
	return Gradient(Render(text), startHex, endHex)
}

// RenderRainbow renders text with the built-in font and applies rainbow colors.
func RenderRainbow(text string) string {
	return Rainbow(Render(text))
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func hexRGB(hex string) (int, int, int) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 255, 255, 255
	}
	val := func(s string) int {
		n := 0
		for _, c := range s {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= int(c - '0')
			case c >= 'a' && c <= 'f':
				n |= int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n |= int(c-'A') + 10
			}
		}
		return n
	}
	return val(h[0:2]), val(h[2:4]), val(h[4:6])
}

func hsvToRGB(h, s, v float64) (int, int, int) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return int(math.Round((r + m) * 255)), int(math.Round((g + m) * 255)), int(math.Round((b + m) * 255))
}
