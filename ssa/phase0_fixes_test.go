package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestIsPackageLevelScope(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/a", "a")
	local := types.NewScope(pkgA.Scope(), token.NoPos, token.NoPos, "func")
	nested := types.NewScope(local, token.NoPos, token.NoPos, "block")

	if !isPackageLevelScope(pkgA.Scope()) {
		t.Fatal("package scope should be recognized")
	}
	if !isPackageLevelScope(pkgB.Scope()) {
		t.Fatal("distinct package scope pointer should be recognized")
	}
	if isPackageLevelScope(local) {
		t.Fatal("function scope must not match package scope")
	}
	if isPackageLevelScope(nested) {
		t.Fatal("nested block scope must not match package scope")
	}
	if isPackageLevelScope(nil) {
		t.Fatal("nil scope must not match package scope")
	}
}

func TestBuilderHasTerminator(t *testing.T) {
	var zero Builder
	if zero.HasTerminator() {
		t.Fatal("nil Builder should not report a terminator")
	}
	if (&aBuilder{}).HasTerminator() {
		t.Fatal("Builder with nil impl should not report a terminator")
	}

	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("term", "example.com/term")
	fn := pkg.NewFunc("f", NoArgsNoRet, GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()

	if b.HasTerminator() {
		t.Fatal("empty block should not report a terminator")
	}
	_ = b.AllocaT(prog.Int())
	if b.HasTerminator() {
		t.Fatal("block ending in alloca should not report a terminator")
	}
	b.Return()
	if !b.HasTerminator() {
		t.Fatal("block ending in ret should report a terminator")
	}
}

func TestTypeAndStructEqualNeedsShortCircuit(t *testing.T) {
	pkg := types.NewPackage("example.com/eq", "eq")
	intField := func(name string) *types.Var {
		return types.NewField(token.NoPos, pkg, name, types.Typ[types.Int], false)
	}
	strField := func(name string) *types.Var {
		return types.NewField(token.NoPos, pkg, name, types.Typ[types.String], false)
	}
	anyField := func(name string) *types.Var {
		return types.NewField(token.NoPos, pkg, name, types.NewInterfaceType(nil, nil).Complete(), false)
	}

	if typeEqualNeedsShortCircuit(types.Typ[types.Int]) {
		t.Fatal("int should not need short-circuit")
	}
	if !typeEqualNeedsShortCircuit(types.Typ[types.String]) {
		t.Fatal("string should need short-circuit")
	}
	if !typeEqualNeedsShortCircuit(types.NewInterfaceType(nil, nil).Complete()) {
		t.Fatal("interface should need short-circuit")
	}
	if typeEqualNeedsShortCircuit(types.NewArray(types.Typ[types.Int], 0)) {
		t.Fatal("zero-length array should not need short-circuit")
	}
	if !typeEqualNeedsShortCircuit(types.NewArray(types.Typ[types.String], 2)) {
		t.Fatal("string array should need short-circuit")
	}
	nested := types.NewStruct([]*types.Var{intField("_"), anyField("v")}, nil)
	if !typeEqualNeedsShortCircuit(nested) {
		t.Fatal("struct with interface field should need short-circuit")
	}
	plain := types.NewStruct([]*types.Var{intField("_"), intField("a"), intField("b")}, nil)
	if typeEqualNeedsShortCircuit(plain) {
		t.Fatal("plain int struct should not need short-circuit")
	}
	if structEqualNeedsShortCircuit(plain, []int{1}) {
		t.Fatal("single-field struct should not need short-circuit")
	}
	if structEqualNeedsShortCircuit(plain, []int{1, 2}) {
		t.Fatal("plain multi-field struct should not need short-circuit")
	}
	withSecondStr := types.NewStruct([]*types.Var{intField("a"), strField("s")}, nil)
	if !structEqualNeedsShortCircuit(withSecondStr, []int{0, 1}) {
		t.Fatal("struct with non-first string field should need short-circuit")
	}
}
