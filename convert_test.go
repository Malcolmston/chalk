package chalk

import (
	"fmt"
	"testing"
)

func TestHexRGBRoundTrip(t *testing.T) {
	cases := []struct {
		hex     string
		r, g, b int
	}{
		{"#ff8800", 255, 136, 0},
		{"ff8800", 255, 136, 0},
		{"#f80", 255, 136, 0},
		{"#000000", 0, 0, 0},
		{"#ffffff", 255, 255, 255},
		{"#123456", 0x12, 0x34, 0x56},
	}
	for _, c := range cases {
		r, g, b := HexToRGB(c.hex)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("HexToRGB(%q) = %d,%d,%d; want %d,%d,%d", c.hex, r, g, b, c.r, c.g, c.b)
		}
	}
	if got := RGBToHex(255, 136, 0); got != "#ff8800" {
		t.Errorf("RGBToHex(255,136,0) = %q; want #ff8800", got)
	}
	if got := RGBToHex(0, 0, 0); got != "#000000" {
		t.Errorf("RGBToHex(0,0,0) = %q; want #000000", got)
	}
	if got := RGBToHex(-5, 300, 128); got != "#00ff80" {
		t.Errorf("RGBToHex clamp = %q; want #00ff80", got)
	}
}

func TestRGBToAnsi256(t *testing.T) {
	cases := []struct {
		r, g, b, want int
	}{
		{0, 0, 0, 16},
		{255, 255, 255, 231},
		{255, 0, 0, 196},
		{0, 255, 0, 46},
		{0, 0, 255, 21},
		{128, 128, 128, 244},
	}
	for _, c := range cases {
		if got := RGBToAnsi256(c.r, c.g, c.b); got != c.want {
			t.Errorf("RGBToAnsi256(%d,%d,%d) = %d; want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestAnsi256ToRGB(t *testing.T) {
	cases := []struct {
		n, r, g, b int
	}{
		{16, 0, 0, 0},
		{231, 255, 255, 255},
		{196, 255, 0, 0},
		{46, 0, 255, 0},
		{21, 0, 0, 255},
	}
	for _, c := range cases {
		r, g, b := Ansi256ToRGB(c.n)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("Ansi256ToRGB(%d) = %d,%d,%d; want %d,%d,%d", c.n, r, g, b, c.r, c.g, c.b)
		}
	}
}

func TestRGBToAnsi16(t *testing.T) {
	cases := []struct {
		r, g, b, want int
	}{
		{0, 0, 0, 30},
		{255, 0, 0, 91},
		{0, 255, 0, 92},
		{0, 0, 255, 94},
		{255, 255, 255, 97},
		{128, 0, 0, 31},
	}
	for _, c := range cases {
		if got := RGBToAnsi16(c.r, c.g, c.b); got != c.want {
			t.Errorf("RGBToAnsi16(%d,%d,%d) = %d; want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestAnsi256ToAnsi16(t *testing.T) {
	cases := []struct {
		n, want int
	}{
		{16, 30},
		{196, 91},
		{231, 97},
		{46, 92},
	}
	for _, c := range cases {
		if got := Ansi256ToAnsi16(c.n); got != c.want {
			t.Errorf("Ansi256ToAnsi16(%d) = %d; want %d", c.n, got, c.want)
		}
	}
}

func TestRGBToHSL(t *testing.T) {
	cases := []struct {
		r, g, b int
		h, s, l int
	}{
		{255, 0, 0, 0, 100, 50},
		{0, 255, 0, 120, 100, 50},
		{0, 0, 255, 240, 100, 50},
		{255, 255, 255, 0, 0, 100},
		{0, 0, 0, 0, 0, 0},
		{128, 128, 128, 0, 0, 50},
	}
	for _, c := range cases {
		h, s, l := RGBToHSL(c.r, c.g, c.b)
		if h != c.h || s != c.s || l != c.l {
			t.Errorf("RGBToHSL(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.r, c.g, c.b, h, s, l, c.h, c.s, c.l)
		}
	}
}

func TestHSLToRGB(t *testing.T) {
	cases := []struct {
		h, s, l int
		r, g, b int
	}{
		{0, 100, 50, 255, 0, 0},
		{120, 100, 50, 0, 255, 0},
		{240, 100, 50, 0, 0, 255},
		{0, 0, 100, 255, 255, 255},
		{0, 0, 0, 0, 0, 0},
		{60, 100, 50, 255, 255, 0},
	}
	for _, c := range cases {
		r, g, b := HSLToRGB(c.h, c.s, c.l)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("HSLToRGB(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.h, c.s, c.l, r, g, b, c.r, c.g, c.b)
		}
	}
}

func TestRGBToHSV(t *testing.T) {
	cases := []struct {
		r, g, b int
		h, s, v int
	}{
		{255, 0, 0, 0, 100, 100},
		{0, 255, 0, 120, 100, 100},
		{0, 0, 255, 240, 100, 100},
		{255, 255, 255, 0, 0, 100},
		{0, 0, 0, 0, 0, 0},
		{128, 128, 128, 0, 0, 50},
	}
	for _, c := range cases {
		h, s, v := RGBToHSV(c.r, c.g, c.b)
		if h != c.h || s != c.s || v != c.v {
			t.Errorf("RGBToHSV(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.r, c.g, c.b, h, s, v, c.h, c.s, c.v)
		}
	}
}

func TestHSVToRGB(t *testing.T) {
	cases := []struct {
		h, s, v int
		r, g, b int
	}{
		{0, 100, 100, 255, 0, 0},
		{120, 100, 100, 0, 255, 0},
		{240, 100, 100, 0, 0, 255},
		{0, 0, 100, 255, 255, 255},
		{0, 0, 0, 0, 0, 0},
		{60, 100, 100, 255, 255, 0},
	}
	for _, c := range cases {
		r, g, b := HSVToRGB(c.h, c.s, c.v)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("HSVToRGB(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.h, c.s, c.v, r, g, b, c.r, c.g, c.b)
		}
	}
}

func TestRGBToHWB(t *testing.T) {
	cases := []struct {
		r, g, b  int
		h, w, bl int
	}{
		{255, 0, 0, 0, 0, 0},
		{0, 255, 0, 120, 0, 0},
		{0, 0, 255, 240, 0, 0},
		{255, 255, 255, 0, 100, 0},
		{0, 0, 0, 0, 0, 100},
	}
	for _, c := range cases {
		h, w, bl := RGBToHWB(c.r, c.g, c.b)
		if h != c.h || w != c.w || bl != c.bl {
			t.Errorf("RGBToHWB(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.r, c.g, c.b, h, w, bl, c.h, c.w, c.bl)
		}
	}
}

func TestHWBToRGB(t *testing.T) {
	cases := []struct {
		h, w, bl int
		r, g, b  int
	}{
		{0, 0, 0, 255, 0, 0},
		{120, 0, 0, 0, 255, 0},
		{240, 0, 0, 0, 0, 255},
		{0, 100, 0, 255, 255, 255},
		{0, 0, 100, 0, 0, 0},
	}
	for _, c := range cases {
		r, g, b := HWBToRGB(c.h, c.w, c.bl)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("HWBToRGB(%d,%d,%d) = %d,%d,%d; want %d,%d,%d", c.h, c.w, c.bl, r, g, b, c.r, c.g, c.b)
		}
	}
}

// TestConversionRoundTrips verifies primaries survive an RGB->model->RGB trip.
func TestConversionRoundTrips(t *testing.T) {
	primaries := [][3]int{
		{255, 0, 0}, {0, 255, 0}, {0, 0, 255},
		{255, 255, 0}, {0, 255, 255}, {255, 0, 255},
		{0, 0, 0}, {255, 255, 255},
	}
	for _, p := range primaries {
		h, s, l := RGBToHSL(p[0], p[1], p[2])
		if r, g, b := HSLToRGB(h, s, l); r != p[0] || g != p[1] || b != p[2] {
			t.Errorf("HSL round-trip %v -> %d,%d,%d", p, r, g, b)
		}
		h, s, v := RGBToHSV(p[0], p[1], p[2])
		if r, g, b := HSVToRGB(h, s, v); r != p[0] || g != p[1] || b != p[2] {
			t.Errorf("HSV round-trip %v -> %d,%d,%d", p, r, g, b)
		}
		h, w, bl := RGBToHWB(p[0], p[1], p[2])
		if r, g, b := HWBToRGB(h, w, bl); r != p[0] || g != p[1] || b != p[2] {
			t.Errorf("HWB round-trip %v -> %d,%d,%d", p, r, g, b)
		}
	}
}

func BenchmarkRGBToHSL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = RGBToHSL(i&0xff, (i>>8)&0xff, (i>>16)&0xff)
	}
}

func BenchmarkHSLToRGB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = HSLToRGB(i%360, i%101, (i*7)%101)
	}
}

func ExampleRGBToHex() {
	fmt.Println(RGBToHex(255, 136, 0))
	// Output: #ff8800
}
