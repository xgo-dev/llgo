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

func concatStrings(parts []rt.String, buffer *[32]byte) rt.String {
	return rt.StringConcat(unsafe.SliceData(parts), len(parts), buffer)
}

func TestStringConcat(t *testing.T) {
	var empty rt.String
	if got := concatStrings(nil, nil); got != empty {
		t.Fatalf("StringConcat(empty) returned a nonzero header: %v", got)
	}
	if rt.StringCat(empty, empty) != empty || rt.StringCatBuffer(empty, empty, nil) != empty {
		t.Fatal("binary concat of empty strings returned a nonzero header")
	}
	// Exercise both paths with empty operands, NUL and invalid UTF-8.
	parts := []rt.String{stringRuntimeValue("a"), {}, stringRuntimeValue("\x00\xff"), stringRuntimeValue("z")}
	for i, want := range []string{"a\x00\xff", "a\x00\xffz"} {
		if got := runtimeStringValue(concatStrings(parts[:3+i], nil)); got != want {
			t.Fatalf("StringConcat(%d parts) = %q, want %q", 3+i, got, want)
		}
	}
}

func TestStringConcatBufferBoundary(t *testing.T) {
	for _, size := range []int{0, 1, 31, 32, 33, 64} {
		text := strings.Repeat("x", size)
		parts := []rt.String{stringRuntimeValue(text[:size/2]), stringRuntimeValue(""), stringRuntimeValue(text[size/2:])}
		var buffer [32]byte
		got := runtimeStringValue(concatStrings(parts, &buffer))
		if got != text {
			t.Fatalf("size %d: got %q, want %q", size, got, text)
		}
		if size == 0 {
			continue
		}
		inBuffer := unsafe.StringData(got) == &buffer[0]
		if inBuffer != (size <= len(buffer)) {
			t.Fatalf("size %d: caller buffer used = %v", size, inBuffer)
		}
	}
}

func TestStringConcatCopiesSingleNonEmptyInput(t *testing.T) {
	var source, destination [32]byte
	copy(source[:], "owned")
	input := unsafe.String(&source[0], len("owned"))
	parts := []rt.String{{}, stringRuntimeValue(input), {}}
	check := func(name string, result rt.String, buffer *[32]byte) {
		t.Helper()
		got := runtimeStringValue(result)
		if unsafe.StringData(got) == &source[0] {
			t.Fatalf("%s: result borrowed input storage", name)
		}
		if got != "owned" {
			t.Fatalf("%s: got %q, want owned", name, got)
		}
		if buffer != nil && unsafe.StringData(got) != &buffer[0] {
			t.Fatalf("%s: result did not use caller buffer", name)
		}
	}
	check("StringCat", rt.StringCat(parts[1], parts[2]), nil)
	for _, buffer := range []*[32]byte{nil, &destination} {
		check("StringCatBuffer", rt.StringCatBuffer(parts[0], parts[1], buffer), buffer)
		check("StringConcat", concatStrings(parts, buffer), buffer)
	}
	runtime.KeepAlive(parts)
}

func invalidLengthString(length int) rt.String {
	// No data backs this header. Tests must reject the length before reading
	// its data or attempting a large allocation.
	header := struct {
		data unsafe.Pointer
		len  int
	}{len: length}
	return *(*rt.String)(unsafe.Pointer(&header))
}

func checkStringConcatLengthPanic(t *testing.T, a, b int) {
	t.Helper()
	left, right := invalidLengthString(a), invalidLengthString(b)
	for _, call := range []struct {
		name string
		fn   func()
	}{
		{"binary", func() { rt.StringCat(left, right) }},
		{"buffered binary", func() {
			var buffer [32]byte
			rt.StringCatBuffer(left, right, &buffer)
		}},
		{"multiple", func() {
			parts := []rt.String{left, right}
			concatStrings(parts, nil)
		}},
		{"three, first sum", func() { concatStrings([]rt.String{left, right, {}}, nil) }},
		{"three, final sum", func() { concatStrings([]rt.String{{}, left, right}, nil) }},
	} {
		func() {
			defer func() {
				if got := recover(); got != "runtime: string concat length out of range" {
					t.Errorf("%s lengths (%d, %d): panic = %v", call.name, a, b, got)
				}
			}()
			call.fn()
		}()
	}
}

func TestStringConcatInvalidLength(t *testing.T) {
	checkStringConcatLengthPanic(t, -1, 0)
	checkStringConcatLengthPanic(t, 0, -1)
	checkStringConcatLengthPanic(t, int(^uint(0)>>1), 1)
	if ^uint(0)>>63 != 0 {
		// Both lengths fit a 64-bit Go int, but their sum exceeds the
		// runtime's 48-bit allocation limit (or Wasm32's narrower limit).
		part := int64(1) << 47
		checkStringConcatLengthPanic(t, int(part), int(part+1))
	}
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
			var buffer [32]byte
			buffered := runtimeStringValue(rt.StringCatBuffer(stringRuntimeValue(text), stringRuntimeValue(suffix), &buffer))
			if buffered != text+suffix {
				t.Fatalf("StringCatBuffer(%q, %q) = %q", text, suffix, buffered)
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
		fused  string
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
		parts := []rt.String{stringRuntimeValue(text[:17]), stringRuntimeValue(text[17:]), stringRuntimeValue("tail")}
		fused := runtimeStringValue(concatStrings(parts, nil))
		values[i] = liveString{
			copied: copied,
			joined: joined,
			fused:  fused,
			tail:   joined[17:],
			cstr:   rt.CStrDup(stringRuntimeValue(text)),
		}
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	for i, value := range values {
		want := strings.Repeat(string(rune('a'+i)), 4096) + "\x00界🙂"
		if value.copied != want || value.joined != want+"tail" || value.fused != want+"tail" || value.tail != (want + "tail")[17:] {
			t.Fatalf("string %d changed after GC", i)
		}
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(value.cstr)), len(want)+1)
		if string(bytes[:len(want)]) != want || bytes[len(want)] != 0 {
			t.Fatalf("C string %d changed after GC", i)
		}
	}
	runtime.KeepAlive(values)
}
