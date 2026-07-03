package figlet

import (
	"io"
	"os"
	"strings"
	"sync"
)

// builtin is the default 5-row block font, covering space, A–Z, 0–9 and common
// punctuation. Lowercase input falls back to uppercase glyphs. Glyphs use '#'
// for filled cells.
var builtinGlyphs = map[rune][]string{
	' ':  {"   ", "   ", "   ", "   ", "   "},
	'A':  {" ### ", "#   #", "#####", "#   #", "#   #"},
	'B':  {"#### ", "#   #", "#### ", "#   #", "#### "},
	'C':  {" ####", "#    ", "#    ", "#    ", " ####"},
	'D':  {"#### ", "#   #", "#   #", "#   #", "#### "},
	'E':  {"#####", "#    ", "#### ", "#    ", "#####"},
	'F':  {"#####", "#    ", "#### ", "#    ", "#    "},
	'G':  {" ####", "#    ", "#  ##", "#   #", " ####"},
	'H':  {"#   #", "#   #", "#####", "#   #", "#   #"},
	'I':  {"###", " # ", " # ", " # ", "###"},
	'J':  {"  ###", "   # ", "   # ", "#  # ", " ##  "},
	'K':  {"#   #", "#  # ", "###  ", "#  # ", "#   #"},
	'L':  {"#    ", "#    ", "#    ", "#    ", "#####"},
	'M':  {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'N':  {"#   #", "##  #", "# # #", "#  ##", "#   #"},
	'O':  {" ### ", "#   #", "#   #", "#   #", " ### "},
	'P':  {"#### ", "#   #", "#### ", "#    ", "#    "},
	'Q':  {" ### ", "#   #", "#   #", "#  ##", " ####"},
	'R':  {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'S':  {" ####", "#    ", " ### ", "    #", "#### "},
	'T':  {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'U':  {"#   #", "#   #", "#   #", "#   #", " ### "},
	'V':  {"#   #", "#   #", "#   #", " # # ", "  #  "},
	'W':  {"#   #", "#   #", "# # #", "## ##", "#   #"},
	'X':  {"#   #", " # # ", "  #  ", " # # ", "#   #"},
	'Y':  {"#   #", " # # ", "  #  ", "  #  ", "  #  "},
	'Z':  {"#####", "   # ", "  #  ", " #   ", "#####"},
	'0':  {" ### ", "#  ##", "# # #", "##  #", " ### "},
	'1':  {"  #  ", " ##  ", "  #  ", "  #  ", " ### "},
	'2':  {" ### ", "#   #", "  ## ", " #   ", "#####"},
	'3':  {"#### ", "    #", " ### ", "    #", "#### "},
	'4':  {"#  # ", "#  # ", "#####", "   # ", "   # "},
	'5':  {"#####", "#    ", "#### ", "    #", "#### "},
	'6':  {" ####", "#    ", "#### ", "#   #", " ### "},
	'7':  {"#####", "   # ", "  #  ", " #   ", " #   "},
	'8':  {" ### ", "#   #", " ### ", "#   #", " ### "},
	'9':  {" ### ", "#   #", " ####", "    #", " ### "},
	'.':  {"  ", "  ", "  ", "##", "##"},
	',':  {"  ", "  ", "  ", "##", " #"},
	'!':  {"#", "#", "#", " ", "#"},
	'?':  {"### ", "   #", " ## ", "    ", " #  "},
	'-':  {"    ", "    ", "####", "    ", "    "},
	'_':  {"     ", "     ", "     ", "     ", "#####"},
	':':  {"  ", "##", "  ", "##", "  "},
	'+':  {"     ", "  #  ", "#####", "  #  ", "     "},
	'=':  {"     ", "#####", "     ", "#####", "     "},
	'*':  {"     ", "# # #", " ### ", "# # #", "     "},
	'/':  {"    #", "   # ", "  #  ", " #   ", "#    "},
	'(':  {" #", "# ", "# ", "# ", " #"},
	')':  {"# ", " #", " #", " #", "# "},
	'#':  {" # # ", "#####", " # # ", "#####", " # # "},
	'@':  {" ### ", "#   #", "# ###", "#    ", " ####"},
	'\'': {"#", "#", " ", " ", " "},
}

var (
	builtinOnce sync.Once
	builtin     *Font
)

// BuiltinFont returns the built-in block font.
func BuiltinFont() *Font {
	builtinOnce.Do(func() {
		// Give each glyph a one-column right margin and render at full width so
		// letters are cleanly separated.
		chars := make(map[rune][]string, len(builtinGlyphs))
		for r, rows := range builtinGlyphs {
			padded := make([]string, len(rows))
			for i, line := range rows {
				padded[i] = line + " "
			}
			chars[r] = padded
		}
		builtin = &Font{
			hardblank: 0,
			height:    5,
			baseline:  5,
			oldLayout: -1, // full width
			chars:     chars,
		}
	})
	return builtin
}

// Render renders text with the built-in font.
func Render(text string, opts ...Options) string {
	return BuiltinFont().Render(text, opts...)
}

// LoadFont parses a FIGfont from a reader.
func LoadFont(r io.Reader) (*Font, error) {
	return ParseFont(r)
}

// LoadFontFile parses a FIGfont from a .flf file on disk.
func LoadFontFile(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseFont(strings.NewReader(string(data)))
}
