package build

import (
	stdcontext "context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxFutexTimeoutABI(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler unavailable")
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	bin := filepath.Join(t.TempDir(), "futex-wait")
	if out, err := exec.CommandContext(ctx, cc, "-std=c11", "-Wall", "-Wextra", "-Werror", "testdata/futex_wait.c", "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("compile futex test: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, bin).CombinedOutput(); err != nil {
		t.Fatalf("futex wait: %v\n%s", err, out)
	}
}
