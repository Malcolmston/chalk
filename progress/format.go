package progress

import (
	"fmt"
	"strconv"
	"time"
)

// clamp keeps a value in [0, total], leaving it unbounded above when total is
// unknown (non-positive).
func clamp(n, total int64) int64 {
	if n < 0 {
		return 0
	}
	if total > 0 && n > total {
		return total
	}
	return n
}

// iecUnits are the binary (1024-based) units. IEC spelling is deliberate: a
// progress bar over bytes counts memory and file sizes, where 1536 bytes is
// 1.5 KiB, and calling it "1.5 KB" would be a factor-1.024 lie.
var iecUnits = [...]string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

// FormatBytes formats a byte count with IEC units: 512 becomes "512 B", 1536
// becomes "1.5 KiB". Bytes are shown as a whole number because a fraction of a
// byte is meaningless; larger units keep one decimal so the number moves while a
// transfer is running.
func FormatBytes(n int64) string { return formatBytesFloat(float64(n)) }

func formatBytesFloat(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	i := 0
	for f >= 1024 && i < len(iecUnits)-1 {
		f /= 1024
		i++
	}
	var s string
	if i == 0 {
		s = strconv.FormatFloat(f, 'f', 0, 64) + " B"
	} else {
		s = strconv.FormatFloat(f, 'f', 1, 64) + " " + iecUnits[i]
	}
	if neg {
		return "-" + s
	}
	return s
}

// formatAmount renders a counter either as a plain integer or as a byte size.
func formatAmount(n int64, bytes bool) string {
	if bytes {
		return FormatBytes(n)
	}
	return strconv.FormatInt(n, 10)
}

// formatRate renders a per-second rate. Counts get at most one decimal (and none
// once they are large enough that a decimal is noise); byte rates reuse the IEC
// formatting.
func formatRate(perSec float64, bytes bool) string {
	if bytes {
		return formatBytesFloat(perSec) + "/s"
	}
	if perSec >= 100 || perSec == 0 {
		return strconv.FormatFloat(perSec, 'f', 0, 64) + "/s"
	}
	return strconv.FormatFloat(perSec, 'f', 1, 64) + "/s"
}

// formatDuration renders a duration compactly and at fixed width per magnitude:
// "9s", "2m03s", "1h02m03s". Sub-second durations round down to "0s" rather than
// printing a jittery fraction, and a negative duration (a clock that went
// backwards) is treated as zero.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int64(d / time.Second)
	h, m, s := total/3600, (total%3600)/60, total%60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// spinnerChars is an ASCII spinner: it needs no font support and cannot be
// mistaken for a box-drawing glyph in a log.
var spinnerChars = [...]string{"|", "/", "-", "\\"}

// spinnerFrame picks a spinner character from elapsed time, so it advances on
// the injected clock like the rest of the rendering.
func spinnerFrame(elapsed time.Duration) string {
	if elapsed < 0 {
		elapsed = 0
	}
	return spinnerChars[int(elapsed/pulseInterval)%len(spinnerChars)]
}
