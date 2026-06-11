//go:build !llgo
// +build !llgo

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestFieldOutOfRangePanicsWithTypeString(t *testing.T) {
	st := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil)
	typ := &aType{raw: rawType{Type: st}}

	var p Program
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic")
		}
		msg := r.(string)
		if !strings.Contains(msg, "Field: struct index out of range") {
			t.Fatalf("panic = %q, want out-of-range message", msg)
		}
	}()
	_ = p.Field(typ, 1)
}

func TestTypeStringWithPkgAndIsPkgScope(t *testing.T) {
	obj := types.NewTypeName(token.NoPos, nil, "Local", nil)
	local := types.NewNamed(obj, types.Typ[types.Int], nil)
	if got := typeStringWithPkg(local); got != "Local" {
		t.Fatalf("typeStringWithPkg(local) = %q, want %q", got, "Local")
	}

	pkg1 := types.NewPackage("example.com/p", "p")
	if !isPkgScope(pkg1.Scope(), pkg1.Scope()) {
		t.Fatalf("isPkgScope(scope, scope) should be true")
	}
	if isPkgScope(nil, pkg1.Scope()) {
		t.Fatalf("isPkgScope(nil, scope) should be false")
	}
}

func TestNamedTypeEquivalent(t *testing.T) {
	pkg1 := types.NewPackage("example.com/p", "p")
	pkg2 := types.NewPackage("example.com/p", "p")
	obj1 := types.NewTypeName(token.NoPos, pkg1, "T", nil)
	obj2 := types.NewTypeName(token.NoPos, pkg2, "T", nil)
	a := types.NewNamed(obj1, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil), nil)
	b := types.NewNamed(obj2, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil), nil)
	if !namedTypeEquivalent(a, b) {
		t.Fatalf("namedTypeEquivalent should be true for equivalent named structs")
	}

	c := types.NewNamed(types.NewTypeName(token.NoPos, pkg2, "T", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.String], false),
	}, nil), nil)
	if namedTypeEquivalent(a, c) {
		t.Fatalf("namedTypeEquivalent should be false for different underlying types")
	}
}

func TestNamedTypeEquivalentRecursiveSignature(t *testing.T) {
	pkg1 := types.NewPackage("example.com/p", "p")
	pkg2 := types.NewPackage("example.com/p", "p")
	a := types.NewNamed(types.NewTypeName(token.NoPos, pkg1, "F", nil), nil, nil)
	b := types.NewNamed(types.NewTypeName(token.NoPos, pkg2, "F", nil), nil, nil)
	a.SetUnderlying(types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", a)),
		types.NewTuple(types.NewParam(token.NoPos, nil, "", a)),
		false))
	b.SetUnderlying(types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", b)),
		types.NewTuple(types.NewParam(token.NoPos, nil, "", b)),
		false))
	if namedTypeEquivalent(a, b) {
		t.Fatalf("namedTypeEquivalent should be false for recursive function signatures")
	}
}

func TestNamedTypeEquivalentRejectsAnySignature(t *testing.T) {
	pkg1 := types.NewPackage("example.com/p", "p")
	pkg2 := types.NewPackage("example.com/p", "p")
	sigNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkg1, "F", nil),
		types.NewSignatureType(nil, nil, nil, nil, nil, false), nil)
	structNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkg2, "F", nil),
		types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
		}, nil), nil)
	if namedTypeEquivalent(sigNamed, structNamed) {
		t.Fatalf("namedTypeEquivalent should be false when only one side is a signature")
	}
	if namedTypeEquivalent(structNamed, sigNamed) {
		t.Fatalf("namedTypeEquivalent should be false when only one side is a signature")
	}
}

func TestToLLVMFuncPtrUsesVoidPtr(t *testing.T) {
	prog := NewProgram(nil)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	if got, want := prog.toLLVMFuncPtr(sig).String(), prog.tyVoidPtr().String(); got != want {
		t.Fatalf("toLLVMFuncPtr() = %q, want %q", got, want)
	}
}

