package pipe

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// blockingReader blocks in Read until release is closed, then reports EOF. It
// stands in for a pipe nobody is writing to, which is the situation the timeout
// and cancellation paths exist for -- and it needs no real pipe, so the test is
// deterministic.
type blockingReader struct {
	release chan struct{}
}

func newBlockingReader(t *testing.T) *blockingReader {
	t.Helper()
	br := &blockingReader{release: make(chan struct{})}
	// Releasing on cleanup lets every parked reader goroutine finish and exit, so
	// the package's tests do not accumulate blocked goroutines.
	t.Cleanup(func() { close(br.release) })
	return br
}

func (b *blockingReader) Read([]byte) (int, error) {
	<-b.release
	return 0, io.EOF
}

func TestReadAll(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ReadConfig
		want     string
		wantErr  error
		wantSame bool
	}{
		{name: "reads a fake reader", cfg: ReadConfig{In: strings.NewReader("a\nb\n")}, want: "a\nb\n"},
		{name: "empty input", cfg: ReadConfig{In: strings.NewReader("")}, want: ""},
		{name: "buffer source", cfg: ReadConfig{In: bytes.NewBufferString("data")}, want: "data"},
		{name: "MaxBytes truncates", cfg: ReadConfig{In: strings.NewReader("0123456789"), MaxBytes: 4}, want: "0123"},
		{name: "MaxBytes above size", cfg: ReadConfig{In: strings.NewReader("hi"), MaxBytes: 100}, want: "hi"},
		{name: "reader error surfaces", cfg: ReadConfig{In: errReader{}}, wantErr: errBoom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadAll(tc.cfg)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && string(got) != tc.want {
				t.Errorf("data = %q, want %q", got, tc.want)
			}
		})
	}
}

var errBoom = errors.New("boom")

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBoom }

// TestReadAllRedirectedFileIsNotRefused proves the non-TTY path: a real
// descriptor that is a file, not a terminal, is read rather than rejected.
func TestReadAllRedirectedFileIsNotRefused(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/in"
	if err := os.WriteFile(path, []byte("from a file\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	got, err := ReadAll(ReadConfig{In: f})
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "from a file\n" {
		t.Errorf("data = %q", got)
	}
}

// TestTerminalInputIsRefused covers the hang bug: a real terminal is refused
// with ErrNoInput instead of blocking. It needs a controlling terminal, so it
// skips where there is none (CI), and the AllowTerminal opt-out is asserted
// without reading anything.
func TestTerminalInputIsRefused(t *testing.T) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("no controlling terminal available")
	}
	defer func() { _ = tty.Close() }()
	if !IsTerminal(tty) {
		t.Skip("/dev/tty is not a terminal here")
	}

	if _, err := ReadAll(ReadConfig{In: tty}); !errors.Is(err, ErrNoInput) {
		t.Errorf("ReadAll err = %v, want ErrNoInput", err)
	}
	var buf bytes.Buffer
	if _, err := Lines(LinesConfig{In: tty, Out: &buf}); !errors.Is(err, ErrNoInput) {
		t.Errorf("Lines err = %v, want ErrNoInput", err)
	}
}

// TestReaderIsTerminalNeverTrueForFakes documents that every injected reader in
// these tests takes the non-terminal path.
func TestReaderIsTerminalNeverTrueForFakes(t *testing.T) {
	for _, r := range []io.Reader{strings.NewReader(""), &bytes.Buffer{}, newBlockingReader(t), errReader{}} {
		if readerIsTerminal(r) {
			t.Errorf("%T reported as a terminal", r)
		}
	}
}

func TestReadAllTimeout(t *testing.T) {
	start := time.Now()
	got, err := ReadAll(ReadConfig{In: newBlockingReader(t), Timeout: 20 * time.Millisecond})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
	if got != nil {
		t.Errorf("data = %q, want nil", got)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v, expected to give up promptly", elapsed)
	}
}

func TestReadAllCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if _, err := ReadAll(ReadConfig{In: newBlockingReader(t), Ctx: ctx}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
}

// TestReadAllConcurrent runs many timed reads at once; under -race it proves the
// reader goroutine shares nothing with the caller that returns before it.
func TestReadAllConcurrent(t *testing.T) {
	br := newBlockingReader(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ReadAll(ReadConfig{In: br, Timeout: 5 * time.Millisecond}); !errors.Is(err, ErrTimeout) {
				t.Errorf("err = %v, want ErrTimeout", err)
			}
		}()
	}
	wg.Wait()
}

// TestReadAllCtxDoneAlreadyCanceled proves a canceled context short-circuits
// rather than reading.
func TestReadAllCtxDoneAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadAll(ReadConfig{In: newBlockingReader(t), Ctx: ctx}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
}

