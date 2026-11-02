package fmt_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

type literalWriter struct {
	writes, strings int
	text            string
	n               int
	err             error
}

func (w *literalWriter) Write(p []byte) (int, error) {
	w.writes++
	w.text = string(p)
	return w.n, w.err
}

func (w *literalWriter) WriteString(s string) (int, error) {
	w.strings++
	return len(s), nil
}

func TestLiteralFormattingWriteContract(t *testing.T) {
	errWrite := errors.New("writer failure")
	w := &literalWriter{n: 2, err: errWrite}
	n, err := fmt.Fprintf(w, "你好\x00%%!")
	if n != 2 || err != errWrite || w.writes != 1 || w.strings != 0 || w.text != "你好\x00%!" {
		t.Fatalf("write = %d, %v, writer=%+v", n, err, w)
	}
	w = &literalWriter{err: errWrite}
	n, err = fmt.Fprintf(w, "")
	if n != 0 || err != errWrite || w.writes != 1 || w.strings != 0 || w.text != "" {
		t.Fatalf("empty write = %d, %v, writer=%+v", n, err, w)
	}
	w = &literalWriter{n: 1}
	n, err = fmt.Fprintf(w, "abcdef")
	if n != 1 || err != nil || w.writes != 1 {
		t.Fatalf("short write = %d, %v, writer=%+v", n, err, w)
	}
	if got := fmt.Sprintf("你好\x00%%%%"); got != "你好\x00%%" {
		t.Fatalf("literal Sprintf = %q", got)
	}
}

func TestLiteralPrintfCurrentStdout(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "literal")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	previous := os.Stdout
	defer func() { os.Stdout = previous }()
	os.Stdout = f
	n, err := fmt.Printf("literal%%\n")
	os.Stdout = previous
	if n != len("literal%\n") || err != nil {
		t.Fatalf("Printf = %d, %v", n, err)
	}
	got, err := os.ReadFile(f.Name())
	if err != nil || string(got) != "literal%\n" {
		t.Fatalf("redirected stdout = %q, %v", got, err)
	}
}

type literalFormatter struct{ calls *int }

func (v literalFormatter) Format(s fmt.State, verb rune) {
	*v.calls++
	s.Write([]byte("formatted"))
}

func TestLiteralFormattingFallback(t *testing.T) {
	calls := 0
	if got := fmt.Sprintf("%v", literalFormatter{&calls}); got != "formatted" || calls != 1 {
		t.Fatalf("Formatter = %q, calls=%d", got, calls)
	}
	for _, format := range []string{"%", "%d", "%[1]s", "%5%", "%.2f"} {
		var w literalWriter
		fmt.Fprintf(&w, format, []any(nil)...)
		if want := fmt.Sprintf(format, []any(nil)...); w.text != want || w.writes != 1 {
			t.Fatalf("format %q = %q, want %q", format, w.text, want)
		}
	}
}

type literalNotifyWriter struct{ writes chan string }

func (w literalNotifyWriter) Write(p []byte) (int, error) {
	w.writes <- string(p)
	return len(p), nil
}

func TestLiteralFormattingDeferredAndGo(t *testing.T) {
	w := literalNotifyWriter{writes: make(chan string, 1)}
	func() { defer fmt.Fprintf(w, "deferred%%") }()
	if got := <-w.writes; got != "deferred%" {
		t.Fatalf("deferred write = %q", got)
	}
	go fmt.Fprintf(w, "goroutine%%")
	if got := <-w.writes; got != "goroutine%" {
		t.Fatalf("goroutine write = %q", got)
	}
}

// The parameter keeps the reference call on fmt's dynamic-format path.
func dynamicPrintf(format string) (int, error) { return fmt.Printf(format, []any(nil)...) }

func capturePrintf(t *testing.T, closed bool, print func() (int, error)) (string, int, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "printf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if closed {
		f.Close()
	}
	previous := os.Stdout
	defer func() { os.Stdout = previous }()
	os.Stdout = f
	n, writeErr := print()
	os.Stdout = previous
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), n, writeErr
}

func TestLiteralPrintfEmptyAndClosedStdout(t *testing.T) {
	for _, test := range []struct {
		name, format, want string
		literal            func() (int, error)
	}{
		{"empty", "", "", func() (int, error) { return fmt.Printf("") }},
		{"escaped", "literal%%", "literal%", func() (int, error) { return fmt.Printf("literal%%") }},
	} {
		for _, closed := range []bool{false, true} {
			text, n, err := capturePrintf(t, closed, test.literal)
			reference, rn, re := capturePrintf(t, closed, func() (int, error) { return dynamicPrintf(test.format) })
			if text != reference || n != rn || (err == nil) != (re == nil) ||
				errors.Is(err, os.ErrClosed) != errors.Is(re, os.ErrClosed) {
				t.Fatalf("%s closed=%v: literal=(%q,%d,%v), dynamic=(%q,%d,%v)", test.name, closed, text, n, err, reference, rn, re)
			}
			if closed {
				if text != "" || n != 0 || !errors.Is(err, os.ErrClosed) {
					t.Fatalf("closed stdout accepted write: %q,%d,%v", text, n, err)
				}
			} else if text != test.want || n != len(test.want) || err != nil {
				t.Fatalf("open stdout: %q,%d,%v", text, n, err)
			}
		}
	}
}

func TestLiteralPrintfDeferredAndGo(t *testing.T) {
	literal := func() (int, error) {
		func() { defer fmt.Printf("deferred%%") }()
		done := make(chan struct{})
		var n int
		var err error
		go func() {
			n, err = fmt.Printf("goroutine%%")
			close(done)
		}()
		<-done // The write and its result must complete before restoring stdout.
		return n, err
	}
	dynamic := func() (int, error) {
		func() { defer dynamicPrintf("deferred%%") }()
		done := make(chan struct{})
		var n int
		var err error
		go func() { n, err = dynamicPrintf("goroutine%%"); close(done) }()
		<-done
		return n, err
	}
	text, n, err := capturePrintf(t, false, literal)
	reference, rn, re := capturePrintf(t, false, dynamic)
	if text != "deferred%goroutine%" || text != reference || n != len("goroutine%") || n != rn || err != nil || re != nil {
		t.Fatalf("literal=(%q,%d,%v), dynamic=(%q,%d,%v)", text, n, err, reference, rn, re)
	}
}