func TestPointerTypeDoesNotExpandRecursiveNamedElement(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "recursive.go", `package p

type T18 *[10]T19
type T19 T18
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("example.com/p", "p")
	if err := types.NewChecker(&types.Config{}, fset, pkg, nil).Files([]*ast.File{f}); err != nil {
		t.Fatal(err)
	}

	prog := NewProgram(nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	t18 := pkg.Scope().Lookup("T18").Type()
	if got, want := prog.Type(t18, InGo).ll.String(), prog.tyVoidPtr().String(); got != want {
		t.Fatalf("recursive named pointer LLVM type = %q, want %q", got, want)
	}
}

func TestGoProgramSizesUnaliasFunctionFieldStruct(t *testing.T) {
	prog := NewProgram(nil)
	sizes := prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fnStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Fn", sig, false),
	}, nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "FnStruct", nil), fnStruct)
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "A", alias, false),
		types.NewField(token.NoPos, nil, "B", types.Typ[types.Int], false),
	}

	want := int64(prog.PointerSize() * 2)
	if got := sizes.Sizeof(alias); got != want {
		t.Fatalf("Sizeof(alias to func-field struct) = %d, want %d", got, want)
	}
	if got := sizes.Offsetsof(fields)[1]; got != want {
		t.Fatalf("Offsetsof(field after alias to func-field struct) = %d, want %d", got, want)
	}
}

func TestGoProgramSizesNamedFunctionUsesPointerSize(t *testing.T) {
	prog := NewProgram(nil)
	sizes := prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Fn", nil), sig, nil)
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "Fn", fn, false),
		types.NewField(token.NoPos, nil, "V", types.Typ[types.Int], false),
	}

	want := int64(prog.PointerSize())
	if got := sizes.Sizeof(fn); got != want {
		t.Fatalf("Sizeof(named func) = %d, want %d", got, want)
	}
	if got := sizes.Offsetsof(fields)[1]; got != want {
		t.Fatalf("Offsetsof(field after named func) = %d, want %d", got, want)
	}
}

func TestNamedStructLayoutEquivalent(t *testing.T) {
	prog := NewProgram(nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))

	pkg := types.NewPackage("example.com/p", "p")
	s1 := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil)
	n1 := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), s1, nil)
	llStruct, _ := prog.toLLVMStruct(s1)
	existing := &aType{
		ll:  llStruct,
		raw: rawType{Type: n1},
	}

	same := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil), nil)
	if !prog.namedStructLayoutEquivalent(existing, same) {
		t.Fatalf("namedStructLayoutEquivalent should be true for same layout")
	}

	diff := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
		types.NewField(token.NoPos, nil, "B", types.Typ[types.Int], false),
	}, nil), nil)
	if prog.namedStructLayoutEquivalent(existing, diff) {
		t.Fatalf("namedStructLayoutEquivalent should be false for different layout")
	}
}

func buildGoSSAPackageForOpaque(t *testing.T, src string) *gossa.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "opaque.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage("foo", "foo")
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg
}

func TestGoSSAOpaqueTypeConversion(t *testing.T) {
	ssapkg := buildGoSSAPackageForOpaque(t, `package foo

func seq(yield func(int) bool) { _ = yield(1) }

func f() {
	for v := range seq {
		defer func() { _ = v }()
	}
}
`)

	var deferStackTy types.Type
	for fn := range ssautil.AllFunctions(ssapkg.Prog) {
		if fn == nil {
			continue
		}
		for _, blk := range fn.Blocks {
			for _, instr := range blk.Instrs {
				if d, ok := instr.(*gossa.Defer); ok && d.DeferStack != nil {
					deferStackTy = d.DeferStack.Type()
					break
				}
			}
			if deferStackTy != nil {
				break
			}
		}
		if deferStackTy != nil {
			break
		}
	}
	if deferStackTy == nil {
		t.Fatal("missing defer stack type")
	}

	ptrTy, ok := deferStackTy.(*types.Pointer)
	if !ok {
		t.Fatalf("expected pointer defer stack type, got %T", deferStackTy)
	}
	if !isGoSSAOpaqueType(ptrTy.Elem()) {
		t.Fatalf("expected opaque defer stack elem type, got %T", ptrTy.Elem())
	}
	if raw, ok := cvtGoSSAOpaqueType(deferStackTy); !ok || raw != types.Typ[types.UnsafePointer] {
		t.Fatalf("cvtGoSSAOpaqueType = (%v, %v), want (unsafe.Pointer, true)", raw, ok)
	}
	if raw, ok := cvtGoSSAOpaqueType(ptrTy.Elem()); !ok || raw != types.Typ[types.UnsafePointer] {
		t.Fatalf("cvtGoSSAOpaqueType(ptr) = (%v, %v), want (unsafe.Pointer, true)", raw, ok)
	}
	if isGoSSAOpaqueType(types.Typ[types.Int]) {
		t.Fatal("plain int must not be treated as opaque type")
	}
	if raw, ok := cvtGoSSAOpaqueType(types.Typ[types.Int]); ok || raw != nil {
		t.Fatalf("cvtGoSSAOpaqueType(int) = (%v, %v), want (nil, false)", raw, ok)
	}

	prog := NewProgram(nil)
	if got := prog.toType(deferStackTy).RawType(); got != types.Typ[types.UnsafePointer] {
		t.Fatalf("toType(opaque) raw = %v, want unsafe.Pointer", got)
	}
	if got := prog.toType(ptrTy.Elem()).RawType(); got != types.Typ[types.UnsafePointer] {
		t.Fatalf("toType(opaque elem) raw = %v, want unsafe.Pointer", got)
	}
}

func TestCvtStructPreservesTags(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	st := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Fn", sig, false),
	}, []string{`protobuf:"bytes,1,opt,name=fn" json:"fn,omitempty"`})

	raw, cvt := newGoTypes().cvtType(st)
	if !cvt {
		t.Fatalf("cvtType did not convert function field")
	}
	got := raw.(*types.Struct).Tag(0)
	want := `protobuf:"bytes,1,opt,name=fn" json:"fn,omitempty"`
	if got != want {
		t.Fatalf("converted struct tag = %q, want %q", got, want)
	}
}
