package chalk_test

import (
	"fmt"

	"github.com/malcolmston/chalk"
)

// ExampleStyle demonstrates the fluent styling API end to end. It pins the
// global color level to LevelBasic with SetLevel so the emitted ANSI escape
// codes are deterministic regardless of the terminal running the test, then
// resets detection afterward so other tests are unaffected. A single Style is
// built by chaining Red and Bold and rendered with Sprint, which wraps the text
// in the corresponding open/close SGR sequences. The result is printed with the
// %q verb so the normally invisible escape bytes show up literally, making clear
// that Red opens code 31 and Bold opens code 1, each closed in reverse order.
// The takeaway is that a chained Style is immutable and produces predictable
// escape sequences once the color level is fixed.
func ExampleStyle() {
	chalk.SetLevel(chalk.LevelBasic)
	defer chalk.ResetDetection()

	styled := chalk.New().Red().Bold().Sprint("error!")
	fmt.Printf("%q\n", styled)
	// Output: "\x1b[31m\x1b[1merror!\x1b[22m\x1b[39m"
}

// ExampleGreen demonstrates the package-level shortcuts for one-off styling. It
// forces the basic color level so the output is deterministic, then colors a
// short string green using the Green shortcut, which is equivalent to
// New().Green().Sprint. It also shows Strip, which removes every ANSI escape
// sequence from a styled string, recovering the original plain text. Printing
// both the quoted styled form and the stripped form side by side highlights the
// difference between the on-the-wire bytes and what the user sees. The takeaway
// is that shortcuts are a concise way to style a single value and that Strip
// reverses the styling for measurement or logging.
func ExampleGreen() {
	chalk.SetLevel(chalk.LevelBasic)
	defer chalk.ResetDetection()

	styled := chalk.Green("ok")
	fmt.Printf("%q\n", styled)
	fmt.Printf("%q\n", chalk.Strip(styled))
	// Output:
	// "\x1b[32mok\x1b[39m"
	// "ok"
}

// ExampleStyle_Level shows that a Style resolves its color codes when it
// renders, not when it is built. The style below is created while color output
// is off — at that moment chalk would have emitted nothing at all — and is then
// rendered twice, once pinned to truecolor and once to the 256-color palette.
// The same value produces a 24-bit "38;2;r;g;b" sequence in the first case and
// the nearest palette index in the second, which is what makes it safe to store
// configured styles in package-level variables and decide the terminal's
// capability later. Style.Level pins one style without touching the global
// setting, so the two renders do not interfere.
func ExampleStyle_Level() {
	chalk.SetLevel(chalk.LevelNone)
	defer chalk.ResetDetection()

	warn := chalk.New().Hex("#ff8800")
	fmt.Printf("%q\n", warn.Level(chalk.LevelTrueColor).Sprint("hot"))
	fmt.Printf("%q\n", warn.Level(chalk.Level256).Sprint("hot"))
	// Output:
	// "\x1b[38;2;255;136;0mhot\x1b[39m"
	// "\x1b[38;5;214mhot\x1b[39m"
}

// ExampleStrip shows that Strip removes whole ANSI escape sequences, not just
// the color codes chalk itself writes. The input below mixes an SGR color pair
// with a cursor-movement sequence and an OSC hyperlink of the kind terminals use
// to make text clickable; all three vanish and only the characters a user would
// see remain. This is the function to reach for before measuring, logging or
// comparing styled output.
func ExampleStrip() {
	styled := "\x1b[31mred\x1b[39m \x1b[2Cshifted \x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\"
	fmt.Printf("%q\n", chalk.Strip(styled))
	// Output: "red shifted link"
}
