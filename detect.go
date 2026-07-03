package chalk

import (
	"os"
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

// detectLevel determines the color level from environment and terminal state,
// following the conventions of NO_COLOR, FORCE_COLOR, COLORTERM and TERM.
func detectLevel(f *os.File) Level {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return LevelNone
	}
	if fc, ok := os.LookupEnv("FORCE_COLOR"); ok {
		switch fc {
		case "0", "false":
			return LevelNone
		case "1", "true", "":
			return LevelBasic
		case "2":
			return Level256
		case "3":
			return LevelTrueColor
		default:
			return LevelBasic
		}
	}

	if !isTerminal(f) {
		return LevelNone
	}

	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return LevelNone
	}

	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorterm == "truecolor" || colorterm == "24bit" {
		return LevelTrueColor
	}
	if strings.Contains(term, "256") {
		return Level256
	}
	if term != "" {
		return LevelBasic
	}
	// Terminal detected but no TERM hint: assume basic color.
	return LevelBasic
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
