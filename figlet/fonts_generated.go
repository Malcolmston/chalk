package figlet

import "strings"

// This file programmatically generates a large family of named FIGlet-style
// fonts so that ~1,000 distinct fonts are importable through the registry
// (see Fonts, GetFont and RenderFont).
//
// Each generated font is a systematic, visually-distinct variant built by
// combining three independent axes over the hand-authored glyph designs already
// bundled in this package:
//
//   - base shape:  the 5-row block glyphs (fonts.go), the 3-row "small" glyphs
//     (fonts_small.go) and the 5-row "banner" outline glyphs (fonts_banner.go).
//   - fill/ink:    the character every inked cell is drawn with (# █ ▓ ● ★ …).
//   - decoration:  a cheap transform of the rendered glyph map — bold, shadow,
//     box, invert, outline, mirror, flip and friends.
//
// Names are deterministic: "<base>-<fill>" for the plain variant and
// "<base>-<fill>-<decoration>" otherwise, e.g. block-hash, small-dots,
// banner-star-shadow, block-plus-outline. All names are lowercase and unique;
// registration guards against collisions with the hand-authored fonts.

// genBase names a base glyph map and its short label used in font names.
type genBase struct {
	name   string
	glyphs map[rune][]string
}

// genFill names an ink rune and its short label used in font names.
type genFill struct {
	name string
	r    rune
}

// genDecorator names a glyph transform. An empty name marks the plain (no-op)
// variant, whose font is registered as "<base>-<fill>" with no decoration
// suffix. fn transforms one glyph (its rows) into the decorated rows; it must
// add a fixed number of rows so every glyph in a font keeps a uniform height.
type genDecorator struct {
	name string
	fn   func(rows []string, fill rune) []string
}

// generatedBases lists the base glyph shapes the variants are built from.
func generatedBases() []genBase {
	return []genBase{
		{"block", builtinGlyphs},
		{"small", smallGlyphs},
		{"banner", bannerGlyphs},
	}
}

// generatedFills lists the ink runes used to draw each variant.
func generatedFills() []genFill {
	return []genFill{
		{"hash", '#'},
		{"block", '█'},
		{"dark", '▓'},
		{"medium", '▒'},
		{"light", '░'},
		{"dot", '●'},
		{"ring", '○'},
		{"star", '★'},
		{"starlit", '☆'},
		{"asterisk", '*'},
		{"plus", '+'},
		{"equals", '='},
		{"tilde", '~'},
		{"dash", '-'},
		{"middot", '·'},
		{"tick", '▪'},
		{"diamond", '◆'},
		{"diamondlit", '◇'},
		{"at", '@'},
		{"percent", '%'},
		{"amp", '&'},
		{"caret", '^'},
		{"triangle", '▲'},
		{"square", '■'},
		{"pipe", '|'},
		{"ex", 'X'},
	}
}

// generatedDecorators lists the glyph transforms applied to each base+fill.
func generatedDecorators() []genDecorator {
	return []genDecorator{
		{"", decPlain},
		{"bold", decBold},
		{"wide", decWide},
		{"tall", decTall},
		{"shadow", decShadow},
		{"box", decBox},
		{"underline", decUnderline},
		{"overline", decOverline},
		{"frame", decFrame},
		{"invert", decInvert},
		{"outline", decOutline},
		{"mirror", decMirror},
		{"flip", decFlip},
	}
}

// init builds every base × fill × decoration combination and registers it under
// its deterministic name. With 3 bases, 26 fills and 13 decorations this adds
// 1,014 uniquely-named fonts to the registry.
func init() {
	for _, base := range generatedBases() {
		norm := normalizeGlyphs(base.glyphs)
		for _, fill := range generatedFills() {
			filled := inkReplace(norm, fill.r)
			for _, dec := range generatedDecorators() {
				name := base.name + "-" + fill.name
				if dec.name != "" {
					name += "-" + dec.name
				}
				if _, exists := GetFont(name); exists {
					continue // never overwrite a hand-authored or earlier font
				}
				Register(name, buildGeneratedFont(filled, fill.r, dec.fn))
			}
		}
	}
}

// buildGeneratedFont applies dec to every glyph of filled and returns a *Font.
// The font uses full-width layout (no smushing) so the transformed glyphs render
// predictably, and a nil hardblank because normalizeGlyphs already removed the
// bundled fonts' '$' gutters.
func buildGeneratedFont(filled map[rune][]string, fill rune, dec func([]string, rune) []string) *Font {
	chars := make(map[rune][]string, len(filled))
	height := 0
	for r, rows := range filled {
		out := dec(rows, fill)
		chars[r] = out
		height = len(out) // uniform across glyphs by construction
	}
	return &Font{
		hardblank: 0,
		height:    height,
		baseline:  height,
		oldLayout: -1,
		chars:     chars,
	}
}

// normalizeGlyphs copies glyphs, replacing the '$' hardblank gutter with a plain
// space. Combined with full-width layout this reproduces the original spacing
// without needing hardblank handling.
func normalizeGlyphs(glyphs map[rune][]string) map[rune][]string {
	out := make(map[rune][]string, len(glyphs))
	for r, rows := range glyphs {
		cp := make([]string, len(rows))
		for i, line := range rows {
			cp[i] = strings.ReplaceAll(line, "$", " ")
		}
		out[r] = cp
	}
	return out
}

// inkReplace copies glyphs, replacing every inked (non-space) cell with fill.
func inkReplace(glyphs map[rune][]string, fill rune) map[rune][]string {
	out := make(map[rune][]string, len(glyphs))
	for r, rows := range glyphs {
		cp := make([]string, len(rows))
		for i, line := range rows {
			runes := []rune(line)
			for j, c := range runes {
				if c != ' ' {
					runes[j] = fill
				}
			}
			cp[i] = string(runes)
		}
		out[r] = cp
	}
	return out
}

