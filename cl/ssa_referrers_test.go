package cl

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestNonDebugReferrers(t *testing.T) {
	constantValue := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	if refs, available := nonDebugReferrers(constantValue); available || refs != nil {
		t.Fatalf("constant referrers = %v, available = %v", refs, available)
	}
	if singleNonDebugReferrer(constantValue) != nil {
		t.Fatal("constant unexpectedly has a sole referrer")
	}
	const source = `package refs
func f(p int) int {
	x := p + 1
	println(x)
	return x
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "refs.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("refs", "refs"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}

	var sum *ssa.BinOp
	for _, block := range pkg.Func("f").Blocks {
		for _, instr := range block.Instrs {
			if candidate, ok := instr.(*ssa.BinOp); ok && candidate.Op == token.ADD {
				sum = candidate
			}
		}
	}
	if sum == nil {
		t.Fatal("sum instruction not found")
	}
	if singleNonDebugReferrer(sum) != nil {
		t.Fatal("multiply used value has a sole referrer")
	}
	if singleNonDebugReferrer(pkg.Func("f").Params[0]) != sum {
		t.Fatal("debug references hid the parameter's sole executable use")
	}
	all := *sum.Referrers()
	refs, available := nonDebugReferrers(sum)
	if !available {
		t.Fatal("referrers unexpectedly unavailable")
	}
	debugRefs := 0
	for _, ref := range all {
		if _, ok := ref.(*ssa.DebugRef); ok {
			debugRefs++
		}
	}
	if debugRefs == 0 {
		t.Fatalf("GlobalDebug produced no DebugRef for %s", sum)
	}
	if len(refs) != len(all)-debugRefs {
		t.Fatalf("filtered referrers = %d, all = %d, debug = %d", len(refs), len(all), debugRefs)
	}
	for _, ref := range refs {
		if _, ok := ref.(*ssa.DebugRef); ok {
			t.Fatalf("filtered referrers still contain %T", ref)
		}
	}
}
