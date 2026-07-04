package figlet

import (
	"strings"
	"testing"
)

// TestGeneratedFontCount asserts the registry exposes at least 1,000 fonts once
// the programmatically-generated family is registered.
func TestGeneratedFontCount(t *testing.T) {
	n := len(Fonts())
	t.Logf("registered fonts: %d", n)
	if n < 1000 {
		t.Fatalf("len(Fonts()) = %d, want >= 1000", n)
	}
}

// TestGeneratedFontNamesUnique verifies every registered name is unique and
// non-empty.
func TestGeneratedFontNamesUnique(t *testing.T) {
	names := Fonts()
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if n == "" {
			t.Error("registered an empty font name")
		}
		if seen[n] {
			t.Errorf("duplicate font name %q", n)
		}
		seen[n] = true
	}
}

// TestGeneratedFontsRender renders a deterministic sample of generated fonts and
// checks each returns without error and produces exactly its declared height in
// rows.
func TestGeneratedFontsRender(t *testing.T) {
	names := Fonts()
	// Fixed indices give a deterministic, well-spread sample across the sorted
	// name space.
	for _, idx := range []int{0, 1, 7, 42, 100, 250, 333, 500, 613, 777, 900, 999} {
		if idx >= len(names) {
			continue
		}
		name := names[idx]
		f, ok := GetFont(name)
		if !ok {
			t.Errorf("GetFont(%q) missing", name)
			continue
		}
		out, err := RenderFont(name, "AB1")
		if err != nil {
			t.Errorf("RenderFont(%q): %v", name, err)
			continue
		}
		if got := len(strings.Split(out, "\n")); got != f.Height() {
			t.Errorf("font %q rendered %d rows, want height %d:\n%s", name, got, f.Height(), out)
		}
	}
}

// TestGeneratedFontExamples spot-checks that a few representative names from the
// documented naming scheme are actually registered.
func TestGeneratedFontExamples(t *testing.T) {
	for _, name := range []string{
		"block-hash",
		"small-dot",
		"banner-star-shadow",
		"block-plus-outline",
		"small-square-box",
	} {
		if _, ok := GetFont(name); !ok {
			t.Errorf("expected generated font %q to be registered", name)
		}
	}
}
