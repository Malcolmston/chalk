// Package image renders images in a terminal, using the sibling chalk package's
// color-support detection to pick the best encoding the terminal can actually
// display.
//
//	img, _ := image.Open("logo.png")
//	_ = image.Fprint(img, image.Config{Width: 40})
//
// Two layers are provided. The universal layer downscales the picture to a grid
// of character cells and paints it with colored half blocks (U+2580, "upper half
// block"), which gives two vertical pixels per cell by coloring the glyph's
// foreground and its background separately; this works on every color terminal.
// It degrades on its own to the 256-color palette, then to the 16 ANSI colors,
// and finally — with no color at all — to a greyscale ASCII ramp, following
// [chalk.GetLevel] rather than a second detection scheme of its own. The
// optional native layer emits the image itself through a terminal graphics
// protocol ([ProtocolITerm2] or [ProtocolKitty]) for pixel-accurate output.
//
// Sizing is in cells, and a cell is roughly twice as tall as it is wide. With
// two half-block pixels stacked per cell the sampled pixels are square again, so
// a picture of aspect ratio w/h needs rows = cols*h/(2*w) to keep its shape. Set
// [Config.Width] to fit a width, [Config.Height] to fit a height, or both to fit
// inside that box; with neither, the terminal's width (or 80 columns when that
// cannot be measured) is used.
//
// Nothing is written when the output is not a terminal. A screenful of color
// escapes in a CI log or a redirected file is noise at best and garbled at
// worst, so [Fprint] returns without writing unless [Config.Force] says
// otherwise — and even when forced, a non-terminal output never receives a
// native graphics protocol, because a wrong guess there dumps raw base64 across
// the screen. Cursor movement is likewise only ever emitted to a real terminal:
// the still renderers use newlines only, and [Player], which animates a GIF in
// place, refuses to animate anywhere else.
//
// Formats are whatever the standard library decodes: PNG, JPEG and GIF are
// registered by this package. WebP, AVIF and friends are not supported, since
// decoding them would mean a third-party dependency.
package image

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"

	// Registered for the side effect of adding decoders to image.Decode.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/malcolmston/chalk"
	"golang.org/x/term"
)

// ErrEmptyImage is returned when an image has no pixels to draw.
var ErrEmptyImage = errors.New("image: empty image")

// DefaultWidth is the cell width used when no width is requested and the
// terminal size cannot be measured (a pipe, a file, a CI log).
const DefaultWidth = 80

// MaxCells caps the number of character cells a single render may produce. The
// cell count comes from caller-supplied numbers and sizes allocations, so an
// absurd Width would otherwise be an out-of-memory bug rather than a bad
// picture.
const MaxCells = 1 << 20

// Protocol selects how an image is encoded for the terminal.
type Protocol int

const (
	// ProtocolAuto picks a native protocol when the output is a terminal that is
	// known to support one, and the universal ANSI renderer otherwise.
	ProtocolAuto Protocol = iota
	// ProtocolANSI draws with colored half-block characters. Works everywhere.
	ProtocolANSI
	// ProtocolITerm2 emits an inline image in an OSC 1337 sequence.
	ProtocolITerm2
	// ProtocolKitty emits an image through the Kitty graphics protocol.
	ProtocolKitty
)

// String names the protocol, for tests and error messages.
func (p Protocol) String() string {
	switch p {
	case ProtocolAuto:
		return "auto"
	case ProtocolANSI:
		return "ansi"
	case ProtocolITerm2:
		return "iterm2"
	case ProtocolKitty:
		return "kitty"
	}
	return fmt.Sprintf("Protocol(%d)", int(p))
}

// Config configures a render. The zero value renders with the half-block ANSI
// renderer, at the terminal's width, to os.Stdout.
type Config struct {
	// Out is where [Fprint] writes; defaults to os.Stdout. It is an io.Writer so
	// a test can render into a buffer with no terminal involved.
	Out io.Writer
	// Width is the target width in character cells. Zero means "derive it": from
	// Height when that is set, otherwise from the terminal (or [DefaultWidth]).
	Width int
	// Height is the target height in character cells. Zero means "derive it from
	// Width". When both are set the image is fitted inside that box.
	Height int
	// Level pins the color level instead of using chalk's detected one. Zero
	// ([chalk.LevelNone]) means "use [chalk.GetLevel]"; pass a level explicitly to
	// render color into a buffer in tests. To ask for the ASCII ramp on purpose,
	// use [Config.ASCII].
	Level chalk.Level
	// ASCII forces the no-color greyscale ramp even when the terminal supports
	// color. This is the escape hatch that Level cannot express, because the zero
	// Level already means "auto".
	ASCII bool
	// Protocol selects the encoding; [ProtocolAuto] by default.
	Protocol Protocol
	// Force renders even when Out is not a terminal. Native graphics protocols
	// are still never auto-selected in that case.
	Force bool
	// Background is composited under transparent pixels; black when nil. A
	// terminal cannot report its own background color, so a concrete choice beats
	// guessing.
	Background color.Color
}

