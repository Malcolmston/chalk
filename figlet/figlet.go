// Package figlet renders text as ASCII-art banners using FIGfont, a Go port of
// the classic figlet / Node figlet library. It ships a built-in font and can
// load any standard .flf FIGfont.
//
//	fmt.Println(figlet.Render("Hi!"))                    // built-in font
//	f, _ := figlet.LoadFontFile("slant.flf")
//	fmt.Println(f.Render("Hello"))
package figlet

import (
	"bufio"
	"fmt"
	"io"
	"strings"
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

func parseHeader(header string) (*Font, int, error) {
	if !strings.HasPrefix(header, "flf2a") {
		return nil, 0, fmt.Errorf("figlet: not a FIGfont (bad signature)")
	}
	rest := header[len("flf2a"):]
	if rest == "" {
		return nil, 0, fmt.Errorf("figlet: malformed header")
	}
	hardblank := rune(rest[0])
	fields := strings.Fields(rest[1:])
	if len(fields) < 5 {
		return nil, 0, fmt.Errorf("figlet: incomplete header")
	}
	f := &Font{hardblank: hardblank}
	nums := make([]int, len(fields))
	for i, s := range fields {
		fmt.Sscanf(s, "%d", &nums[i])
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
		return nil, 0, fmt.Errorf("figlet: invalid height")
	}
	return f, commentLines, nil
}

func parseCharCode(line string) (rune, error) {
	// A code tag begins with the code (decimal, hex 0x, or octal 0) then a space.
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty")
	}
	var code int
	tok := fields[0]
	switch {
	case strings.HasPrefix(tok, "0x") || strings.HasPrefix(tok, "0X"):
		if _, err := fmt.Sscanf(tok, "0x%x", &code); err != nil {
			return 0, err
		}
	case strings.HasPrefix(tok, "-0x"):
		if _, err := fmt.Sscanf(tok, "-0x%x", &code); err != nil {
			return 0, err
		}
		code = -code
	default:
		if _, err := fmt.Sscanf(tok, "%d", &code); err != nil {
			return 0, err
		}
		if tok != fmt.Sprintf("%d", code) {
			return 0, fmt.Errorf("not a code tag")
		}
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
