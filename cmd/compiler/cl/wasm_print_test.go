package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
)

func TestWasmPrintStringScalarArguments(t *testing.T) {
	for _, arch := range []string{"wasm", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, `package main
func literal() { println("你好\x00", "") }
func dynamic(s string) { print(s) }
`, nil, &llvmgen.Target{GOOS: "linux", GOARCH: arch})
			if got := strings.Contains(ir, llvmgen.PkgRuntime+".PrintString32"); got != (arch == "wasm") {
				t.Fatalf("scalar print entry present = %v for %s:\n%s", got, arch, ir)
			}
			if !strings.Contains(ir, llvmgen.PkgRuntime+`.PrintString"`) {
				t.Fatalf("dynamic strings must retain the checked entry:\n%s", ir)
			}
		})
	}
}
