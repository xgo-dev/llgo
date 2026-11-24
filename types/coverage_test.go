package types

import (
	"go/token"
	gotypes "go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/target"
)

func TestAllTypeMethodsAndLayouts(t *testing.T) {
	for _, cc := range []CallConv{CallConvUnknown, GoDecl, CDecl, PyDecl, Stdcall} {
		if cc.String() == "" {
			t.Fatalf("CallConv(%d).String() is empty", cc)
		}
		wantNative := cc == CDecl || cc == Stdcall
		if cc.IsNative() != wantNative {
			t.Fatalf("CallConv(%v).IsNative() = %v, want %v", cc, cc.IsNative(), wantNative)
		}
	}
	if GoPtr.String() != "goptr" || NativePtr.String() != "nativeptr" {
		t.Fatalf("PtrKind.String() = %q, %q", GoPtr.String(), NativePtr.String())
	}

	m := NewMapper(nil)
	goPkg := gotypes.NewPackage("example.com/cov", "cov")
	namedObj := gotypes.NewTypeName(token.NoPos, goPkg, "Node", nil)
	nodeNamed := gotypes.NewNamed(namedObj, nil, nil)
	nodeStruct := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(token.NoPos, goPkg, "Next", gotypes.NewPointer(nodeNamed), false),
		gotypes.NewField(token.NoPos, goPkg, "Tag", gotypes.Typ[gotypes.Uint8], false),
		gotypes.NewField(token.NoPos, goPkg, "Data", gotypes.NewSlice(gotypes.Typ[gotypes.Byte]), false),
		gotypes.NewField(token.NoPos, goPkg, "Dict", gotypes.NewMap(gotypes.Typ[gotypes.String], gotypes.Typ[gotypes.Int32]), false),
		gotypes.NewField(token.NoPos, goPkg, "Ch", gotypes.NewChan(gotypes.SendRecv, gotypes.Typ[gotypes.Int]), false),
		gotypes.NewField(token.NoPos, goPkg, "Any", gotypes.NewInterfaceType(nil, nil).Complete(), false),
		gotypes.NewField(token.NoPos, goPkg, "Err", gotypes.Universe.Lookup("error").Type(), false),
		gotypes.NewField(token.NoPos, goPkg, "Arr", gotypes.NewArray(gotypes.Typ[gotypes.Float32], 3), false),
		gotypes.NewField(token.NoPos, goPkg, "Raw", gotypes.Typ[gotypes.UnsafePointer], false),
		gotypes.NewField(token.NoPos, goPkg, "C128", gotypes.Typ[gotypes.Complex128], false),
		gotypes.NewField(token.NoPos, goPkg, "C64", gotypes.Typ[gotypes.Complex64], false),
		gotypes.NewField(token.NoPos, goPkg, "Flag", gotypes.Typ[gotypes.Bool], false),
		gotypes.NewField(token.NoPos, goPkg, "I16", gotypes.Typ[gotypes.Int16], false),
		gotypes.NewField(token.NoPos, goPkg, "U64", gotypes.Typ[gotypes.Uint64], false),
	}, nil)
	nodeNamed.SetUnderlying(nodeStruct)

	sig := gotypes.NewSignatureType(
		gotypes.NewVar(token.NoPos, goPkg, "n", nodeNamed),
		nil, nil,
		gotypes.NewTuple(gotypes.NewParam(token.NoPos, goPkg, "x", gotypes.NewSlice(gotypes.Typ[gotypes.Int]))),
		gotypes.NewTuple(gotypes.NewParam(token.NoPos, goPkg, "ok", gotypes.Typ[gotypes.Bool])),
		true,
	)

	llNode := m.Map(nodeNamed, GoDecl).(*Named)
	llStruct := llNode.Underlying().(*Struct)
	llSig := m.MapSignature(sig, GoDecl)
	llClosure := m.Map(sig, GoDecl).(*Closure)
	llSIMD := &SIMDVec{Elem: &Basic{Kind: gotypes.Float32}, Len: 4}
	llTuple := &Tuple{Elems: []*Var{{Name: "a", Type: &Basic{Kind: gotypes.Int}}, {Name: "b", Type: &String{}}}}
	emptyNamed := &Named{Name: "Anon"}
	bareIface := &Interface{}

	allTypes := []Type{
		llNode, llStruct, llSig, llClosure, llSIMD, llTuple, emptyNamed, bareIface,
		&Basic{Kind: gotypes.Int64},
		&Basic{Kind: gotypes.UnsafePointer, PtrKind: NativePtr},
		&Pointer{Elem: &Basic{Kind: gotypes.Int}, Kind: GoPtr},
		&Pointer{Elem: &Basic{Kind: gotypes.Int}, Kind: NativePtr},
		&String{},
		&Slice{Elem: &Basic{Kind: gotypes.Byte}},
		&Map{Key: &String{}, Elem: &Basic{Kind: gotypes.Int}},
		&Chan{Elem: &Basic{Kind: gotypes.Int}},
		&Array{Elem: &Pointer{Elem: &Basic{Kind: gotypes.Int}, Kind: GoPtr}, Len: 2},
	}
	for _, f := range llStruct.Fields {
		allTypes = append(allTypes, f.Type)
	}

	for _, spec := range []target.Spec{
		target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
		target.ResolveSpec(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"}),
	} {
		sz := NewSizes(spec)
		for _, typ := range allTypes {
			typ.llTypeMarker()
			_ = typ.GoType()
			_ = typ.Underlying()
			_ = typ.HasPtrData()
			if typ.String() == "" {
				t.Fatalf("empty String() for %T", typ)
			}
			_ = sz.Sizeof(typ)
			_ = sz.Alignof(typ)
			_ = sz.PtrDataSize(typ)
			_ = sz.PtrOffsets(typ)
		}
		if offs := sz.Offsetsof(llStruct); len(offs) != len(llStruct.Fields) {
			t.Fatalf("Offsetsof len = %d, want %d", len(offs), len(llStruct.Fields))
		}
	}

	if emptyNamed.Underlying() != nil || emptyNamed.HasPtrData() || emptyNamed.String() != "Anon" {
		t.Fatalf("unexpected empty Named behavior: %v", emptyNamed.String())
	}
	if !strings.Contains(bareIface.String(), "interface") {
		t.Fatalf("bare Interface.String() = %q", bareIface.String())
	}
	if ContainsHostLayout(nodeStruct) {
		t.Fatal("plain struct unexpectedly reports ContainsHostLayout")
	}
	if ContainsHostLayout(nil) || HasHostLayout(nil) || IsHostLayoutMarker(nil) {
		t.Fatal("nil unexpectedly reports HostLayout")
	}
	hlPkg := gotypes.NewPackage("structs", "structs")
	hlInner := gotypes.NewNamed(gotypes.NewTypeName(token.NoPos, hlPkg, "hostLayout", nil), gotypes.NewStruct(nil, nil), nil)
	hlType := gotypes.NewNamed(
		gotypes.NewTypeName(token.NoPos, hlPkg, "HostLayout", nil),
		gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, hlPkg, "_", hlInner, false)}, nil),
		nil,
	)
	hlStruct := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(token.NoPos, goPkg, "_", hlType, false),
		gotypes.NewField(token.NoPos, goPkg, "X", gotypes.Typ[gotypes.Int32], false),
	}, nil)
	outerStruct := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(token.NoPos, goPkg, "Inner", gotypes.NewArray(hlStruct, 2), false),
	}, nil)
	if !ContainsHostLayout(outerStruct) {
		t.Fatal("expected ContainsHostLayout for struct wrapping [2]HostLayoutStruct")
	}
}
