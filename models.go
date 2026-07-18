package chalk

// This file adds the remaining color-model entry points that Node chalk exposes
// (chalk.hsl / chalk.hsv / chalk.hwb and their bg* variants) plus the ".visible"
// modifier. Each model method converts to RGB and then defers to [Style.RGB] /
// [Style.BgRGB], so the usual truecolor→256→16 downgrade applies automatically.

// HSL sets the foreground from an HSL color: hue 0–360, saturation 0–100 and
// lightness 0–100.
func (s *Style) HSL(h, sat, l int) *Style {
	r, g, b := HSLToRGB(h, sat, l)
	return s.RGB(r, g, b)
}

// BgHSL sets the background from an HSL color: hue 0–360, saturation 0–100 and
// lightness 0–100.
func (s *Style) BgHSL(h, sat, l int) *Style {
	r, g, b := HSLToRGB(h, sat, l)
	return s.BgRGB(r, g, b)
}

// HSV sets the foreground from an HSV color: hue 0–360, saturation 0–100 and
// value/brightness 0–100.
func (s *Style) HSV(h, sat, v int) *Style {
	r, g, b := HSVToRGB(h, sat, v)
	return s.RGB(r, g, b)
}

// BgHSV sets the background from an HSV color: hue 0–360, saturation 0–100 and
// value/brightness 0–100.
func (s *Style) BgHSV(h, sat, v int) *Style {
	r, g, b := HSVToRGB(h, sat, v)
	return s.BgRGB(r, g, b)
}

// HWB sets the foreground from an HWB color: hue 0–360, whiteness 0–100 and
// blackness 0–100.
func (s *Style) HWB(h, w, bl int) *Style {
	r, g, b := HWBToRGB(h, w, bl)
	return s.RGB(r, g, b)
}

// BgHWB sets the background from an HWB color: hue 0–360, whiteness 0–100 and
// blackness 0–100.
func (s *Style) BgHWB(h, w, bl int) *Style {
	r, g, b := HWBToRGB(h, w, bl)
	return s.BgRGB(r, g, b)
}

// Visible returns a style that emits its text only when color output is enabled
// (the color level is greater than [LevelNone]). When color is disabled the
// render methods return an empty string instead of the raw text. This mirrors
// Node chalk's ".visible" modifier, handy for decorations that should vanish in
// plain, piped output.
func (s *Style) Visible() *Style {
	cp := *s
	cp.visibleOnly = true
	return &cp
}

// ---- package-level shortcuts ------------------------------------------------

// HSL styles text with an HSL foreground color (hue 0–360, saturation 0–100,
// lightness 0–100).
func HSL(h, sat, l int, a ...any) string { return New().HSL(h, sat, l).Sprint(a...) }

// HSV styles text with an HSV foreground color (hue 0–360, saturation 0–100,
// value 0–100).
func HSV(h, sat, v int, a ...any) string { return New().HSV(h, sat, v).Sprint(a...) }

// HWB styles text with an HWB foreground color (hue 0–360, whiteness 0–100,
// blackness 0–100).
func HWB(h, w, bl int, a ...any) string { return New().HWB(h, w, bl).Sprint(a...) }

// Visible styles text so that it renders only when color output is enabled,
// returning an empty string otherwise.
func Visible(a ...any) string { return New().Visible().Sprint(a...) }
