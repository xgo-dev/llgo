package cl

import (
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestImportedFrontendPreservesCallerAndRecoverContracts(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example/dep", `package dep
import "runtime"
func Where(){runtime.Caller(0)}
func Catch(){recover();runtime.Caller(0)}
func Add(x int)int{return x+1}
`, "example/root", `package root
import "example/dep"
func Run(){defer dep.Catch();dep.Where()}
`)
	cold := NewCallerTracking()
	cold.Precompute([]*ssa.Package{dep, root})
	want := cold.ExportFrontendFacts(root, root.Func("Run"))
	imports := make(map[*ssa.Function]FrontendFacts)
	for _, member := range dep.Members {
		if fn, ok := member.(*ssa.Function); ok {
			imports[fn] = cold.ExportFrontendFacts(dep, fn)
		}
	}
	if !imports[dep.Func("Where")].CallerBase || !imports[dep.Func("Catch")].UsesRecover {
		t.Fatal("fixture did not exercise caller/recover contracts")
	}
	if imports[dep.Func("Add")].InlineCost > inlineBodyBudget {
		t.Fatal("eligible inline-body contract was not exported")
	}
	for fn := range imports {
		fn.Blocks = nil
	}
	warm := NewCallerTracking()
	warm.ImportFrontendFacts(dep, imports)
	warm.Precompute([]*ssa.Package{dep, root})
	if got := warm.ExportFrontendFacts(root, root.Func("Run")); got != want {
		t.Fatalf("caller behavior changed with opaque dependency: got %+v want %+v", got, want)
	}
	if !warm.recoverAnalysis().functionUsesRecover(dep.Func("Catch")) {
		t.Fatal("recover classification was inferred from an empty body")
	}
}
