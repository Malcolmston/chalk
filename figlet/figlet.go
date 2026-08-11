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
// leaves them separate, kerning slides them together until they touch, and
// smushing overlaps their touching edges and fuses the overlapping cells using
// the font's "smushing rules". [LayoutDefault] honors whatever the font's header
// specifies. Fonts may use a "hardblank" character that occupies space during
// layout but prints as a blank, which is how figlet keeps letters from fusing
// into an unreadable blob; hardblanks are replaced with spaces in the final
// output.
//
// Important semantics and edge cases: input is split on newlines and each line
// is rendered as its own block, so multi-line input produces stacked banners.
// Characters the font does not define fall back to the uppercase form (so fonts
// that only define capitals still render mixed-case text) and finally to a
// space, meaning unknown runes never abort a render. [ParseFont] validates the
// flf2a signature and header and returns an error for a malformed font, but it
// tolerates a truncated glyph table by rendering whatever characters it managed
// to read. Output is plain text containing no ANSI codes unless you pass it
// through one of the color helpers.
//
// Parity with the original tools is partial and deliberate. The .flf format,
// the four layout modes and the standard horizontal smushing rules (equal
// character, underscore, hierarchy, opposite pair, big-X and hardblank) are all
// implemented, so real-world fonts render faithfully. What is intentionally left
// out is vertical smushing, right-to-left print direction, control files (.flc)
// and character-code remapping. In exchange this port bundles its own fonts and
// programmatically generates roughly a thousand named variants (see
// fonts_generated.go) so useful output is available with zero external font
// files.
package figlet

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Layout selects how adjacent characters are combined horizontally.
type Layout int

const (
	// LayoutDefault uses the font's own layout settings.
	LayoutDefault Layout = iota
	// LayoutFull places characters at full width (no overlap).
	LayoutFull
	// LayoutKerning moves characters together until they touch.
	LayoutKerning
	// LayoutSmush overlaps and smushes adjacent characters.
	LayoutSmush
)

// Font is a parsed FIGfont.
type Font struct {
	hardblank  rune
	height     int
	baseline   int
	maxLen     int
	oldLayout  int
	fullLayout int
	hasFull    bool
	chars      map[rune][]string
	comment    string
}

// Height returns the number of rows in a rendered line.
func (f *Font) Height() int { return f.height }

// Options configures a render.
type Options struct {
	// Layout selects how adjacent characters are combined horizontally;
	// LayoutDefault uses the font's own settings.
	Layout Layout
	// Width, when > 0, wraps output to this many columns.
	Width int
}

// ParseFont reads a FIGfont from r.
func ParseFont(r io.Reader) (*Font, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	if !sc.Scan() {
		return nil, fmt.Errorf("figlet: empty font")
	}
	header := sc.Text()
	f, commentLines, err := parseHeader(header)
	if err != nil {
		return nil, err
	}

	var comment strings.Builder
	for i := 0; i < commentLines; i++ {
		if !sc.Scan() {
			return nil, fmt.Errorf("figlet: truncated comment block")
		}
		comment.WriteString(sc.Text())
		comment.WriteByte('\n')
	}
	f.comment = comment.String()

	f.chars = make(map[rune][]string)
	// Required characters: ASCII 32..126. A well-formed font defines them all;
	// we tolerate a short font by stopping when the input is exhausted.
	for c := rune(32); c <= 126; c++ {
		glyph, err := readGlyph(sc, f.height)
		if err != nil {
			break
		}
		f.chars[c] = glyph
	}
	// Optional additional code-tagged characters.
	for {
		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		code, err := parseCharCode(line)
		if err != nil {
			// Not a code tag; stop.
			break
		}
		glyph, err := readGlyph(sc, f.height)
		if err != nil {
			break
		}
		f.chars[code] = glyph
	}
	return f, sc.Err()
}

// MaxFontHeight is the largest character height ParseFont accepts. Real FIGfonts
// are at most a few dozen rows tall; the limit exists because the height is read
// from the font file and then used to size allocations, so an absurd declared
// height in a hostile or corrupt file would otherwise try to allocate gigabytes
// before the parser ever discovered the file was short.
const MaxFontHeight = 1000

func parseHeader(header string) (*Font, int, error) {
	if !strings.HasPrefix(header, "flf2a") {
		return nil, 0, fmt.Errorf("figlet: not a FIGfont (bad signature)")
	}
	rest := header[len("flf2a"):]
	if rest == "" {
		return nil, 0, fmt.Errorf("figlet: malformed header: missing hardblank")
	}
	// The hardblank is the single character following the signature. Decoding it
	// as a rune (rather than taking one byte) keeps fonts that use a non-ASCII
	// hardblank working: taking rest[0] split a multi-byte character in half and
	// left its tail in the numeric-field list.
	hardblank, hbSize := utf8.DecodeRuneInString(rest)
	if hardblank == utf8.RuneError && hbSize <= 1 {
		return nil, 0, fmt.Errorf("figlet: malformed header: invalid hardblank")
	}
	fields := strings.Fields(rest[hbSize:])
	if len(fields) < 5 {
		return nil, 0, fmt.Errorf("figlet: incomplete header: got %d numeric fields, want at least 5", len(fields))
	}
	f := &Font{hardblank: hardblank}
	nums := make([]int, len(fields))
	for i, s := range fields {
		// Strict parsing: fmt.Sscanf("%d") accepts trailing garbage ("12abc"
		// scans as 12) and, with its error ignored, silently produced 0 for a
		// non-numeric field. strconv.Atoi rejects both.
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, 0, fmt.Errorf("figlet: malformed header field %d: %q is not a number", i+1, s)
		}
		nums[i] = n
	}
	f.height = nums[0]
	f.baseline = nums[1]
	f.maxLen = nums[2]
	f.oldLayout = nums[3]
	commentLines := nums[4]
	// nums[5] printDirection, nums[6] fullLayout, nums[7] codeTagCount (optional)
	if len(fields) >= 7 {
		f.fullLayout = nums[6]
		f.hasFull = true
	}
	if f.height <= 0 {
		return nil, 0, fmt.Errorf("figlet: invalid height %d: must be positive", f.height)
	}
	if f.height > MaxFontHeight {
		return nil, 0, fmt.Errorf("figlet: font height %d exceeds the maximum of %d", f.height, MaxFontHeight)
	}
	if commentLines < 0 {
		return nil, 0, fmt.Errorf("figlet: invalid comment line count %d", commentLines)
	}
	return f, commentLines, nil
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
	// Negative codes are legal in the FIGfont spec (they tag glyphs that are not
	// reachable as ordinary characters), so they are kept as-is; they simply
	// never match a rune during rendering. Codes above the Unicode maximum are
	// rejected because rune() would turn them into a different, valid-looking
	// code point.
	if code > unicode.MaxRune {
		return 0, fmt.Errorf("figlet: character code %d out of range", code)
	}
	return rune(code), nil
}

// readGlyph reads height sub-lines, stripping the trailing end-mark characters.
func readGlyph(sc *bufio.Scanner, height int) ([]string, error) {
	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		if !sc.Scan() {
			return nil, fmt.Errorf("unexpected EOF")
		}
		line := sc.Text()
		rows = append(rows, stripEndmark(line))
	}
	return rows, nil
}

// stripEndmark removes the trailing end-mark run (the last visible char repeated
// once or twice at the end of the line).
func stripEndmark(line string) string {
	if line == "" {
		return line
	}
	r := []rune(line)
	mark := r[len(r)-1]
	i := len(r)
	for i > 0 && r[i-1] == mark {
		i--
	}
	return string(r[:i])
}
