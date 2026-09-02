//go:build windows

package gotest

import (
	"regexp"
	"runtime"
	"testing"
)

const recoverableWindowsExceptionChildEnv = "LLGO_TEST_WINDOWS_EXCEPTION_CHILD"

func enterWindowsExceptionTest(t *testing.T) bool {
	t.Helper()
	return enterWindowsFaultTest(
		t,
		recoverableWindowsExceptionChildEnv,
		"^"+regexp.QuoteMeta(t.Name())+"$",
	)
}

//go:noinline
func growWindowsExceptionStack(depth int) {
	var frame [1024]byte
	frame[0] = byte(depth)
	if depth != 0 {
		growWindowsExceptionStack(depth - 1)
	}
	runtime.KeepAlive(&frame)
}

func ensureWindowsExceptionStackHeadroom() {
	// Grow and then unwind the current goroutine stack before recovering a
	// hardware exception. About 64 one-KiB frames provide headroom for the
	// host-dependent Windows exception context suspected in golang/go#81238.
	// Without that headroom, exception dispatch can write below stack.lo and
	// corrupt unrelated heap state on some AMX-capable hosted runners.
	growWindowsExceptionStack(64)
}
