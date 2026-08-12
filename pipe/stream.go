package pipe

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Errors returned by the streaming helpers. ErrCanceled matches the sibling
// prompts package's name for "the user or the caller gave up", so a program can
// treat cancellation the same way whether it came from Ctrl-C at a prompt or
// from a canceled context here.
var (
	// ErrNoInput reports that there is nothing to read because the input is an
	// interactive terminal rather than a pipe or a file. It is returned instead
	// of blocking, which is the entire point: a CLI that waits silently for
	// input the user never intended to type looks like a hang.
	ErrNoInput = errors.New("pipe: no piped input (stdin is a terminal)")
	// ErrCanceled reports that the caller's context was canceled.
	ErrCanceled = errors.New("pipe: canceled")
	// ErrTimeout reports that Timeout elapsed before the read finished.
	ErrTimeout = errors.New("pipe: timed out waiting for input")
)

// ReadConfig configures [ReadAll].
type ReadConfig struct {
	// In is the stream to read. Defaults to os.Stdin.
	In io.Reader
	// Ctx cancels the read. Defaults to context.Background().
	Ctx context.Context
	// Timeout bounds the read. Zero means no bound, which is safe because a
	// terminal input is refused up front (see AllowTerminal).
	Timeout time.Duration
	// MaxBytes caps how much is read; further bytes are discarded and no error
	// is reported. Zero means unlimited. Use it when the input is untrusted:
	// "read all of stdin" is an unbounded allocation controlled by whoever is on
	// the other end of the pipe.
	MaxBytes int64
	// AllowTerminal permits reading from an interactive terminal. Off by
	// default so a program that expects piped data fails fast with [ErrNoInput]
	// instead of hanging; turn it on for a tool that legitimately reads typed
	// input until EOF, the way cat does.
	AllowTerminal bool
}

// ReadAll reads the whole of cfg.In and returns it.
//
// When the input is an interactive terminal and AllowTerminal is not set, it
// returns [ErrNoInput] immediately without reading a byte. When Timeout or Ctx
// is in play the read happens on a goroutine so the wait can be abandoned;
// see the note on that below.
//
// The goroutine cannot be killed — no read in Go can be interrupted from
// outside — so on timeout or cancellation it stays parked in the blocking read
// until the writer closes or writes. It holds only its own buffer, writes only
// to a buffered channel of size one, and touches nothing the caller can see, so
// it neither leaks unboundedly nor races with the returning caller. That is the
// honest trade: the *caller* is freed on time, the descriptor is not.
func ReadAll(cfg ReadConfig) ([]byte, error) {
	in := cfg.In
	if in == nil {
		in = os.Stdin
	}
	if !cfg.AllowTerminal && readerIsTerminal(in) {
		return nil, ErrNoInput
	}

	r := io.Reader(in)
	if cfg.MaxBytes > 0 {
		r = io.LimitReader(in, cfg.MaxBytes)
	}

	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Timeout <= 0 && ctx.Done() == nil {
		return io.ReadAll(r)
	}

	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	type result struct {
		data []byte
		err  error
	}
	// Buffered so the reader goroutine can always finish and exit even after the
	// caller has stopped listening.
	done := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		done <- result{data, err}
	}()

	select {
	case res := <-done:
		return res.data, res.err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) && cfg.Timeout > 0 {
			return nil, ErrTimeout
		}
		return nil, ErrCanceled
	}
}

// isTerminalFile is the seam the streaming helpers ask about interactivity
// through. It is a variable only so the refusal path can be exercised without a
// pseudo-terminal, which not every CI image provides; nothing outside tests
// reassigns it.
var isTerminalFile = IsTerminal

// readerIsTerminal reports whether r is an interactive terminal. A non-file
// reader (a strings.Reader in a test, a bytes.Buffer, a network stream) is never
// a terminal, which is what makes every path here testable without a TTY.
func readerIsTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && isTerminalFile(f)
}

// LinesConfig configures [Lines].
type LinesConfig struct {
	// In is the stream to read. Defaults to os.Stdin.
	In io.Reader
	// Out is where transformed lines are written. Defaults to os.Stdout.
	Out io.Writer
	// Transform maps one input line to its output. Returning false drops the
	// line entirely, which is how a filter is written. A nil Transform passes
	// every line through unchanged, making Lines a plain copy that normalizes
	// line endings.
	Transform func(line string) (string, bool)
	// MaxLineBytes caps the length of a single line. Defaults to
	// DefaultMaxLineBytes. A line longer than this fails with an error rather
	// than being silently truncated.
	MaxLineBytes int
	// Ctx cancels between lines, yielding [ErrCanceled] with the lines written
	// so far already flushed.
	Ctx context.Context
	// AllowTerminal permits reading from an interactive terminal; see
	// [ReadConfig.AllowTerminal].
	AllowTerminal bool
}

// DefaultMaxLineBytes is the default per-line cap for [Lines]: generous for real
// text, and an explicit ceiling rather than bufio.Scanner's silent 64 KiB one.
const DefaultMaxLineBytes = 1 << 20

// Lines reads cfg.In line by line, applies cfg.Transform, and writes the
// surviving lines to cfg.Out. It returns the number of lines written.
//
// Line endings are normalized: a trailing CR is stripped from the input line
// before Transform sees it, and every written line is terminated with a single
// LF, including the last one even when the input did not end with a newline.
// This is what makes the output usable as a pipe stage — a partial final line
// would otherwise glue itself to the next tool's first line.
//
// Output is buffered and flushed before returning, on the error paths too, so a
// caller that cancels still sees everything that was already transformed.
func Lines(cfg LinesConfig) (int, error) {
	in := cfg.In
	if in == nil {
		in = os.Stdin
	}
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	if !cfg.AllowTerminal && readerIsTerminal(in) {
		return 0, ErrNoInput
	}
	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	maxLine := cfg.MaxLineBytes
	if maxLine <= 0 {
		maxLine = DefaultMaxLineBytes
	}

	sc := bufio.NewScanner(in)
	// The starting buffer must not exceed the cap, or bufio.Scanner would only
	// notice an over-long line once it needed to grow past the initial capacity,
	// silently accepting lines up to that capacity.
	initial := 4096
	if maxLine < initial {
		initial = maxLine
	}
	sc.Buffer(make([]byte, 0, initial), maxLine)
	bw := bufio.NewWriter(out)

	written := 0
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return written, flushWith(bw, ErrCanceled)
		}
		line := strings.TrimSuffix(sc.Text(), "\r")
		if cfg.Transform != nil {
			s, keep := cfg.Transform(line)
			if !keep {
				continue
			}
			line = s
		}
		if _, err := bw.WriteString(line); err != nil {
			return written, flushWith(bw, err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			return written, flushWith(bw, err)
		}
		written++
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			err = fmt.Errorf("pipe: line exceeds %d bytes: %w", maxLine, err)
		}
		return written, flushWith(bw, err)
	}
	return written, bw.Flush()
}

// flushWith flushes bw and returns err, preferring err over any flush failure
// because err is the reason the caller stopped.
func flushWith(bw *bufio.Writer, err error) error {
	if ferr := bw.Flush(); ferr != nil && err == nil {
		return ferr
	}
	return err
}

// SplitLines splits s the way [Lines] does: on LF, tolerating CRLF, with no
// empty trailing element for a string that ends in a newline. It is exported
// because a caller writing a Transform usually needs the same rule for a
// multi-line replacement.
func SplitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	parts := strings.Split(s, "\n")
	for i, p := range parts {
		parts[i] = strings.TrimSuffix(p, "\r")
	}
	return parts
}
