package chalk

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// Level is the color capability of the output terminal.
type Level int

const (
	// LevelNone disables color output.
	LevelNone Level = iota
	// LevelBasic supports the 16 ANSI colors.
	LevelBasic
	// Level256 supports the 256-color palette.
	Level256
	// LevelTrueColor supports 24-bit color.
	LevelTrueColor
)

var (
	levelMu    sync.RWMutex
	levelSet   bool
	levelValue Level
)

// currentLevel returns the effective global color level, detecting it once from
// the environment if not explicitly set.
func currentLevel() Level {
	levelMu.RLock()
	if levelSet {
		v := levelValue
		levelMu.RUnlock()
		return v
	}
	levelMu.RUnlock()

	levelMu.Lock()
	defer levelMu.Unlock()
	if !levelSet {
		levelValue = detectLevel(os.Stdout)
		levelSet = true
	}
	return levelValue
}

// SetLevel forces the global color level, overriding auto-detection.
func SetLevel(l Level) {
	levelMu.Lock()
	levelValue = l
	levelSet = true
	levelMu.Unlock()
}

// GetLevel returns the current global color level.
func GetLevel() Level { return currentLevel() }

// Enabled reports whether any color will be emitted.
func Enabled() bool { return currentLevel() > LevelNone }

// SetEnabled turns color on (auto-detected level, minimum basic) or off.
func SetEnabled(on bool) {
	if !on {
		SetLevel(LevelNone)
		return
	}
	if l := detectLevel(os.Stdout); l > LevelNone {
		SetLevel(l)
	} else {
		SetLevel(LevelBasic)
	}
}

// ResetDetection clears any forced level so the next use re-detects.
func ResetDetection() {
	levelMu.Lock()
	levelSet = false
	levelMu.Unlock()
}

// envForceColor interprets the FORCE_COLOR variable. The second result reports
// whether it was set at all, which matters independently of its value: a set
// FORCE_COLOR suppresses the "not a terminal" bail-out even when it does not
// name a level.
//
// Values follow Node's supports-color: "false" and "0" disable color, "true" and
// the empty string mean basic color, and anything numeric is clamped into the
// 0–3 range (so FORCE_COLOR=4 is truecolor rather than, as before, basic).
// Any other string means basic color.
func envForceColor() (Level, bool) {
	v, ok := os.LookupEnv("FORCE_COLOR")
	if !ok {
		return LevelNone, false
	}
	switch v {
	case "false":
		return LevelNone, true
	case "true", "":
		return LevelBasic, true
	}
	if n, err := strconv.Atoi(v); err == nil {
		switch {
		case n < 0:
			n = 0
		case n > int(LevelTrueColor):
			n = int(LevelTrueColor)
		}
		return Level(n), true
	}
	return LevelBasic, true
}

