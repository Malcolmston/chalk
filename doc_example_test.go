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
