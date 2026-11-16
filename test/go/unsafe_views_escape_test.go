package gotest

import (
	"runtime"
	"testing"
	"unsafe"
)

//go:noinline
func unsafeLocalViews(value byte) int {
	var data [16]byte
	for i := range data {
		data[i] = value + byte(i)
	}
	view := unsafe.Slice(unsafe.SliceData(data[:]), len(data))
	p := (*byte)(unsafe.Add(unsafe.Pointer(&view[0]), 1))
	text := unsafe.String(p, len(view)-1)
	return int(*unsafe.StringData(text)) + int(view[15])
}

//go:noinline
func unsafeReturnedViews(value byte) ([]byte, string, *byte) {
	data := [4]byte{value, value + 1, value + 2, value + 3}
	view := unsafe.Slice(&data[0], len(data))
	text := unsafe.String(unsafe.SliceData(view), len(view))
	p := (*byte)(unsafe.Add(unsafe.Pointer(unsafe.StringData(text)), 1))
	return view, text, p
}

var unsafeViewsChurn []byte

func TestUnsafeViewsEscapeAndGC(t *testing.T) {
	for value := byte(1); value <= 4; value++ {
		if got, want := unsafeLocalViews(value), 2*int(value)+16; got != want {
			t.Fatalf("local views=%d, want %d", got, want)
		}
		view, text, p := unsafeReturnedViews(value)
		runtime.GC()
		for range 128 {
			unsafeViewsChurn = make([]byte, 4)
		}
		if len(view) != 4 || len(text) != 4 || p != &view[1] || unsafe.StringData(text) != &view[0] {
			t.Fatal("returned views lost backing identity")
		}
		for i := range view {
			if view[i] != value+byte(i) || text[i] != view[i] {
				t.Fatalf("returned backing was reclaimed: %v %q", view, text)
			}
		}
	}
}
