package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
)

// kittyChunk is the payload size per Kitty escape sequence. The protocol requires
// base64 data to be split into chunks of at most 4096 bytes.
const kittyChunk = 4096

// DetectProtocol reports the graphics protocol the current environment is known
// to support, or [ProtocolANSI] when it cannot tell.
//
// The bias is deliberate and one-sided: guessing "no native protocol" costs a
// slightly blocky picture, while guessing wrong the other way sprays kilobytes of
// base64 across the user's screen. So only terminals that identify themselves
// unambiguously are accepted, and any multiplexer in the way (tmux or screen,
// which need explicit passthrough to forward these sequences) disqualifies the
// terminal outright.
func DetectProtocol() Protocol {
	return detectProtocolEnv(os.Getenv, os.LookupEnv)
}

// detectProtocolEnv is DetectProtocol with the environment injected, so the
// decision table can be tested without mutating the process environment.
func detectProtocolEnv(getenv func(string) string, lookup func(string) (string, bool)) Protocol {
	has := func(k string) bool { _, ok := lookup(k); return ok }
	termName := strings.ToLower(getenv("TERM"))

	// A multiplexer swallows unknown escape sequences (or worse, redraws around
	// them). Never send graphics through one.
	if has("TMUX") || strings.HasPrefix(termName, "screen") || strings.HasPrefix(termName, "tmux") {
		return ProtocolANSI
	}

	// Kitty and Ghostty both implement the Kitty graphics protocol natively.
	if termName == "xterm-kitty" || has("KITTY_WINDOW_ID") {
		return ProtocolKitty
	}
	if termName == "xterm-ghostty" || getenv("TERM_PROGRAM") == "ghostty" {
		return ProtocolKitty
	}

	// iTerm2 (which also sets LC_TERMINAL when logging in over ssh) and WezTerm
	// both implement the OSC 1337 inline-image sequence.
	if getenv("LC_TERMINAL") == "iTerm2" {
		return ProtocolITerm2
	}
	switch getenv("TERM_PROGRAM") {
	case "iTerm.app":
		// Inline images arrived in iTerm 3; refuse when the version is unreadable.
		major, err := strconv.Atoi(strings.SplitN(getenv("TERM_PROGRAM_VERSION"), ".", 2)[0])
		if err == nil && major >= 3 {
			return ProtocolITerm2
		}
		return ProtocolANSI
	case "WezTerm":
		return ProtocolITerm2
	}
	return ProtocolANSI
}

// encodePNG re-encodes an image for a native protocol. Both protocols accept PNG,
// which is lossless and in the standard library, so there is no reason to pick
// anything else.
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("image: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// renderITerm2 builds an iTerm2 inline-image sequence sized to cols by rows
// cells: OSC 1337 ; File = <args> : <base64> BEL.
func renderITerm2(img image.Image, cols, rows int) (string, error) {
	data, err := encodePNG(img)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("\x1b]1337;File=inline=1;preserveAspectRatio=1")
	sb.WriteString(";size=" + strconv.Itoa(len(data)))
	sb.WriteString(";width=" + strconv.Itoa(cols))
	sb.WriteString(";height=" + strconv.Itoa(rows))
	sb.WriteByte(':')
	sb.WriteString(base64.StdEncoding.EncodeToString(data))
	sb.WriteString("\a\n")
	return sb.String(), nil
}

// renderKitty builds a Kitty graphics sequence: a=T transmits and displays,
// f=100 says the payload is a PNG file, c/r place it in a cols by rows cell box.
// The base64 payload is split into m=1 continuation chunks with a final m=0,
// which the protocol requires above 4096 bytes per escape.
func renderKitty(img image.Image, cols, rows int) (string, error) {
	data, err := encodePNG(img)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(data)

	var sb strings.Builder
	first := true
	for len(b64) > 0 {
		n := kittyChunk
		if n > len(b64) {
			n = len(b64)
		}
		chunk, rest := b64[:n], b64[n:]
		more := 0
		if len(rest) > 0 {
			more = 1
		}
		sb.WriteString("\x1b_G")
		if first {
			sb.WriteString("a=T,f=100,c=" + strconv.Itoa(cols) + ",r=" + strconv.Itoa(rows) + ",")
			first = false
		}
		sb.WriteString("m=" + strconv.Itoa(more) + ";")
		sb.WriteString(chunk)
		sb.WriteString("\x1b\\")
		b64 = rest
	}
	// The image is drawn without moving the cursor down, so advance past it the
	// way the ANSI renderer does: one newline per occupied row.
	sb.WriteString(strings.Repeat("\n", rows))
	return sb.String(), nil
}
