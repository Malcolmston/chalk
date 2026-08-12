package image

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/malcolmston/chalk"
)

// halfBlock is U+2580 UPPER HALF BLOCK. Drawn with a foreground and a background
// color it shows two pixels in one cell: the foreground fills the top half, the
// background shows through the bottom half. Using the *upper* half block (rather
// than the lower one, U+2584) keeps the mapping "foreground = top pixel", which
// matches how the grid is stored.
const halfBlock = "▀"

// ramp is the greyscale ladder used with no color available, darkest first. A
// leading space is the darkest step so an unpainted area stays empty.
const ramp = " .:-=+*#%@"

// Cell is one character cell of a rendered image: two vertically stacked pixels,
// already composited over the configured background so both are opaque.
type Cell struct {
	Upper color.RGBA
	Lower color.RGBA
}

// Grid downscales img to the cell grid chosen by [Fit] and returns it row by row
// (rows of cols cells each). It is exported because it is the honest unit to test
// against: the sampled colors are the render, and everything after this is escape
// sequence formatting.
func Grid(img image.Image, cfg Config) ([][]Cell, error) {
	cols, rows, err := Fit(img, cfg)
	if err != nil {
		return nil, err
	}
	return gridAt(img, cols, rows, cfg.Background), nil
}

// gridAt is Grid with the cell size already decided. [Player] needs that: every
// frame of an animation has to land on the same grid, so the size is computed
// once from the canvas and reused rather than re-fitted (and possibly rounded
// differently) per frame.
func gridAt(img image.Image, cols, rows int, background color.Color) [][]Cell {
	// Default background is black: a terminal cannot be asked what its own
	// background is, so the alternative is guessing wrong half the time.
	br, bg, bb := 0.0, 0.0, 0.0
	if background != nil {
		r, g, b, a := background.RGBA()
		// Ignore a fully transparent background color: there is nothing behind it.
		if a > 0 {
			br, bg, bb = float64(r>>8), float64(g>>8), float64(b>>8)
		}
	}

	px := sample(img, cols, rows*2, br, bg, bb)
	grid := make([][]Cell, rows)
	for y := 0; y < rows; y++ {
		row := make([]Cell, cols)
		for x := 0; x < cols; x++ {
			row[x] = Cell{Upper: px[2*y][x], Lower: px[2*y+1][x]}
		}
		grid[y] = row
	}
	return grid
}

// sample box-averages img down to w by h pixels, compositing over the background
// color (br, bg, bb). Averaging the whole source box rather than picking one
// pixel per cell matters at these sizes: nearest-neighbour sampling of a 40-cell
// thumbnail drops most of the image and turns fine detail into noise.
func sample(img image.Image, w, h int, br, bg, bb float64) [][]color.RGBA {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	out := make([][]color.RGBA, h)
	for y := 0; y < h; y++ {
		row := make([]color.RGBA, w)
		y0 := b.Min.Y + y*ih/h
		y1 := b.Min.Y + (y+1)*ih/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*iw/w
			x1 := b.Min.X + (x+1)*iw/w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sr, sg, sb, sa, n float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					r, g, bl, a := img.At(sx, sy).RGBA()
					// RGBA() is alpha-premultiplied, so these sums stay a valid
					// premultiplied color and can be composited directly below.
					sr += float64(r >> 8)
					sg += float64(g >> 8)
					sb += float64(bl >> 8)
					sa += float64(a >> 8)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			sr, sg, sb, sa = sr/n, sg/n, sb/n, sa/n
			t := 1 - sa/255
			row[x] = color.RGBA{
				R: clamp8(sr + br*t),
				G: clamp8(sg + bg*t),
				B: clamp8(sb + bb*t),
				A: 255,
			}
		}
		out[y] = row
	}
	return out
}

func clamp8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v + 0.5)
}

// renderANSI paints a grid at the given color level. Each row ends with a reset
// so a background color can never bleed past the picture, and codes are only
// re-emitted when they change, which roughly halves the output on flat areas.
func renderANSI(grid [][]Cell, level chalk.Level) string {
	if level <= chalk.LevelNone {
		return renderRamp(grid)
	}
	var sb strings.Builder
	for _, row := range grid {
		lastFG, lastBG := "", ""
		for _, c := range row {
			fg := fgCode(c.Upper, level)
			bg := bgCode(c.Lower, level)
			if fg != lastFG || bg != lastBG {
				sb.WriteString("\x1b[")
				sb.WriteString(fg)
				sb.WriteByte(';')
				sb.WriteString(bg)
				sb.WriteByte('m')
				lastFG, lastBG = fg, bg
			}
			sb.WriteString(halfBlock)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return sb.String()
}

// fgCode renders the SGR parameters for a foreground color at the given level.
func fgCode(c color.RGBA, level chalk.Level) string {
	switch {
	case level >= chalk.LevelTrueColor:
		return "38;2;" + strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
	case level >= chalk.Level256:
		return "38;5;" + strconv.Itoa(chalk.RGBToAnsi256(int(c.R), int(c.G), int(c.B)))
	default:
		return strconv.Itoa(chalk.RGBToAnsi16(int(c.R), int(c.G), int(c.B)))
	}
}

// bgCode is fgCode's background twin. At basic level the background code is the
// foreground code plus 10, the standard SGR offset (30->40, 90->100).
func bgCode(c color.RGBA, level chalk.Level) string {
	switch {
	case level >= chalk.LevelTrueColor:
		return "48;2;" + strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
	case level >= chalk.Level256:
		return "48;5;" + strconv.Itoa(chalk.RGBToAnsi256(int(c.R), int(c.G), int(c.B)))
	default:
		return strconv.Itoa(chalk.RGBToAnsi16(int(c.R), int(c.G), int(c.B)) + 10)
	}
}

// renderRamp draws the grid with no escape sequences at all, one ASCII character
// per cell chosen from [ramp] by the luminance of the two pixels averaged. This
// is what lands in a log file or a TERM=dumb session.
func renderRamp(grid [][]Cell) string {
	steps := []rune(ramp)
	var sb strings.Builder
	for _, row := range grid {
		for _, c := range row {
			l := (luma(c.Upper) + luma(c.Lower)) / 2
			i := int(l * float64(len(steps)-1) / 255)
			if i < 0 {
				i = 0
			}
			if i >= len(steps) {
				i = len(steps) - 1
			}
			sb.WriteRune(steps[i])
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// luma is Rec. 601 relative luminance, the weighting that matches perceived
// brightness closely enough for a ten-step ramp.
func luma(c color.RGBA) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}
