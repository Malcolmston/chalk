// Package figlet renders text as ASCII-art banners using FIGfont, a Go port of
// the classic figlet program and the Node figlet library. It ships a built-in
// block font plus a large registry of bundled variants, and it can load any
// standard .flf FIGfont from a file, directory or reader.
//
//	fmt.Println(figlet.Render("Hi!"))                    // built-in font
//	f, _ := figlet.LoadFontFile("slant.flf")
//	fmt.Println(f.Render("Hello"))
//
// Use figlet to draw large banner text for CLI splash screens, section headers
// in logs, or generated README art. The simplest entry point is [Render], which
// renders a string with the built-in font; [RenderFont] renders with a named
// font from the registry (see [Fonts] and [GetFont]); and a [Font] value
// obtained from [LoadFont], [LoadFontFile] or [ParseFont] can render directly
// with [Font.Render]. The companion helpers [RenderRainbow] and [RenderGradient]
// (and the lower-level [Rainbow] and [Gradient]) colorize a finished banner with
// the sibling chalk package.
//
// A FIGfont describes each printable character as a small block of text rows,
// all the same height. Rendering lays the glyphs for the input left to right and
// combines each pair of adjacent glyphs according to a [Layout]: full width
// leaves them separate, fitting slides them together until they touch, and
// smushing overlaps their touching edges and fuses the overlapping cells using
// the font's "smushing rules". Multi-line input is stacked the same way
// vertically, under Options.VerticalLayout. [LayoutDefault] honors whatever the
// font's header specifies. Fonts may use a "hardblank" character that occupies
// space during layout but prints as a blank, which is how figlet keeps letters
// from fusing into an unreadable blob; hardblanks are replaced with spaces in the
// final output unless Options.ShowHardBlanks asks otherwise.
//
// Important semantics and edge cases: input is split on newlines (CRLF and CR are
// normalised first) and each line is rendered as its own block, then the blocks
// are stacked — which pads the narrower ones, so every row of the result has the
// same width. A character the font does not define contributes nothing at all,
// so unknown runes never abort a render; the capitals-only fonts this package
// bundles itself additionally fall back to the uppercase glyph and then to the
// space glyph. [ParseFont] validates the flf2a signature, the header and the
// completeness of the glyph table, and returns an error rather than half-loading a
// truncated font. Output is plain text containing no ANSI codes unless you pass it
// through one of the color helpers.
//
// Parity with the reference JavaScript implementation (patorjk/figlet.js, npm
// "figlet") is exact for the rendering engine: the .flf format, all five layout
// modes, the six horizontal and five vertical smushing rules, width wrapping
// (with and without Options.WhitespaceBreak) and right-to-left print direction
// are ported line for line and measured by a cross-language harness that renders
// the same fonts through both implementations. What is deliberately left out is
// control files (.flc) and character-code remapping. This port additionally
// bundles its own fonts and programmatically generates roughly a thousand named
// variants (see fonts_generated.go) so useful output is available with zero
// external font files; those fonts are the port's own art and do not match the
// upstream fonts of the same name, which is recorded in API-DEVIATIONS.md.
package figlet

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Layout selects how adjacent characters are combined, horizontally by
// Options.Layout and vertically by Options.VerticalLayout. The names correspond
// one-to-one with the reference implementation's horizontalLayout /
// verticalLayout option strings.
type Layout int

const (
	// LayoutDefault uses the font's own layout settings ("default").
	LayoutDefault Layout = iota
	// LayoutFull places characters at full width, with no overlap ("full").
	LayoutFull
	// LayoutFitted moves characters together until they touch ("fitted").
	LayoutFitted
	// LayoutControlledSmush overlaps characters and fuses the touching cells
	// with all six standard smushing rules enabled ("controlled smushing").
	LayoutControlledSmush
	// LayoutUniversalSmush overlaps characters and lets the later one win
	// ("universal smushing").
	LayoutUniversalSmush
)

const (
	// LayoutKerning is the historical name for [LayoutFitted].
	LayoutKerning = LayoutFitted
	// LayoutSmush is the historical name for [LayoutControlledSmush].
	LayoutSmush = LayoutControlledSmush
)

