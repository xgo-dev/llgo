package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"
)

func TestNativeRepresentationInsideHostLayout(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	sizes := prog.TypeSizes(types.SizesFor("gc", "386"))
	markerPkg, err := importer.Default().Import("structs")
	if err != nil {
		t.Fatal(err)
	}
	marker := markerPkg.Scope().Lookup("HostLayout").Type()
	pkg := types.NewPackage("p", "p")
	field := func(name string, typ types.Type) *types.Var { return types.NewField(0, pkg, name, typ, false) }
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), types.NewStruct([]*types.Var{field("Fn", NoArgsNoRet), field("Tail", types.Typ[types.Uint64])}, nil), nil)
	prog.SetTypeCallConv("p.Native", CDecl)
	fields := []*types.Var{field("_", marker), field("B", types.Typ[types.Byte]), field("F", types.Typ[types.Float64]), field("N", native)}
	st := types.NewStruct(fields, nil)
	physical := prog.Type(st, GoDecl)
	if got, want := sizes.Offsetsof(fields)[2], int64(prog.OffsetOf(physical, 2)); got != want || got != 8 {
		t.Fatalf("HostLayout float64 source/LLVM offset = %d/%d, want 8/8", got, want)
	}
	if uint64(sizes.Sizeof(st)) != prog.SizeOf(physical) || uint64(sizes.Alignof(st)) != prog.AlignOf(physical) {
		t.Fatal("native child replaced the containing HostLayout policy")
	}
}

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

func TestImportedGoRepresentationPreservesLocalShadow(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("p", "p")
	prog.SetTypeCallConv("p.Native", CDecl)
	// The type graph retains the original local identity, but reconstructed
	// objects may lack a Parent scope. A same-spelling package root is unrelated.
	local := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), types.NewStruct(nil, nil), nil)
	prog.SetImportedRepresentation(local.Obj(), TypeRepresentation{})
	if prog.RepresentationOf(local).Storage != GoRepresentation {
		t.Fatal("imported local type inherited an unrelated package directive")
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

func TestNativeRepresentationSourceLayoutAcrossTargets(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			pkg := types.NewPackage("p", "p")
			field := func(name string, typ types.Type) *types.Var {
				return types.NewField(0, pkg, name, typ, false)
			}
			fields := []*types.Var{
				field("Fn", NoArgsNoRet), field("Tail", types.Typ[types.Uint64]),
				field("Next", types.NewPointer(types.Typ[types.Byte])),
			}
			st := types.NewStruct(fields, nil)
			native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), st, nil)
			copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), st, nil)
			prog.SetTypeCallConv("p.Native", CDecl, fields...)
			prog.SetTypeDefinition(copyType.Obj(), native)
			empty := types.NewNamed(types.NewTypeName(0, pkg, "EmptyTail", nil), types.NewStruct([]*types.Var{field("X", types.Typ[types.Float64]), field("Z", types.NewStruct(nil, nil))}, nil), nil)
			prog.SetTypeCallConv("p.EmptyTail", CDecl)
			pointerOnly := types.NewNamed(types.NewTypeName(0, pkg, "PointerOnly", nil), types.NewStruct([]*types.Var{field("P", types.NewPointer(types.Typ[types.Byte]))}, nil), nil)
			prog.SetTypeCallConv("p.PointerOnly", CDecl)
			container := types.NewStruct([]*types.Var{
				field("Prefix", types.Typ[types.Byte]), field("Values", types.NewArray(copyType, 3)), field("Z", types.NewStruct(nil, nil)),
			}, nil)
			for _, typ := range []types.Type{native, copyType, empty, pointerOnly, types.NewArray(copyType, 3), container} {
				raw := prog.Type(typ, GoDecl)
				if size, alignment := uint64(sizes.Sizeof(typ)), uint64(sizes.Alignof(typ)); size != prog.SizeOf(raw) || alignment != prog.AlignOf(raw) {
					t.Fatalf("%s source = %d/%d; LLVM = %d/%d", typ, size, alignment, prog.SizeOf(raw), prog.AlignOf(raw))
				}
				if got := prog.abi.Size(raw.RawType()); uint64(got) != prog.SizeOf(raw) {
					t.Fatalf("%s ABI size = %d; LLVM = %d", typ, got, prog.SizeOf(raw))
				}
				if got := prog.abi.PtrBytes(raw.RawType()); uint64(got) > prog.SizeOf(raw) {
					t.Fatalf("%s pointer bytes = %d exceed LLVM storage %d", typ, got, prog.SizeOf(raw))
				}
			}
			if bytes := prog.abi.PtrBytes(pointerOnly); bytes != uintptr(prog.PointerSize()) {
				t.Fatalf("native pointer bytes = %d; pointer width = %d", bytes, prog.PointerSize())
			}
			for i, offset := range sizes.Offsetsof(fields) {
				if uint64(offset) != prog.OffsetOf(prog.Type(copyType, GoDecl), i) {
					t.Fatalf("field %d source offset = %d; LLVM = %d", i, offset, prog.OffsetOf(prog.Type(copyType, GoDecl), i))
				}
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

func TestNativeVectorPointerStorageInheritance(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "amd64"))
	_, vector := layoutTestVector()
	copyType := types.NewNamed(types.NewTypeName(0, types.NewPackage("p", "p"), "Copy", nil), vector.Underlying(), nil)
	prog.SetTypeCallConv("simd/archsimd.Float32x4", CDecl)
	prog.SetTypeDefinition(copyType.Obj(), vector)
	if prog.Type(vector, GoDecl).kind != vkSIMD || prog.Type(copyType, GoDecl).kind != vkStruct {
		t.Fatal("fixture no longer exercises different register representations")
	}
	for _, pair := range [][2]types.Type{{copyType, vector}, {vector, copyType}} {
		if err := prog.CheckNativeConversion(types.NewPointer(pair[0]), types.NewPointer(pair[1])); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativePointerViewsUseTargetCallbackABI(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			cdecl := newStdcallType(prog, "p", "Cdecl", NoArgsNoRet)
			prog.SetTypeCallConv("p.Cdecl", CDecl)
			stdcall := newStdcallType(prog, "p", "Stdcall", NoArgsNoRet)
			err := prog.CheckNativeConversion(types.NewPointer(cdecl), types.NewPointer(stdcall))
			if (err != nil) != (arch == "386") {
				t.Fatalf("callback pointer conversion error = %v", err)
			}
			if err := prog.CheckNativeConversion(cdecl, stdcall); (err != nil) != (arch == "386") {
				t.Fatalf("callback value conversion error = %v", err)
			}
		})
	}
}

