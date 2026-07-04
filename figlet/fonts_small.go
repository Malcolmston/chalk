package figlet

// smallGlyphs is a compact, three-row "mini" alphabet. Where the bundled block
// font (fonts.go) draws each character five rows tall out of solid fill, this
// font sketches the same characters in only three rows using line-drawing
// punctuation ( | _ / \ ( ) ), giving a small, low-profile look that suits
// tight spaces, inline labels and single-line banners.
//
// Every glyph is exactly three rows tall (the rows of one glyph need not share
// a width; they are padded when rendered). The set covers space, A-Z, 0-9 and
// common punctuation; lowercase input falls back to the uppercase glyph
// automatically, so only capitals are defined here.
//
// The font registers with kerning layout (0) so characters slide together, and
// each glyph carries a trailing '$' column. '$' is the font hardblank: it reads
// as a space on screen but blocks the kerning overlap, so it acts as a thin,
// consistent gutter that keeps neighbouring characters legibly separated
// instead of fusing into one blob. The space character is drawn from hardblanks
// for the same reason, producing a clear word gap. It is registered under both
// "small" and "mini".
var smallGlyphs = map[rune][]string{
	' ':  {"$$", "$$", "$$"},
	'A':  {" _ $", "|_|$", "| |$"},
	'B':  {"|_ $", "|_)$", "|_)$"},
	'C':  {" _ $", "|  $", "|_ $"},
	'D':  {"|_ $", "| )$", "|_)$"},
	'E':  {" _ $", "|_ $", "|_ $"},
	'F':  {" _ $", "|_ $", "|  $"},
	'G':  {" _ $", "(_ $", "(_|$"},
	'H':  {"| |$", "|_|$", "| |$"},
	'I':  {" _ $", " | $", " _ $"},
	'J':  {" _ $", "  |$", "|_|$"},
	'K':  {"|  $", "|/ $", "|\\ $"},
	'L':  {"|  $", "|  $", "|_ $"},
	'M':  {"|\\/|$", "|  |$", "|  |$"},
	'N':  {"|\\|$", "| |$", "| |$"},
	'O':  {" _ $", "( )$", "\\_/$"},
	'P':  {"|_ $", "|_)$", "|  $"},
	'Q':  {" _ $", "( )$", "\\_\\$"},
	'R':  {"|_ $", "|_)$", "| \\$"},
	'S':  {" _ $", "(_ $", " _)$"},
	'T':  {"___$", " | $", " | $"},
	'U':  {"| |$", "| |$", "|_|$"},
	'V':  {"| |$", "\\ /$", " v $"},
	'W':  {"|  |$", "|  |$", "|\\/|$"},
	'X':  {"\\ /$", " x $", "/ \\$"},
	'Y':  {"\\ /$", " | $", " | $"},
	'Z':  {"___$", " / $", "___$"},
	'0':  {" _ $", "|/|$", "|_|$"},
	'1':  {" /|$", "  |$", "  |$"},
	'2':  {" _ $", " _)$", "(_ $"},
	'3':  {" _ $", " _)$", " _)$"},
	'4':  {"| |$", "|_|$", "  |$"},
	'5':  {" _ $", "|_ $", " _)$"},
	'6':  {" _ $", "|_ $", "|_)$"},
	'7':  {"___$", "  /$", " / $"},
	'8':  {" _ $", "(_)$", "(_)$"},
	'9':  {" _ $", "(_|$", "  |$"},
	'.':  {"  $", "  $", ". $"},
	',':  {"  $", "  $", " ,$"},
	'!':  {"|$", "|$", ".$"},
	'?':  {" _ $", " _)$", " . $"},
	'-':  {"  $", "__$", "  $"},
	'_':  {"  $", "  $", "__$"},
	':':  {"  $", ". $", ". $"},
	'+':  {" | $", "_|_$", " | $"},
	'=':  {"   $", "___$", "___$"},
	'*':  {"\\|/$", " * $", "/|\\$"},
	'/':  {"  /$", " / $", "/  $"},
	'(':  {" /$", "| $", " \\$"},
	')':  {"\\ $", " |$", " /$"},
	'#':  {"# #$", "###$", "# #$"},
	'@':  {" _ $", "(@)$", "\\_/$"},
	'\'': {"|$", " $", " $"},
}

func init() {
	Register("small", FontFromGlyphs(3, 0, smallGlyphs))
	Register("mini", FontFromGlyphs(3, 0, smallGlyphs))
}
