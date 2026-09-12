//go:build !llgo
// +build !llgo

package build

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWasmFSHostShim(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	script := filepath.Join(filepath.Dir(file), "testdata", "wasm_fs_host_test.mjs")
	cmd := exec.Command(node, script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node %s: %v\n%s", script, err, out)
	}
}
