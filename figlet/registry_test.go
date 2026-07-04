package figlet

import (
	"strings"
	"testing"

	"github.com/malcolmston/chalk"
)

func TestRegistryHasBuiltins(t *testing.T) {
	names := Fonts()
	want := []string{"standard", "block", "dark", "light", "dots", "stars"}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("font %q not registered; have %v", w, names)
		}
	}
}

func TestRenderFontVariant(t *testing.T) {
	out, err := RenderFont("block", "A")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "█") {
		t.Fatalf("block variant should use █:\n%s", out)
	}
	if strings.Contains(out, "#") {
		t.Fatalf("block variant should not contain #:\n%s", out)
	}
}

func TestRenderFontUnknown(t *testing.T) {
	if _, err := RenderFont("nope", "x"); err == nil {
		t.Fatal("expected error for unknown font")
	}
}

func TestGradientColorsOutput(t *testing.T) {
	chalk.SetLevel(chalk.LevelTrueColor)
	defer chalk.SetLevel(chalk.LevelNone)

	colored := RenderGradient("HI", "#ff0000", "#0000ff")
	// Colored output contains ANSI escapes; stripping them yields the plain art.
	if !strings.Contains(colored, "\x1b[38;2;") {
		t.Fatal("gradient did not emit truecolor codes")
	}
	if chalk.Strip(colored) != Render("HI") {
		t.Fatal("stripping color should reproduce the plain banner")
	}
}

func TestRainbowStripsToPlain(t *testing.T) {
	chalk.SetLevel(chalk.LevelTrueColor)
	defer chalk.SetLevel(chalk.LevelNone)
	if chalk.Strip(RenderRainbow("GO")) != Render("GO") {
		t.Fatal("rainbow strip mismatch")
	}
}
