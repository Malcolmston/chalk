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
	// MaxVisible caps how many choices are shown at once, scrolling the list
	// around the highlighted entry. Zero shows every choice. This is upstream
	// prompts' "limit" option.
	MaxVisible int
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
	// MaxVisible caps how many choices are shown at once, scrolling the list
	// around the highlighted entry. Zero shows every choice.
	MaxVisible int
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// entriesToDisplay determines which slice of a list to show given the currently
// selected cursor, the total number of entries, and the maximum visible window.
// It is a direct port of upstream terkelg/prompts' lib/util/entriesToDisplay.js
// and is used to page long Select / MultiSelect lists. A maxVisible of 0 means
// "show everything" (the upstream default when the option is omitted).
func entriesToDisplay(cursor, total, maxVisible int) (startIndex, endIndex int) {
	if maxVisible == 0 {
		maxVisible = total
	}
	startIndex = min(total-maxVisible, cursor-maxVisible/2)
	if startIndex < 0 {
		startIndex = 0
	}
	endIndex = min(startIndex+maxVisible, total)
	return startIndex, endIndex
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

// hasSelectable reports whether at least one choice can be picked.
func hasSelectable(choices []Choice) bool {
	for _, c := range choices {
		if !c.Disabled {
			return true
		}
	}
	return false
}

// visibleWindow returns the half-open range of choices to draw, together with
// flags reporting whether entries are hidden above or below it.
func visibleWindow(cur, total, maxVisible int) (start, end int, more, less bool) {
	if maxVisible < 0 {
		maxVisible = 0
	}
	start, end = entriesToDisplay(cur, total, maxVisible)
	return start, end, start > 0, end < total
}

// overflowHint is the dim marker drawn where a scrolled list continues.
func overflowHint(arrow string) string { return styleDim.Sprint("  "+arrow) + "\r\n" }

// Select presents a single-choice list navigated with the arrow keys. It
// returns the selected index and choice.
func Select(cfg SelectConfig) (int, Choice, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	if len(cfg.Choices) == 0 {
		return -1, Choice{}, fmt.Errorf("prompts: Select requires at least one choice")
	}
	// A list in which everything is disabled has no valid answer. Without this
	// check the cursor sat on a disabled entry and Enter "selected" it, so the
	// caller got back a choice it had explicitly marked unselectable.
	if !hasSelectable(cfg.Choices) {
		return -1, Choice{}, fmt.Errorf("prompts: Select requires at least one choice that is not disabled")
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
	start, end, more, less := visibleWindow(cur, len(cfg.Choices), cfg.MaxVisible)
	if more {
		b.WriteString(overflowHint("↑"))
	}
	for i := start; i < end; i++ {
		c := cfg.Choices[i]
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
	if less {
		b.WriteString(overflowHint("↓"))
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
	if !hasSelectable(cfg.Choices) {
		return nil, nil, fmt.Errorf("prompts: MultiSelect requires at least one choice that is not disabled")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	checked := make([]bool, len(cfg.Choices))
	for i, c := range cfg.Choices {
		// A disabled choice cannot be toggled on, so it must not start on
		// either: Checked plus Disabled otherwise returned a selection the user
		// had no way to remove.
		checked[i] = c.Checked && !c.Disabled
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
	start, end, more, less := visibleWindow(cur, len(cfg.Choices), cfg.MaxVisible)
	if more {
		b.WriteString(overflowHint("↑"))
	}
	for i := start; i < end; i++ {
		c := cfg.Choices[i]
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
	if less {
		b.WriteString(overflowHint("↓"))
	}
	return b.String()
}
