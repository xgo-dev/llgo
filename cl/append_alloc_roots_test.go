package cl_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestAppendTemporaryExplicitRoots(t *testing.T) {
	for _, elem := range []string{"int", "*int"} {
		ir := cltest.CompileIREx(t, "package appendroots\nfunc appendOne(s []"+elem+", x "+elem+") []"+elem+" { return append(s, x) }", "appendroots.go", false, func(prog llssa.Program) {
			prog.EnableGCRoots(true)
			prog.Target().GOARCH = "wasm"
		})
		if got := strings.Contains(ir, "runtime.AllocZ"); got != (elem == "*int") {
			t.Fatalf("%s: temporary heap allocation = %v:\n%s", elem, got, ir)
		}
	}
}
