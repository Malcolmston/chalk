// Command table prints a colored, aligned table and a report of what the
// current terminal supports.
//
// It is the worked example for two things that are easy to get wrong by hand:
// measuring styled text, and adapting to the terminal's color capability.
// Column padding is computed with chalk.VisibleWidth, which ignores ANSI escape
// sequences and counts wide characters (CJK, emoji) as the two cells they
// actually occupy — using len() or a rune count here would misalign every row
// that contains either. The capability report below the table shows the level
// chalk detected and how the same 24-bit color degrades at each level.
//
// Run it, then run it again with the environment overridden to see detection
// change:
//
//	go run ./examples/table
//	NO_COLOR=1 go run ./examples/table
//	FORCE_COLOR=1 go run ./examples/table | cat
package main

import (
	"fmt"
	"strings"

	"github.com/malcolmston/chalk"
)

// row is one line of the table.
type row struct {
	name   string
	status string
	note   string
}

func main() {
	rows := []row{
		{"build", "ok", "cached"},
		{"tests", "ok", "142 passed"},
		{"lint", "warn", "3 suggestions"},
		{"deploy", "fail", "credentials rejected"},
		{"图表 chart", "ok", "wide characters align too"},
		{"emoji 🚀", "ok", "so do emoji"},
	}

	headers := []string{"TASK", "STATUS", "NOTE"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = chalk.VisibleWidth(h)
	}
	for _, r := range rows {
		for i, cell := range []string{r.name, r.status, r.note} {
			if w := chalk.VisibleWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	header := chalk.New().Bold().Underline()
	line := make([]string, len(headers))
	for i, h := range headers {
		line[i] = pad(header.Sprint(h), widths[i])
	}
	fmt.Println(strings.Join(line, "  "))

	for _, r := range rows {
		cells := []string{
			pad(chalk.New().Cyan().Sprint(r.name), widths[0]),
			pad(statusStyle(r.status).Sprint(r.status), widths[1]),
			pad(chalk.New().Gray().Sprint(r.note), widths[2]),
		}
		fmt.Println(strings.Join(cells, "  "))
	}

	fmt.Println()
	report()
}

// pad right-pads a styled cell to width columns on screen.
func pad(cell string, width int) string {
	gap := width - chalk.VisibleWidth(cell)
	if gap <= 0 {
		return cell
	}
	return cell + strings.Repeat(" ", gap)
}

// statusStyle picks a color for a status word.
func statusStyle(status string) *chalk.Style {
	switch status {
	case "ok":
		return chalk.New().Green()
	case "warn":
		return chalk.New().Yellow()
	default:
		return chalk.New().Red().Bold()
	}
}

// report prints the detected color level and how one color renders at each of
// them, which is the clearest way to see the truecolor -> 256 -> 16 downgrade.
func report() {
	names := map[chalk.Level]string{
		chalk.LevelNone:      "none (no color)",
		chalk.LevelBasic:     "basic (16 colors)",
		chalk.Level256:       "256-color palette",
		chalk.LevelTrueColor: "truecolor (24-bit)",
	}

	fmt.Printf("detected level: %s\n", names[chalk.GetLevel()])
	fmt.Printf("supports color: %v  256: %v  truecolor: %v\n",
		chalk.SupportsColor(), chalk.Has256(), chalk.HasTrueColor())

	orange := chalk.New().Hex("#ff8800")
	for _, l := range []chalk.Level{chalk.LevelNone, chalk.LevelBasic, chalk.Level256, chalk.LevelTrueColor} {
		styled := orange.Level(l).Sprint("#ff8800")
		fmt.Printf("  %-20s %s  (%q)\n", names[l], styled, styled)
	}
}
