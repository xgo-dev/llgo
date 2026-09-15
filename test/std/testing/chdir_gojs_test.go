//go:build llgo && js && wasm && !llgo.wasm.emscripten

package testing_test

import (
	"errors"
	"os"
	"syscall"
)

type chdirTestContext interface {
	Helper()
	Fatalf(string, ...any)
}

func testChdir(tb chdirTestContext, dir string, _ func()) {
	tb.Helper()
	// Go's browser process shim deliberately exposes chdir as ENOSYS. Exercise
	// that host contract without calling testing.Chdir, which reports the same
	// unsupported operation as a fatal test error.
	if err := os.Chdir(dir); !errors.Is(err, syscall.ENOSYS) {
		tb.Fatalf("Chdir = %v, want ENOSYS for the browser host", err)
	}
}
