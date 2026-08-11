package figlet

// This file is a faithful port of the rendering engine in the reference
// JavaScript implementation, patorjk/figlet.js (npm "figlet"), as measured by
// the cross-language parity harness in the aggregator repo
// (parity/chalk/nested/figlet). Function names mirror the upstream ones —
// getHorizontalSmushLength, horizontalSmush, generateFigTextLines,
// smushVerticalFigLines — so the two can be diffed side by side.
//
// The engine works on FIG-blocks: a block is `height` rows of runes, one row per
// sub-line of the FIGfont. Rows within a block are deliberately *not* padded to
// a common width; upstream leaves them ragged and the smushing arithmetic
// depends on the real lengths.

import (
	"strings"
	"unicode"
	"unicode/utf16"
)

// Horizontal/vertical layout modes, mirroring upstream's LAYOUT enum.
const (
	layoutFullWidth = iota
	layoutFitting
	layoutSmushing
	layoutControlledSmushing
)

// fittingRules is upstream's `fittingRules` object: the effective layout mode
// plus the six horizontal and five vertical smushing-rule switches. hRule[0] is
// rule 1 (code value 1), hRule[5] is rule 6 (code value 32); vRule[0] is
// vertical rule 1 (code value 256), vRule[4] is rule 5 (code value 4096).
type fittingRules struct {
	hLayout int
	hRule   [6]bool
	vLayout int
	vRule   [5]bool
}

func (r fittingRules) anyHRule() bool {
	for _, b := range r.hRule {
		if b {
			return true
		}
	}
	return false
}

func (r fittingRules) anyVRule() bool {
	for _, b := range r.vRule {
		if b {
			return true
		}
	}
	return false
}

// getSmushingRules decodes the FIGfont header's layout fields into the rule set,
// a direct port of upstream's function of the same name. fullLayout is nil when
// the header omits the extended field, in which case oldLayout decides.
func getSmushingRules(oldLayout int, fullLayout *int) fittingRules {
	var r fittingRules
	var hSet, vSet bool

	val := oldLayout
	if fullLayout != nil {
		val = *fullLayout
	}

	// The codes are consumed from the most significant downwards, each one
	// subtracted from the running value, exactly as upstream does. A layout code
	// only wins if no larger layout code already claimed the slot.
	if val >= 16384 {
		val -= 16384
		r.vLayout, vSet = layoutSmushing, true
	}
	if val >= 8192 {
		val -= 8192
		if !vSet {
			r.vLayout, vSet = layoutFitting, true
		}
	}
	for i, code := range []int{4096, 2048, 1024, 512, 256} {
		idx := 4 - i // 4096 is vertical rule 5
		if val >= code {
			val -= code
			r.vRule[idx] = true
		}
	}
	if val >= 128 {
		val -= 128
		r.hLayout, hSet = layoutSmushing, true
	}
	if val >= 64 {
		val -= 64
		if !hSet {
			r.hLayout, hSet = layoutFitting, true
		}
	}
	for i, code := range []int{32, 16, 8, 4, 2, 1} {
		idx := 5 - i // 32 is horizontal rule 6
		if val >= code {
			val -= code
			r.hRule[idx] = true
		}
	}

	switch {
	case !hSet:
		switch {
		case oldLayout == 0:
			r.hLayout = layoutFitting
		case oldLayout == -1:
			r.hLayout = layoutFullWidth
		case r.anyHRule():
			r.hLayout = layoutControlledSmushing
		default:
			r.hLayout = layoutSmushing
		}
	case r.hLayout == layoutSmushing && r.anyHRule():
		r.hLayout = layoutControlledSmushing
	}

	switch {
	case !vSet:
		if r.anyVRule() {
			r.vLayout = layoutControlledSmushing
		} else {
			r.vLayout = layoutFullWidth
		}
	case r.vLayout == layoutSmushing && r.anyVRule():
		r.vLayout = layoutControlledSmushing
	}

	return r
}

// ------------------------------------------------------- horizontal smushing

