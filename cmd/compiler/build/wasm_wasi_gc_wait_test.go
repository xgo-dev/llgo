//go:build darwin || linux

package build

import (
	stdcontext "context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWASIGCWaitsSuspendGoAcrossMutexReacquisition(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "wasi-gc-wait")
	args := []string{"-std=c11", "-O2", "-Wall", "-Wextra", "-Werror", "-pthread",
		"testdata/wasm-wasi-gc-wait/main.c",
		"../../../runtime/internal/runtime/_wrap/wasi_gc_world.c", "-o", bin}
	if out, err := exec.Command(clang, args...).CombinedOutput(); err != nil {
		t.Fatalf("compile wait regression: %v\n%s", err, out)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, bin).CombinedOutput(); err != nil {
		t.Fatalf("wait regression: %v\n%s", err, out)
	}
}
