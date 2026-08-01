package ssa

import (
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestAssertNilDerefZeroExprNoPanic(t *testing.T) {
	var b Builder
	b.AssertNilDeref(Expr{})
	b.AssertNilDerefBranch(Expr{})
}

func TestAssertNilDerefBranchIR(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	runtimePkg := types.NewPackage(PkgRuntime, "runtime")
	params := types.NewTuple(types.NewVar(token.NoPos, runtimePkg, "isNil", types.Typ[types.Bool]))
	assertNilDeref := types.NewFunc(token.NoPos, runtimePkg, "AssertNilDeref", types.NewSignatureType(nil, nil, nil, params, nil, false))
	runtimePkg.Scope().Insert(assertNilDeref)
	prog.SetRuntime(runtimePkg)

	ptrParams := types.NewTuple(types.NewVar(token.NoPos, nil, "ptr", types.Typ[types.UnsafePointer]))
	pkg := prog.NewPackage("main", "main")
	fn := pkg.NewFunc("check", types.NewSignatureType(nil, nil, nil, ptrParams, nil, false), InGo)
	b := fn.MakeBody(1)
	b.AssertNilDerefBranch(b.Const(constant.MakeInt64(1), prog.VoidPtr()))
	b.AssertNilDerefBranch(fn.Param(0))
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	if got := strings.Count(ir, "AssertNilDeref"); got != 2 {
		t.Fatalf("AssertNilDeref references = %d, want declaration and one call:\n%s", got, ir)
	}
	if !strings.Contains(ir, "br i1") || !strings.Contains(ir, "unreachable") {
		t.Fatalf("AssertNilDerefBranch did not emit a nil-only panic path:\n%s", ir)
	}
}
