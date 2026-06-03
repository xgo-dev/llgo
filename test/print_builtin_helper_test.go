package test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBuiltinPrintHelper(t *testing.T) {
	if os.Getenv("LLGO_PRINT_HELPER") == "" {
		t.Skip("helper process")
	}
	zero := 0.0
	nan := zero / zero
	posInf := 1.0 / zero
	negInf := -1.0 / zero

	print(1e7, "\n")
	print(complex(1e7, -1e7), "\n")
	print(complex(1.5, -0.0), "\n")
	print(nan, "\n")
	print(posInf, "\n")
	print(negInf, "\n")
	print(complex(1, nan), "\n")
	print(complex(1, posInf), "\n")
	print(complex(1, negInf), "\n")
}

func runBuiltinPrintProbe(t *testing.T) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestBuiltinPrintHelper$")
	cmd.Env = append(os.Environ(), "LLGO_PRINT_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("builtin print probe failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return strings.ReplaceAll(stderr.String(), "\r\n", "\n")
}
