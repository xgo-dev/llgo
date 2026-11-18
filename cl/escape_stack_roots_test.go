package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestEscapeTypedFieldRootsAndBudget(t *testing.T) {
	const src = `package stackfields
func replace(p **int,q *int){*p=q}
func read(p *int)int{return *p}
func local(p *int)int {var q *int;replace(&q,p);return read(q)}
func large(p *int)int {var a [16384]*int;a[0]=p;a[16383]=p;return *a[0]+*a[16383]}
`
	ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, true)
	if body := llvmFunction(t, ir, "stackfields.local"); strings.Contains(body, "runtime.Alloc") || !strings.Contains(body, "gcroot.init") {
		t.Fatalf("mutable field not in typed frame:\n%s", body)
	}
	if body := llvmFunction(t, ir, "stackfields.large"); !strings.Contains(body, "runtime.AllocZ") {
		t.Fatal("root slots exceeded the function budget")
	}
	if strings.Contains(llvmFunction(t, ir, "stackfields.large"), `alloca [16384 x ptr]`) {
		t.Fatal("large local introduced unbudgeted root slots")
	}
}