// hRule1Smush is EQUAL CHARACTER SMUSHING (code value 1): two identical
// sub-characters become one. Hardblanks are excluded (that is rule 6).
func hRule1Smush(ch1, ch2, hardBlank rune) (rune, bool) {
	if ch1 == ch2 && ch1 != hardBlank {
		return ch1, true
	}
	return 0, false
}

// hRule2Smush is UNDERSCORE SMUSHING (code value 2): "_" gives way to any of
// the border characters.
func hRule2Smush(ch1, ch2 rune) (rune, bool) {
	const border = `|/\[]{}()<>`
	if ch1 == '_' && strings.ContainsRune(border, ch2) {
		return ch2, true
	}
	if ch2 == '_' && strings.ContainsRune(border, ch1) {
		return ch1, true
	}
	return 0, false
}

// rule3Classes is upstream's hierarchy string. The *position* in this exact
// string, not a hand-written rank table, decides the outcome: two characters
// smush only when their positions differ by more than one, and the winner is
// the character at the later position.
var rule3Classes = []rune(`| /\ [] {} () <>`)

// hRule3Smush is HIERARCHY SMUSHING (code value 4).
func hRule3Smush(ch1, ch2 rune) (rune, bool) {
	p1 := runeIndex(rule3Classes, ch1)
	p2 := runeIndex(rule3Classes, ch2)
	if p1 == -1 || p2 == -1 {
		return 0, false
	}
	if p1 != p2 && abs(p1-p2) != 1 {
		return rule3Classes[max(p1, p2)], true
	}
	return 0, false
}

// rule4Str is upstream's opposite-pair string; as with rule 3 the comparison is
// positional.
var rule4Str = []rune(`[] {} ()`)

// hRule4Smush is OPPOSITE PAIR SMUSHING (code value 8).
func hRule4Smush(ch1, ch2 rune) (rune, bool) {
	p1 := runeIndex(rule4Str, ch1)
	p2 := runeIndex(rule4Str, ch2)
	if p1 == -1 || p2 == -1 {
		return 0, false
	}
	if abs(p1-p2) <= 1 {
		return '|', true
	}
	return 0, false
}

// hRule5Smush is BIG X SMUSHING (code value 16).
func hRule5Smush(ch1, ch2 rune) (rune, bool) {
	switch {
	case ch1 == '/' && ch2 == '\\':
		return '|', true
	case ch1 == '\\' && ch2 == '/':
		return 'Y', true
	case ch1 == '>' && ch2 == '<':
		return 'X', true
	}
	return 0, false
}

// hRule6Smush is HARDBLANK SMUSHING (code value 32).
func hRule6Smush(ch1, ch2, hardBlank rune) (rune, bool) {
	if ch1 == hardBlank && ch2 == hardBlank {
		return hardBlank, true
	}
	return 0, false
}

// hRuleSmush applies the enabled horizontal rules in order, returning the first
// hit. It mirrors upstream's `||` chain.
func hRuleSmush(ch1, ch2, hardBlank rune, r fittingRules) (rune, bool) {
	if r.hRule[0] {
		if c, ok := hRule1Smush(ch1, ch2, hardBlank); ok {
			return c, true
		}
	}
	if r.hRule[1] {
		if c, ok := hRule2Smush(ch1, ch2); ok {
			return c, true
		}
	}
	if r.hRule[2] {
		if c, ok := hRule3Smush(ch1, ch2); ok {
			return c, true
		}
	}
	if r.hRule[3] {
		if c, ok := hRule4Smush(ch1, ch2); ok {
			return c, true
		}
	}
	if r.hRule[4] {
		if c, ok := hRule5Smush(ch1, ch2); ok {
			return c, true
		}
	}
	if r.hRule[5] {
		if c, ok := hRule6Smush(ch1, ch2, hardBlank); ok {
			return c, true
		}
	}
	return 0, false
}

