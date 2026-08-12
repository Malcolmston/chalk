package image

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malcolmston/chalk"
)

// syncWriter is a buffer that is safe to read while the player goroutine writes,
// so the race detector has something honest to check.
type syncWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// palettedFrame builds one GIF frame of a solid color over the whole canvas.
func palettedFrame(w, h int, c color.RGBA) *image.Paletted {
	pal := color.Palette{color.RGBA{0, 0, 0, 0}, c}
	f := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	for i := range f.Pix {
		f.Pix[i] = 1
	}
	return f
}

// twoFrameGIF is red then green, 10ms per frame.
func twoFrameGIF() *gif.GIF {
	return &gif.GIF{
		Image:    []*image.Paletted{palettedFrame(2, 2, red), palettedFrame(2, 2, green)},
		Delay:    []int{1, 1},
		Disposal: []byte{gif.DisposalNone, gif.DisposalNone},
		Config:   image.Config{Width: 2, Height: 2},
	}
}

func TestNewPlayerFrames(t *testing.T) {
	p, err := NewPlayer(twoFrameGIF(), PlayerConfig{Width: 2, Level: chalk.LevelTrueColor})
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if p.Frames() != 2 {
		t.Fatalf("Frames = %d, want 2", p.Frames())
	}
	if cols, rows := p.Size(); cols != 2 || rows != 1 {
		t.Fatalf("Size = %dx%d, want 2x1", cols, rows)
	}
	want := []string{
		"\x1b[38;2;255;0;0;48;2;255;0;0m▀▀\x1b[0m\n",
		"\x1b[38;2;0;255;0;48;2;0;255;0m▀▀\x1b[0m\n",
	}
	for i, w := range want {
		if got := p.Frame(i); got != w {
			t.Errorf("Frame(%d) = %q, want %q", i, got, w)
		}
	}
}

func TestNewPlayerErrors(t *testing.T) {
	if _, err := NewPlayer(nil, PlayerConfig{}); err != ErrEmptyImage {
		t.Fatalf("nil GIF: err = %v, want ErrEmptyImage", err)
	}
	if _, err := NewPlayer(&gif.GIF{}, PlayerConfig{}); err != ErrEmptyImage {
		t.Fatalf("frameless GIF: err = %v, want ErrEmptyImage", err)
	}
}

// A GIF with no logical screen size still plays: the canvas is the union of the
// frames.
func TestNewPlayerWithoutLogicalScreen(t *testing.T) {
	g := twoFrameGIF()
	g.Config = image.Config{}
	p, err := NewPlayer(g, PlayerConfig{Width: 2, Level: chalk.LevelTrueColor})
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if cols, rows := p.Size(); cols != 2 || rows != 1 {
		t.Fatalf("Size = %dx%d, want 2x1", cols, rows)
	}
}

// Frames are composited: a partial second frame must leave the first frame's
// pixels visible where it does not paint, and DisposalBackground must clear them.
func TestNewPlayerComposition(t *testing.T) {
	half := image.NewPaletted(image.Rect(0, 0, 1, 2), color.Palette{color.RGBA{0, 0, 0, 0}, green})
	for i := range half.Pix {
		half.Pix[i] = 1
	}
	tests := []struct {
		name      string
		disposal  []byte
		wantThird string
	}{
		{
			"partial frame keeps the background",
			[]byte{gif.DisposalNone, gif.DisposalNone, gif.DisposalNone},
			"\x1b[38;2;0;255;0;48;2;0;255;0m▀\x1b[38;2;255;0;0;48;2;255;0;0m▀\x1b[0m\n",
		},
		{
			// The partial frame is erased after it is shown, so frame three is the
			// still-undisturbed red frame with a transparent (black) left column.
			"disposal to background clears it",
			[]byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalNone},
			"\x1b[38;2;0;0;0;48;2;0;0;0m▀\x1b[38;2;255;0;0;48;2;255;0;0m▀\x1b[0m\n",
		},
		{
			"disposal to previous restores it",
			[]byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalNone},
			"\x1b[38;2;255;0;0;48;2;255;0;0m▀▀\x1b[0m\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Frame three paints nothing at all, so it shows the composited canvas.
			empty := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.RGBA{0, 0, 0, 0}})
			g := &gif.GIF{
				Image:    []*image.Paletted{palettedFrame(2, 2, red), half, empty},
				Delay:    []int{1, 1, 1},
				Disposal: tc.disposal,
				Config:   image.Config{Width: 2, Height: 2},
			}
			p, err := NewPlayer(g, PlayerConfig{Width: 2, Level: chalk.LevelTrueColor})
			if err != nil {
				t.Fatalf("NewPlayer: %v", err)
			}
			if got := p.Frame(2); got != tc.wantThird {
				t.Fatalf("Frame(2) = %q, want %q", got, tc.wantThird)
			}
		})
	}
}

