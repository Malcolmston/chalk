package figlet_test

import (
	"fmt"
	"strings"

	"github.com/malcolmston/chalk/figlet"
)

// ExampleRender demonstrates rendering a word as an ASCII-art banner with the
// built-in block font. It calls Render on the two-character string "Hi", which
// lays out the glyph for each letter side by side and joins their rows into a
// five-line block. Because the built-in font is deterministic, the exact art is
// known ahead of time and checked with an Output comment. Each rendered row is
// right-trimmed before printing so that the font's internal padding spaces do
// not appear as trailing whitespace, which keeps the expected output stable. The
// takeaway is that Render turns ordinary text into multi-row banner art using
// the bundled font with no font files or configuration required.
func ExampleRender() {
	banner := figlet.Render("Hi")
	for _, line := range strings.Split(banner, "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	// Output:
	// #   # ###
	// #   #  #
	// #####  #
	// #   #  #
	// #   # ###
}

// ExampleRenderFont demonstrates rendering with a named font drawn from the
// registry rather than the default. It renders the word "Go" with the bundled
// "banner" outline font by calling RenderFont, which looks the font up by
// case-insensitive name and returns an error if the name is unknown. The number
// of output rows equals the font's Height, so the code prints that height to
// show the render succeeded without depending on the exact glyph art, which is
// less stable across fonts. This also illustrates the two-value contract of
// RenderFont: a rendered string plus an error that is nil on success. The
// takeaway is that many named fonts are available through the registry via a
// single call.
func ExampleRenderFont() {
	banner, err := figlet.RenderFont("banner", "Go")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(strings.Count(banner, "\n") + 1)
	// Output: 5
}

// ExampleFont_Render demonstrates Options.Width, which wraps a banner at word
// boundaries so it fits a given number of columns. The phrase below renders 60
// columns wide as a single block, but with Width set to 30 the words are laid
// out as two stacked blocks of the font's height, breaking between words and
// never in the middle of one. A word too wide for the limit is still emitted
// whole on its own line rather than being split or dropped. The takeaway is that
// Width lets a banner adapt to a narrow terminal without the caller having to
// pre-chop the text.
func ExampleFont_Render() {
	f := figlet.BuiltinFont()
	narrow := f.Render("go chalk", figlet.Options{Width: 30})
	fmt.Println(len(strings.Split(narrow, "\n")) / f.Height())

	wide := f.Render("go chalk")
	fmt.Println(len(strings.Split(wide, "\n")) / f.Height())
	// Output:
	// 2
	// 1
}
