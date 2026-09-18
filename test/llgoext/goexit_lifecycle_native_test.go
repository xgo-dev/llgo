//go:build llgo && !wasm

package llgoext

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainGoexitLifecycleReleasedOnce(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), mainGoexitLifecycleChild+"=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("main Goexit child unexpectedly succeeded:\n%s", output)
	}
	worker := strings.Index(string(output), "WORKER_RETURNING")
	deadlock := strings.Index(string(output), "no goroutines (main called runtime.Goexit) - deadlock!")
	if worker < 0 || deadlock < 0 || worker > deadlock {
		t.Fatalf("worker must return before the last-goroutine deadlock:\n%s", output)
	}
}
