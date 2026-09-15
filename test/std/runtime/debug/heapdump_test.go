//go:build !js

package debug_test

import (
	"os"
	"runtime/debug"
	"testing"
)

// Go's js/wasm runtime fatally rejects runtime.write to fd > 2 (os_js.go),
// so dumping a heap to a file is not a supported JS host operation. Keep the
// descriptor-ownership test on native and WASI hosts; JS still tests Stack,
// PrintStack, and the explicit SetCrashOutput error contract in debug_test.go.
func TestHeapDumpOutputDescriptor(t *testing.T) {
	heapFile, err := os.CreateTemp(t.TempDir(), "heap-*.dump")
	if err != nil {
		t.Fatal(err)
	}
	debug.WriteHeapDump(heapFile.Fd())
	if _, err := heapFile.WriteString("fd-remains-open"); err != nil {
		t.Fatalf("heap dump closed its output descriptor: %v", err)
	}
	if err := heapFile.Close(); err != nil {
		t.Fatal(err)
	}
}