// hasEnv reports whether the named variable is present, regardless of value.
func hasEnv(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

// ciLevel reports the color level implied by continuous-integration environment
// variables, mirroring the CI block of Node's supports-color. The bool result
// reports whether this looked like a recognized CI environment at all.
func ciLevel() (Level, bool) {
	if !hasEnv("CI") {
		return LevelNone, false
	}
	if os.Getenv("CI_NAME") == "codeship" {
		return LevelBasic, true
	}
	// GitHub and Gitea Actions render 24-bit color in their log viewers.
	if hasEnv("GITHUB_ACTIONS") || hasEnv("GITEA_ACTIONS") {
		return LevelTrueColor, true
	}
	for _, v := range []string{"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE", "DRONE"} {
		if hasEnv(v) {
			return LevelBasic, true
		}
	}
	// Some other CI: recognized, but nothing known about its capabilities.
	return LevelNone, true
}

// detectLevel determines the color level from the environment and the terminal
// state of f, following Node supports-color's order of precedence:
//
//  1. NO_COLOR (set at all) disables color outright.
//  2. FORCE_COLOR=0/false disables color; any other FORCE_COLOR value becomes a
//     floor for the detected level and suppresses the TTY requirement.
//  3. Azure Pipelines (TF_BUILD together with AGENT_NAME) gets basic color even
//     when the output is redirected.
//  4. Otherwise output that is not a terminal gets no color.
//  5. TERM=dumb gets no more than the floor.
//  6. CI providers, TEAMCITY_VERSION, COLORTERM, known truecolor terminals,
//     TERM_PROGRAM, then the TERM name itself.
//
// Steps 2 and 6 are the ones worth remembering: FORCE_COLOR raises the floor but
// does not cap detection (FORCE_COLOR=1 in a truecolor terminal still yields
// [LevelTrueColor]), and the level never drops below the floor once set.
func detectLevel(f *os.File) Level {
	if hasEnv("NO_COLOR") {
		return LevelNone
	}

	floor, forced := envForceColor()
	if forced && floor == LevelNone {
		return LevelNone
	}

	// Azure Pipelines colorizes its log view without presenting a TTY.
	if hasEnv("TF_BUILD") && hasEnv("AGENT_NAME") {
		return maxLevel(floor, LevelBasic)
	}

	if !forced && !isTerminal(f) {
		return LevelNone
	}

	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return floor
	}

	if lvl, isCI := ciLevel(); isCI {
		return maxLevel(floor, lvl)
	}

	if v, ok := os.LookupEnv("TEAMCITY_VERSION"); ok {
		if teamCitySupportsColor(v) {
			return maxLevel(floor, LevelBasic)
		}
		return floor
	}

	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	// "24bit" is not recognized by supports-color but is a common spelling; it is
	// accepted here as a deliberate superset.
	if colorterm == "truecolor" || colorterm == "24bit" {
		return LevelTrueColor
	}

	// Terminals known to advertise truecolor through TERM alone.
	switch term {
	case "xterm-kitty", "xterm-ghostty", "wezterm":
		return LevelTrueColor
	}
	// Windows Terminal sets WT_SESSION and supports 24-bit color.
	if hasEnv("WT_SESSION") {
		return LevelTrueColor
	}

	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app":
		// iTerm 3 and later support truecolor; earlier versions only 256.
		if major, err := strconv.Atoi(strings.SplitN(os.Getenv("TERM_PROGRAM_VERSION"), ".", 2)[0]); err == nil && major >= 3 {
			return LevelTrueColor
		}
		return maxLevel(floor, Level256)
	case "Apple_Terminal":
		return maxLevel(floor, Level256)
	}

	if strings.HasSuffix(term, "-256color") || strings.HasSuffix(term, "-256") {
		return maxLevel(floor, Level256)
	}
	if termLooksColorful(term) {
		return maxLevel(floor, LevelBasic)
	}
	if hasEnv("COLORTERM") {
		return maxLevel(floor, LevelBasic)
	}
	return floor
}

// termPrefixes are the TERM name prefixes supports-color treats as color-capable.
var termPrefixes = []string{"screen", "xterm", "vt100", "vt220", "rxvt"}

// termSubstrings are the TERM name fragments supports-color treats as
// color-capable wherever they appear.
var termSubstrings = []string{"color", "ansi", "cygwin", "linux"}

// termLooksColorful reports whether a (lowercased) TERM name matches the
// heuristic supports-color uses for basic color support.
func termLooksColorful(term string) bool {
	if term == "" {
		return false
	}
	for _, p := range termPrefixes {
		if strings.HasPrefix(term, p) {
			return true
		}
	}
	for _, s := range termSubstrings {
		if strings.Contains(term, s) {
			return true
		}
	}
	return false
}

// teamCitySupportsColor reports whether a TEAMCITY_VERSION string is 9.1 or
// newer, the first release whose build log renders ANSI color. supports-color
// tests this with /^(9\.(0*[1-9]\d*)\.|\d{2,}\.)/; this is the same rule spelled
// out: a two-or-more-digit major version, or major 9 with a nonzero minor.
func teamCitySupportsColor(v string) bool {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	if major >= 10 {
		return true
	}
	if major != 9 {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	return err == nil && minor > 0
}

// maxLevel returns the greater of two levels.
func maxLevel(a, b Level) Level {
	if a > b {
		return a
	}
	return b
}

// isTerminal reports whether f is attached to a terminal (character device).
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