// Font is a parsed FIGfont.
type Font struct {
	hardblank      rune
	height         int
	baseline       int
	maxLen         int
	oldLayout      int
	fullLayout     int
	hasFull        bool
	printDirection int
	numComments    int
	rules          fittingRules
	// fallback enables the uppercase-then-space glyph fallback used by the
	// capitals-only fonts this package bundles. Fonts parsed from a .flf file
	// leave it off so an undefined character is skipped, as upstream does.
	fallback bool
	chars    map[rune][]string
	comment  string
}

// Height returns the number of rows in a rendered line.
func (f *Font) Height() int { return f.height }

// Metadata reports the FIGfont header fields, matching the object the reference
// implementation's figlet.metadata returns.
type Metadata struct {
	// HardBlank is the character that occupies space during layout but prints
	// as a blank.
	HardBlank rune
	// Height is the number of sub-lines in every glyph.
	Height int
	// Baseline is the row the glyphs sit on.
	Baseline int
	// MaxLength is the declared maximum glyph width.
	MaxLength int
	// OldLayout is the legacy layout field.
	OldLayout int
	// NumCommentLines is the length of the font's comment block.
	NumCommentLines int
	// PrintDirection is 0 for left-to-right and 1 for right-to-left.
	PrintDirection int
	// FullLayout is the extended layout field, or -1 when the header omits it.
	FullLayout int
	// Comment is the font's comment block.
	Comment string
}

// Metadata returns the font's header fields.
func (f *Font) Metadata() Metadata {
	full := -1
	if f.hasFull {
		full = f.fullLayout
	}
	return Metadata{
		HardBlank:       f.hardblank,
		Height:          f.height,
		Baseline:        f.baseline,
		MaxLength:       f.maxLen,
		OldLayout:       f.oldLayout,
		NumCommentLines: f.numComments,
		PrintDirection:  f.printDirection,
		FullLayout:      full,
		Comment:         f.comment,
	}
}

// FittingRules reports the layout modes and smushing-rule switches a font's
// header resolves to, the equivalent of the `fittingRules` object the reference
// implementation attaches to a loaded font. The layout values are 0 full width,
// 1 fitting, 2 universal smushing, 3 controlled smushing.
type FittingRules struct {
	// HorizontalLayout is how adjacent characters are combined.
	HorizontalLayout int
	// HorizontalRules enables horizontal smushing rules 1 to 6 in order.
	HorizontalRules [6]bool
	// VerticalLayout is how stacked lines are combined.
	VerticalLayout int
	// VerticalRules enables vertical smushing rules 1 to 5 in order.
	VerticalRules [5]bool
}

// FittingRules returns the font's resolved layout rules.
func (f *Font) FittingRules() FittingRules {
	return FittingRules{
		HorizontalLayout: f.rules.hLayout,
		HorizontalRules:  f.rules.hRule,
		VerticalLayout:   f.rules.vLayout,
		VerticalRules:    f.rules.vRule,
	}
}

// Options configures a render.
type Options struct {
	// Layout selects how adjacent characters are combined horizontally;
	// LayoutDefault uses the font's own settings.
	Layout Layout
	// VerticalLayout selects how the blocks of a multi-line input are stacked;
	// LayoutDefault uses the font's own settings.
	VerticalLayout Layout
	// Width, when > 0, wraps output to this many columns.
	Width int
	// WhitespaceBreak makes width wrapping break between words rather than
	// mid-character.
	WhitespaceBreak bool
	// ShowHardBlanks leaves the font's hardblank character in the output
	// instead of replacing it with a space.
	ShowHardBlanks bool
	// PrintDirection, when non-nil, overrides the font's own direction: 0 for
	// left-to-right, 1 for right-to-left.
	PrintDirection *int
}

// requiredChars is the glyph table every FIGfont must define, in file order:
// printable ASCII followed by the seven German characters the format mandates
// (Ä Ö Ü ä ö ü ß). They are stored positionally, with no code tag of their own,
// so a parser that stops at 126 both loses them and mis-reads everything after.
var requiredChars = func() []rune {
	out := make([]rune, 0, 102)
	for c := rune(32); c <= 126; c++ {
		out = append(out, c)
	}
	return append(out, 196, 214, 220, 228, 246, 252, 223)
}()

