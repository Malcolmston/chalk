package prompts

import (
	"fmt"
	"io"
	"strings"
)

// SlidesConfig configures Slides.
type SlidesConfig struct {
	// Title is an optional heading shown above every page.
	Title string
	// Pages are the slides shown one at a time. A page may contain newlines and
	// styled (chalk) text.
	Pages []string
	// Start is the index of the first page shown (clamped into range).
	Start int
	// Loop, when set, wraps navigation around the ends instead of stopping at
	// the first and last page.
	Loop bool
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// Slides presents a paged, in-place viewer for a sequence of text pages. The
// reader navigates with the arrow keys (or space / n for next, p for previous)
// and quits with q, Esc or Ctrl-C. It returns the index of the page that was on
// screen when the viewer closed, and a nil error on a clean quit.
//
// Like the other prompts, Slides reads keys from any io.Reader, so it is fully
// scriptable and testable without a terminal. When the input is not a TTY raw
// mode is skipped, and reaching end of input closes the viewer gracefully rather
// than hanging — a piped or redirected stdin therefore shows the first page (and
// any scripted navigation) and then returns.
func Slides(cfg SlidesConfig) (int, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	if len(cfg.Pages) == 0 {
		return -1, fmt.Errorf("prompts: Slides requires at least one page")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	cur := cfg.Start
	if cur < 0 {
		cur = 0
	}
	if cur >= len(cfg.Pages) {
		cur = len(cfg.Pages) - 1
	}

	move := func(dir int) {
		next := cur + dir
		if cfg.Loop {
			n := len(cfg.Pages)
			cur = (next%n + n) % n
			return
		}
		if next >= 0 && next < len(cfg.Pages) {
			cur = next
		}
	}

	lines := renderFrame(out, 0, slidesFrame(cfg, cur))
	for {
		k := kr.read()
		switch k.typ {
		case keyRight, keyDown, keySpace, keyTab, keyEnter:
			move(1)
		case keyLeft, keyUp:
			move(-1)
		case keyRune:
			switch k.r {
			case 'n', 'j', 'l':
				move(1)
			case 'p', 'k', 'h':
				move(-1)
			case 'q', 'Q':
				renderFrame(out, lines, "")
				return cur, nil
			}
		case keyEsc, keyCtrlC, keyEOF:
			// Esc/Ctrl-C is an explicit quit; EOF means a scripted or piped
			// input ran out — both close the viewer cleanly without hanging.
			renderFrame(out, lines, "")
			return cur, nil
		}
		lines = renderFrame(out, lines, slidesFrame(cfg, cur))
	}
}

// slidesFrame renders a single page: an optional title, a "page i/N" counter,
// the page body (normalized so every line returns the cursor to column 0), and a
// help footer. The returned string ends in "\r\n" so renderFrame can measure it.
func slidesFrame(cfg SlidesConfig, cur int) string {
	var b strings.Builder
	header := styleMessage.Sprint(cfg.Title)
	if cfg.Title != "" {
		header += " "
	}
	header += styleDim.Sprint(fmt.Sprintf("(page %d/%d)", cur+1, len(cfg.Pages)))
	b.WriteString(header + "\r\n")
	b.WriteString(styleDim.Sprint(strings.Repeat("─", 40)) + "\r\n")

	// Normalize the page body to CRLF line endings so each line starts at column
	// zero in raw mode, and so renderFrame's line count matches what is drawn.
	body := strings.ReplaceAll(cfg.Pages[cur], "\r\n", "\n")
	for _, line := range strings.Split(body, "\n") {
		b.WriteString(line + "\r\n")
	}

	b.WriteString(styleDim.Sprint(strings.Repeat("─", 40)) + "\r\n")
	b.WriteString(styleHelp.Sprint("(←/→ or space to move, q to quit)") + "\r\n")
	return b.String()
}