// uniSmush is UNIVERSAL SMUSHING: the later sub-character wins, except that a
// space or a hardblank never overwrites real ink.
func uniSmush(ch1, ch2, hardBlank rune) rune {
	if ch2 == ' ' {
		return ch1
	}
	if ch2 == hardBlank && ch1 != ' ' {
		return ch1
	}
	return ch2
}

// --------------------------------------------------------- vertical smushing

func vRule1Smush(ch1, ch2 rune) (rune, bool) {
	if ch1 == ch2 {
		return ch1, true
	}
	return 0, false
}

func vRule2Smush(ch1, ch2 rune) (rune, bool) { return hRule2Smush(ch1, ch2) }

func vRule3Smush(ch1, ch2 rune) (rune, bool) { return hRule3Smush(ch1, ch2) }

// vRule4Smush is HORIZONTAL LINE SMUSHING (code value 2048).
func vRule4Smush(ch1, ch2 rune) (rune, bool) {
	if (ch1 == '-' && ch2 == '_') || (ch1 == '_' && ch2 == '-') {
		return '=', true
	}
	return 0, false
}

// vRule5Smush is VERTICAL LINE SUPERSMUSHING (code value 4096).
func vRule5Smush(ch1, ch2 rune) (rune, bool) {
	if ch1 == '|' && ch2 == '|' {
		return '|', true
	}
	return 0, false
}

// smush verdicts returned by canVerticalSmush.
const (
	smushInvalid = "invalid"
	smushValid   = "valid"
	smushEnd     = "end"
)

// canVerticalSmush reports whether two sub-lines may be stacked: "valid" (yes,
// and more may follow), "end" (yes, but stop here) or "invalid".
func canVerticalSmush(txt1, txt2 []rune, r fittingRules) string {
	if r.vLayout == layoutFullWidth {
		return smushInvalid
	}
	n := min(len(txt1), len(txt2))
	if n == 0 {
		return smushInvalid
	}
	endSmush := false
	for ii := 0; ii < n; ii++ {
		ch1, ch2 := txt1[ii], txt2[ii]
		if ch1 == ' ' || ch2 == ' ' {
			continue
		}
		switch r.vLayout {
		case layoutFitting:
			return smushInvalid
		case layoutSmushing:
			return smushEnd
		}
		// Upstream checks supersmushing unconditionally here, before the
		// rule switches; keep that.
		if _, ok := vRule5Smush(ch1, ch2); ok {
			continue
		}
		valid := false
		if r.vRule[0] {
			_, valid = vRule1Smush(ch1, ch2)
		}
		if !valid && r.vRule[1] {
			_, valid = vRule2Smush(ch1, ch2)
		}
		if !valid && r.vRule[2] {
			_, valid = vRule3Smush(ch1, ch2)
		}
		if !valid && r.vRule[3] {
			_, valid = vRule4Smush(ch1, ch2)
		}
		endSmush = true
		if !valid {
			return smushInvalid
		}
	}
	if endSmush {
		return smushEnd
	}
	return smushValid
}

// getVerticalSmushDist returns how many rows of the two blocks may overlap.
func getVerticalSmushDist(lines1, lines2 [][]rune, r fittingRules) int {
	maxDist := len(lines1)
	len1 := len(lines1)
	curDist := 1
	for curDist <= maxDist {
		start := max(0, len1-curDist)
		subLines1 := lines1[start:len1]
		n := min(min(maxDist, curDist), len(lines2))
		subLines2 := lines2[:n]
		if len(subLines2) == 0 {
			break
		}
		result := ""
		for ii := 0; ii < len(subLines2); ii++ {
			switch canVerticalSmush(subLines1[ii], subLines2[ii], r) {
			case smushEnd:
				result = smushEnd
			case smushInvalid:
				result = smushInvalid
			default:
				if result == "" {
					result = smushValid
				}
			}
			if result == smushInvalid {
				break
			}
		}
		if result == smushInvalid {
			curDist--
			break
		}
		if result == smushEnd {
			break
		}
		if result == smushValid {
			curDist++
		}
	}
	return min(maxDist, curDist)
}

