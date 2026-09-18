//go:build !wasm

package gotest

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRuntimeSetFinalizerRejectsInvalidArgumentTypes(t *testing.T) {
	for _, name := range finalizerInvalidCases {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeSetFinalizerRejectsInvalidArgumentTypes$")
			cmd.Env = append(os.Environ(), finalizerInvalidCaseEnv+"="+name)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("SetFinalizer accepted invalid %s", name)
			}
			if !strings.Contains(string(out), "runtime.SetFinalizer:") {
				t.Fatalf("SetFinalizer error for %s:\n%s", name, out)
			}
		})
	}
}
