package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWasmPrintStringScalarArguments(t *testing.T) {
	for _, tc := range []struct {
		name       string
		target     llssa.Target
		wantScalar bool
		wantString bool
	}{
		{"wasm32", llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"}, true, false},
		{"wasm64", llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", LLVMTarget: "wasm64-unknown-emscripten"}, true, true},
		{"native", llssa.Target{GOOS: "linux", GOARCH: "amd64"}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, `package main
func literal() { println("你好\x00", "") }
func dynamic(s string) { print(s) }
`, nil, &tc.target)
			if got := strings.Contains(ir, llssa.PkgRuntime+".PrintString32"); got != tc.wantScalar {
				t.Fatalf("scalar print entry present = %v:\n%s", got, ir)
			}
			if got := strings.Contains(ir, llssa.PkgRuntime+`.PrintString"`); got != tc.wantString {
				t.Fatalf("full-width print entry present = %v:\n%s", got, ir)
			}
		})
	}
}