// TestInteractiveInputIsRefusedWithoutATTY exercises the refusal path with no
// pseudo-terminal at all, by swapping the interactivity seam. This is the bug the
// package exists to prevent: a program handed a keyboard where it expected a pipe
// must fail fast, not sit there silently.
func TestInteractiveInputIsRefusedWithoutATTY(t *testing.T) {
	f := tempFile(t)
	prev := isTerminalFile
	isTerminalFile = func(g *os.File) bool { return g == f }
	t.Cleanup(func() { isTerminalFile = prev })

	if _, err := ReadAll(ReadConfig{In: f}); !errors.Is(err, ErrNoInput) {
		t.Errorf("ReadAll err = %v, want ErrNoInput", err)
	}
	var out bytes.Buffer
	if n, err := Lines(LinesConfig{In: f, Out: &out}); !errors.Is(err, ErrNoInput) || n != 0 {
		t.Errorf("Lines = %d, %v; want 0, ErrNoInput", n, err)
	}
	if out.Len() != 0 {
		t.Errorf("out = %q, want nothing written", out.String())
	}

	// AllowTerminal opts back in: the same descriptor is now read (it is an empty
	// temp file, so the read completes immediately).
	if _, err := ReadAll(ReadConfig{In: f, AllowTerminal: true}); err != nil {
		t.Errorf("ReadAll with AllowTerminal: %v", err)
	}
	if _, err := Lines(LinesConfig{In: f, Out: &out, AllowTerminal: true}); err != nil {
		t.Errorf("Lines with AllowTerminal: %v", err)
	}
}

func TestLines(t *testing.T) {
	upper := func(s string) (string, bool) { return strings.ToUpper(s), true }
	dropBlank := func(s string) (string, bool) { return s, strings.TrimSpace(s) != "" }

	tests := []struct {
		name      string
		in        string
		transform func(string) (string, bool)
		wantOut   string
		wantCount int
	}{
		{name: "nil transform passes through", in: "a\nb\n", wantOut: "a\nb\n", wantCount: 2},
		{name: "adds missing final newline", in: "a\nb", wantOut: "a\nb\n", wantCount: 2},
		{name: "normalizes CRLF", in: "a\r\nb\r\n", wantOut: "a\nb\n", wantCount: 2},
		{name: "transform applied", in: "ab\ncd\n", transform: upper, wantOut: "AB\nCD\n", wantCount: 2},
		{name: "filter drops lines", in: "a\n\n \nb\n", transform: dropBlank, wantOut: "a\nb\n", wantCount: 2},
		{name: "empty input writes nothing", in: "", wantOut: "", wantCount: 0},
		{name: "blank lines preserved", in: "\n\n", wantOut: "\n\n", wantCount: 2},
		{name: "transform may expand", in: "x\n", transform: func(s string) (string, bool) { return s + s, true }, wantOut: "xx\n", wantCount: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			n, err := Lines(LinesConfig{In: strings.NewReader(tc.in), Out: &out, Transform: tc.transform})
			if err != nil {
				t.Fatalf("Lines: %v", err)
			}
			if n != tc.wantCount {
				t.Errorf("count = %d, want %d", n, tc.wantCount)
			}
			if out.String() != tc.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tc.wantOut)
			}
		})
	}
}

func TestLinesLineTooLong(t *testing.T) {
	var out bytes.Buffer
	n, err := Lines(LinesConfig{
		In:           strings.NewReader("ok\n" + strings.Repeat("x", 50) + "\n"),
		Out:          &out,
		MaxLineBytes: 8,
	})
	if err == nil {
		t.Fatal("want an error for an over-long line")
	}
	if !strings.Contains(err.Error(), "exceeds 8 bytes") {
		t.Errorf("err = %v, want it to name the limit", err)
	}
	if n != 1 || out.String() != "ok\n" {
		t.Errorf("count = %d, out = %q; the lines before the failure should be flushed", n, out.String())
	}
}

func TestLinesCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	n, err := Lines(LinesConfig{
		In:  strings.NewReader("a\nb\nc\n"),
		Out: &out,
		Ctx: ctx,
		Transform: func(s string) (string, bool) {
			if s == "b" {
				cancel()
			}
			return s, true
		},
	})
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
	if n != 2 || out.String() != "a\nb\n" {
		t.Errorf("count = %d, out = %q; want the lines up to cancellation flushed", n, out.String())
	}
	cancel()
}

func TestLinesWriteError(t *testing.T) {
	// A tiny transform feeding a writer that always fails: the error must reach
	// the caller rather than being swallowed by the buffered writer.
	_, err := Lines(LinesConfig{In: strings.NewReader(strings.Repeat("line\n", 10000)), Out: errWriter{}})
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want errBoom", err)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errBoom }

func TestSplitLines(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a"}},
		{"a\nb", []string{"a", "b"}},
		{"a\r\nb\r\n", []string{"a", "b"}},
		{"\n", []string{""}},
	}
	for _, tc := range tests {
		got := SplitLines(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("SplitLines(%q) = %q, want %q", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("SplitLines(%q) = %q, want %q", tc.in, got, tc.want)
				break
			}
		}
	}
}
