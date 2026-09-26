package wasmtest

import (
	"os"
	"testing"
)

// TestFailureExit is normally a no-op. The host-runner acceptance invokes it
// with LLGO_WASM_TEST_FAIL set to prove that a failed wasm test propagates a
// nonzero status through Emscripten, Node, and the llgo test command.
func TestFailureExit(t *testing.T) {
	if os.Getenv("LLGO_WASM_TEST_FAIL") != "" {
		t.Fatal("intentional WebAssembly test failure")
	}
}
