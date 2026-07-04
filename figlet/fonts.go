package figlet

// This file provides the shared constructor used by the bundled font data files
// (fonts_*.go). Each bundled font supplies a glyph map — rows of ASCII art, one
// slice per rune, every glyph the same number of rows — and registers itself
// under one or more names in its own init().

// FontFromGlyphs builds a *Font from a glyph map of the given row height. Rows
// within a glyph need not be equal width (they are padded at render time), but
// every glyph must have exactly height rows. layout selects horizontal spacing:
// use 0 for kerning (characters slide together until they touch, the natural
// look for proportional fonts) or -1 for full fixed width.
//
// Lowercase input automatically falls back to the uppercase glyph, so fonts
// that define only capitals still render mixed-case text.
func FontFromGlyphs(height, layout int, glyphs map[rune][]string) *Font {
	chars := make(map[rune][]string, len(glyphs))
	for r, rows := range glyphs {
		cp := make([]string, len(rows))
		copy(cp, rows)
		chars[r] = cp
	}
	return &Font{
		hardblank: '$',
		height:    height,
		baseline:  height,
		oldLayout: layout,
		chars:     chars,
	}
}
