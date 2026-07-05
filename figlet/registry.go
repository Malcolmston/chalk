package figlet

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[string]*Font{}
)

// Register adds a named font to the registry, making it available to
// RenderFont and GetFont.
func Register(name string, f *Font) {
	registryMu.Lock()
	registry[strings.ToLower(name)] = f
	registryMu.Unlock()
}

// GetFont returns a registered font by name (case-insensitive).
func GetFont(name string) (*Font, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[strings.ToLower(name)]
	return f, ok
}

// Fonts returns the sorted names of all registered fonts.
func Fonts() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RenderFont renders text with a named registered font.
func RenderFont(name, text string, opts ...Options) (string, error) {
	f, ok := GetFont(name)
	if !ok {
		return "", &unknownFontError{name}
	}
	return f.Render(text, opts...), nil
}

type unknownFontError struct{ name string }

// Error implements the error interface.
func (e *unknownFontError) Error() string { return "figlet: unknown font " + e.name }

// LoadFontDir parses every .flf FIGfont in dir and registers each under its
// base file name (without extension). It returns the names loaded.
func LoadFontDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var loaded []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".flf") {
			continue
		}
		f, err := LoadFontFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		Register(name, f)
		loaded = append(loaded, name)
	}
	sort.Strings(loaded)
	return loaded, nil
}

// init registers the built-in fonts: the standard block font plus a set of
// style variants that render the same glyphs with different fill characters.
func init() {
	Register("standard", BuiltinFont())
	Register("block", variantFont('█'))
	Register("dark", variantFont('▓'))
	Register("medium", variantFont('▒'))
	Register("light", variantFont('░'))
	Register("dots", variantFont('●'))
	Register("stars", variantFont('*'))
	Register("plus", variantFont('+'))
	Register("at", variantFont('@'))
}

// variantFont builds a copy of the built-in font with '#' replaced by fill.
func variantFont(fill rune) *Font {
	base := BuiltinFont()
	chars := make(map[rune][]string, len(base.chars))
	for r, rows := range base.chars {
		cp := make([]string, len(rows))
		for i, line := range rows {
			cp[i] = strings.ReplaceAll(line, "#", string(fill))
		}
		chars[r] = cp
	}
	return &Font{
		hardblank: base.hardblank,
		height:    base.height,
		baseline:  base.baseline,
		oldLayout: base.oldLayout,
		chars:     chars,
	}
}