// verticallySmushLines fuses one pair of stacked sub-lines.
func verticallySmushLines(line1, line2 []rune, r fittingRules) []rune {
	n := min(len(line1), len(line2))
	out := make([]rune, 0, n)
	for ii := 0; ii < n; ii++ {
		ch1, ch2 := line1[ii], line2[ii]
		if ch1 == ' ' || ch2 == ' ' {
			// Upstream passes no hardblank here, so a hardblank is ordinary ink.
			out = append(out, uniSmush(ch1, ch2, 0))
			continue
		}
		if r.vLayout == layoutFitting || r.vLayout == layoutSmushing {
			out = append(out, uniSmush(ch1, ch2, 0))
			continue
		}
		var c rune
		var ok bool
		if r.vRule[4] {
			c, ok = vRule5Smush(ch1, ch2)
		}
		if !ok && r.vRule[0] {
			c, ok = vRule1Smush(ch1, ch2)
		}
		if !ok && r.vRule[1] {
			c, ok = vRule2Smush(ch1, ch2)
		}
		if !ok && r.vRule[2] {
			c, ok = vRule3Smush(ch1, ch2)
		}
		if !ok && r.vRule[3] {
			c, ok = vRule4Smush(ch1, ch2)
		}
		if !ok {
			// Unreachable: getVerticalSmushDist only hands over row pairs it has
			// already validated. Upstream concatenates the boolean `false` here,
			// which would corrupt the row; fall back to universal smushing.
			c = uniSmush(ch1, ch2, 0)
		}
		out = append(out, c)
	}
	return out
}

// verticalSmush stacks two blocks with the given row overlap.
func verticalSmush(lines1, lines2 [][]rune, overlap int, r fittingRules) [][]rune {
	len1, len2 := len(lines1), len(lines2)
	cut := max(0, len1-overlap)
	piece1 := lines1[:cut]
	piece21 := lines1[cut:len1]
	piece22 := lines2[:min(overlap, len2)]

	out := make([][]rune, 0, len1+len2)
	out = append(out, piece1...)
	for ii := range piece21 {
		if ii >= len2 || ii >= len(piece22) {
			out = append(out, piece21[ii])
		} else {
			out = append(out, verticallySmushLines(piece21[ii], piece22[ii], r))
		}
	}
	out = append(out, lines2[min(overlap, len2):]...)
	return out
}

// padLines right-pads every row of a block with n spaces.
func padLines(lines [][]rune, n int) [][]rune {
	pad := make([]rune, n)
	for i := range pad {
		pad[i] = ' '
	}
	out := make([][]rune, len(lines))
	for i, l := range lines {
		row := make([]rune, 0, len(l)+n)
		row = append(row, l...)
		row = append(row, pad...)
		out[i] = row
	}
	return out
}

// smushVerticalFigLines stacks the block `lines` under `output`, padding the
// narrower of the two to a common width first. This padding is why upstream's
// multi-line output has uniformly wide rows.
func smushVerticalFigLines(output, lines [][]rune, r fittingRules) [][]rune {
	if len(output) == 0 || len(lines) == 0 {
		if len(output) == 0 {
			return lines
		}
		return output
	}
	len1 := len(output[0])
	len2 := len(lines[0])
	switch {
	case len1 > len2:
		lines = padLines(lines, len1-len2)
	case len2 > len1:
		output = padLines(output, len2-len1)
	}
	return verticalSmush(output, lines, getVerticalSmushDist(output, lines, r), r)
}

// ------------------------------------------------------- horizontal assembly

