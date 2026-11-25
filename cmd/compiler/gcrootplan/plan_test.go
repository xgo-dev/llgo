package gcrootplan

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestPlanStraightLine(t *testing.T) {
	fn := buildFunction(t, `package p
func keep(*int)
func f(live, dead *int) *int {
	_ = dead
	keep(live)
	return live
}`)
	roots := Plan(fn, pointerValue, isCall)
	assertRootNames(t, roots, "live")
}

func TestPlanPhiEdges(t *testing.T) {
	fn := buildFunction(t, `package p
func keep(*int)
func f(cond bool, left, right *int) *int {
	var value *int
	if cond {
		value = left
	} else {
		value = right
	}
	keep(value)
	return value
	}`)
	roots := Plan(fn, pointerValue, isCall)
	var foundPhi bool
	for value := range roots {
		if _, ok := value.(*ssa.Phi); ok {
			foundPhi = true
		}
	}
	if !foundPhi {
		t.Fatal("merged pointer is not rooted at the call")
	}
}

func TestPlanPhiEdgeUse(t *testing.T) {
	fn := buildFunction(t, `package p
func keep(*int)
func f(cond bool, left, right *int) *int {
	var value *int
	if cond {
		keep(nil)
		value = left
	} else {
		keep(nil)
		value = right
	}
	return value
}`)
	roots := Plan(fn, pointerValue, isCall)
	assertRootNames(t, roots, "left", "right")
}

func TestPlanLoop(t *testing.T) {
	fn := buildFunction(t, `package p
func keep(*int)
func f(head *int, n int) *int {
	for n > 0 {
		keep(head)
		n--
	}
	return head
}`)
	roots := Plan(fn, pointerValue, isCall)
	assertRootNames(t, roots, "head")
}

func TestPlanNoSafepoint(t *testing.T) {
	fn := buildFunction(t, `package p
func f(value *int) *int { return value }`)
	if roots := Plan(fn, pointerValue, isCall); len(roots) != 0 {
		t.Fatalf("Plan returned %d roots without a safepoint", len(roots))
	}
}

func buildFunction(t *testing.T, src string) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("p", "p"),
		[]*ast.File{file},
		ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ssaPkg.Func("f")
}

func pointerValue(value ssa.Value) bool {
	_, ok := value.Type().Underlying().(*types.Pointer)
	return ok
}

func isCall(instr ssa.Instruction) bool {
	_, ok := instr.(*ssa.Call)
	return ok
}

func assertRootNames(t *testing.T, roots map[ssa.Value]struct{}, names ...string) {
	t.Helper()
	for _, name := range names {
		var found bool
		for value := range roots {
			if value.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("root %q not found", name)
		}
	}
}
