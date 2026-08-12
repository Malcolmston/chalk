package image

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malcolmston/chalk"
)

// swatch builds a tiny opaque test image from a row-major list of colors, so no
// binary fixture has to be committed.
func swatch(w, h int, px ...color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, c := range px {
		img.SetRGBA(i%w, i/w, c)
	}
	return img
}

var (
	red   = color.RGBA{255, 0, 0, 255}
	green = color.RGBA{0, 255, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
	white = color.RGBA{255, 255, 255, 255}
)

// quad is red/green over blue/white: one cell row of two cells once rendered.
func quad() *image.RGBA { return swatch(2, 2, red, green, blue, white) }

func TestFit(t *testing.T) {
	tests := []struct {
		name       string
		w, h       int
		cfg        Config
		cols, rows int
	}{
		// A cell is twice as tall as it is wide and holds two stacked pixels, so a
		// square image is half as many rows as columns.
		{"width square", 100, 100, Config{Width: 20}, 20, 10},
		{"width wide", 200, 100, Config{Width: 20}, 20, 5},
		{"width tall", 100, 200, Config{Width: 20}, 20, 20},
		{"height square", 100, 100, Config{Height: 10}, 20, 10},
		{"height tall", 100, 200, Config{Height: 20}, 20, 20},
		{"box height binds", 100, 200, Config{Width: 40, Height: 10}, 10, 10},
		{"box width binds", 200, 100, Config{Width: 10, Height: 40}, 10, 3},
		{"default width", 160, 80, Config{}, DefaultWidth, DefaultWidth / 4},
		{"rounds up to one row", 100, 1, Config{Width: 4}, 4, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols, rows, err := Fit(image.NewRGBA(image.Rect(0, 0, tc.w, tc.h)), tc.cfg)
			if err != nil {
				t.Fatalf("Fit: %v", err)
			}
			if cols != tc.cols || rows != tc.rows {
				t.Fatalf("Fit = %dx%d, want %dx%d", cols, rows, tc.cols, tc.rows)
			}
		})
	}
}

func TestFitErrors(t *testing.T) {
	if _, _, err := Fit(image.NewRGBA(image.Rect(0, 0, 0, 0)), Config{Width: 4}); err != ErrEmptyImage {
		t.Fatalf("empty image: err = %v, want ErrEmptyImage", err)
	}
	if _, _, err := Fit(image.NewRGBA(image.Rect(0, 0, 10, 10)), Config{Width: 1 << 12, Height: 1 << 12}); err == nil {
		t.Fatal("oversized render: want an error, got nil")
	}
	if _, err := Render(nil, Config{Width: 4, Force: true}); err != ErrEmptyImage {
		t.Fatalf("Render(nil): err = %v, want ErrEmptyImage", err)
	}
}

func TestGridCells(t *testing.T) {
	g, err := Grid(quad(), Config{Width: 2})
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	if len(g) != 1 || len(g[0]) != 2 {
		t.Fatalf("grid is %d rows, first row %d cells; want 1x2", len(g), len(g[0]))
	}
	want := []Cell{{Upper: red, Lower: blue}, {Upper: green, Lower: white}}
	for i, c := range g[0] {
		if c != want[i] {
			t.Errorf("cell %d = %+v, want %+v", i, c, want[i])
		}
	}
}

func TestGridAveragesAndComposites(t *testing.T) {
	// Four pixels averaged into one half-block pixel: box sampling, not a pick.
	flat := swatch(2, 2, red, red, red, red)
	g, err := Grid(flat, Config{Width: 1, Height: 1})
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	if g[0][0].Upper != red || g[0][0].Lower != red {
		t.Fatalf("averaged cell = %+v, want red/red", g[0][0])
	}

	// A half-transparent pixel is composited over the configured background.
	trans := swatch(1, 2, color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 0, 0})
	g, err = Grid(trans, Config{Width: 1, Height: 1, Background: white})
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	if g[0][0].Upper != white {
		t.Fatalf("transparent over white = %+v, want opaque white", g[0][0].Upper)
	}
	// With no Background the default is black, and always opaque.
	g, _ = Grid(trans, Config{Width: 1, Height: 1})
	if got := g[0][0].Upper; got != (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("transparent default = %+v, want opaque black", got)
	}
}