// getHorizontalSmushLength returns how many columns two sub-lines may overlap.
func getHorizontalSmushLength(txt1, txt2 []rune, hardBlank rune, r fittingRules) int {
	if r.hLayout == layoutFullWidth {
		return 0
	}
	len1, len2 := len(txt1), len(txt2)
	if len1 == 0 {
		return 0
	}
	maxDist := len1
	curDist := 1
	breakAfter := false

	for curDist <= maxDist {
		seg1 := txt1[len1-curDist:]
		n := min(curDist, len2)
		seg2 := txt2[:n]
		stop := false
		for ii := 0; ii < n; ii++ {
			ch1, ch2 := seg1[ii], seg2[ii]
			if ch1 == ' ' || ch2 == ' ' {
				continue
			}
			switch r.hLayout {
			case layoutFitting:
				curDist--
				stop = true
			case layoutSmushing:
				if ch1 == hardBlank || ch2 == hardBlank {
					curDist--
				}
				stop = true
			default:
				breakAfter = true
				if _, ok := hRuleSmush(ch1, ch2, hardBlank, r); !ok {
					curDist--
					stop = true
				}
			}
			if stop {
				break
			}
		}
		if stop || breakAfter {
			break
		}
		curDist++
	}
	return min(maxDist, curDist)
}

// horizontalSmush lays block2 to the right of block1 with the given overlap.
func horizontalSmush(block1, block2 [][]rune, overlap int, hardBlank rune, r fittingRules, height int) [][]rune {
	out := make([][]rune, height)
	for ii := 0; ii < height; ii++ {
		var txt1, txt2 []rune
		if ii < len(block1) {
			txt1 = block1[ii]
		}
		if ii < len(block2) {
			txt2 = block2[ii]
		}
		len1, len2 := len(txt1), len(txt2)

		row := make([]rune, 0, len1+len2)
		row = append(row, txt1[:max(0, min(len1, len1-overlap))]...)

		seg1 := txt1[max(0, len1-overlap):]
		seg2 := txt2[:min(overlap, len2)]
		for jj := 0; jj < overlap; jj++ {
			ch1, ch2 := ' ', ' '
			if jj < len1 && jj < len(seg1) {
				ch1 = seg1[jj]
			}
			if jj < len2 && jj < len(seg2) {
				ch2 = seg2[jj]
			}
			if ch1 != ' ' && ch2 != ' ' && r.hLayout != layoutFitting && r.hLayout != layoutSmushing {
				if c, ok := hRuleSmush(ch1, ch2, hardBlank, r); ok {
					row = append(row, c)
					continue
				}
			}
			row = append(row, uniSmush(ch1, ch2, hardBlank))
		}
		if overlap < len2 {
			row = append(row, txt2[overlap:]...)
		}
		out[ii] = row
	}
	return out
}

// figWord is one piece queued for horizontal assembly: a block and the overlap
// to use when joining it to whatever came before.
type figWord struct {
	fig     [][]rune
	overlap int
}

// newFigChar is an empty block of the given height.
func newFigChar(height int) [][]rune {
	out := make([][]rune, height)
	for i := range out {
		out[i] = []rune{}
	}
	return out
}

// figLinesWidth is the width of the widest row of a block.
func figLinesWidth(lines [][]rune) int {
	w := 0
	for _, l := range lines {
		if len(l) > w {
			w = len(l)
		}
	}
	return w
}

// joinFigArray folds a list of pieces into one block.
func joinFigArray(items []figWord, height int, hardBlank rune, r fittingRules) [][]rune {
	acc := newFigChar(height)
	for _, it := range items {
		acc = horizontalSmush(acc, it.fig, it.overlap, hardBlank, r, height)
	}
	return acc
}

// breakWord splits an over-long word, returning the part that fits and the
// characters left over.
func breakWord(chars []figWord, height int, hardBlank rune, r fittingRules, width int) ([][]rune, []figWord) {
	for i := len(chars) - 1; i > 0; i-- {
		w := joinFigArray(chars[:i], height, hardBlank, r)
		if figLinesWidth(w) <= width {
			return w, chars[i:]
		}
	}
	if len(chars) > 0 {
		return joinFigArray(chars[:1], height, hardBlank, r), chars[1:]
	}
	return newFigChar(height), chars
}

// renderOpts is the resolved option set for one render, the equivalent of the
// object upstream's _reworkFontOpts builds.
type renderOpts struct {
	rules           fittingRules
	width           int
	whitespaceBreak bool
	showHardBlanks  bool
	printDirection  int
}

