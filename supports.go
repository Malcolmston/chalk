package chalk

// This file mirrors Node chalk's `supportsColor` object, whose `.hasBasic`,
// `.has256` and `.has16m` flags let callers branch on terminal capability
// without inspecting the raw [Level]. Each predicate consults the current
// global level (see [GetLevel] / [SetLevel]).

// SupportsColor reports whether any color will be emitted, i.e. the current
// level is greater than [LevelNone]. It is an alias for [Enabled] provided for
// parity with Node chalk's supportsColor.
func SupportsColor() bool { return currentLevel() > LevelNone }

// HasBasic reports whether the terminal supports at least the 16 basic ANSI
// colors ([LevelBasic] or higher). This is Node chalk's supportsColor.hasBasic.
func HasBasic() bool { return currentLevel() >= LevelBasic }

// Has256 reports whether the terminal supports the 256-color palette
// ([Level256] or higher). This is Node chalk's supportsColor.has256.
func Has256() bool { return currentLevel() >= Level256 }

// HasTrueColor reports whether the terminal supports 24-bit truecolor
// ([LevelTrueColor]). This is Node chalk's supportsColor.has16m.
func HasTrueColor() bool { return currentLevel() >= LevelTrueColor }
