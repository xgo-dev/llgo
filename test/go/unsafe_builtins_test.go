package gotest

import (
	"math"
	"testing"
	"unsafe"
)

var (
	unsafeBuiltinByteSink   []byte
	unsafeBuiltinUint64Sink []uint64
	unsafeBuiltinStringSink string
)

func TestUnsafeBuiltins(t *testing.T) {
	var buf [16]byte
	for i := range buf {
		buf[i] = byte('a' + i)
	}

	p := unsafe.Pointer(&buf[1])
	if got, want := unsafe.Add(p, 1), unsafe.Pointer(&buf[2]); got != want {
		t.Fatalf("unsafe.Add(+1) = %p, want %p", got, want)
	}
	if got, want := unsafe.Add(p, -1), unsafe.Pointer(&buf[0]); got != want {
		t.Fatalf("unsafe.Add(-1) = %p, want %p", got, want)
	}

	s := unsafe.Slice(&buf[0], len(buf))
	if len(s) != len(buf) || cap(s) != len(buf) || &s[0] != &buf[0] {
		t.Fatalf("unsafe.Slice header = len %d cap %d data %p, want len %d cap %d data %p",
			len(s), cap(s), unsafe.SliceData(s), len(buf), len(buf), &buf[0])
	}
	if unsafe.Slice((*int)(nil), 0) != nil {
		t.Fatal("unsafe.Slice(nil, 0) is not nil")
	}

	str := unsafe.String(&buf[0], len(buf))
	if got, want := len(str), len(buf); got != want {
		t.Fatalf("unsafe.String len = %d, want %d", got, want)
	}
	if unsafe.String(nil, 0) != "" {
		t.Fatal("unsafe.String(nil, 0) is not empty")
	}

	text := "unsafe string data"
	if got := string(unsafe.Slice(unsafe.StringData(text), len(text))); got != text {
		t.Fatalf("string(unsafe.Slice(unsafe.StringData(text), len(text))) = %q, want %q", got, text)
	}
	if got := unsafe.String(unsafe.SliceData(s), len(s)); got != string(s) {
		t.Fatalf("unsafe.String(unsafe.SliceData(s), len(s)) = %q, want %q", got, string(s))
	}
}

func TestUnsafeBuiltinPanics(t *testing.T) {
	negative := -1
	tooBig := uint64(math.MaxUint64)
	maxUintptr := ^uintptr(0)
	last := (*byte)(unsafe.Pointer(maxUintptr))

	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "slice nil non-zero length",
			f: func() {
				unsafeBuiltinByteSink = unsafe.Slice((*byte)(nil), 1)
			},
		},
		{
			name: "slice negative length",
			f: func() {
				unsafeBuiltinByteSink = unsafe.Slice(new(byte), negative)
			},
		},
		{
			name: "slice oversized length",
			f: func() {
				unsafeBuiltinByteSink = unsafe.Slice(new(byte), tooBig)
			},
		},
		{
			name: "slice element size overflow at max length",
			f: func() {
				unsafeBuiltinUint64Sink = unsafe.Slice(new(uint64), maxUintptr/8)
			},
		},
		{
			name: "slice element size overflow above max length",
			f: func() {
				unsafeBuiltinUint64Sink = unsafe.Slice(new(uint64), maxUintptr/8+1)
			},
		},
		{
			name: "slice memory end overflow",
			f: func() {
				unsafeBuiltinByteSink = unsafe.Slice(last, 2)
			},
		},
		{
			name: "string nil non-zero length",
			f: func() {
				unsafeBuiltinStringSink = unsafe.String(nil, 1)
			},
		},
		{
			name: "string negative length",
			f: func() {
				unsafeBuiltinStringSink = unsafe.String(new(byte), negative)
			},
		},
		{
			name: "string oversized length",
			f: func() {
				unsafeBuiltinStringSink = unsafe.String(new(byte), tooBig)
			},
		},
		{
			name: "string memory end overflow",
			f: func() {
				unsafeBuiltinStringSink = unsafe.String(last, 2)
			},
		},
	}

	_ = unsafe.Slice(last, 1)
	_ = unsafe.String(last, 1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectUnsafeBuiltinPanic(t, tt.f)
		})
	}
}

func expectUnsafeBuiltinPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
}