// generateFigTextLines renders one newline-free input line into one or more
// blocks (more than one only when width wrapping splits it).
func generateFigTextLines(txt string, f *Font, o renderOpts) [][][]rune {
	height := f.height
	r := o.rules
	hb := f.hardblank

	// Upstream indexes the font by UTF-16 code unit, because it walks the string
	// with substring(i, i+1). Doing the same keeps astral characters (which split
	// into surrogates there) behaving identically.
	units := utf16.Encode([]rune(txt))
	if o.printDirection == 1 {
		for i, j := 0, len(units)-1; i < j; i, j = i+1, j-1 {
			units[i], units[j] = units[j], units[i]
		}
	}

	var outputFigLines [][][]rune
	outputFigText := newFigChar(height)
	overlap := 0
	var nextChars []figWord
	nextOverlap := 0
	var figWords []figWord

	for charIndex := 0; charIndex < len(units); charIndex++ {
		u := units[charIndex]
		isSpace := isJSSpace(rune(u))
		glyph := f.glyphFor(rune(u))
		if glyph == nil {
			continue
		}
		figChar := toBlock(glyph)

		if r.hLayout != layoutFullWidth {
			overlap = 10000
			for row := 0; row < height; row++ {
				var acc, gl []rune
				if row < len(outputFigText) {
					acc = outputFigText[row]
				}
				if row < len(figChar) {
					gl = figChar[row]
				}
				if n := getHorizontalSmushLength(acc, gl, hb, r); n < overlap {
					overlap = n
				}
			}
			if overlap == 10000 {
				overlap = 0
			}
		}

		var textFigLine [][]rune
		maxWidth := 0
		if o.width > 0 {
			if o.whitespaceBreak {
				textFigWord := joinFigArray(append(append([]figWord{}, nextChars...), figWord{figChar, overlap}), height, hb, r)
				textFigLine = joinFigArray(append(append([]figWord{}, figWords...), figWord{textFigWord, nextOverlap}), height, hb, r)
			} else {
				textFigLine = horizontalSmush(outputFigText, figChar, overlap, hb, r, height)
			}
			maxWidth = figLinesWidth(textFigLine)

			if maxWidth >= o.width && charIndex > 0 {
				if o.whitespaceBreak {
					outputFigText = joinFigArray(figWords[:max(0, len(figWords)-1)], height, hb, r)
					if len(figWords) > 1 {
						outputFigLines = append(outputFigLines, outputFigText)
						outputFigText = newFigChar(height)
					}
					figWords = nil
				} else {
					outputFigLines = append(outputFigLines, outputFigText)
					outputFigText = newFigChar(height)
				}
			}
		}

		if o.width > 0 && o.whitespaceBreak {
			if !isSpace || charIndex == len(units)-1 {
				nextChars = append(nextChars, figWord{figChar, overlap})
			}
			if isSpace || charIndex == len(units)-1 {
				broke := false
				for {
					textFigLine = joinFigArray(nextChars, height, hb, r)
					maxWidth = figLinesWidth(textFigLine)
					if maxWidth < o.width {
						break
					}
					var fitted [][]rune
					fitted, nextChars = breakWord(nextChars, height, hb, r, o.width)
					broke = true
					outputFigLines = append(outputFigLines, fitted)
				}
				if maxWidth > 0 {
					ov := nextOverlap
					if broke {
						ov = 1
					}
					figWords = append(figWords, figWord{textFigLine, ov})
				}
				if isSpace {
					figWords = append(figWords, figWord{figChar, overlap})
					outputFigText = newFigChar(height)
				}
				if charIndex == len(units)-1 {
					outputFigText = joinFigArray(figWords, height, hb, r)
				}
				nextChars = nil
				nextOverlap = overlap
				continue
			}
		}

		outputFigText = horizontalSmush(outputFigText, figChar, overlap, hb, r, height)
	}

	if figLinesWidth(outputFigText) > 0 {
		outputFigLines = append(outputFigLines, outputFigText)
	}
	if !o.showHardBlanks && hb != 0 {
		for _, block := range outputFigLines {
			for _, row := range block {
				for i, c := range row {
					if c == hb {
						row[i] = ' '
					}
				}
			}
		}
	}
	if txt == "" && len(outputFigLines) == 0 {
		outputFigLines = append(outputFigLines, newFigChar(height))
	}
	return outputFigLines
}

