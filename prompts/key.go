package prompts

import (
	"bufio"
	"io"
)

// keyType enumerates the recognized key categories.
type keyType int

const (
	keyRune keyType = iota
	keyEnter
	keyBackspace
	keySpace
	keyTab
	keyUp
	keyDown
	keyLeft
	keyRight
	keyEsc
	keyCtrlC
	keyEOF
)

// key is a single decoded keypress.
type key struct {
	typ keyType
	r   rune
}

// keyReader decodes keypresses (including arrow escape sequences) from a byte
// stream. It works on a raw-mode terminal as well as a scripted byte reader,
// which makes the prompts testable without a TTY.
type keyReader struct {
	br *bufio.Reader
}

func newKeyReader(r io.Reader) *keyReader {
	return &keyReader{br: bufio.NewReader(r)}
}

// read decodes the next key.
func (k *keyReader) read() key {
	r, _, err := k.br.ReadRune()
	if err != nil {
		return key{typ: keyEOF}
	}
	switch r {
	case '\r', '\n':
		return key{typ: keyEnter}
	case 127, 8:
		return key{typ: keyBackspace}
	case 3:
		return key{typ: keyCtrlC}
	case 9:
		return key{typ: keyTab}
	case ' ':
		return key{typ: keySpace, r: ' '}
	case 27: // ESC — possibly a CSI arrow sequence.
		next, _, err := k.br.ReadRune()
		if err != nil {
			return key{typ: keyEsc}
		}
		if next != '[' && next != 'O' {
			return key{typ: keyEsc}
		}
		dir, _, err := k.br.ReadRune()
		if err != nil {
			return key{typ: keyEsc}
		}
		switch dir {
		case 'A':
			return key{typ: keyUp}
		case 'B':
			return key{typ: keyDown}
		case 'C':
			return key{typ: keyRight}
		case 'D':
			return key{typ: keyLeft}
		default:
			return key{typ: keyEsc}
		}
	default:
		return key{typ: keyRune, r: r}
	}
}