// The degraded path: no terminal means no cursor movement and no animation.
func TestPlayerNonTerminal(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		want  int // number of frames expected in the output
	}{
		{"unforced writes nothing", false, 0},
		{"forced writes one still frame", true, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := &syncWriter{}
			p, err := NewPlayer(twoFrameGIF(), PlayerConfig{
				Out: out, Width: 2, Level: chalk.LevelTrueColor, Force: tc.force, Loops: 3,
			})
			if err != nil {
				t.Fatalf("NewPlayer: %v", err)
			}
			if p.Animates() {
				t.Fatal("Animates = true for a non-terminal Out")
			}
			p.Start()
			p.Wait()
			got := out.String()
			if n := strings.Count(got, "▀▀\x1b[0m\n"); n != tc.want {
				t.Fatalf("output %q has %d frames, want %d", got, n, tc.want)
			}
			if strings.Contains(got, "\x1b[1A") || strings.Contains(got, "A\r") {
				t.Fatalf("cursor movement leaked into a non-terminal: %q", got)
			}
			p.Stop()
		})
	}
}

// Start/Stop must be safe and leak-free from several goroutines at once.
func TestPlayerConcurrentStartStop(t *testing.T) {
	base := runtime.NumGoroutine()
	out := &syncWriter{}
	p, err := NewPlayer(twoFrameGIF(), PlayerConfig{
		Out: out, Width: 2, Level: chalk.LevelTrueColor, Force: true, Loops: 0,
	})
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				p.Start()
				return
			}
			time.Sleep(time.Millisecond)
			p.Stop()
		}(i)
	}
	wg.Wait()
	p.Stop() // idempotent, and waits for the goroutine
	// Reading the buffer after Stop must be race-free: the goroutine is gone.
	_ = out.String()
	p.Start() // after Stop, a restart is a no-op rather than a second goroutine
	p.Wait()
	if got := runtime.NumGoroutine(); got > base {
		t.Fatalf("goroutines = %d after Stop, started from %d: the player leaked one", got, base)
	}
}

// A bounded animation on a fake terminal finishes on its own, repaints in place,
// and leaves the last frame drawn.
func TestPlayerAnimatesInPlace(t *testing.T) {
	p, err := NewPlayer(twoFrameGIF(), PlayerConfig{Width: 2, Level: chalk.LevelTrueColor})
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	out := &syncWriter{}
	// Pretend the output is a terminal; isTerminal cannot be true for a buffer, so
	// the animate flag is set directly, which is exactly what a TTY would do.
	p.out = out
	p.animate = true
	p.loops = 2
	p.Start()
	p.Wait()

	got := out.String()
	if n := strings.Count(got, "\x1b[1A\r"); n != 3 {
		t.Fatalf("output %q repositions %d times, want 3 (4 frames drawn)", got, n)
	}
	if n := strings.Count(got, "▀▀\x1b[0m\n"); n != 4 {
		t.Fatalf("output %q has %d frames, want 4", got, n)
	}
	if !strings.HasSuffix(got, "\x1b[38;2;0;255;0;48;2;0;255;0m▀▀\x1b[0m\n") {
		t.Fatalf("output %q should end on the last frame", got)
	}
}

// A frame with no declared delay must not spin: it gets the default interval.
func TestPlayerDefaultDelay(t *testing.T) {
	g := twoFrameGIF()
	g.Delay = []int{0, 0}
	p, err := NewPlayer(g, PlayerConfig{Width: 2, Level: chalk.LevelTrueColor})
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	for i, d := range p.delays {
		if d != defaultFrameDelay {
			t.Fatalf("delay %d = %v, want %v", i, d, defaultFrameDelay)
		}
	}
}
