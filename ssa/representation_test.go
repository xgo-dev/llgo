package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestRepresentationRequiresDefinitionProvenance(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("example.com/p", "p")
	st := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Fn", types.NewSignatureType(nil, nil, nil, nil, nil, false), false),
		types.NewField(token.NoPos, pkg, "Tail", types.Typ[types.Uint64], false),
	}, nil)
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), st, nil)
	copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), st, nil)
	independent := types.NewNamed(types.NewTypeName(0, pkg, "Independent", nil), st, nil)
	prog.SetTypeCallConv("example.com/p.Native", CDecl)
	prog.SetTypeDefinition(copyType.Obj(), native)
	if got := prog.RepresentationOf(copyType); got.Storage != NativeRepresentation || got.Origin != native.Obj() {
		t.Fatalf("derived representation = %+v", got)
	}
	if prog.RepresentationOf(independent).Storage != GoRepresentation {
		t.Fatal("shared fields were mistaken for a definition edge")
	}
	if _, registered := prog.packageTypeCallConv("example.com/p.Copy"); registered {
		t.Fatal("representation inheritance registered a new calling convention")
	}
	ordinaryCallback := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
		types.NewVar(0, pkg, "value", copyType),
		types.NewVar(0, pkg, "callback", ordinaryCallback),
	), nil, false)
	raw := prog.FuncDecl(sig, GoDecl).RawType().(*types.Signature)
	if raw.Params().At(0).Type() != copyType || !IsClosure(raw.Params().At(1).Type().Underlying().(*types.Struct)) {
		t.Fatal("native argument changed ordinary Go function parameter lowering")
	}
}

func TestRepresentationDoesNotCacheIncompleteSource(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("p", "p")
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), types.Typ[types.Uint64], nil)
	copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), types.Typ[types.Uint64], nil)
	prog.SetTypeCallConv("p.Native", CDecl)
	var rhs types.Type
	prog.SetTypeDefinitionSource(pkg, func(obj *types.TypeName) types.Type { return rhs })
	if prog.RepresentationOf(copyType).Storage != GoRepresentation {
		t.Fatal("unresolved source is native")
	}
	rhs = native
	if prog.RepresentationOf(copyType).Storage != NativeRepresentation {
		t.Fatal("unresolved source result was cached")
	}
	rhs = copyType
	if prog.RepresentationOf(copyType).Storage != GoRepresentation {
		t.Fatal("cyclic source resolved to a native root")
	}
}

func TestDerivedNativeStorageAcrossTargets(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			pkg := types.NewPackage("p", "p")
			st := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "Fn", types.NewSignatureType(nil, nil, nil, nil, nil, false), false),
				types.NewField(0, pkg, "Tail", types.Typ[types.Uint64], false),
			}, nil)
			native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), st, nil)
			copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), st, nil)
			prog.SetTypeCallConv("p.Native", CDecl)
			prog.SetTypeDefinition(copyType.Obj(), native)
			a, b := prog.Type(native, GoDecl), prog.Type(copyType, GoDecl)
			if prog.SizeOf(a) != prog.SizeOf(b) || prog.AlignOf(a) != prog.AlignOf(b) || prog.OffsetOf(a, 1) != prog.OffsetOf(b, 1) {
				t.Fatalf("inherited layout differs: Native %d/%d/%d, Copy %d/%d/%d",
					prog.SizeOf(a), prog.AlignOf(a), prog.OffsetOf(a, 1),
					prog.SizeOf(b), prog.AlignOf(b), prog.OffsetOf(b, 1))
			}
		})
	}
}

func TestDerivedNativeCallbackKeepsStdcallABI(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	native := newStdcallType(prog, "p", "Callback", sig)
	copyType := types.NewNamed(types.NewTypeName(0, native.Obj().Pkg(), "Copy", nil), sig, nil)
	prog.SetTypeDefinition(copyType.Obj(), native)
	if !prog.isStdcallType(copyType) || prog.Type(copyType, GoDecl).kind != vkFuncPtr {
		t.Fatal("derived native callback lost its pointer representation or invocation ABI")
	}
}
