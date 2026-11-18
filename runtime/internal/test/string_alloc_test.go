//go:build llgo

package test

import (
	"runtime"
	"strings"
	"testing"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func stringRuntimeValue(s string) rt.String {
	return *(*rt.String)(unsafe.Pointer(&s))
}

func runtimeStringValue(s rt.String) string {
	return *(*string)(unsafe.Pointer(&s))
}

func TestStringAllocationContents(t *testing.T) {
	for _, text := range []string{"", "hello", "a\x00b", "中文🙂", strings.Repeat("a\x00界🙂", 1024)} {
		data := []byte(text)
		copied := runtimeStringValue(rt.StringFrom(unsafe.Pointer(unsafe.SliceData(data)), len(data)))
		for i := range data {
			data[i] ^= 0xff
		}
		if copied != text {
			t.Fatalf("StringFrom did not preserve an independent copy of %d bytes", len(text))
		}

		for _, suffix := range []string{"", "\x00尾🙂"} {
			joined := runtimeStringValue(rt.StringCat(stringRuntimeValue(text), stringRuntimeValue(suffix)))
			if joined != text+suffix {
				t.Fatalf("StringCat(%q, %q) = %q", text, suffix, joined)
			}
		}

		cstr := rt.CStrDup(stringRuntimeValue(text))
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(cstr)), len(text)+1)
		if string(bytes[:len(text)]) != text || bytes[len(text)] != 0 {
			t.Fatalf("CStrDup did not preserve %d bytes and a NUL terminator", len(text))
		}
		runtime.KeepAlive(cstr)
	}
}

func TestStringAllocationsSurviveGC(t *testing.T) {
	type liveString struct {
		copied string
		joined string
		tail   string
		cstr   *int8
	}
	const count = 32
	values := make([]liveString, count)
	for i := range values {
		text := strings.Repeat(string(rune('a'+i)), 4096) + "\x00界🙂"
		data := []byte(text)
		copied := runtimeStringValue(rt.StringFrom(unsafe.Pointer(unsafe.SliceData(data)), len(data)))
		joined := runtimeStringValue(rt.StringCat(stringRuntimeValue(text), stringRuntimeValue("tail")))
		values[i] = liveString{
			copied: copied,
			joined: joined,
			tail:   joined[17:],
			cstr:   rt.CStrDup(stringRuntimeValue(text)),
		}
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	for i, value := range values {
		want := strings.Repeat(string(rune('a'+i)), 4096) + "\x00界🙂"
		if value.copied != want || value.joined != want+"tail" || value.tail != (want + "tail")[17:] {
			t.Fatalf("string %d changed after GC", i)
		}
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(value.cstr)), len(want)+1)
		if string(bytes[:len(want)]) != want || bytes[len(want)] != 0 {
			t.Fatalf("C string %d changed after GC", i)
		}
	}
	runtime.KeepAlive(values)
}
