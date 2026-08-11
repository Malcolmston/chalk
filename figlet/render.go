package figlet

import (
	"strings"
	"unicode"
)

// Render renders text using the font.
//
// Each line of text becomes its own block of [Font.Height] rows, and the blocks
// are stacked. When Options.Width is positive the text is additionally wrapped
// at word boundaries so no block is wider than that many columns; a single word
// too wide to fit is emitted on a line of its own rather than being broken.
func (f *Font) Render(text string, opts ...Options) string {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	layout, rules := f.resolveLayout(o.Layout)

	// Render each line of input separately.
	var blocks []string
	for _, line := range strings.Split(text, "\n") {
		if o.Width > 0 {
			blocks = append(blocks, f.wrapLine(line, layout, rules, o.Width)...)
		} else {
			blocks = append(blocks, f.renderLine(line, layout, rules))
		}
	}
	return strings.Join(blocks, "\n")
}

// renderLine renders a single (newline-free) line.
func (f *Font) renderLine(text string, layout Layout, rules int) string {
	return f.finish(f.appendRunes(nil, text, layout, rules))
}

// appendRunes lays the glyphs for text onto rows, which may be nil to start a
// new block, and returns the new rows. Hardblanks are still present in the
// result; call finish to produce displayable text.
func (f *Font) appendRunes(rows []string, text string, layout Layout, rules int) []string {
	for _, ch := range text {
		glyph := f.glyphFor(ch)
		if glyph == nil {
			continue
		}
		glyph = padGlyph(glyph)
		if rows == nil {
			rows = make([]string, f.height)
			copy(rows, glyph)
			continue
		}
		rows = f.merge(rows, glyph, layout, rules)
	}
	return rows
}

// finish turns a block of rows into displayable text, replacing hardblanks with
// spaces. A nil block renders as an empty block of the font's height.
func (f *Font) finish(rows []string) string {
	if rows == nil {
		rows = make([]string, f.height)
	}
	hb := string(f.hardblank)
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = strings.ReplaceAll(row, hb, " ")
	}
	return strings.Join(out, "\n")
}

// blockWidth returns the number of columns a block occupies, ignoring the
// trailing blanks that padding leaves on each row.
func blockWidth(rows []string) int {
	w := 0
	for _, row := range rows {
		if n := len([]rune(strings.TrimRight(row, " "))); n > w {
			w = n
		}
	}
	return w
}

// wrapLine renders one input line as one or more blocks, none wider than width
// columns, breaking between words.
//
// Words are merged into the current block one at a time and the block is
// measured as it grows, so each word is laid out at most twice however long the
// input is — deliberately not "re-render the whole candidate line for every
// word", which is quadratic in the number of words and is exactly the kind of
// loop a caller-supplied width could blow up.
func (f *Font) wrapLine(text string, layout Layout, rules, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{f.renderLine(text, layout, rules)}
	}

	var blocks []string
	var cur []string
	inLine := 0
	for _, w := range words {
		next := f.appendWord(cur, w, layout, rules)
		if inLine > 0 && blockWidth(next) > width {
			blocks = append(blocks, f.finish(cur))
			cur, inLine = f.appendWord(nil, w, layout, rules), 0
		} else {
			cur = next
		}
		inLine++
	}
	return append(blocks, f.finish(cur))
}

// appendWord adds word to a block, separated by a space when the block is not
// empty.
func (f *Font) appendWord(rows []string, word string, layout Layout, rules int) []string {
	if rows != nil {
		rows = f.appendRunes(rows, " ", layout, rules)
	}
	return f.appendRunes(rows, word, layout, rules)
}

// glyphFor returns the glyph for ch, falling back to uppercase (for fonts that
// only define capitals) and then to space.
func (f *Font) glyphFor(ch rune) []string {
	if g, ok := f.chars[ch]; ok {
		return g
	}
	if up := unicode.ToUpper(ch); up != ch {
		if g, ok := f.chars[up]; ok {
			return g
		}
	}
	if g, ok := f.chars[' ']; ok {
		return g
	}
	return nil
}

// resolveLayout determines the effective layout and smushing rule bits.
func (f *Font) resolveLayout(requested Layout) (Layout, int) {
	if requested != LayoutDefault {
		rules := f.oldLayout
		if f.hasFull {
			rules = f.fullLayout & 63
		}
		if rules < 0 {
			rules = 0
		}
		return requested, rules
	}
	if f.hasFull {
		switch {
		case f.fullLayout&128 != 0:
			return LayoutSmush, f.fullLayout & 63
		case f.fullLayout&64 != 0:
			return LayoutKerning, 0
		default:
			return LayoutFull, 0
		}
	}
	switch {
	case f.oldLayout < 0:
		return LayoutFull, 0
	case f.oldLayout == 0:
		return LayoutKerning, 0
	default:
		return LayoutSmush, f.oldLayout & 63
	}
}