// Open decodes an image file. It is a convenience wrapper over image.Decode with
// this package's registered decoders (PNG, JPEG, GIF).
func Open(path string) (image.Image, error) {
	f, err := os.Open(path) // #nosec G304 -- the caller names the file it wants
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("image: decode %s: %w", path, err)
	}
	return img, nil
}

// Decode decodes an image from r, reporting the format name that matched.
func Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, "", fmt.Errorf("image: decode: %w", err)
	}
	return img, format, nil
}

// Render returns the escape sequence for img, without writing anything.
//
// The protocol is resolved first: [ProtocolAuto] consults [DetectProtocol] only
// when cfg.Out is a real terminal, so rendering into a buffer or a pipe always
// yields the universal ANSI form instead of base64 that nothing will decode.
func Render(img image.Image, cfg Config) (string, error) {
	if img == nil {
		return "", ErrEmptyImage
	}
	cols, rows, err := Fit(img, cfg)
	if err != nil {
		return "", err
	}
	switch resolveProtocol(cfg) {
	case ProtocolITerm2:
		return renderITerm2(img, cols, rows)
	case ProtocolKitty:
		return renderKitty(img, cols, rows)
	default:
		g, err := Grid(img, cfg)
		if err != nil {
			return "", err
		}
		return renderANSI(g, resolveLevel(cfg)), nil
	}
}

// Fprint renders img to cfg.Out (os.Stdout by default).
//
// When Out is not a terminal nothing is written and the error is nil: see the
// package documentation for why. Set cfg.Force to write anyway.
func Fprint(img image.Image, cfg Config) error {
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	if !isTerminal(out) && !cfg.Force {
		return nil
	}
	s, err := Render(img, cfg)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, s)
	return err
}

// Fit returns the cell grid size chosen for img under cfg: cols columns by rows
// rows, with the aspect ratio corrected for a cell being about twice as tall as
// it is wide.
func Fit(img image.Image, cfg Config) (cols, rows int, err error) {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw <= 0 || ih <= 0 {
		return 0, 0, ErrEmptyImage
	}

	switch {
	case cfg.Width > 0 && cfg.Height > 0:
		// Fit inside the box: take whichever axis binds first.
		cols, rows = cfg.Width, rowsFor(cfg.Width, iw, ih)
		if rows > cfg.Height {
			cols, rows = colsFor(cfg.Height, iw, ih), cfg.Height
			if cols > cfg.Width {
				cols = cfg.Width
			}
		}
	case cfg.Height > 0:
		cols, rows = colsFor(cfg.Height, iw, ih), cfg.Height
	default:
		w := cfg.Width
		if w <= 0 {
			w = terminalWidth(cfg.Out)
		}
		cols, rows = w, rowsFor(w, iw, ih)
	}

	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols*rows > MaxCells {
		return 0, 0, fmt.Errorf("image: %dx%d cells exceeds MaxCells (%d)", cols, rows, MaxCells)
	}
	return cols, rows, nil
}

// rowsFor is the aspect correction: cols columns of two stacked half-block
// pixels are cols wide and 2*rows tall in square pixels, so rows = cols*ih/(2*iw).
func rowsFor(cols, iw, ih int) int {
	return int(math.Round(float64(cols) * float64(ih) / (2 * float64(iw))))
}

// colsFor is rowsFor solved for cols.
func colsFor(rows, iw, ih int) int {
	return int(math.Round(2 * float64(rows) * float64(iw) / float64(ih)))
}

// resolveLevel picks the color level: an explicit cfg.Level wins, ASCII forces
// none, otherwise chalk's own detection decides so that this package has exactly
// one notion of terminal capability.
func resolveLevel(cfg Config) chalk.Level {
	if cfg.ASCII {
		return chalk.LevelNone
	}
	if cfg.Level != chalk.LevelNone {
		return cfg.Level
	}
	return chalk.GetLevel()
}

// resolveProtocol turns [ProtocolAuto] into a concrete protocol. An explicit
// choice is honored as given — a caller who names Kitty has already decided.
func resolveProtocol(cfg Config) Protocol {
	if cfg.Protocol != ProtocolAuto {
		return cfg.Protocol
	}
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	if !isTerminal(out) {
		return ProtocolANSI
	}
	return DetectProtocol()
}

// isTerminal reports whether w is a character device, i.e. a terminal. Anything
// that is not an *os.File (a bytes.Buffer in a test, a pipe wrapper) is by
// definition not one.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// terminalWidth measures the usable width of w in cells, falling back to
// [DefaultWidth] when w is not a terminal or the query fails.
func terminalWidth(w io.Writer) int {
	if w == nil {
		w = os.Stdout
	}
	f, ok := w.(*os.File)
	if !ok {
		return DefaultWidth
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return DefaultWidth
	}
	cols, _, err := term.GetSize(fd)
	if err != nil || cols <= 0 {
		return DefaultWidth
	}
	return cols
}
