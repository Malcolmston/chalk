package figlet

import (
	"strings"
	"unicode"
)

// Render renders text using the font.
func (f *Font) Render(text string, opts ...Options) string {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	layout, rules := f.resolveLayout(o.Layout)

	// Render each line of input separately.
	var blocks []string
	for _, line := range strings.Split(text, "\n") {
		blocks = append(blocks, f.renderLine(line, layout, rules))
	}
	return strings.Join(blocks, "\n")
}

// renderLine renders a single (newline-free) line.
func (f *Font) renderLine(text string, layout Layout, rules int) string {
	out := make([]string, f.height)
	first := true
	for _, ch := range text {
		glyph := f.glyphFor(ch)
		if glyph == nil {
			continue
		}
		glyph = padGlyph(glyph)
		if first {
			copy(out, glyph)
			first = false
			continue
		}
		out = f.merge(out, glyph, layout, rules)
	}
	// Replace hardblanks with spaces for display.
	hb := string(f.hardblank)
	for i := range out {
		out[i] = strings.ReplaceAll(out[i], hb, " ")
	}
	return strings.Join(out, "\n")
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
		for i := range out {
			out[i] += glyph[i]
		}
		return out
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
