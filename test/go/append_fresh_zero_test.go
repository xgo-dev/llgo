package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func appendFreshZeroBytes(n int) []byte { return append([]byte(nil), make([]byte, n)...) }

//go:noinline
func appendFreshZeroConstant() []byte { return append([]byte(nil), make([]byte, 4)...) }

//go:noinline
func appendFreshZeroPointers(n int) []*int { return append([]*int(nil), make([]*int, n)...) }

//go:noinline
func appendFreshZeroWide(n uint64) []byte { return append([]byte(nil), make([]byte, n)...) }

//go:noinline
func appendFreshZeroOverflow(n int) [][16]byte {
	return append([][16]byte(nil), make([][16]byte, n)...)
}

func appendFreshZeroGeneric[T any](n int) []T { return append([]T(nil), make([]T, n)...) }

//go:noinline
func appendFreshZeroLoopCopies(n int) [][]byte {
	source := make([]byte, n)
	var out [][]byte
	for i := 0; i < 3; i++ {
		out = append(out, append([]byte(nil), source...))
	}
	return out
}

//go:noinline
func appendFreshZeroConstantLoopCopies() [][]byte {
	source := make([]byte, 8)
	var out [][]byte
	for i := 0; i < 3; i++ {
		out = append(out, append([]byte(nil), source...))
	}
	return out
}

func TestAppendFreshZeroLoopCopies(t *testing.T) {
	for _, copies := range [][][]byte{appendFreshZeroLoopCopies(8), appendFreshZeroConstantLoopCopies()} {
		copies[0][0] = 7
		runtime.GC()
		if copies[0][0] != 7 || copies[1][0] != 0 || copies[2][0] != 0 {
			t.Fatalf("loop copies share storage: %d %d %d", copies[0][0], copies[1][0], copies[2][0])
		}
	}
}

//go:noinline
func appendFreshZeroNext(calls *int) int {
	*calls++
	return 4
}

//go:noinline
func appendFreshZeroClobber() int {
	var values [256]byte
	for i := range values {
		values[i] = byte(i)
	}
	runtime.KeepAlive(values)
	return int(values[255])
}

func TestAppendFreshZeroResult(t *testing.T) {
	for _, n := range []int{0, 1, 4, 16, 32, 33, 4096} {
		s := appendFreshZeroBytes(n)
		if len(s) != n || cap(s) < n || (s == nil) != (n == 0) {
			t.Fatalf("n=%d: len=%d cap=%d nil=%v", n, len(s), cap(s), s == nil)
		}
		for i := range s {
			s[i] = byte(i + 1)
		}
		appendFreshZeroClobber()
		runtime.GC()
		for i, value := range s {
			if value != byte(i+1) {
				t.Fatalf("returned temporary storage: n=%d index=%d value=%d", n, i, value)
			}
		}
	}
	s := appendFreshZeroConstant()
	s[0], s[3] = 17, 23
	appendFreshZeroClobber()
	if s[0] != 17 || s[3] != 23 {
		t.Fatal("constant append returned temporary storage")
	}
	p := appendFreshZeroPointers(4)
	for _, value := range p {
		if value != nil {
			t.Fatal("pointer slice was not zeroed")
		}
	}
	x := new(int)
	*x = 42
	p[3] = x
	runtime.GC()
	if *p[3] != 42 {
		t.Fatal("pointer slice lost its contents")
	}
	if s := appendFreshZeroGeneric[struct{ P *int }](4); len(s) != 4 || s[3].P != nil {
		t.Fatal("generic append did not return zeroed storage")
	}
}

func TestAppendFreshZeroChecksAndOwnership(t *testing.T) {
	for _, n := range []int{-1, -17} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("negative make length %d did not panic", n)
				}
			}()
			_ = appendFreshZeroBytes(n)
		}()
	}
	calls := 0
	if s := append([]byte(nil), make([]byte, appendFreshZeroNext(&calls))...); len(s) != 4 || calls != 1 {
		t.Fatalf("make length evaluated %d times", calls)
	}
	source := make([]byte, 4)
	copy := append([]byte(nil), source...)
	source[0] = 99
	if copy[0] != 0 {
		t.Fatal("append aliased shared source storage")
	}
	backing := []byte{1, 2, 3, 4}
	got := append(backing[:1], make([]byte, 2)...)
	if got[0] != 1 || got[1] != 0 || got[2] != 0 || backing[3] != 4 {
		t.Fatalf("non-nil append changed ownership or failed to zero: %v", backing)
	}
	type Bytes []byte
	if s := append(Bytes(nil), make(Bytes, 0)...); s != nil {
		t.Fatal("empty named append is non-nil")
	}
	max := int(^uint(0) >> 1)
	if s := append([]struct{}(nil), make([]struct{}, max)...); len(s) != max {
		t.Fatal("zero-sized elements lost their length")
	}
	for _, call := range []func(){
		func() { _ = appendFreshZeroWide(^uint64(0)) },
		func() { _ = appendFreshZeroOverflow(max) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("overflowing make length did not panic")
				}
			}()
			call()
		}()
	}
}