func TestRenderLevels(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			"truecolor",
			Config{Width: 2, Level: chalk.LevelTrueColor},
			"\x1b[38;2;255;0;0;48;2;0;0;255m▀\x1b[38;2;0;255;0;48;2;255;255;255m▀\x1b[0m\n",
		},
		{
			"256 color",
			Config{Width: 2, Level: chalk.Level256},
			"\x1b[38;5;196;48;5;21m▀\x1b[38;5;46;48;5;231m▀\x1b[0m\n",
		},
		{
			// Basic level uses the 16-color SGR codes; background is foreground + 10.
			"basic",
			Config{Width: 2, Level: chalk.LevelBasic},
			"\x1b[91;104m▀\x1b[92;107m▀\x1b[0m\n",
		},
		{
			// No color: an ASCII ramp with no escape sequences whatsoever.
			"ascii ramp",
			Config{Width: 2, ASCII: true},
			".#\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(quad(), tc.cfg)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Render = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRenderRepeatsNoRedundantCodes(t *testing.T) {
	// A flat image should emit one SGR sequence per row, not one per cell.
	flat := swatch(4, 2, red, red, red, red, red, red, red, red)
	got, err := Render(flat, Config{Width: 4, Height: 1, Level: chalk.LevelTrueColor})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if n := strings.Count(got, "38;2;"); n != 1 {
		t.Fatalf("Render = %q, has %d color sequences, want 1", got, n)
	}
	if n := strings.Count(got, halfBlock); n != 4 {
		t.Fatalf("Render = %q, has %d blocks, want 4", got, n)
	}
	if !strings.HasSuffix(got, "\x1b[0m\n") {
		t.Fatalf("Render = %q, want a reset at end of row", got)
	}
}

func TestFprintNonTerminal(t *testing.T) {
	tests := []struct {
		name  string
		cfg   Config
		empty bool
	}{
		{"not a terminal writes nothing", Config{Width: 4, Level: chalk.LevelTrueColor}, true},
		{"forced writes", Config{Width: 4, Level: chalk.LevelTrueColor, Force: true}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := tc.cfg
			cfg.Out = &buf
			if err := Fprint(quad(), cfg); err != nil {
				t.Fatalf("Fprint: %v", err)
			}
			if got := buf.Len() == 0; got != tc.empty {
				t.Fatalf("Fprint wrote %q; empty = %v, want %v", buf.String(), got, tc.empty)
			}
		})
	}
}

// A pipe is the realistic non-terminal case: an *os.File that is not a TTY.
func TestFprintToPipeWritesNothing(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	if err := Fprint(quad(), Config{Out: w, Width: 4, Level: chalk.LevelTrueColor}); err != nil {
		t.Fatalf("Fprint: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("wrote %q to a pipe, want nothing", buf.String())
	}
}

func TestForcedNonTerminalNeverPicksNativeProtocol(t *testing.T) {
	// Even inside a terminal that supports Kitty graphics, a forced write to a
	// buffer must degrade: base64 in a log file is the failure this guards.
	t.Setenv("TERM", "xterm-kitty")
	var buf bytes.Buffer
	if err := Fprint(quad(), Config{Out: &buf, Width: 2, Level: chalk.LevelTrueColor, Force: true}); err != nil {
		t.Fatalf("Fprint: %v", err)
	}
	if strings.Contains(buf.String(), "\x1b_G") || strings.Contains(buf.String(), "\x1b]1337") {
		t.Fatalf("auto protocol emitted a native sequence into a buffer: %q", buf.String())
	}
	if !strings.Contains(buf.String(), halfBlock) {
		t.Fatalf("want half blocks, got %q", buf.String())
	}
}

func TestRenderITerm2(t *testing.T) {
	got, err := Render(quad(), Config{Width: 8, Protocol: ProtocolITerm2})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(got, "\x1b]1337;File=inline=1;preserveAspectRatio=1;size=") {
		t.Fatalf("bad prefix: %q", got[:min(60, len(got))])
	}
	if !strings.Contains(got, ";width=8;height=4:") {
		t.Fatalf("missing cell size: %q", got)
	}
	if !strings.HasSuffix(got, "\a\n") {
		t.Fatal("iTerm2 sequence must end with BEL")
	}
	payload := got[strings.Index(got, ":")+1 : len(got)-2]
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(raw)); err != nil {
		t.Fatalf("payload is not a PNG: %v", err)
	}
}

