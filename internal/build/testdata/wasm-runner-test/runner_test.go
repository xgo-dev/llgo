package wasmrunner

import (
	"os"
	"runtime"
	"testing"
)

func TestRawWasmRunner(t *testing.T) {
	if runtime.GOARCH != "wasm" {
		t.Fatalf("GOARCH = %q, want wasm", runtime.GOARCH)
	}
	// Match Go's go_wasip1_wasm_exec contract: the default Wasmtime adapter
	// inherits PWD and PATH without exposing every host environment variable.
	if got := os.Getenv("PWD"); got == "" {
		t.Fatal("WASI runner did not inherit PWD")
	}
	done := make(chan struct{})
	go func() { close(done) }()
	<-done
}
