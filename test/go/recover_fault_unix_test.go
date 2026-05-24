//go:build linux || darwin

package gotest

import (
	"runtime/debug"
	"syscall"
	"testing"
)

func faultCopy(dst, src []byte) (n int, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()

	for i := 0; i < len(dst) && i < len(src); i++ {
		dst[i] = src[i]
		n++
	}
	return
}

func TestRecoverAfterFaultPreservesNamedResult(t *testing.T) {
	old := debug.SetPanicOnFault(true)
	defer debug.SetPanicOnFault(old)

	size := syscall.Getpagesize()
	data, err := syscall.Mmap(-1, 0, 16*size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatalf("mmap: %v", err)
	}
	defer syscall.Munmap(data)

	hole := data[len(data)/2 : 3*(len(data)/4)]
	if err := syscall.Mprotect(hole, syscall.PROT_NONE); err != nil {
		t.Fatalf("mprotect: %v", err)
	}

	const offset = 5
	n, err := faultCopy(data[offset:], make([]byte, len(data)))
	if err == nil {
		t.Fatal("no error from copy across memory hole")
	}
	if want := len(data)/2 - offset; n != want {
		t.Fatalf("copy returned %d, want %d", n, want)
	}
}
