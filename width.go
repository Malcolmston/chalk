package chalk

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// This file covers the "how wide is this styled string?" side of chalk, the job
// the Node ecosystem splits across strip-ansi, string-width and ansi-regex.
// Getting it right matters whenever colored text is laid out in columns: escape
// sequences occupy bytes but no screen cells, while a single CJK ideograph or
// emoji occupies two.

// Strip removes ANSI escape sequences from s, returning the plain text a
// terminal would display.
//
// It removes complete escape sequences rather than only SGR color codes: CSI
// sequences (ESC [ … final byte, which covers colors as well as cursor movement
// and erase commands), OSC sequences terminated by BEL or ST (used for window
// titles and hyperlinks), and the short two-character escapes. Anything that is
// not part of a recognized sequence is preserved verbatim, including a trailing
// lone ESC.
func Strip(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			i++
			continue
		}
		if n := escapeLen(s[i:]); n > 0 {
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// escapeLen returns the byte length of the ANSI escape sequence at the start of
// s, or 0 when s does not begin with a complete, recognized sequence.
func escapeLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[': // CSI: parameter bytes, intermediate bytes, then a final byte.
		i := 2
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3f {
			i++
		}
		if i < len(s) && s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1
		}
		return 0
	case ']': // OSC: terminated by BEL or by ST (ESC \).
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return 0
	case 'P', 'X', '^', '_': // DCS / SOS / PM / APC: terminated by ST.
		for i := 2; i < len(s); i++ {
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
			if s[i] == 0x07 {
				return i + 1
			}
		}
		return 0
	default:
		// Two-character escape such as ESC c (reset) or ESC ( B (charset).
		if s[1] >= 0x20 && s[1] <= 0x7e {
			return 2
		}
		return 0
	}
}

// VisibleLength returns the number of visible characters in s, ignoring ANSI
// escape codes. It counts runes, not bytes and not screen columns: a CJK
// ideograph counts as one even though it occupies two terminal cells. Use
// [VisibleWidth] when laying out columns.
func VisibleLength(s string) int { return utf8.RuneCountInString(Strip(s)) }

// VisibleWidth returns the number of terminal cells s occupies, ignoring ANSI
// escape codes. Wide characters (CJK ideographs, Hangul syllables, most emoji
// and fullwidth forms) count as two cells; combining marks, variation selectors
// and other zero-width characters count as none; and an emoji joined with
// zero-width joiners counts once for the whole sequence, so "👨‍👩‍👧" is 2 rather
// than 6.
//
// This is the measurement to use when padding or aligning colored output. Tabs
// and other control characters have no well-defined width and are counted as 0.
func VisibleWidth(s string) int {
	width := 0
	prevZWJ := false
	last := 0 // cells contributed by the previous non-joiner rune
	for _, r := range Strip(s) {
		if r == zeroWidthJoiner {
			prevZWJ = true
			continue
		}
		if prevZWJ {
			// The joined component merges into the preceding glyph.
			prevZWJ = false
			continue
		}
		if r == variationSelector16 && last == 1 {
			// U+FE0F requests emoji presentation, which terminals draw two cells
			// wide even for a base character that is narrow on its own ("☂" is
			// one cell, "☂️" is two). This matches npm string-width.
			width++
			last = 2
			continue
		}
		last = RuneWidth(r)
		width += last
	}
	return width
}

const (
	zeroWidthJoiner     = 0x200d
	variationSelector16 = 0xfe0f
)

// RuneWidth returns the number of terminal cells r occupies: 0 for combining
// marks, control characters and other zero-width runes, 2 for East Asian wide
// and fullwidth characters and for emoji, and 1 for everything else.
func RuneWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		// C0 and C1 control characters occupy no column of their own.
		return 0
	case r < 0x7f:
		return 1
	}
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	if inRanges(r, zeroWidthRanges) {
		return 0
	}
	if inRanges(r, wideRanges) {
		return 2
	}
	return 1
}

// rrange is an inclusive rune range.
type rrange struct{ lo, hi rune }

// inRanges reports whether r falls inside any of the (sorted, non-overlapping)
// ranges, using a binary search.
func inRanges(r rune, ranges []rrange) bool {
	lo, hi := 0, len(ranges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch {
		case r < ranges[mid].lo:
			hi = mid - 1
		case r > ranges[mid].hi:
			lo = mid + 1
		default:
			return true
		}
	}
	return false
}

// zeroWidthRanges are runes that occupy no column but are not classified as
// Mn/Me/Cf by the unicode tables.
var zeroWidthRanges = []rrange{
	{0x200b, 0x200f}, // zero-width space through RTL mark
	{0x2028, 0x202e}, // line/paragraph separators and bidi overrides
	{0x2060, 0x2064}, // word joiner and invisible operators
	{0xfeff, 0xfeff}, // zero-width no-break space (BOM)
}

