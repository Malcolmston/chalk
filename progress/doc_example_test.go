package progress_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/malcolmston/chalk/progress"
)

// ExampleNew renders a determinate bar without a terminal. The Config points Out
// at a discarded buffer and supplies a fixed clock through Now, which is what
// makes the derived numbers — the rate and the ETA — reproducible: with 4 of 8
// units done after two seconds the bar is halfway, moving at two units a second,
// so two seconds remain. Render returns the line as plain text with no escape
// sequences, so it is the value to assert on in a test; the live program would
// simply let Add draw.
func ExampleNew() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsed := time.Duration(0) // New records the start time, so advance after it
	bar := progress.New(progress.Config{
		Total: 8,
		Width: 8,
		Out:   &bytes.Buffer{},
		Now:   func() time.Time { return now.Add(elapsed) },
	})
	elapsed = 2 * time.Second
	bar.Set(4)
	fmt.Println(bar.Render())
	// Output: [████░░░░]  50% 4/8 2.0/s eta 2s
}

// ExampleBar_Write shows the byte-oriented mode. Bytes makes the counters and the
// rate use IEC units, so 1536 bytes reads as "1.5 KiB" rather than as a raw
// number, and a Bar is itself an io.Writer, so wrapping a copy in
// io.MultiWriter(dst, bar) measures a download with no extra bookkeeping. Here
// the writes come from strings.NewReader through io.Copy in the same way.
func ExampleBar_Write() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsed := time.Duration(0)
	bar := progress.New(progress.Config{
		Total:    4096,
		Width:    8,
		Bytes:    true,
		Out:      &bytes.Buffer{},
		Template: "{bar} {current} of {total} at {rate}",
		Now:      func() time.Time { return now.Add(elapsed) },
	})
	elapsed = 2 * time.Second
	fmt.Fprint(bar, strings.Repeat("x", 1536))
	fmt.Println(bar.Render())
	// Output: [███░░░░░] 1.5 KiB of 4.0 KiB at 768 B/s
}

// ExampleConfig_Mode demonstrates the non-terminal path, which is how this
// package behaves in a CI log or a pipe by default. ModePlain (the automatic
// choice whenever Out is not a terminal) emits whole lines terminated by "\n"
// and never a cursor-movement escape, and it throttles: with the clock frozen the
// throttle never opens, so only the first frame and the final frame written by
// Finish appear. In a real run the default PlainInterval of five seconds keeps a
// long job to a few dozen lines.
func ExampleConfig_Mode() {
	var out bytes.Buffer
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bar := progress.New(progress.Config{
		Total:    3,
		Out:      &out,
		Mode:     progress.ModePlain,
		Template: "step {current}/{total}",
		Now:      func() time.Time { return now },
	})
	for i := 0; i < 3; i++ {
		bar.Add(1)
	}
	bar.Finish()
	fmt.Printf("%q\n", out.String())
	// Output: "step 1/3\nstep 3/3\n"
}

// ExampleNewMulti shows several bars sharing one writer. A group is required for
// that: two independent bars both rewriting "the current line" would overwrite
// each other, so the MultiBar owns the cursor and repaints all of its bars as one
// block. Render returns the block as plain text, one line per bar, which keeps a
// group as testable as a single bar.
func ExampleNewMulti() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	group := progress.NewMulti(progress.MultiConfig{
		Out: &bytes.Buffer{},
		Now: func() time.Time { return now },
	})
	cfg := progress.Config{Total: 4, Width: 4, Template: "{desc}{bar} {percent}"}
	cfg.Description = "fetch"
	fetch := group.New(cfg)
	cfg.Description = "build"
	build := group.New(cfg)

	fetch.Set(4)
	build.Set(1)
	fmt.Println(group.Render())
	// Output:
	// fetch [████] 100%
	// build [█░░░]  25%
}

// ExampleBar_Start animates an indeterminate bar: with no Total there is no
// percentage and no ETA, so the bar becomes a block bouncing inside its track and
// the position comes from elapsed time. Start runs the repaint from its own
// goroutine so the bar keeps moving while the caller is busy; Stop joins that
// goroutine, and Finish stops it too. Because the position is a function of the
// injected clock rather than of a redraw counter, the frame below is exact.
func ExampleBar_Start() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// The animation goroutine reads the clock, so the test clock has to be safe
	// for concurrent use: an atomic holds the elapsed nanoseconds.
	var elapsed atomic.Int64
	bar := progress.New(progress.Config{
		Width:    10,
		Out:      &bytes.Buffer{},
		Template: "{spinner} working {bar} {current}",
		Now:      func() time.Time { return now.Add(time.Duration(elapsed.Load())) },
	})
	bar.Start()
	elapsed.Store(int64(500 * time.Millisecond))
	bar.Add(42)
	bar.Stop()
	fmt.Println(bar.Render())
	// Output: / working [░░░░░███░░] 42
}