// glyphWidth returns the widest row of a glyph in runes.
func glyphWidth(rows []string) int {
	w := 0
	for _, row := range rows {
		if n := len([]rune(row)); n > w {
			w = n
		}
	}
	return w
}

// padRow right-pads row with spaces to w runes.
func padRow(row string, w int) string {
	if n := len([]rune(row)); n < w {
		return row + strings.Repeat(" ", w-n)
	}
	return row
}

// decPlain returns the glyph unchanged (the "<base>-<fill>" variant).
func decPlain(rows []string, _ rune) []string {
	out := make([]string, len(rows))
	copy(out, rows)
	return out
}

// decBold thickens ink by overlaying a copy shifted one column to the right.
func decBold(rows []string, _ rune) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		src := []rune(row)
		res := make([]rune, len(src)+1)
		for j := range res {
			res[j] = ' '
		}
		for j, c := range src {
			if c != ' ' {
				res[j] = c
				res[j+1] = c
			}
		}
		out[i] = string(res)
	}
	return out
}

// decWide doubles every column, stretching the glyph horizontally.
func decWide(rows []string, _ rune) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		var b strings.Builder
		for _, c := range row {
			b.WriteRune(c)
			b.WriteRune(c)
		}
		out[i] = b.String()
	}
	return out
}

// decTall doubles every row, stretching the glyph vertically (height ×2).
func decTall(rows []string, _ rune) []string {
	out := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		out = append(out, row, row)
	}
	return out
}

// decShadow overlays an offset duplicate down-and-right for a drop shadow
// (height +1).
func decShadow(rows []string, _ rune) []string {
	w := glyphWidth(rows)
	h := len(rows)
	grid := make([][]rune, h+1)
	for i := range grid {
		grid[i] = make([]rune, w+1)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	for i, row := range rows {
		for j, c := range []rune(row) {
			if c != ' ' {
				grid[i+1][j+1] = c
			}
		}
	}
	for i, row := range rows {
		for j, c := range []rune(row) {
			if c != ' ' {
				grid[i][j] = c
			}
		}
	}
	out := make([]string, h+1)
	for i := range grid {
		out[i] = string(grid[i])
	}
	return out
}

// decBox wraps the glyph in a +--+ border (height +2).
func decBox(rows []string, _ rune) []string {
	w := glyphWidth(rows)
	border := "+" + strings.Repeat("-", w) + "+"
	out := make([]string, 0, len(rows)+2)
	out = append(out, border)
	for _, row := range rows {
		out = append(out, "|"+padRow(row, w)+"|")
	}
	out = append(out, border)
	return out
}

// decUnderline adds a fill rule beneath the glyph (height +1).
func decUnderline(rows []string, fill rune) []string {
	w := glyphWidth(rows)
	out := make([]string, 0, len(rows)+1)
	out = append(out, rows...)
	out = append(out, strings.Repeat(string(fill), w))
	return out
}

// decOverline adds a fill rule above the glyph (height +1).
func decOverline(rows []string, fill rune) []string {
	w := glyphWidth(rows)
	out := make([]string, 0, len(rows)+1)
	out = append(out, strings.Repeat(string(fill), w))
	out = append(out, rows...)
	return out
}

// decFrame brackets the glyph with '=' rules top and bottom (height +2).
func decFrame(rows []string, _ rune) []string {
	w := glyphWidth(rows)
	rule := strings.Repeat("=", w)
	out := make([]string, 0, len(rows)+2)
	out = append(out, rule)
	out = append(out, rows...)
	out = append(out, rule)
	return out
}

// decInvert swaps ink and background within the glyph's bounding box, producing
// a negative.
func decInvert(rows []string, fill rune) []string {
	w := glyphWidth(rows)
	out := make([]string, len(rows))
	for i, row := range rows {
		runes := []rune(padRow(row, w))
		for j, c := range runes {
			if c == ' ' {
				runes[j] = fill
			} else {
				runes[j] = ' '
			}
		}
		out[i] = string(runes)
	}
	return out
}

// decOutline hollows solid shapes, keeping only ink cells on an edge (adjacent
// to background or the glyph border).
func decOutline(rows []string, _ rune) []string {
	w := glyphWidth(rows)
	h := len(rows)
	grid := make([][]rune, h)
	for i, row := range rows {
		grid[i] = []rune(padRow(row, w))
	}
	isInk := func(i, j int) bool {
		if i < 0 || i >= h || j < 0 || j >= w {
			return false
		}
		return grid[i][j] != ' '
	}
	out := make([]string, h)
	for i := 0; i < h; i++ {
		res := make([]rune, w)
		for j := 0; j < w; j++ {
			switch {
			case grid[i][j] == ' ':
				res[j] = ' '
			case isInk(i-1, j) && isInk(i+1, j) && isInk(i, j-1) && isInk(i, j+1):
				res[j] = ' ' // fully surrounded interior cell
			default:
				res[j] = grid[i][j]
			}
		}
		out[i] = string(res)
	}
	return out
}

// decMirror reverses each row, flipping the glyph left-to-right.
func decMirror(rows []string, _ rune) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		runes := []rune(row)
		for a, b := 0, len(runes)-1; a < b; a, b = a+1, b-1 {
			runes[a], runes[b] = runes[b], runes[a]
		}
		out[i] = string(runes)
	}
	return out
}

// decFlip reverses row order, flipping the glyph top-to-bottom.
func decFlip(rows []string, _ rune) []string {
	h := len(rows)
	out := make([]string, h)
	for i, row := range rows {
		out[h-1-i] = row
	}
	return out
}