func TestNativePointerTypedefKeepsPointeeStorage(t *testing.T) {
	for _, target := range []*Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"}} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			pkg := types.NewPackage("p", "p")
			pointer := types.NewPointer(types.Typ[types.Uintptr])
			native := types.NewNamed(types.NewTypeName(0, pkg, "NativePointer", nil), pointer, nil)
			copyType := types.NewNamed(types.NewTypeName(0, pkg, "Copy", nil), pointer, nil)
			prog.SetTypeCallConv("p.NativePointer", CDecl)
			prog.SetTypeDefinition(copyType.Obj(), native)
			if err := prog.CheckNativeConversion(copyType, native); err != nil {
				t.Fatal(err)
			}
			err := prog.CheckNativeConversion(native, pointer)
			if (err != nil) != prog.usesWideGoStorage() {
				t.Fatalf("native pointee view conversion error = %v", err)
			}
		})
	}
}

func TestNativeAnonymousLayoutDoesNotReuseGoStorage(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
	} {
		for _, nativeFirst := range []bool{false, true} {
			name := target.GOOS + "/" + target.GOARCH
			if nativeFirst {
				name += "/native-first"
			}
			t.Run(name, func(t *testing.T) {
				prog := NewProgram(target)
				defer prog.Dispose()
				sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
				pkg := types.NewPackage("p", "p")
				second := types.Type(types.Typ[types.Float64])
				if target.GOARCH == "wasm" {
					second = types.NewPointer(types.Typ[types.Byte])
				}
				fields := func(pos token.Pos) []*types.Var {
					return []*types.Var{types.NewField(pos, pkg, "B", types.Typ[types.Byte], false), types.NewField(pos+1, pkg, "V", second, false)}
				}
				goStruct := types.NewStruct(fields(1), nil)
				nativeFields := fields(3)
				nativeStruct := types.NewStruct(nativeFields, nil)
				alias := types.NewAlias(types.NewTypeName(0, pkg, "Native", nil), nativeStruct)
				if !types.Identical(goStruct, alias) {
					t.Fatal("fixture does not exercise identical gc types")
				}
				prog.SetTypeCallConv("p.Native", CDecl)
				prog.SetNativeStructFields([]token.Pos{3, 4}, true)
				var ordinary, native Type
				if nativeFirst {
					native, ordinary = prog.Type(alias, GoDecl), prog.Type(goStruct, GoDecl)
				} else {
					ordinary, native = prog.Type(goStruct, GoDecl), prog.Type(alias, GoDecl)
				}
				for _, typ := range []Type{ordinary, native} {
					if uint64(sizes.Sizeof(typ.RawType())) != prog.SizeOf(typ) {
						t.Fatalf("%s source size %d differs from storage %d", typ.RawType(), sizes.Sizeof(typ.RawType()), prog.SizeOf(typ))
					}
				}
				if prog.SizeOf(ordinary) == prog.SizeOf(native) || prog.OffsetOf(ordinary, 1) == prog.OffsetOf(native, 1) {
					t.Fatal("native layout reused the ordinary storage view")
				}
				if _, wrapped := prog.structLayout(native); wrapped {
					t.Fatal("native layout inherited ordinary Go field wrappers")
				}
				nativeName, _ := prog.abi.TypeName(alias)
				goName, _ := prog.abi.TypeName(goStruct)
				if nativeName == goName {
					t.Fatal("different layouts share one runtime type descriptor")
				}
			})
		}
	}
}
