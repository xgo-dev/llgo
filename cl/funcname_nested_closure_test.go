//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/llgo/internal/typepatch"
	"github.com/goplus/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildSSAPackage(t *testing.T, src string) *ssa.Package {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage("foo", "foo")
	imp := packages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg
}

func collectFuncNamesBySSAName(ssapkg *ssa.Package, ssaName string) []string {
	var got []string
	for fn := range ssautil.AllFunctions(ssapkg.Prog) {
		if fn == nil || fn.Name() != ssaName {
			continue
		}
		got = append(got, funcName(ssapkg.Pkg, fn, false))
	}
	sort.Strings(got)
	return got
}

func expectTwoDistinctNamesForTypes(t *testing.T, got []string, suffix string) {
	t.Helper()
	if len(got) != 2 {
		t.Fatalf("got %d function names, want 2: %v", len(got), got)
	}
	if got[0] == got[1] {
		t.Fatalf("got duplicate names: %v", got)
	}

	var hasA, hasB bool
	for _, name := range got {
		if !strings.Contains(name, suffix) {
			t.Fatalf("name %q missing %q", name, suffix)
		}
		if strings.Contains(name, "A") {
			hasA = true
		}
		if strings.Contains(name, "B") {
			hasB = true
		}
	}
	if !hasA || !hasB {
		t.Fatalf("expected names for both A and B, got %v", got)
	}
}

func TestGenericLocalTypePatchHelpers(t *testing.T) {
	ssapkg := buildSSAPackage(t, `package foo

func use() {
	_ = localType[int]()
	_ = localType[string]()
}

func localType[T any]() any {
	type local struct {
		value T
	}
	return local{}
}
`)

	fn, local := firstGenericLocalNamed(t, ssapkg)
	ctx := &context{goFn: fn}
	if !ctx.isGenericLocalType(local.Obj()) {
		t.Fatalf("%s should be detected as a generic local type", local.Obj().Name())
	}

	patched, ok := ctx.patchLocalGenericNamed(local)
	if !ok {
		t.Fatalf("patchLocalGenericNamed(%v) was not patched", local)
	}
	name := patched.Obj().Name()
	if !strings.Contains(name, "[") || !strings.Contains(name, "·") {
		t.Fatalf("patched local generic name = %q, want type args and ordinal suffix", name)
	}
	if _, ok := ctx.patchLocalGenericNamed(patched); ok {
		t.Fatalf("already-patched local generic name %q should not be patched again", name)
	}

	for _, typ := range []types.Type{
		types.NewPointer(local),
		types.NewSlice(local),
		types.NewArray(local, 2),
		types.NewMap(types.Typ[types.String], local),
		types.NewChan(types.SendRecv, local),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, ssapkg.Pkg, "field", local, false)}, []string{`json:"field"`}),
	} {
		if _, ok := ctx._patchType(typ); !ok {
			t.Fatalf("_patchType(%v) did not patch nested local generic type", typ)
		}
	}

	argName := ctx.typeArgName(types.NewMap(
		types.NewChan(types.RecvOnly, types.Typ[types.Int]),
		types.NewArray(types.NewSlice(local), 3),
	))
	if !strings.Contains(argName, "<-chan int") || !strings.Contains(argName, "[]") {
		t.Fatalf("typeArgName did not format nested composite type: %q", argName)
	}

	if got := typeListArgs(nil, ctx.typeArgName); got != nil {
		t.Fatalf("typeListArgs(nil) = %v, want nil", got)
	}
}

