package image

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/malcolmston/chalk"
)

// defaultFrameDelay is used for a GIF frame that declares no delay, matching the
// long-standing browser behaviour of treating 0 as "about a tenth of a second"
// rather than "as fast as possible".
const defaultFrameDelay = 100 * time.Millisecond

// PlayerConfig configures a [Player]. The sizing, color and background fields
// mean exactly what they do in [Config].
type PlayerConfig struct {
	Out        io.Writer
	Width      int
	Height     int
	Level      chalk.Level
	ASCII      bool
	Background color.Color
	// Loops is how many times to play the animation; 0 plays until Stop is
	// called. A GIF's own loop count is not consulted, because the caller of a
	// blocking-ish API wants to decide when it ends.
	Loops int
	// Force draws the first frame even when Out is not a terminal. It never
	// animates there: repainting in place needs cursor-movement escapes, and those
	// in a log file produce a wall of garbage rather than a moving picture.
	Force bool
}

// Player animates a decoded GIF in a terminal by repainting the same rows in
// place. Frames are rendered once, up front, so the animation loop only writes
// bytes; that keeps the frame interval honest and keeps the goroutine cheap.
//
// A Player is safe to use from several goroutines: [Player.Start] and
// [Player.Stop] may be called in any order, from any goroutine, more than once.
// Stop waits for the animation goroutine to finish, so no goroutine outlives it.
type Player struct {
	frames  []string
	delays  []time.Duration
	rows    int
	cols    int
	out     io.Writer
	animate bool
	force   bool
	loops   int

	mu      sync.Mutex
	started bool
	stopped bool
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewPlayer pre-renders every frame of g and returns a Player ready to Start.
//
// GIF frames are partial updates: each one may cover a sub-rectangle of the
// canvas and may ask, through its disposal method, for the area to be cleared or
// restored afterwards. They are composited here onto a persistent canvas so each
// pre-rendered frame is a whole picture.
func NewPlayer(g *gif.GIF, cfg PlayerConfig) (*Player, error) {
	if g == nil || len(g.Image) == 0 {
		return nil, ErrEmptyImage
	}
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}

	canvasRect := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	if canvasRect.Dx() <= 0 || canvasRect.Dy() <= 0 {
		// A GIF with no logical screen size: take the union of its frames.
		canvasRect = g.Image[0].Bounds()
		for _, f := range g.Image[1:] {
			canvasRect = canvasRect.Union(f.Bounds())
		}
	}
	if canvasRect.Dx() <= 0 || canvasRect.Dy() <= 0 {
		return nil, ErrEmptyImage
	}

	canvas := image.NewRGBA(canvasRect)
	sizing := Config{Out: cfg.Out, Width: cfg.Width, Height: cfg.Height}
	cols, rows, err := Fit(canvas, sizing)
	if err != nil {
		return nil, err
	}
	level := resolveLevel(Config{Level: cfg.Level, ASCII: cfg.ASCII})

	p := &Player{
		rows:    rows,
		cols:    cols,
		out:     out,
		animate: isTerminal(out),
		force:   cfg.Force,
		loops:   cfg.Loops,
	}
	if p.loops < 0 {
		p.loops = 0
	}

	for i, frame := range g.Image {
		var snapshot *image.RGBA
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			snapshot = image.NewRGBA(canvas.Rect)
			copy(snapshot.Pix, canvas.Pix)
		}

		r := frame.Bounds().Intersect(canvas.Rect)
		draw.Draw(canvas, r, frame, r.Min, draw.Over)
		p.frames = append(p.frames, renderANSI(gridAt(canvas, cols, rows, cfg.Background), level))

		d := defaultFrameDelay
		if i < len(g.Delay) && g.Delay[i] > 0 {
			d = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}
		p.delays = append(p.delays, d)

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, r, image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, snapshot.Pix)
		}
	}
	return p, nil
}

// Frames reports how many frames were pre-rendered.
func (p *Player) Frames() int { return len(p.frames) }

// Frame returns the pre-rendered escape sequence for frame i, which is the whole
// picture (not the GIF's partial update). It panics for an out-of-range index,
// like any slice.
func (p *Player) Frame(i int) string { return p.frames[i] }

// Size reports the cell grid the animation occupies.
func (p *Player) Size() (cols, rows int) { return p.cols, p.rows }

// Animates reports whether Start will actually animate. It is false when Out is
// not a terminal, where Start draws at most a single frame.
func (p *Player) Animates() bool { return p.animate }

// Start begins playback in a background goroutine and returns immediately.
// Calling it again while playing, or after Stop, does nothing.
func (p *Player) Start() {
	p.mu.Lock()
	if p.started || p.stopped {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.done = make(chan struct{})
	done := p.done
	p.wg.Add(1)
	p.mu.Unlock()

	go p.run(done)
}

// Stop ends playback and waits for the goroutine to exit. It is idempotent and
// safe to call from a different goroutine than Start, including before it.
func (p *Player) Stop() {
	p.mu.Lock()
	if !p.stopped {
		p.stopped = true
		if p.done != nil {
			close(p.done)
		}
	}
	p.mu.Unlock()
	p.wg.Wait()
}

// Wait blocks until playback finishes on its own. With Loops == 0 it only
// returns after Stop, so callers of an endless animation should Stop instead.
func (p *Player) Wait() { p.wg.Wait() }

// run is the animation loop. It repaints by moving the cursor back up over the
// picture before each frame after the first, so the final frame is left on screen
// and the cursor ends below it exactly as a still render would leave it.
func (p *Player) run(done chan struct{}) {
	defer p.wg.Done()

	if !p.animate {
		// Not a terminal: no cursor movement, no animation. One frame if forced,
		// otherwise nothing at all.
		if p.force {
			p.write(p.frames[0])
		}
		return
	}

	up := "\x1b[" + strconv.Itoa(p.rows) + "A\r"
	drawn := false
	for loop := 0; p.loops == 0 || loop < p.loops; loop++ {
		for i, frame := range p.frames {
			if drawn {
				p.write(up)
			}
			p.write(frame)
			drawn = true

			timer := time.NewTimer(p.delays[i])
			select {
			case <-done:
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

func (p *Player) write(s string) { _, _ = io.WriteString(p.out, s) }