// merge appends glyph to out using the layout, returning the new rows.
func (f *Font) merge(out, glyph []string, layout Layout, rules int) []string {
	if layout == LayoutFull {
		// Build a new slice rather than writing back into out: callers keep
		// earlier blocks around (see wrapLine, which holds the last block that
		// fitted while it tries the next word), and mutating in place corrupted
		// them.
		res := make([]string, len(out))
		for i := range out {
			res[i] = out[i] + glyph[i]
		}
		return res
	}

	smush := layout == LayoutSmush
	amount := f.overlap(out, glyph, smush, rules)

	res := make([]string, len(out))
	for i := range out {
		left := []rune(out[i])
		right := []rune(glyph[i])
		keep := len(left) - amount
		if keep < 0 {
			keep = 0
		}
		var b strings.Builder
		b.WriteString(string(left[:keep]))
		for k := 0; k < amount; k++ {
			li := keep + k
			var lc, rc rune = ' ', ' '
			if li >= 0 && li < len(left) {
				lc = left[li]
			}
			if k < len(right) {
				rc = right[k]
			}
			b.WriteRune(f.smushem(lc, rc, smush, rules))
		}
		if amount < len(right) {
			b.WriteString(string(right[amount:]))
		}
		res[i] = b.String()
	}
	return res
}

// overlap computes the number of columns to overlap across all rows.
func (f *Font) overlap(out, glyph []string, smush bool, rules int) int {
	amount := 1 << 30
	for i := range out {
		a := rowOverlap([]rune(out[i]), []rune(glyph[i]), smush, rules, f.hardblank, f)
		if a < amount {
			amount = a
		}
	}
	if amount < 0 {
		amount = 0
	}
	return amount
}

// rowOverlap returns the max columns two rows can overlap.
func rowOverlap(left, right []rune, smush bool, rules int, hb rune, f *Font) int {
	// Trailing blanks in left.
	lt := 0
	for lt < len(left) && left[len(left)-1-lt] == ' ' {
		lt++
	}
	// Leading blanks in right.
	rl := 0
	for rl < len(right) && right[rl] == ' ' {
		rl++
	}
	amt := lt + rl
	if !smush {
		return amt
	}
	// Try to smush one more column where the non-blank edges meet.
	li := len(left) - lt - 1
	ri := rl
	if li >= 0 && ri < len(right) {
		if f.smushem(left[li], right[ri], true, rules) != 0 {
			amt++
		}
	}
	return amt
}

// smushem combines two overlapping characters, returning the smushed rune or 0
// when they cannot be smushed (used during amount computation).
func (f *Font) smushem(a, b rune, smush bool, rules int) rune {
	if a == ' ' {
		return b
	}
	if b == ' ' {
		return a
	}
	hb := f.hardblank

	// Hardblank handling.
	if a == hb || b == hb {
		if smush && rules&32 != 0 && a == hb && b == hb {
			return hb
		}
		return 0
	}

	if !smush {
		return 0
	}

	if rules == 0 {
		// Universal smushing: the later character wins.
		return b
	}
	// Rule 1: equal character.
	if rules&1 != 0 && a == b {
		return a
	}
	// Rule 2: underscore.
	if rules&2 != 0 {
		const border = "|/\\[]{}()<>"
		if a == '_' && strings.ContainsRune(border, b) {
			return b
		}
		if b == '_' && strings.ContainsRune(border, a) {
			return a
		}
	}
	// Rule 4: hierarchy.
	if rules&4 != 0 {
		if ra, rb := rank(a), rank(b); ra > 0 && rb > 0 && ra != rb {
			if ra > rb {
				return a
			}
			return b
		}
	}
	// Rule 8: opposite pair.
	if rules&8 != 0 {
		if isOppositePair(a, b) {
			return '|'
		}
	}
	// Rule 16: big X.
	if rules&16 != 0 {
		switch {
		case a == '/' && b == '\\':
			return '|'
		case a == '\\' && b == '/':
			return 'Y'
		case a == '>' && b == '<':
			return 'X'
		}
	}
	return 0
}

// rank returns the hierarchy class rank of a bracket-like character.
func rank(r rune) int {
	switch r {
	case '|':
		return 1
	case '/', '\\':
		return 2
	case '[', ']':
		return 3
	case '{', '}':
		return 4
	case '(', ')':
		return 5
	case '<', '>':
		return 6
	default:
		return 0
	}
}

func isOppositePair(a, b rune) bool {
	pairs := map[rune]rune{'[': ']', ']': '[', '{': '}', '}': '{', '(': ')', ')': '('}
	return pairs[a] == b
}

// padGlyph pads all rows of a glyph to equal width.
func padGlyph(glyph []string) []string {
	max := 0
	for _, r := range glyph {
		if n := len([]rune(r)); n > max {
			max = n
		}
	}
	out := make([]string, len(glyph))
	for i, r := range glyph {
		out[i] = r + strings.Repeat(" ", max-len([]rune(r)))
	}
	return out
}