// ------------------------------------------------------------- public render

// Render renders text using the font.
//
// The output matches the reference figlet implementation byte for byte: each
// input line becomes a block of [Font.Height] rows, the blocks are stacked with
// the font's vertical layout (which may fuse their touching rows), and stacking
// pads the narrower blocks so every row of the result is the same width. When
// Options.Width is positive the text is wrapped to that many columns, breaking
// mid-word unless Options.WhitespaceBreak asks for word boundaries.
func (f *Font) Render(text string, opts ...Options) string {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	ro := renderOpts{
		rules:           f.effectiveRules(o),
		width:           o.Width,
		whitespaceBreak: o.WhitespaceBreak,
		showHardBlanks:  o.ShowHardBlanks,
		printDirection:  f.printDirection,
	}
	if o.PrintDirection != nil {
		ro.printDirection = *o.PrintDirection
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var figLines [][][]rune
	for _, line := range strings.Split(text, "\n") {
		figLines = append(figLines, generateFigTextLines(line, f, ro)...)
	}
	if len(figLines) == 0 {
		return ""
	}
	out := figLines[0]
	for _, block := range figLines[1:] {
		out = smushVerticalFigLines(out, block, ro.rules)
	}

	rows := make([]string, len(out))
	for i, row := range out {
		rows[i] = string(row)
	}
	return strings.Join(rows, "\n")
}

// effectiveRules applies the caller's layout overrides on top of the font's own
// rules, mirroring upstream's getHorizontalFittingRules/getVerticalFittingRules.
func (f *Font) effectiveRules(o Options) fittingRules {
	r := f.rules
	switch o.Layout {
	case LayoutFull:
		r.hLayout, r.hRule = layoutFullWidth, [6]bool{}
	case LayoutFitted:
		r.hLayout, r.hRule = layoutFitting, [6]bool{}
	case LayoutControlledSmush:
		r.hLayout, r.hRule = layoutControlledSmushing, [6]bool{true, true, true, true, true, true}
	case LayoutUniversalSmush:
		r.hLayout, r.hRule = layoutSmushing, [6]bool{}
	}
	switch o.VerticalLayout {
	case LayoutFull:
		r.vLayout, r.vRule = layoutFullWidth, [5]bool{}
	case LayoutFitted:
		r.vLayout, r.vRule = layoutFitting, [5]bool{}
	case LayoutControlledSmush:
		r.vLayout, r.vRule = layoutControlledSmushing, [5]bool{true, true, true, true, true}
	case LayoutUniversalSmush:
		r.vLayout, r.vRule = layoutSmushing, [5]bool{}
	}
	return r
}

// glyphFor returns the sub-lines for ch, or nil when the font has no glyph.
//
// A FIGfont parsed from a .flf file behaves exactly like upstream: an undefined
// character contributes nothing at all. The fonts this package bundles itself
// define only capitals, so they additionally fall back to the uppercase form and
// then to the space glyph (see Font.fallback).
func (f *Font) glyphFor(ch rune) []string {
	if g, ok := f.chars[ch]; ok {
		return g
	}
	if !f.fallback {
		return nil
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

// ------------------------------------------------------------------ helpers

// toBlock converts a glyph's rows to rune slices.
func toBlock(rows []string) [][]rune {
	out := make([][]rune, len(rows))
	for i, r := range rows {
		out[i] = []rune(r)
	}
	return out
}

func runeIndex(s []rune, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// isJSSpace reports whether r is whitespace to JavaScript's `\s`, which is the
// test upstream uses to find word boundaries.
func isJSSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r',
		0x00a0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}
