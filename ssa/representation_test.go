package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestRepresentationUsesResolvedAttributes(t *testing.T) {
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
	prog.SetTypeRepresentation(copyType.Obj(), prog.RepresentationOf(native))
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

func TestRepresentationCheckingAndPublication(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("p", "p")
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), types.Typ[types.Uint64], nil)
	copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), types.Typ[types.Uint64], nil)
	prog.SetTypeCallConv("p.Native", CDecl)
	var bound bool
	calls := 0
	nativeRep := prog.DeclaredTypeRepresentation(native.Obj())
	prog.SetTypeRepresentationChecker(pkg, func(obj *types.TypeName) TypeRepresentation {
		calls++
		if bound {
			return nativeRep
		}
		return TypeRepresentation{}
	})
	if prog.sourceRepresentationOf(copyType).Storage != GoRepresentation {
		t.Fatal("unresolved source is native")
	}
	bound = true
	if prog.sourceRepresentationOf(copyType) != nativeRep {
		t.Fatal("source sizes cached an incomplete binding")
	}
	for _, query := range []func(){
		func() { prog.RepresentationOf(copyType) },
		func() { prog.NewBackendProgram().Dispose() },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("backend accepted unfinished attributes")
				}
			}()
			query()
		}()
	}
	snapshot := map[*types.TypeName]TypeRepresentation{copyType.Obj(): nativeRep}
	prog.PublishTypeRepresentations(pkg, snapshot)
	// Neither the old callback nor the caller's mutable map may affect layout.
	bound = false
	snapshot[copyType.Obj()] = TypeRepresentation{}
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	for _, p := range []Program{prog, backend} {
		if p.RepresentationOf(copyType) != nativeRep || p.sourceRepresentationOf(copyType) != nativeRep {
			t.Fatal("published attributes depend on checker state")
		}
	}
	if calls != 2 || len(prog.checkingRepresentations) != 0 || len(backend.checkingRepresentations) != 0 {
		t.Fatal("checker survived frontend completion")
	}
	defer func() {
		if recover() == nil {
			t.Error("published representation was mutable")
		}
	}()
	prog.SetTypeRepresentation(copyType.Obj(), TypeRepresentation{})
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
			prog.SetTypeRepresentation(copyType.Obj(), prog.RepresentationOf(native))
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
	prog.SetTypeRepresentation(copyType.Obj(), prog.RepresentationOf(native))
	if !prog.isStdcallType(copyType) || prog.Type(copyType, GoDecl).kind != vkFuncPtr {
		t.Fatal("derived native callback lost its pointer representation or invocation ABI")
	}
}

func TestRepresentationRejectsInvalidAttributes(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	obj := types.NewTypeName(0, types.NewPackage("p", "p"), "Copy", nil)
	for _, tc := range []struct {
		name string
		obj  *types.TypeName
		rep  TypeRepresentation
	}{
		{"missing owner", nil, TypeRepresentation{}},
		{"missing native origin", obj, TypeRepresentation{Storage: NativeRepresentation, NativeABI: CDecl}},
		{"missing native ABI", obj, TypeRepresentation{Storage: NativeRepresentation, Origin: obj}},
		{"Go storage with native ABI", obj, TypeRepresentation{NativeABI: CDecl}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != "ssa: invalid type representation" {
					t.Fatalf("invalid attributes accepted: %v", got)
				}
			}()
			prog.SetTypeRepresentation(tc.obj, tc.rep)
		})
	}
	if len(prog.packageSyntax.typeRepresentations) != 0 {
		t.Fatal("rejected attributes were partially published")
	}
}

func TestRepresentationAdapterIsBackendLocal(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("p", "p")
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), types.Typ[types.Uint64], nil)
	prog.SetTypeCallConv("p.Native", CDecl)
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	other := prog.NewBackendProgram()
	defer other.Dispose()
	adapter := types.NewNamed(types.NewTypeName(0, pkg, "Copy[int]", nil), native.Underlying(), nil)
	backend.CopyTypeRepresentation(adapter.Obj(), native)
	alias := types.NewAlias(types.NewTypeName(0, pkg, "Alias", nil), adapter)
	if backend.RepresentationOf(alias).Storage != NativeRepresentation {
		t.Fatal("alias lost its backend adapter's representation")
	}
	if backend.RepresentationOf(adapter).Storage != NativeRepresentation || backend.gocvt.shouldConvertNamed(adapter) {
		t.Fatal("backend adapter lost the resolved representation")
	}
	if prog.RepresentationOf(adapter).Storage != GoRepresentation || other.RepresentationOf(adapter).Storage != GoRepresentation {
		t.Fatal("backend adapter mutated the shared frontend snapshot")
	}
}
