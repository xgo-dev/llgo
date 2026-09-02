//go:build linux || darwin || windows

// This test uses platform-specific protected-memory helpers.
package gotest

import (
	"runtime/debug"
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
	if !enterRecoverableFaultTest(t) {
		return
	}

	// Automatic GC is orthogonal to this test and can perturb the host's
	// signal/exception recovery path while the fault is being converted into
	// a panic. Disable it only around the intentional fault so Go and LLGo run
	// the same deterministic recovery check.
	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)
	ensureWindowsExceptionStackHeadroom()

	old := debug.SetPanicOnFault(true)
	defer debug.SetPanicOnFault(old)

	data, _ := protectedMemory(t, 16, 8, 4)

	const offset = 5
	n, err := faultCopy(data[offset:], make([]byte, len(data)))
	if err == nil {
		t.Fatal("no error from copy across memory hole")
	}
	checkRecoveredFaultAddress(t, err, &data[len(data)/2])
	if want := len(data)/2 - offset; n != want {
		t.Fatalf("copy returned %d, want %d", n, want)
	}
}