func TestGenericLocalTypePatchHelperBranches(t *testing.T) {
	ssapkg := buildSSAPackage(t, `package foo

func outer[T any]() any {
	type one struct{ value T }
	type two struct{ value T }
	_ = one{}
	return two{}
}

func use() {
	_ = outer[int]()
}
`)
	fn, local := firstGenericLocalNamed(t, ssapkg)
	ctx := &context{goFn: fn}

	if ctx.isGenericLocalType(nil) {
		t.Fatal("nil object should not be a generic local type")
	}
	topObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "Top", nil)
	if ctx.isLocalType(topObj) {
		t.Fatal("top-level type should not be local")
	}
	if ctx.localTypeOuterArgs(topObj) != nil {
		t.Fatal("top-level type should not report outer type arguments")
	}
	if _, ok := (&context{}).patchLocalGenericNamed(local); ok {
		t.Fatal("patchLocalGenericNamed should require a current generic function")
	}

	if _, ok := ctx._patchType(types.NewMap(local, types.Typ[types.Int])); !ok {
		t.Fatal("_patchType should patch map key local type")
	}
	if _, ok := ctx._patchType(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, ssapkg.Pkg, "plain", types.Typ[types.Int], false),
	}, nil)); ok {
		t.Fatal("_patchType should not patch structs without local generic fields")
	}
	if _, ok := ctx._patchType(types.NewTuple(types.NewVar(token.NoPos, ssapkg.Pkg, "v", local))); !ok {
		t.Fatal("_patchType should patch tuple elements")
	}
	if _, ok := ctx._patchType(types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, ssapkg.Pkg, "p", local)),
		types.NewTuple(types.NewVar(token.NoPos, ssapkg.Pkg, "r", local)),
		false)); !ok {
		t.Fatal("_patchType should patch signature params and results")
	}
	if _, ok := ctx._patchType(types.NewTuple(types.NewVar(token.NoPos, ssapkg.Pkg, "v", types.Typ[types.Int]))); ok {
		t.Fatal("_patchType should not patch tuple without local generic elements")
	}

	oldPkg := types.NewPackage("example.com/patched", "patched")
	oldObj := types.NewTypeName(token.NoPos, oldPkg, "N", nil)
	oldNamed := types.NewNamed(oldObj, types.Typ[types.Int], nil)
	oldPkg.Scope().Insert(oldObj)
	patchPkg := types.NewPackage(oldPkg.Path(), oldPkg.Name())
	patchObj := types.NewTypeName(token.NoPos, patchPkg, "N", nil)
	patchNamed := types.NewNamed(patchObj, types.Typ[types.String], nil)
	patchPkg.Scope().Insert(patchObj)
	typepatch.Merge(patchPkg, oldPkg, nil, false)
	patchCtx := &context{
		prog:    ssatest.NewProgram(t, nil),
		patches: Patches{oldPkg.Path(): {Types: patchPkg}},
	}
	if got, ok := patchCtx._patchType(oldNamed); !ok || !types.Identical(got, patchNamed) {
		t.Fatalf("_patchType(patched named) = %v, %v, want patched named", got, ok)
	}

	pkglessObj := types.NewTypeName(token.NoPos, nil, "Pkgless", nil)
	pkgless := types.NewNamed(pkglessObj, types.Typ[types.Int], nil)
	if got := ctx.typeArgName(pkgless); !strings.Contains(got, "Pkgless") {
		t.Fatalf("typeArgName(pkgless) = %q, want local name", got)
	}
	aliasObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "Alias", nil)
	alias := types.NewAlias(aliasObj, local)
	if got := ctx.typeArgName(alias); !strings.Contains(got, "foo.two[") {
		t.Fatalf("typeArgName(alias) = %q, want unaliased local named type", got)
	}
	if got := ctx.typeArgName(types.NewPointer(local)); !strings.HasPrefix(got, "*foo.two[") {
		t.Fatalf("typeArgName(pointer) = %q, want pointer to patched local name", got)
	}
	send := types.NewChan(types.SendOnly, types.Typ[types.Int])
	recv := types.NewChan(types.RecvOnly, types.Typ[types.Int])
	if got := ctx.typeArgName(send); !strings.Contains(got, "chan<-") {
		t.Fatalf("typeArgName(send chan) = %q, want chan<-", got)
	}
	if got := ctx.typeArgName(types.NewChan(types.SendRecv, recv)); !strings.Contains(got, "chan (<-chan int)") {
		t.Fatalf("typeArgName(chan recv) = %q, want parenthesized recv channel", got)
	}
	if got := ctx.typeArgName(types.NewSignatureType(nil, nil, nil, nil, nil, false)); !strings.Contains(got, "func") {
		t.Fatalf("typeArgName(signature) = %q, want fallback TypeString", got)
	}
	otherPkg := types.NewPackage("example.com/bar", "bar")
	other := types.NewNamed(types.NewTypeName(token.NoPos, otherPkg, "Other", nil), types.Typ[types.Int], nil)
	sigWithPkg := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, otherPkg, "x", other)),
		nil, false)
	if got := ctx.typeArgName(sigWithPkg); !strings.Contains(got, "bar.Other") {
		t.Fatalf("typeArgName(signature with package) = %q, want package-qualified param", got)
	}

	patchedObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "Already[int]", nil)
	patched := types.NewNamed(patchedObj, types.Typ[types.Int], nil)
	if got := ctx.localNamedName(patched, true); got != "Already[int]" {
		t.Fatalf("localNamedName(already patched) = %q, want unchanged name", got)
	}

	topTypeParamObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "V", nil)
	topTypeParam := types.NewTypeParam(topTypeParamObj, types.NewInterfaceType(nil, nil))
	topBoxObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "TopBox", nil)
	ssapkg.Pkg.Scope().Insert(topBoxObj)
	topBox := types.NewNamed(topBoxObj, types.Typ[types.Int], nil)
	topBox.SetTypeParams([]*types.TypeParam{topTypeParam})
	topBoxInt, err := types.Instantiate(types.NewContext(), topBox, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate(TopBox[int]) failed: %v", err)
	}
	if got := ctx.localNamedName(topBoxInt.(*types.Named), false); !strings.Contains(got, "[int]") || strings.Contains(got, ";") {
		t.Fatalf("localNamedName(top-level generic) = %q, want own args only", got)
	}

	tpObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "T", nil)
	tparam := types.NewTypeParam(tpObj, types.NewInterfaceType(nil, nil))
	scopeWithTypeParam := types.NewScope(ssapkg.Pkg.Scope(), token.NoPos, token.NoPos, "typed")
	scopeWithTypeParam.Insert(types.NewTypeName(token.NoPos, ssapkg.Pkg, "N", types.Typ[types.Int]))
	scopeWithTypeParam.Insert(types.NewTypeName(token.NoPos, ssapkg.Pkg, "T", tparam))
	if !scopeHasTypeParams(scopeWithTypeParam) {
		t.Fatal("scopeHasTypeParams should detect type parameters")
	}
	if !isTypeParamObject(tpObj) {
		t.Fatal("isTypeParamObject should detect type parameter type names")
	}
	if isTypeParamObject(types.NewVar(token.NoPos, ssapkg.Pkg, "v", types.Typ[types.Int])) {
		t.Fatal("isTypeParamObject should ignore non-type names")
	}

	localScope := types.NewScope(ssapkg.Pkg.Scope(), token.NoPos, token.NoPos, "local")
	localScope.Insert(types.NewVar(token.NoPos, ssapkg.Pkg, "v", types.Typ[types.Int]))
	localScope.Insert(tpObj)
	localScope.Insert(types.NewTypeName(token.Pos(10), ssapkg.Pkg, "Before", types.Typ[types.Int]))
	manualObj := types.NewTypeName(token.Pos(20), ssapkg.Pkg, "Manual", nil)
	localScope.Insert(manualObj)
	localScope.Insert(types.NewTypeName(token.Pos(30), ssapkg.Pkg, "After", types.Typ[types.Int]))
	manual := types.NewNamed(manualObj, types.Typ[types.Int], nil)
	if got := ctx.localTypeOrdinal(manual.Obj()); got != 2 {
		t.Fatalf("localTypeOrdinal(manual) = %d, want 2", got)
	}
	if !ctx.isGenericLocalType(manual.Obj()) {
		t.Fatal("manual object in type-param scope should be generic local")
	}

	localPlainScope := types.NewScope(nil, token.NoPos, token.NoPos, "plain")
	plainObj := types.NewTypeName(token.Pos(1), ssapkg.Pkg, "Plain", nil)
	localPlainScope.Insert(plainObj)
	if ctx.isGenericLocalType(plainObj) {
		t.Fatal("local type without type-param scope should not be generic local")
	}
	childScope := types.NewScope(ssapkg.Pkg.Scope(), token.NoPos, token.NoPos, "child")
	childObj := types.NewTypeName(token.Pos(2), ssapkg.Pkg, "Child", nil)
	childScope.Insert(childObj)
	if ctx.isGenericLocalType(childObj) {
		t.Fatal("local type whose scope reaches package scope without type params should not be generic local")
	}
	if !ctx.isGenericLocalType(types.NewTypeName(fn.Syntax().Pos()+1, ssapkg.Pkg, "SyntaxLocal", nil)) {
		t.Fatal("parentless object inside current function syntax should be generic local")
	}
	if ctx.inCurrentFunction(token.NoPos) {
		t.Fatal("invalid position should not be inside current function")
	}
	if (&context{}).currentFunctionSyntax() != nil {
		t.Fatal("currentFunctionSyntax without goFn should be nil")
	}
	if (&context{}).localTypeOrdinalBySyntax(token.Pos(1)) != 0 {
		t.Fatal("localTypeOrdinalBySyntax without current function should be zero")
	}
	if !ctx.inCurrentFunction(fn.Syntax().Pos() + 1) {
		t.Fatal("position inside current function should be detected")
	}

	typeParamObj := types.NewTypeName(token.NoPos, ssapkg.Pkg, "U", nil)
	typeParam := types.NewTypeParam(typeParamObj, types.NewInterfaceType(nil, nil))
	genericScope := types.NewScope(ssapkg.Pkg.Scope(), token.NoPos, token.NoPos, "generic")
	genericScope.Insert(typeParamObj)
	boxObj := types.NewTypeName(token.Pos(40), ssapkg.Pkg, "Box", nil)
	genericScope.Insert(boxObj)
	box := types.NewNamed(boxObj, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, ssapkg.Pkg, "value", typeParam, false),
	}, nil), nil)
	box.SetTypeParams([]*types.TypeParam{typeParam})
	boxInt, err := types.Instantiate(types.NewContext(), box, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate(Box[int]) failed: %v", err)
	}
	if got := ctx.localNamedName(boxInt.(*types.Named), false); !strings.Contains(got, ";int]") {
		t.Fatalf("localNamedName(local generic with own args) = %q, want outer and own args", got)
	}
	if got := typeListArgs(boxInt.(*types.Named).TypeArgs(), ctx.typeArgName); len(got) != 1 || got[0] != "int" {
		t.Fatalf("typeListArgs(non-nil) = %v, want [int]", got)
	}

	mustPanicContains(t, "invalid channel direction", func() {
		_ = chanDirName(types.ChanDir(99))
	})
}