// wideRanges are the East Asian Wide and Fullwidth blocks plus the emoji blocks
// that terminals render two cells wide. Derived from Unicode's
// EastAsianWidth.txt (W and F classes) collapsed to whole blocks.
var wideRanges = []rrange{
	{0x1100, 0x115f},   // Hangul Jamo initial consonants
	{0x231a, 0x231b},   // watch, hourglass
	{0x2329, 0x232a},   // angle brackets
	{0x23e9, 0x23ec},   // media control emoji
	{0x25fd, 0x25fe},   // small squares (emoji presentation)
	{0x2614, 0x2615},   // umbrella, hot beverage
	{0x2648, 0x2653},   // zodiac
	{0x267f, 0x267f},   // wheelchair
	{0x2693, 0x2693},   // anchor
	{0x26a1, 0x26a1},   // high voltage
	{0x26aa, 0x26ab},   // circles
	{0x26bd, 0x26be},   // soccer, baseball
	{0x26c4, 0x26c5},   // snowman, sun behind cloud
	{0x26ce, 0x26ce},   // ophiuchus
	{0x26d4, 0x26d4},   // no entry
	{0x26ea, 0x26ea},   // church
	{0x26f2, 0x26f3},   // fountain, golf
	{0x26f5, 0x26f5},   // sailboat
	{0x26fa, 0x26fa},   // tent
	{0x26fd, 0x26fd},   // fuel pump
	{0x2705, 0x2705},   // check mark button
	{0x270a, 0x270b},   // raised fist, raised hand
	{0x2728, 0x2728},   // sparkles
	{0x274c, 0x274c},   // cross mark
	{0x274e, 0x274e},   // cross mark button
	{0x2753, 0x2755},   // question/exclamation marks
	{0x2757, 0x2757},   // exclamation mark
	{0x2795, 0x2797},   // plus, minus, divide
	{0x27b0, 0x27b0},   // curly loop
	{0x27bf, 0x27bf},   // double curly loop
	{0x2b1b, 0x2b1c},   // large squares
	{0x2b50, 0x2b50},   // star
	{0x2b55, 0x2b55},   // hollow red circle
	{0x2e80, 0x303e},   // CJK radicals, Kangxi, CJK symbols
	{0x3041, 0x33ff},   // kana through CJK compatibility
	{0x3400, 0x4dbf},   // CJK extension A
	{0x4e00, 0x9fff},   // CJK unified ideographs
	{0xa000, 0xa4cf},   // Yi
	{0xa960, 0xa97f},   // Hangul Jamo extended-A
	{0xac00, 0xd7a3},   // Hangul syllables
	{0xf900, 0xfaff},   // CJK compatibility ideographs
	{0xfe10, 0xfe19},   // vertical forms
	{0xfe30, 0xfe6f},   // CJK compatibility forms, small form variants
	{0xff00, 0xff60},   // fullwidth forms
	{0xffe0, 0xffe6},   // fullwidth signs
	{0x16fe0, 0x16fe4}, // Tangut/Nushu marks
	{0x17000, 0x18cd5}, // Tangut, Khitan small script
	{0x1b000, 0x1b2ff}, // kana supplements
	{0x1f004, 0x1f004}, // mahjong red dragon
	{0x1f0cf, 0x1f0cf}, // joker
	{0x1f18e, 0x1f18e}, // AB blood type
	{0x1f191, 0x1f19a}, // squared symbols
	{0x1f200, 0x1f320}, // enclosed ideographic supplement, weather emoji
	{0x1f32d, 0x1f335}, // food emoji
	{0x1f337, 0x1f37c}, // plants and drinks
	{0x1f37e, 0x1f393}, // celebration through graduation cap
	{0x1f3a0, 0x1f3ca}, // activities
	{0x1f3cf, 0x1f3d3}, // sports equipment
	{0x1f3e0, 0x1f3f0}, // buildings
	{0x1f3f4, 0x1f3f4}, // waving black flag
	{0x1f3f8, 0x1f43e}, // sports and animals
	{0x1f440, 0x1f440}, // eyes
	{0x1f442, 0x1f4fc}, // people, objects
	{0x1f4ff, 0x1f53d}, // more objects and symbols
	{0x1f54b, 0x1f54e}, // religious buildings
	{0x1f550, 0x1f567}, // clock faces
	{0x1f57a, 0x1f57a}, // man dancing
	{0x1f595, 0x1f596}, // hand gestures
	{0x1f5a4, 0x1f5a4}, // black heart
	{0x1f5fb, 0x1f64f}, // emoticons
	{0x1f680, 0x1f6c5}, // transport
	{0x1f6cc, 0x1f6cc}, // person in bed
	{0x1f6d0, 0x1f6d2}, // place of worship, shopping cart
	{0x1f6d5, 0x1f6d7}, // hindu temple, hut
	{0x1f6eb, 0x1f6ec}, // airplane departure/arrival
	{0x1f6f4, 0x1f6fc}, // scooters, skates
	{0x1f7e0, 0x1f7eb}, // colored circles and squares
	{0x1f90c, 0x1f93a}, // more emoticons and gestures
	{0x1f93c, 0x1f945}, // sports
	{0x1f947, 0x1f9ff}, // medals through supplemental symbols
	{0x1fa70, 0x1faff}, // symbols and pictographs extended-A
	{0x20000, 0x2fffd}, // CJK extension B and beyond
	{0x30000, 0x3fffd}, // CJK extension G and beyond
}
