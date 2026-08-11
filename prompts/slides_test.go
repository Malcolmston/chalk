package prompts

import (
	"bytes"
	"strings"
	"testing"
)

func TestSlidesNavigate(t *testing.T) {
	// Right, Right, then quit -> ends on page index 2.
	got, err := Slides(SlidesConfig{
		Title: "Deck",
		Pages: []string{"one", "two", "three"},
		In:    strings.NewReader("\x1b[C\x1b[Cq"),
		Out:   &bytes.Buffer{},
	})
	if err != nil || got != 2 {
		t.Fatalf("slides navigate = %d err=%v, want 2", got, err)
	}
}

func TestSlidesClampAtEnds(t *testing.T) {
	// Left at the first page stays put (no loop); then quit.
	got, _ := Slides(SlidesConfig{
		Pages: []string{"a", "b"},
		In:    strings.NewReader("\x1b[Dq"),
		Out:   &bytes.Buffer{},
	})
	if got != 0 {
		t.Fatalf("clamp = %d, want 0", got)
	}
}

func TestSlidesLoop(t *testing.T) {
	// Left from page 0 with Loop wraps to the last page, then quit.
	got, _ := Slides(SlidesConfig{
		Pages: []string{"a", "b", "c"},
		Loop:  true,
		In:    strings.NewReader("\x1b[Dq"),
		Out:   &bytes.Buffer{},
	})
	if got != 2 {
		t.Fatalf("loop = %d, want 2", got)
	}
}

func TestSlidesEOFDoesNotHang(t *testing.T) {
	// Input with no quit key: reaching EOF must close the viewer, not hang.
	got, err := Slides(SlidesConfig{
		Pages: []string{"only"},
		In:    strings.NewReader(""), // immediate EOF
		Out:   &bytes.Buffer{},
	})
	if err != nil || got != 0 {
		t.Fatalf("eof = %d err=%v, want 0", got, err)
	}
}

func TestSlidesSpaceAndLetters(t *testing.T) {
	// space advances, 'p' goes back, 'n' advances -> 0->1->0->1, quit at 1.
	got, _ := Slides(SlidesConfig{
		Pages: []string{"a", "b", "c"},
		In:    strings.NewReader(" pnq"),
		Out:   &bytes.Buffer{},
	})
	if got != 1 {
		t.Fatalf("space/letters = %d, want 1", got)
	}
}

func TestSlidesEmptyPagesError(t *testing.T) {
	_, err := Slides(SlidesConfig{Pages: nil, In: strings.NewReader(""), Out: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error for empty pages")
	}
}

func TestSlidesFrameContent(t *testing.T) {
	// The rendered frame should include the page body and the counter.
	out := &bytes.Buffer{}
	Slides(SlidesConfig{
		Title: "T",
		Pages: []string{"hello\nworld", "second"},
		In:    strings.NewReader("q"),
		Out:   out,
	})
	s := out.String()
	if !strings.Contains(s, "hello") || !strings.Contains(s, "world") {
		t.Fatalf("frame missing body: %q", s)
	}
	if !strings.Contains(s, "page 1/2") {
		t.Fatalf("frame missing counter: %q", s)
	}
}

func TestSlidesStartClamped(t *testing.T) {
	got, _ := Slides(SlidesConfig{
		Pages: []string{"a", "b"},
		Start: 99, // out of range, clamps to last
		In:    strings.NewReader("q"),
		Out:   &bytes.Buffer{},
	})
	if got != 1 {
		t.Fatalf("start clamp = %d, want 1", got)
	}
}
