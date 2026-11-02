//go:build llgo

package runtime_test

import (
	"runtime"
	"testing"
)

// This suite runs as native executables and under the Wasm providers. Exercise
// constant folding and a type-level use, rather than only a runtime value.
func TestCompilerIdentity(t *testing.T) {
	const name = runtime.Compiler
	const description = name + " toolchain"
	var bytes [len(name)]byte
	if name != "llgo" || description != "llgo toolchain" || len(bytes) != 4 {
		t.Fatalf("Compiler = %q, description = %q, type length = %d", name, description, len(bytes))
	}
	switch runtime.Compiler {
	case "llgo":
	default:
		t.Fatal("Compiler constant selected the wrong branch")
	}
}
