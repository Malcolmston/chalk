package chalk

import "math"

// This file exposes the color-space conversions that Node chalk relies on
// internally (via the ansi-styles / color-convert modules). They are pure
// functions with no dependency on the global color level, useful for building
// gradients, choosing contrasting colors, or feeding [Style.RGB] and friends.
//
// Conventions match color-convert: RGB channels are 0–255, hue is 0–360
// degrees, and saturation / lightness / value / whiteness / blackness are
// 0–100 percentages. All results are rounded to the nearest integer.

// HexToRGB parses a hex color such as "#ff8800", "ff8800" or the shorthand
// "#f80" and returns its red, green and blue channels (each 0–255). Invalid
// input yields (0, 0, 0).
func HexToRGB(hex string) (r, g, b int) {
	return parseHex(hex)
}

// RGBToHex formats an RGB triple as a lowercase "#rrggbb" hex string. Channels
// are clamped to the 0–255 range first.
func RGBToHex(r, g, b int) string {
	const digits = "0123456789abcdef"
	r, g, b = clamp(r), clamp(g), clamp(b)
	buf := []byte("#000000")
	buf[1] = digits[r>>4]
	buf[2] = digits[r&0xf]
	buf[3] = digits[g>>4]
	buf[4] = digits[g&0xf]
	buf[5] = digits[b>>4]
	buf[6] = digits[b&0xf]
	return string(buf)
}

// RGBToAnsi256 maps a 24-bit color to the nearest xterm 256-color palette index
// (0–255), using the same 6×6×6 color cube and grayscale ramp as color-convert.
func RGBToAnsi256(r, g, b int) int {
	return rgbTo256(r, g, b)
}

// Ansi256ToRGB converts an xterm 256-color palette index (0–255) back to an
// approximate RGB triple.
func Ansi256ToRGB(n int) (r, g, b int) {
	return ansi256ToRGB(n)
}

// RGBToAnsi16 maps a 24-bit color to the nearest basic 16-color SGR foreground
// code (30–37 for normal colors, 90–97 for the bright variants).
func RGBToAnsi16(r, g, b int) int {
	return rgbTo16(r, g, b)
}

// Ansi256ToAnsi16 reduces an xterm 256-color palette index to the nearest basic
// 16-color SGR foreground code (30–37 or 90–97).
func Ansi256ToAnsi16(n int) int {
	r, g, b := ansi256ToRGB(n)
	return rgbTo16(r, g, b)
}

// RGBToHSL converts an RGB triple (0–255) to hue (0–360), saturation (0–100)
// and lightness (0–100).
func RGBToHSL(r, g, b int) (h, s, l int) {
	rf := float64(clamp(r)) / 255
	gf := float64(clamp(g)) / 255
	bf := float64(clamp(b)) / 255
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	lf := (max + min) / 2
	var hf, sf float64
	if max == min {
		hf, sf = 0, 0
	} else {
		d := max - min
		if lf > 0.5 {
			sf = d / (2 - max - min)
		} else {
			sf = d / (max + min)
		}
		hf = chalkHue(rf, gf, bf, max, d)
	}
	return int(math.Round(hf * 360)), int(math.Round(sf * 100)), int(math.Round(lf * 100))
}

// HSLToRGB converts hue (0–360), saturation (0–100) and lightness (0–100) to an
// RGB triple (0–255).
func HSLToRGB(h, s, l int) (r, g, b int) {
	hf := chalkWrapHue(float64(h)) / 360
	sf := float64(s) / 100
	lf := float64(l) / 100
	if sf == 0 {
		v := int(math.Round(lf * 255))
		return v, v, v
	}
	var q float64
	if lf < 0.5 {
		q = lf * (1 + sf)
	} else {
		q = lf + sf - lf*sf
	}
	p := 2*lf - q
	rf := chalkHue2RGB(p, q, hf+1.0/3.0)
	gf := chalkHue2RGB(p, q, hf)
	bf := chalkHue2RGB(p, q, hf-1.0/3.0)
	return int(math.Round(rf * 255)), int(math.Round(gf * 255)), int(math.Round(bf * 255))
}

