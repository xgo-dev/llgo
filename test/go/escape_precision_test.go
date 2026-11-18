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
func escapeLoopCaptures(local bool) []func() int {
	var x int
	var fs []func() int
	for i := 0; i < 3; i++ {
		if local {
			x := i
			fs = append(fs, func() int { return x })
		} else {
			x = i
			fs = append(fs, func() int { return x })
		}
	}
	return fs
}

//go:noinline
func escapeGotoCaptures() []func() int {
	var x int
	var fs []func() int
	i := 0
again:
	x = i
	fs = append(fs, func() int { return x })
	i++
	if i < 3 {
		goto again
	}
	return fs
}

//go:noinline
func escapeNestedCaptures() []func() int {
	var fs []func() int
	for i := 0; i < 3; i++ {
		var x int
		for j := 0; j < 3; j++ {
			x = i + j
			fs = append(fs, func() int { return x })
		}
	}
	return fs
}

func TestEscapeRepeatedCaptureInitialization(t *testing.T) {
	for _, tc := range []struct {
		name string
		fs   []func() int
		want []int
	}{
		{"shared loop", escapeLoopCaptures(false), []int{2, 2, 2}},
		{"loop local", escapeLoopCaptures(true), []int{0, 1, 2}},
		{"backward goto", escapeGotoCaptures(), []int{2, 2, 2}},
		{"nested loop", escapeNestedCaptures(), []int{2, 2, 2, 3, 3, 3, 4, 4, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime.GC()
			for i, fn := range tc.fs {
				if got := fn(); got != tc.want[i] {
					t.Fatalf("capture %d = %d, want %d", i, got, tc.want[i])
				}
			}
		})
	}
}

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