// ParseFont reads a FIGfont from r.
//
// The font must be complete: a header, its comment block, and a full glyph table
// for [requiredChars]. A file with fewer lines than that is rejected rather than
// half-loaded, matching the reference implementation — a truncated font that
// silently renders with missing glyphs is worse than an error, because the caller
// cannot tell the difference.
func ParseFont(r io.Reader) (*Font, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxFontBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxFontBytes {
		return nil, fmt.Errorf("figlet: font exceeds the maximum size of %d bytes", MaxFontBytes)
	}
	data := strings.ReplaceAll(string(raw), "\r\n", "\n")
	data = strings.ReplaceAll(data, "\r", "\n")

	lines := strings.Split(data, "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("figlet: invalid font file: missing header")
	}
	f, err := parseHeader(lines[0])
	if err != nil {
		return nil, err
	}
	lines = lines[1:]

	if len(lines) < f.numComments+f.height*len(requiredChars) {
		return nil, fmt.Errorf("figlet: font is missing data: %d lines, %d comment lines, height %d, %d characters",
			len(lines), f.numComments, f.height, len(requiredChars))
	}
	f.comment = strings.Join(lines[:f.numComments], "\n")
	lines = lines[f.numComments:]

	f.chars = make(map[rune][]string, len(requiredChars)+32)
	for _, c := range requiredChars {
		if len(lines) == 0 {
			break
		}
		f.chars[c], lines = readGlyph(lines, f.height)
	}

	// Optional additional code-tagged characters.
	for len(lines) > 0 {
		tag := lines[0]
		lines = lines[1:]
		if strings.TrimSpace(tag) == "" {
			break
		}
		code, err := parseCharCode(tag)
		if err != nil {
			return nil, err
		}
		f.chars[code], lines = readGlyph(lines, f.height)
	}
	return f, nil
}

// MaxFontBytes is the largest FIGfont ParseFont will read. The bundled fonts are
// a few tens of kilobytes; the cap exists because ParseFont buffers the whole
// file, so an unbounded reader (a network stream, say) could otherwise be used to
// exhaust memory.
const MaxFontBytes = 8 << 20

// MaxFontHeight is the largest character height ParseFont accepts. Real FIGfonts
// are at most a few dozen rows tall; the limit exists because the height is read
// from the font file and then used to size allocations, so an absurd declared
// height in a hostile or corrupt file would otherwise try to allocate gigabytes
// before the parser ever discovered the file was short.
const MaxFontHeight = 1000

// parseHeader reads the "flf2a<hardblank> h b l old cmt [dir full tags]" line.
//
// The signature check is stricter than the reference implementation, which only
// looks at the sixth character of the first field and will therefore happily
// "parse" an arbitrary text file. Rejecting a file that does not announce itself
// as a FIGfont is deliberate; it is listed in the port's API-DEVIATIONS.md.
func parseHeader(header string) (*Font, error) {
	if !strings.HasPrefix(header, "flf2a") {
		return nil, fmt.Errorf("figlet: not a FIGfont (bad signature)")
	}
	rest := header[len("flf2a"):]
	if rest == "" {
		return nil, fmt.Errorf("figlet: malformed header: missing hardblank")
	}
	// The hardblank is the single character following the signature. Decoding it
	// as a rune (rather than taking one byte) keeps fonts that use a non-ASCII
	// hardblank working: taking rest[0] split a multi-byte character in half and
	// left its tail in the numeric-field list.
	hardblank, hbSize := utf8.DecodeRuneInString(rest)
	if hardblank == utf8.RuneError && hbSize <= 1 {
		return nil, fmt.Errorf("figlet: malformed header: invalid hardblank")
	}
	fields := strings.Fields(rest[hbSize:])
	if len(fields) < 5 {
		return nil, fmt.Errorf("figlet: incomplete header: got %d numeric fields, want at least 5", len(fields))
	}
	f := &Font{hardblank: hardblank}
	nums := make([]int, len(fields))
	for i, s := range fields {
		// Strict parsing: fmt.Sscanf("%d") accepts trailing garbage ("12abc"
		// scans as 12) and, with its error ignored, silently produced 0 for a
		// non-numeric field. strconv.Atoi rejects both.
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("figlet: malformed header field %d: %q is not a number", i+1, s)
		}
		nums[i] = n
	}
	f.height = nums[0]
	f.baseline = nums[1]
	f.maxLen = nums[2]
	f.oldLayout = nums[3]
	f.numComments = nums[4]
	if len(fields) >= 6 {
		f.printDirection = nums[5]
	}
	if len(fields) >= 7 {
		f.fullLayout = nums[6]
		f.hasFull = true
	}
	// nums[7] is codeTagCount, which the renderer does not need.
	if f.height < 1 {
		return nil, fmt.Errorf("figlet: invalid height %d: must be positive", f.height)
	}
	if f.height > MaxFontHeight {
		return nil, fmt.Errorf("figlet: font height %d exceeds the maximum of %d", f.height, MaxFontHeight)
	}
	if f.baseline < 0 || f.maxLen < 0 || f.numComments < 0 {
		return nil, fmt.Errorf("figlet: header contains invalid values")
	}
	var full *int
	if f.hasFull {
		full = &f.fullLayout
	}
	f.rules = getSmushingRules(f.oldLayout, full)
	return f, nil
}

