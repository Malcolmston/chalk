package spinner_test

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/malcolmston/chalk/spinner"
)

// Example shows the shape a program uses: build a spinner, start it, update the
// label as the work moves on, and resolve it into a final status line. Out here
// is os.Stdout, which under `go test` is a pipe rather than a terminal, so the
// spinner takes its degraded path automatically — no frames, no cursor escapes,
// just the label once and the resolved line once. That is exactly what a CI log
// gets, and it is why the example has a stable Output block at all. Run the same
// code in a terminal and the two lines become an animated spinner that resolves
// in place. The Symbols override replaces the default colored marks with plain
// text so this example does not depend on the terminal's color support.
func Example() {
	s := spinner.New(spinner.Config{
		Text: "Building",
		Out:  os.Stdout,
		Symbols: map[spinner.State]spinner.Symbol{
			spinner.StateSucceeded: {Text: "OK"},
		},
	})

	s.Start()
	s.UpdateText("Linking") // silent when not animating: no log spam
	s.Succeed("Built in 1.2s")
	// Output:
	// Building
	// OK Built in 1.2s
}

// Example_animated drives the animated path deterministically. Mode: Animate
// forces the terminal rendering even though Out is a bytes.Buffer, and
// ManualTicker replaces the clock, so each Tick advances exactly one frame and
// the test never sleeps. The printed output shows what a terminal would receive:
// a carriage return plus an erase-to-end-of-line escape before every frame, and
// on Succeed one more erase followed by the final line. Note that the last frame
// is erased before the status line is written, which is what keeps following
// output from being glued onto a half-drawn spinner.
func Example_animated() {
	var buf bytes.Buffer
	clock := spinner.NewManualTicker()

	s := spinner.New(spinner.Config{
		Text:       "Working",
		Out:        &buf,
		Mode:       spinner.Animate,
		ShowCursor: true, // keep the cursor escapes out of this example
		Frames:     spinner.Line,
		NewTicker:  clock.New,
		Symbols: map[spinner.State]spinner.Symbol{
			spinner.StateFailed: {Text: "ERR"},
		},
	})

	s.Start()
	clock.Tick()
	clock.Tick()
	s.Fail("Timed out")

	fmt.Printf("%q\n", buf.String())
	// Output:
	// "\r\x1b[K- Working\r\x1b[K\\ Working\r\x1b[K| Working\r\x1b[KERR Timed out\n"
}

// Example_frameSets shows a caller-supplied frame set and interval. The bundled
// sets are spinner.Dots (braille), spinner.Line (pure ASCII) and spinner.Bars
// (block elements); any []string works, and the Interval field of the set is the
// default tick, overridable per spinner with Config.Interval.
func Example_frameSets() {
	s := spinner.New(spinner.Config{
		Text:     "Uploading",
		Out:      os.Stdout,
		Frames:   spinner.FrameSet{Frames: []string{"<", "^", ">", "v"}, Interval: 120 * time.Millisecond},
		Interval: 60 * time.Millisecond,
	})
	fmt.Println(s.Interval(), s.Animating())
	// Output: 60ms false
}
