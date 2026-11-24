package types

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/target"
)

type testImporter struct {
	base gotypes.Importer
}

func (ti testImporter) Import(path string) (*gotypes.Package, error) {
	if path == "structs" {
		pkg := gotypes.NewPackage("structs", "structs")
		inner := gotypes.NewNamed(
			gotypes.NewTypeName(token.NoPos, pkg, "hostLayout", nil),
			gotypes.NewStruct(nil, nil),
			nil,
		)
		hl := gotypes.NewNamed(
			gotypes.NewTypeName(token.NoPos, pkg, "HostLayout", nil),
			gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, pkg, "_", inner, false)}, nil),
			nil,
		)
		pkg.Scope().Insert(hl.Obj())
		pkg.MarkComplete()
		return pkg, nil
	}
	return ti.base.Import(path)
}

func checkPkg(t *testing.T, src string) *gotypes.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := gotypes.Config{Importer: testImporter{base: importer.Default()}}
	pkg, err := conf.Check("example.com/sample", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestMapperClosureAndCFunc(t *testing.T) {
	pkg := checkPkg(t, `package sample
type GoCallback func(int) int
type CCallback func(int) int
type Rec struct {
	Next *Rec
	Fn   GoCallback
	CFn  CCallback
}
`)
	mapper := NewMapper(func(name string) (CallConv, bool) {
		if name == "example.com/sample.CCallback" {
			return CDecl, true
		}
		return CallConvUnknown, false
	})

	goCB := mapper.Map(pkg.Scope().Lookup("GoCallback").Type(), GoDecl)
	if _, ok := goCB.Underlying().(*Closure); !ok {
		t.Fatalf("GoCallback underlying = %T, want *Closure", goCB.Underlying())
	}
	cCB := mapper.Map(pkg.Scope().Lookup("CCallback").Type(), GoDecl)
	fn, ok := cCB.Underlying().(*Func)
	if !ok || fn.CallConv != CDecl {
		t.Fatalf("CCallback underlying = %#v, want CDecl *Func", cCB.Underlying())
	}

	rec := mapper.Map(pkg.Scope().Lookup("Rec").Type(), GoDecl)
	st := rec.Underlying().(*Struct)
	if st.Fields[0].Type.(*Pointer).Elem != rec {
		t.Fatal("recursive *Rec back-edge did not preserve identity")
	}
	if _, ok := st.Fields[1].Type.Underlying().(*Closure); !ok {
		t.Fatalf("Rec.Fn = %T, want *Closure", st.Fields[1].Type.Underlying())
	}
	if _, ok := st.Fields[2].Type.Underlying().(*Func); !ok {
		t.Fatalf("Rec.CFn = %T, want *Func", st.Fields[2].Type.Underlying())
	}

	// Verify sizes & GC pointer offsets on amd64.
	amd64 := NewSizes(target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}))
	if got := amd64.Sizeof(goCB); got != 16 {
		t.Fatalf("amd64 sizeof(Closure) = %d, want 16", got)
	}
	if got := amd64.Sizeof(cCB); got != 8 {
		t.Fatalf("amd64 sizeof(CCallback) = %d, want 8", got)
	}
	// Rec = {Next *Rec (0, ptr), Fn Closure (8..24, env ptr at 16), CFn CCallback (24..32, non-GC)}
	if got := amd64.Sizeof(rec); got != 32 {
		t.Fatalf("amd64 sizeof(Rec) = %d, want 32", got)
	}
	if got := amd64.PtrOffsets(rec); !reflect.DeepEqual(got, []int64{0, 16}) {
		t.Fatalf("amd64 PtrOffsets(Rec) = %v, want [0 16]", got)
	}
	if got := amd64.PtrDataSize(rec); got != 24 {
		t.Fatalf("amd64 PtrDataSize(Rec) = %d, want 24", got)
	}

	// Verify wide Go storage on wasm32/w32: Go word=8, native ptr=4.
	w32 := NewSizes(target.ResolveSpec(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}))
	if got := w32.Sizeof(goCB); got != 16 {
		t.Fatalf("w32 sizeof(Closure) = %d, want 16", got)
	}
	if got := w32.Sizeof(cCB); got != 4 {
		t.Fatalf("w32 sizeof(CCallback) = %d, want 4", got)
	}
}

