package image_test

import (
	"bytes"
	"fmt"
	stdimage "image"
	"image/color"

	"github.com/malcolmston/chalk"
	chalkimage "github.com/malcolmston/chalk/image"
)

// checkers builds a small checkerboard to render.
func checkers(w, h, size int) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/size+y/size)%2 == 0 {
				img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return img
}

// Render with ASCII forced needs no color support at all, which makes it the one
// form that is readable as example output.
func Example_asciiRamp() {
	out, _ := chalkimage.Render(checkers(32, 32, 8), chalkimage.Config{Width: 8, ASCII: true})
	// Quoted so the example output keeps its trailing spaces.
	fmt.Printf("%q\n", out)
	// Output: "@@  @@  \n  @@  @@\n@@  @@  \n  @@  @@\n"
}

// Fit shows the aspect correction: a cell is about twice as tall as it is wide,
// and each cell holds two stacked half-block pixels.
func ExampleFit() {
	square := stdimage.NewRGBA(stdimage.Rect(0, 0, 100, 100))
	cols, rows, _ := chalkimage.Fit(square, chalkimage.Config{Width: 40})
	fmt.Println(cols, rows)
	// Output: 40 20
}

// Writing to anything that is not a terminal produces nothing, so a program can
// call Fprint unconditionally and still have clean piped output.
func ExampleFprint() {
	var buf bytes.Buffer
	_ = chalkimage.Fprint(checkers(8, 8, 4), chalkimage.Config{Out: &buf, Width: 4})
	fmt.Printf("piped: %d bytes\n", buf.Len())

	// Force renders anyway, for the caller who really does want the escapes in a file.
	buf.Reset()
	_ = chalkimage.Fprint(checkers(8, 8, 4), chalkimage.Config{
		Out: &buf, Width: 2, Level: chalk.LevelTrueColor, Force: true,
	})
	fmt.Printf("forced: %q\n", buf.String())
	// Output:
	// piped: 0 bytes
	// forced: "\x1b[38;2;255;255;255;48;2;0;0;0m▀\x1b[38;2;0;0;0;48;2;255;255;255m▀\x1b[0m\n"
}
