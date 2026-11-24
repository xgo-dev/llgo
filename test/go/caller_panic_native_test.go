//go:build !wasm

package gotest

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// Wasm guests cannot create the subprocesses required by these checks. The
// Wasm acceptance host launches the fatal paths using retained test artifacts.
func TestCallerPanicTraceback(t *testing.T) {
	testCallerPanicTraceback(t)
}

func TestCallerRepanicTraceback(t *testing.T) {
	testCallerRepanicTraceback(t)
}

func TestPanicNamedIntegerText(t *testing.T) {
	if runtime.Compiler != "llgo" {
		t.Skip("checks LLGo panic formatting")
	}
	type panicInt8 int8
	type panicInt16 int16
	type panicInt32 int32
	type panicUint16 uint16
	const child = "LLGO_TEST_PANIC_INTEGER"
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"int8", panicInt8(-128), ".panicInt8(-128)"},
		{"int16", panicInt16(-32768), ".panicInt16(-32768)"},
		{"int32", panicInt32(-2147483648), ".panicInt32(-2147483648)"},
		{"uint16", panicUint16(65535), ".panicUint16(65535)"},
	}
	if mode := os.Getenv(child); mode != "" {
		for _, tc := range cases {
			if mode == tc.name {
				panic(tc.value)
			}
		}
		t.Fatalf("unexpected panic case %q", mode)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestPanicNamedIntegerText$")
			cmd.Env = append(os.Environ(), child+"="+tc.name)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.want) {
				t.Fatalf("want fatal panic containing %q, got %v:\n%s", tc.want, err, out)
			}
		})
	}
}