// RGBToHSV converts an RGB triple (0–255) to hue (0–360), saturation (0–100)
// and value/brightness (0–100).
func RGBToHSV(r, g, b int) (h, s, v int) {
	rf := float64(clamp(r)) / 255
	gf := float64(clamp(g)) / 255
	bf := float64(clamp(b)) / 255
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	d := max - min
	var hf, sf float64
	if d == 0 {
		hf = 0
	} else {
		hf = chalkHue(rf, gf, bf, max, d)
	}
	if max != 0 {
		sf = d / max
	}
	return int(math.Round(hf * 360)), int(math.Round(sf * 100)), int(math.Round(max * 100))
}

// HSVToRGB converts hue (0–360), saturation (0–100) and value/brightness
// (0–100) to an RGB triple (0–255).
func HSVToRGB(h, s, v int) (r, g, b int) {
	hf := chalkWrapHue(float64(h)) / 60
	sf := float64(s) / 100
	vf := float64(v) / 100
	i := int(math.Floor(hf)) % 6
	if i < 0 {
		i += 6
	}
	f := hf - math.Floor(hf)
	p := vf * (1 - sf)
	q := vf * (1 - f*sf)
	t := vf * (1 - (1-f)*sf)
	var rf, gf, bf float64
	switch i {
	case 0:
		rf, gf, bf = vf, t, p
	case 1:
		rf, gf, bf = q, vf, p
	case 2:
		rf, gf, bf = p, vf, t
	case 3:
		rf, gf, bf = p, q, vf
	case 4:
		rf, gf, bf = t, p, vf
	default:
		rf, gf, bf = vf, p, q
	}
	return int(math.Round(rf * 255)), int(math.Round(gf * 255)), int(math.Round(bf * 255))
}

// RGBToHWB converts an RGB triple (0–255) to hue (0–360), whiteness (0–100) and
// blackness (0–100), the HWB color model used by CSS Color 4 and color-convert.
func RGBToHWB(r, g, b int) (h, w, bl int) {
	rf := float64(clamp(r)) / 255
	gf := float64(clamp(g)) / 255
	bf := float64(clamp(b)) / 255
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	d := max - min
	var hf float64
	if d != 0 {
		hf = chalkHue(rf, gf, bf, max, d)
	}
	return int(math.Round(hf * 360)), int(math.Round(min * 100)), int(math.Round((1 - max) * 100))
}

// HWBToRGB converts hue (0–360), whiteness (0–100) and blackness (0–100) to an
// RGB triple (0–255).
func HWBToRGB(h, w, bl int) (r, g, b int) {
	hf := chalkWrapHue(float64(h)) / 360
	wh := float64(w) / 100
	blk := float64(bl) / 100
	if ratio := wh + blk; ratio > 1 {
		wh /= ratio
		blk /= ratio
	}
	i := int(math.Floor(6 * hf))
	v := 1 - blk
	f := 6*hf - float64(i)
	if i&1 != 0 {
		f = 1 - f
	}
	n := wh + f*(v-wh)
	var rf, gf, bf float64
	switch i % 6 {
	case 0:
		rf, gf, bf = v, n, wh
	case 1:
		rf, gf, bf = n, v, wh
	case 2:
		rf, gf, bf = wh, v, n
	case 3:
		rf, gf, bf = wh, n, v
	case 4:
		rf, gf, bf = n, wh, v
	default:
		rf, gf, bf = v, wh, n
	}
	return int(math.Round(rf * 255)), int(math.Round(gf * 255)), int(math.Round(bf * 255))
}

// chalkHue computes the fractional hue (0–1) shared by the HSL/HSV/HWB
// conversions given the normalized channels, the maximum channel and the delta.
func chalkHue(rf, gf, bf, max, d float64) float64 {
	var hf float64
	switch max {
	case rf:
		hf = (gf - bf) / d
		if gf < bf {
			hf += 6
		}
	case gf:
		hf = (bf-rf)/d + 2
	default:
		hf = (rf-gf)/d + 4
	}
	return hf / 6
}

// chalkHue2RGB is the HSL-to-RGB channel helper.
func chalkHue2RGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

// chalkWrapHue normalizes a hue in degrees into the [0, 360) range.
func chalkWrapHue(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}