func firstGenericLocalNamed(t *testing.T, ssapkg *ssa.Package) (*ssa.Function, *types.Named) {
	t.Helper()
	for fn := range ssautil.AllFunctions(ssapkg.Prog) {
		if fn == nil || len(fn.TypeArgs()) == 0 {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if mi, ok := instr.(*ssa.MakeInterface); ok {
					if named := findLocalNamed(mi.X.Type(), ssapkg.Pkg); named != nil {
						return fn, named
					}
				}
				if val, ok := instr.(ssa.Value); ok {
					if named := findLocalNamed(val.Type(), ssapkg.Pkg); named != nil {
						return fn, named
					}
				}
			}
		}
	}
	t.Fatal("could not find instantiated generic local named type")
	return nil, nil
}

func findLocalNamed(typ types.Type, pkg *types.Package) *types.Named {
	switch t := types.Unalias(typ).(type) {
	case *types.Named:
		if obj := t.Obj(); obj != nil && obj.Pkg() == pkg && obj.Parent() != pkg.Scope() {
			return t
		}
	case *types.Pointer:
		return findLocalNamed(t.Elem(), pkg)
	case *types.Slice:
		return findLocalNamed(t.Elem(), pkg)
	case *types.Array:
		return findLocalNamed(t.Elem(), pkg)
	case *types.Map:
		if named := findLocalNamed(t.Key(), pkg); named != nil {
			return named
		}
		return findLocalNamed(t.Elem(), pkg)
	case *types.Chan:
		return findLocalNamed(t.Elem(), pkg)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if named := findLocalNamed(t.Field(i).Type(), pkg); named != nil {
				return named
			}
		}
	}
	return nil
}

