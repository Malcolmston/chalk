package progress

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1536 * 1024, "1.5 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TiB"},
		{-1536, "-1.5 KiB"},
	}
	for _, tc := range tests {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		in    float64
		bytes bool
		want  string
	}{
		{0, false, "0/s"},
		{2.5, false, "2.5/s"},
		{99.94, false, "99.9/s"},
		{100, false, "100/s"},
		{12345, false, "12345/s"},
		{0, true, "0 B/s"},
		{768, true, "768 B/s"},
		{1536, true, "1.5 KiB/s"},
	}
	for _, tc := range tests {
		if got := formatRate(tc.in, tc.bytes); got != tc.want {
			t.Errorf("formatRate(%v, %v) = %q, want %q", tc.in, tc.bytes, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{-time.Second, "0s"},
		{0, "0s"},
		{999 * time.Millisecond, "0s"},
		{9 * time.Second, "9s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m00s"},
		{90 * time.Second, "1m30s"},
		{59*time.Minute + 59*time.Second, "59m59s"},
		{time.Hour, "1h00m00s"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1h02m03s"},
		{100 * time.Hour, "100h00m00s"},
	}
	for _, tc := range tests {
		if got := formatDuration(tc.in); got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSpinnerFrame(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{-time.Second, "|"},
		{0, "|"},
		{pulseInterval, "/"},
		{2 * pulseInterval, "-"},
		{3 * pulseInterval, "\\"},
		{4 * pulseInterval, "|"},
	}
	for _, tc := range tests {
		if got := spinnerFrame(tc.in); got != tc.want {
			t.Errorf("spinnerFrame(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMergeCharsetKeepsAPairOfBrackets(t *testing.T) {
	tests := []struct {
		name string
		in   Charset
		want Charset
	}{
		{"zero falls back entirely", Charset{}, DefaultCharset},
		{
			"one bracket set keeps the other empty",
			Charset{Left: "<"},
			Charset{Filled: DefaultCharset.Filled, Empty: DefaultCharset.Empty, Left: "<"},
		},
		{
			"filled and empty are independent",
			Charset{Filled: "#"},
			Charset{Filled: "#", Empty: DefaultCharset.Empty, Left: "[", Right: "]"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeCharset(tc.in); got != tc.want {
				t.Errorf("mergeCharset(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}
