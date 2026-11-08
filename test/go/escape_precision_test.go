package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func escapeApply(fn func(*int) int, p *int) int { return fn(p) }

//go:noinline
func escapeRead(p *int) int { return *p }

//go:noinline
func escapeImmutable(x int) func() int { return func() int { return x } }

//go:noinline
func escapeShared(x int) (func() int, func()) { return func() int { return x }, func() { x++ } }

//go:noinline
func escapeEqual(a, b string) bool { return a == b }

func TestEscapeCallSiteAndCapture(t *testing.T) {
	x := 13
	if got := escapeApply(escapeRead, &x); got != 13 {
		t.Fatal(got)
	}
	var closures []func() int
	for i := 0; i < 32; i++ {
		closures = append(closures, escapeImmutable(i))
	}
	runtime.GC()
	for i, fn := range closures {
		if got := fn(); got != i {
			t.Fatalf("capture %d = %d", i, got)
		}
	}
	read, write := escapeShared(20)
	write()
	runtime.GC()
	write()
	if got := read(); got != 22 {
		t.Fatal(got)
	}
}

func TestEscapeStringSnapshot(t *testing.T) {
	b := []byte("before")
	empty := []byte(string(b[:0]))
	if empty == nil || len(empty) != 0 {
		t.Fatal("empty conversion must be non-nil")
	}
	empty = append(empty, 7)
	if empty[0] != 7 {
		t.Fatal(empty)
	}
	s := string(b)
	b[0] = 'B'
	if !escapeEqual(s, "before") || escapeEqual(s, string(b)) {
		t.Fatal("lost string snapshot", s, string(b))
	}
	for _, want := range []string{"Before", "other"} {
		s := string(b)
		if got := escapeEqual(s, want) || s == "Before"; !got {
			t.Fatal("read region", s, want)
		}
	}
}