func TestSizesNativePtrAndHostLayout(t *testing.T) {
	pkg := checkPkg(t, `package sample
import "structs"
type Host struct {
	_ structs.HostLayout
	a byte
	b int64
}
type GoStruct struct {
	a byte
	b int64
	z struct{}
}
`)
	mapper := NewMapper(nil)
	hostT := mapper.Map(pkg.Scope().Lookup("Host").Type(), GoDecl)
	goT := mapper.Map(pkg.Scope().Lookup("GoStruct").Type(), GoDecl)

	if !HasHostLayout(pkg.Scope().Lookup("Host").Type()) || HasHostLayout(pkg.Scope().Lookup("GoStruct").Type()) {
		t.Fatal("HasHostLayout classification mismatch")
	}

	// On Windows/386, Go aligns int64 to 4 bytes while HostLayout aligns int64 to 8 bytes.
	win386 := NewSizes(target.ResolveSpec(target.Spec{GOOS: "windows", GOARCH: "386"}))
	if got := win386.Alignof(hostT); got != 8 {
		t.Fatalf("windows/386 HostLayout align = %d, want 8", got)
	}
	if got := win386.Sizeof(hostT); got != 16 {
		t.Fatalf("windows/386 HostLayout size = %d, want 16", got)
	}
	if got := win386.Alignof(goT); got != 4 {
		t.Fatalf("windows/386 GoStruct align = %d, want 4", got)
	}
	// GoStruct on 386: byte(0), pad(1..3), int64(4..12), z(12, trailing zero -> +1 -> aligned to 16).
	if got := win386.Sizeof(goT); got != 16 {
		t.Fatalf("windows/386 GoStruct size = %d, want 16", got)
	}

	// Native pointers carry zero GC pointer data.
	nativePtr := &Pointer{Elem: &Basic{Kind: gotypes.Int32}, Kind: NativePtr}
	if nativePtr.HasPtrData() || win386.PtrDataSize(nativePtr) != 0 {
		t.Fatal("NativePtr must not report GC pointer data")
	}

	// PtrDataSize on huge arrays runs in O(depth) without materializing offsets.
	amd64 := NewSizes(target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}))
	elem := &Struct{
		Fields: []*Field{
			{Name: "p", Type: &Pointer{Elem: &Basic{Kind: gotypes.Int64}, Kind: GoPtr}},
			{Name: "n", Type: &Basic{Kind: gotypes.Int64}},
		},
	}
	hugeArr := &Array{Elem: elem, Len: 1 << 20}
	wantPtrBytes := ((int64(1)<<20)-1)*16 + 8
	if got := amd64.PtrDataSize(hugeArr); got != wantPtrBytes {
		t.Fatalf("PtrDataSize([1<<20]struct{p *int64; n int64}) = %d, want %d", got, wantPtrBytes)
	}
	smallArr := &Array{Elem: elem, Len: 3}
	if got := amd64.PtrOffsets(smallArr); !reflect.DeepEqual(got, []int64{0, 16, 32}) {
		t.Fatalf("PtrOffsets([3]struct) = %v, want [0 16 32]", got)
	}
	if got := amd64.PtrDataSize(smallArr); got != 40 {
		t.Fatalf("PtrDataSize([3]struct) = %d, want 40", got)
	}

	// Tuple GC pointer maps must match Tuple layout (e.g. (int, *int) -> [8]).
	intT := &Basic{Kind: gotypes.Int}
	byteT := &Basic{Kind: gotypes.Uint8}
	ptrInt := &Pointer{Elem: intT, Kind: GoPtr}
	tup := &Tuple{
		Elems: []*Var{
			{Name: "a", Type: intT},
			{Name: "b", Type: ptrInt},
		},
	}
	if !tup.HasPtrData() {
		t.Fatal("(int, *int) HasPtrData = false, want true")
	}
	if got := amd64.Sizeof(tup); got != 16 {
		t.Fatalf("amd64 Sizeof((int, *int)) = %d, want 16", got)
	}
	if got := amd64.PtrOffsets(tup); !reflect.DeepEqual(got, []int64{8}) {
		t.Fatalf("amd64 PtrOffsets((int, *int)) = %v, want [8]", got)
	}
	if got := amd64.PtrDataSize(tup); got != 16 {
		t.Fatalf("amd64 PtrDataSize((int, *int)) = %d, want 16", got)
	}

	paddedTup := &Tuple{
		Elems: []*Var{
			{Name: "tag", Type: byteT},
			{Name: "p", Type: ptrInt},
			{Name: "tail", Type: intT},
		},
	}
	if got := amd64.PtrOffsets(paddedTup); !reflect.DeepEqual(got, []int64{8}) {
		t.Fatalf("amd64 PtrOffsets((byte, *int, int)) = %v, want [8]", got)
	}
	if got := amd64.PtrDataSize(paddedTup); got != 16 {
		t.Fatalf("amd64 PtrDataSize((byte, *int, int)) = %d, want 16", got)
	}

	nestedTup := &Tuple{
		Elems: []*Var{
			{Name: "first", Type: ptrInt},
			{Name: "inner", Type: &Tuple{Elems: []*Var{{Type: byteT}, {Type: ptrInt}}}},
		},
	}
	if got := amd64.PtrOffsets(nestedTup); !reflect.DeepEqual(got, []int64{0, 16}) {
		t.Fatalf("amd64 PtrOffsets((*int, (byte, *int))) = %v, want [0 16]", got)
	}
	if got := amd64.PtrDataSize(nestedTup); got != 24 {
		t.Fatalf("amd64 PtrDataSize((*int, (byte, *int))) = %d, want 24", got)
	}

	noPtrTup := &Tuple{Elems: []*Var{{Type: intT}, {Type: &Basic{Kind: gotypes.Float64}}}}
	if noPtrTup.HasPtrData() || len(amd64.PtrOffsets(noPtrTup)) != 0 || amd64.PtrDataSize(noPtrTup) != 0 {
		t.Fatal("pointer-free tuple must not report GC pointer data")
	}
}
