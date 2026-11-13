package cl_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestImmediateClosureExplicitRoots(t *testing.T) {
	ir := cltest.CompileIREx(t, `package captureroots
func use(p *int) *int { return func() *int { return p }() }
`, "captureroots.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
		prog.Target().GOARCH = "wasm"
	})
	if strings.Contains(ir, "runtime.AllocZ") || strings.Contains(ir, "runtime.AllocU") {
		t.Fatalf("borrowed capture stayed on heap:\n%s", ir)
	}
	for _, want := range []string{"__llgo_gcroot_stack_pointer_", "llvm_gc_root_chain"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("explicit-root capture lost %s:\n%s", want, ir)
		}
	}
}