// parseCharCode reads the character code that introduces a code-tagged glyph.
// The .flf format writes it in C notation: decimal, hexadecimal with a 0x
// prefix, or octal with a leading 0, optionally negated.
//
// The parse is strict. fmt.Sscanf stops at the first character it cannot use, so
// "0x41zzz" and "12abc" were previously accepted as 0x41 and 12; a garbage line
// in the middle of a font was therefore mistaken for a code tag and the rows
// after it were consumed as a glyph. strconv.ParseInt with an explicit base
// rejects any trailing text. The result is also range-checked, because
// converting an arbitrary integer to a rune yields a meaningless or invalid code
// point (and the value comes straight out of the font file).
func parseCharCode(line string) (rune, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, fmt.Errorf("figlet: empty code tag")
	}
	tok := fields[0]

	neg := false
	digits := tok
	if s, ok := strings.CutPrefix(digits, "-"); ok {
		neg, digits = true, s
	} else if s, ok := strings.CutPrefix(digits, "+"); ok {
		digits = s
	}
	if digits == "" {
		return 0, fmt.Errorf("figlet: %q is not a character code", tok)
	}

	base := 10
	switch {
	case strings.HasPrefix(digits, "0x"), strings.HasPrefix(digits, "0X"):
		base, digits = 16, digits[2:]
	case len(digits) > 1 && digits[0] == '0':
		base, digits = 8, digits[1:]
	}
	code, err := strconv.ParseInt(digits, base, 32)
	if err != nil {
		return 0, fmt.Errorf("figlet: %q is not a character code", tok)
	}
	if neg {
		code = -code
	}
	// The reference implementation rejects -1 outright and bounds codes to the
	// signed 32-bit range. Other negative codes are legal in the FIGfont spec
	// (they tag glyphs that are not reachable as ordinary characters) and are
	// kept as-is; they simply never match a rune during rendering. Codes above
	// the Unicode maximum are rejected because rune() would turn them into a
	// different, valid-looking code point.
	if code == -1 {
		return 0, fmt.Errorf("figlet: the character code -1 is not permitted")
	}
	if code > unicode.MaxRune {
		return 0, fmt.Errorf("figlet: character code %d out of range", code)
	}
	return rune(code), nil
}

// readGlyph takes height sub-lines off the front of lines, stripping each one's
// end mark, and returns the glyph together with the remaining lines. A glyph that
// runs off the end of the file is padded with empty rows, as upstream does.
func readGlyph(lines []string, height int) ([]string, []string) {
	rows := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			rows[i] = removeEndChar(lines[i], i, height)
		} else {
			rows[i] = ""
		}
	}
	if height >= len(lines) {
		return rows, nil
	}
	return rows, lines[height:]
}

// removeEndChar strips a glyph row's end mark: the row's last non-blank character,
// appearing once (twice on the glyph's final row) followed by optional trailing
// whitespace. Some TOIlet fonts put spaces after the end mark, which is why the
// trailing whitespace is part of the match.
//
// This replaces a "strip every repeat of the final character" rule, which ate a
// whole row of "@@@@" and disagreed with the reference implementation on any font
// whose art happens to end in the end-mark character.
func removeEndChar(line string, lineNum, height int) string {
	rs := []rune(line)

	endChar := '@'
	if t := strings.TrimSpace(line); t != "" {
		tr := []rune(t)
		endChar = tr[len(tr)-1]
	}

	i := len(rs)
	for i > 0 && isJSSpace(rs[i-1]) {
		i--
	}
	if i == 0 || rs[i-1] != endChar {
		return line
	}
	i--
	if lineNum == height-1 && i > 0 && rs[i-1] == endChar {
		i--
	}
	return string(rs[:i])
}
