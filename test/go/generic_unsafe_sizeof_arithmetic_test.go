package gotest

import (
	"strings"
	"testing"
	"unsafe"
)

func genericUnsafeSizeofShift[T any]() int64 {
	return 1 << unsafe.Sizeof(*new(T))
}

func genericUnsafeSizeofDiv[T any]() uintptr {
	return 1 / unsafe.Sizeof(*new(T))
}

func genericUnsafeSizeofAdd[T any]() int64 {
	return 1<<63 - 1 + int64(unsafe.Sizeof(*new(T)))
}

func genericUnsafeSizeofAny[T any]() any {
	return unsafe.Sizeof(*new(T))
}

func TestGenericUnsafeSizeofArithmetic(t *testing.T) {
	const minInt64 = -1 << 63

	tests := []struct {
		name string
		got  int64
		want int64
	}{
		{name: "shift 62", got: genericUnsafeSizeofShift[[62]byte](), want: 1 << 62},
		{name: "shift 63", got: genericUnsafeSizeofShift[[63]byte](), want: minInt64},
		{name: "shift 64", got: genericUnsafeSizeofShift[[64]byte](), want: 0},
		{name: "shift 100", got: genericUnsafeSizeofShift[[100]byte](), want: 0},
		{name: "shift large", got: genericUnsafeSizeofShift[[1e6]byte](), want: 0},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Fatalf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}

	if got := genericUnsafeSizeofAdd[[1]byte](); got != minInt64 {
		t.Fatalf("add overflow = %d, want %d", got, minInt64)
	}
	if got := genericUnsafeSizeofAny[[1]byte](); got != uintptr(1) {
		t.Fatalf("Sizeof boxed value = %v (%T), want uintptr(1)", got, got)
	}

	expectGenericUnsafeSizeofDivideByZero(t, func() {
		_ = genericUnsafeSizeofDiv[[0]byte]()
	})
}

func expectGenericUnsafeSizeofDivideByZero(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("divide by zero did not panic")
		}
		runtimeErr, ok := err.(interface{ RuntimeError() })
		if !ok {
			t.Fatalf("panic type = %T, want runtime.Error", err)
		}
		_ = runtimeErr
		msgErr, ok := err.(error)
		if !ok {
			t.Fatalf("panic type = %T, want error", err)
		}
		if got := msgErr.Error(); !strings.Contains(got, "divide by zero") {
			t.Fatalf("panic = %q, want divide by zero", got)
		}
	}()
	f()
}
