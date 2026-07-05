package prompts

import (
	"fmt"
	"io"
	"strings"
)

// SelectConfig configures Select.
type SelectConfig struct {
	// Message is the question shown to the user.
	Message string
	// Choices are the selectable options.
	Choices []Choice
	// Default is the initially highlighted index.
	Default int
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// MultiSelectConfig configures MultiSelect.
type MultiSelectConfig struct {
	// Message is the question shown to the user.
	Message string
	// Choices are the selectable options.
	Choices []Choice
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// renderFrame clears the previous frame (prevLines tall) and writes content,
// returning the new frame's line count. Content lines must end in "\r\n".
func renderFrame(out io.Writer, prevLines int, content string) int {
	if prevLines > 0 {
		writeString(out, fmt.Sprintf("\r\x1b[%dA", prevLines))
	}
	writeString(out, "\r\x1b[J") // clear from cursor to end of screen
	writeString(out, content)
	return strings.Count(content, "\n")
}

// step moves the cursor in dir (+1/-1), wrapping and skipping disabled choices.
func step(choices []Choice, cur, dir int) int {
	n := len(choices)
	if n == 0 {
		return cur
	}
	for i := 0; i < n; i++ {
		cur = (cur + dir + n) % n
		if !choices[cur].Disabled {
			return cur
		}
	}
	return cur
}

// firstSelectable returns the first non-disabled index at or after start.
func firstSelectable(choices []Choice, start int) int {
	if start < 0 || start >= len(choices) {
		start = 0
	}
	if len(choices) == 0 || !choices[start].Disabled {
		return start
	}
	return step(choices, start, 1)
}

// Select presents a single-choice list navigated with the arrow keys. It
// returns the selected index and choice.
func Select(cfg SelectConfig) (int, Choice, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	if len(cfg.Choices) == 0 {
		return -1, Choice{}, fmt.Errorf("prompts: Select requires at least one choice")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	cur := firstSelectable(cfg.Choices, cfg.Default)
	lines := renderFrame(out, 0, selectFrame(cfg, cur))

	for {
		k := kr.read()
		switch k.typ {
		case keyUp, keyLeft:
			cur = step(cfg.Choices, cur, -1)
		case keyDown, keyRight, keyTab:
			cur = step(cfg.Choices, cur, 1)
		case keyEnter, keyEOF:
			summary := stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
				styleAnswer.Sprint(cfg.Choices[cur].Name) + "\r\n"
			renderFrame(out, lines, summary)
			return cur, cfg.Choices[cur], nil
		case keyCtrlC, keyEsc:
			writeString(out, "\r\n")
			return -1, Choice{}, ErrCanceled
		}
		lines = renderFrame(out, lines, selectFrame(cfg, cur))
	}
}

func selectFrame(cfg SelectConfig, cur int) string {
	var b strings.Builder
	b.WriteString(stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
		styleHelp.Sprint("(↑/↓ to move, enter to select)") + "\r\n")
	for i, c := range cfg.Choices {
		pointer := "  "
		label := c.Name
		switch {
		case c.Disabled:
			label = styleDim.Sprint(label + " (disabled)")
		case i == cur:
			pointer = stylePointer.Sprint("❯ ")
			label = styleSelected.Sprint(label)
		}
		b.WriteString(pointer + label + "\r\n")
	}
	return b.String()
}

// MultiSelect presents a checkbox list: arrows to move, space to toggle, enter
// to confirm. It returns the selected indices and choices.
func MultiSelect(cfg MultiSelectConfig) ([]int, []Choice, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	if len(cfg.Choices) == 0 {
		return nil, nil, fmt.Errorf("prompts: MultiSelect requires at least one choice")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	checked := make([]bool, len(cfg.Choices))
	for i, c := range cfg.Choices {
		checked[i] = c.Checked
	}
	cur := firstSelectable(cfg.Choices, 0)
	lines := renderFrame(out, 0, multiFrame(cfg, cur, checked))

	for {
		k := kr.read()
		switch k.typ {
		case keyUp, keyLeft:
			cur = step(cfg.Choices, cur, -1)
		case keyDown, keyRight, keyTab:
			cur = step(cfg.Choices, cur, 1)
		case keySpace:
			if !cfg.Choices[cur].Disabled {
				checked[cur] = !checked[cur]
			}
		case keyEnter, keyEOF:
			var idxs []int
			var chosen []Choice
			var names []string
			for i, on := range checked {
				if on {
					idxs = append(idxs, i)
					chosen = append(chosen, cfg.Choices[i])
					names = append(names, cfg.Choices[i].Name)
				}
			}
			summary := stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
				styleAnswer.Sprint(strings.Join(names, ", ")) + "\r\n"
			renderFrame(out, lines, summary)
			return idxs, chosen, nil
		case keyCtrlC, keyEsc:
			writeString(out, "\r\n")
			return nil, nil, ErrCanceled
		}
		lines = renderFrame(out, lines, multiFrame(cfg, cur, checked))
	}
}

func multiFrame(cfg MultiSelectConfig, cur int, checked []bool) string {
	var b strings.Builder
	b.WriteString(stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
		styleHelp.Sprint("(↑/↓ to move, space to toggle, enter to confirm)") + "\r\n")
	for i, c := range cfg.Choices {
		pointer := "  "
		if i == cur && !c.Disabled {
			pointer = stylePointer.Sprint("❯ ")
		}
		box := "◯ "
		if checked[i] {
			box = styleCheckOn.Sprint("◉ ")
		}
		label := c.Name
		switch {
		case c.Disabled:
			label = styleDim.Sprint(label + " (disabled)")
		case i == cur:
			label = styleSelected.Sprint(label)
		}
		b.WriteString(pointer + box + label + "\r\n")
	}
	return b.String()
}