func TestFuncName_NestedClosureInMethodIncludesRecv(t *testing.T) {
	const src = `package foo

type A struct{}
type B struct{}

func (a *A) marshal() int {
	f := func() int {
		g := func() int { return 1 }
		return g()
	}
	return f()
}

func (b *B) marshal() int {
	f := func() int {
		g := func() int { return 2 }
		return g()
	}
	return f()
}
`

	ssapkg := buildSSAPackage(t, src)

	got := collectFuncNamesBySSAName(ssapkg, "marshal$1$1")

	want := []string{
		"foo.(*A).marshal$1$1",
		"foo.(*B).marshal$1$1",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("got %d closure func(s) named marshal$1$1: %v, want %v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFuncName_TopLevelAndPlainClosureNoRecv(t *testing.T) {
	const src = `package foo

func top() {}

func outer() {
	f := func() {}
	f()
}
`
	ssapkg := buildSSAPackage(t, src)

	topFn := ssapkg.Func("top")
	if topFn == nil {
		t.Fatal("top function not found")
	}
	if got, want := funcName(ssapkg.Pkg, topFn, false), "foo.top"; got != want {
		t.Fatalf("top-level func name = %q, want %q", got, want)
	}

	closures := collectFuncNamesBySSAName(ssapkg, "outer$1")
	if len(closures) != 1 {
		t.Fatalf("got %d closure name(s) for outer$1: %v, want 1", len(closures), closures)
	}
	if got, want := closures[0], "foo.outer$1"; got != want {
		t.Fatalf("closure name = %q, want %q", got, want)
	}
}

func TestFuncName_ThunkAndBoundStillCarryRecv(t *testing.T) {
	const src = `package foo

type A int
type B int

func (A) f() int { return 1 }
func (B) f() int { return 2 }

var ta = A.f
var tb = B.f
var ba = A(0).f
var bb = B(0).f
`
	ssapkg := buildSSAPackage(t, src)

	thunks := collectFuncNamesBySSAName(ssapkg, "f$thunk")
	expectTwoDistinctNamesForTypes(t, thunks, "f$thunk")

	bounds := collectFuncNamesBySSAName(ssapkg, "f$bound")
	expectTwoDistinctNamesForTypes(t, bounds, "f$bound")
}