func TestRenderKittyChunks(t *testing.T) {
	// A noisy image so PNG compression cannot squeeze it under one chunk.
	big := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			big.SetRGBA(x, y, color.RGBA{uint8(x * 7 % 251), uint8(y*13%251 + 3), uint8((x*y)%251 + 1), 255})
		}
	}
	got, err := Render(big, Config{Width: 20, Protocol: ProtocolKitty})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(got, "\x1b_Ga=T,f=100,c=20,r=10,m=") {
		t.Fatalf("bad first chunk: %q", got[:min(60, len(got))])
	}
	if n := strings.Count(got, "\x1b_G"); n < 2 {
		t.Fatalf("want the payload chunked, got %d chunk(s)", n)
	}
	if n := strings.Count(got, "m=1;"); n < 1 {
		t.Fatal("want continuation chunks marked m=1")
	}
	if !strings.HasSuffix(got, "\x1b\\"+strings.Repeat("\n", 10)) {
		t.Fatalf("want a terminated last chunk followed by 10 newlines, got %q", got[len(got)-20:])
	}
	// Reassembling every chunk's payload must give back a decodable PNG.
	var b64 strings.Builder
	for _, part := range strings.Split(got, "\x1b_G")[1:] {
		part = strings.TrimSuffix(strings.TrimRight(part, "\n"), "\x1b\\")
		b64.WriteString(part[strings.Index(part, ";")+1:])
	}
	raw, err := base64.StdEncoding.DecodeString(b64.String())
	if err != nil {
		t.Fatalf("reassembled payload is not base64: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(raw)); err != nil {
		t.Fatalf("reassembled payload is not a PNG: %v", err)
	}
}

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Protocol
	}{
		{"bare xterm", map[string]string{"TERM": "xterm-256color"}, ProtocolANSI},
		{"kitty by TERM", map[string]string{"TERM": "xterm-kitty"}, ProtocolKitty},
		{"kitty by window id", map[string]string{"KITTY_WINDOW_ID": "1"}, ProtocolKitty},
		{"ghostty", map[string]string{"TERM": "xterm-ghostty"}, ProtocolKitty},
		{"iterm2 v3", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.4.19"}, ProtocolITerm2},
		{"iterm2 v2 is too old", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "2.9"}, ProtocolANSI},
		{"iterm2 unknown version", map[string]string{"TERM_PROGRAM": "iTerm.app"}, ProtocolANSI},
		{"iterm2 over ssh", map[string]string{"LC_TERMINAL": "iTerm2"}, ProtocolITerm2},
		{"wezterm", map[string]string{"TERM_PROGRAM": "WezTerm"}, ProtocolITerm2},
		{"tmux hides kitty", map[string]string{"TERM": "xterm-kitty", "TMUX": "/tmp/tmux-0/default,1,0"}, ProtocolANSI},
		{"screen hides iterm2", map[string]string{"TERM": "screen-256color", "LC_TERMINAL": "iTerm2"}, ProtocolANSI},
		{"empty environment", map[string]string{}, ProtocolANSI},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			lookup := func(k string) (string, bool) { v, ok := tc.env[k]; return v, ok }
			if got := detectProtocolEnv(getenv, lookup); got != tc.want {
				t.Fatalf("detectProtocolEnv = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProtocolString(t *testing.T) {
	for _, tc := range []struct {
		p    Protocol
		want string
	}{
		{ProtocolAuto, "auto"},
		{ProtocolANSI, "ansi"},
		{ProtocolITerm2, "iterm2"},
		{ProtocolKitty, "kitty"},
		{Protocol(9), "Protocol(9)"},
	} {
		if got := tc.p.String(); got != tc.want {
			t.Errorf("Protocol(%d).String() = %q, want %q", int(tc.p), got, tc.want)
		}
	}
}

func TestDecodeAndOpen(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, quad()); err != nil {
		t.Fatalf("encode: %v", err)
	}
	img, format, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if format != "png" {
		t.Fatalf("format = %q, want png", format)
	}
	if img.Bounds().Dx() != 2 {
		t.Fatalf("bounds = %v", img.Bounds())
	}

	path := filepath.Join(t.TempDir(), "quad.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Open(path); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := Open(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Fatal("Open of a missing file: want an error")
	}
	if _, _, err := Decode(strings.NewReader("not an image")); err == nil {
		t.Fatal("Decode of junk: want an error")
	}
}
