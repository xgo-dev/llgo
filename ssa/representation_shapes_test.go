package ssa

import (
	"go/types"
	"testing"
)

// Check source sizes against storage rather than trusting a second source-size
// query. Arrays and pointer-bearing Go headers exercise different native views.
func TestNativeRepresentationStorageShapes(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.SetRuntime(newDebugRuntimePackage())
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			pkg := types.NewPackage("shapes", "shapes")
			native := func(name string, underlying types.Type) *types.Named {
				n := types.NewNamed(types.NewTypeName(0, pkg, name, nil), underlying, nil)
				prog.SetTypeCallConv(pkg.Path()+"."+name, CDecl)
				return n
			}
			for _, tc := range []struct {
				name string
				typ  types.Type
			}{
				{"Int", types.Typ[types.Int]},
				{"Uint", types.Typ[types.Uint]},
				{"Uintptr", types.Typ[types.Uintptr]},
				{"UnsafePointer", types.Typ[types.UnsafePointer]},
				{"String", types.Typ[types.String]},
				{"Slice", types.NewSlice(types.Typ[types.Byte])},
				{"Interface", types.NewInterfaceType(nil, nil).Complete()},
				{"Map", types.NewMap(types.Typ[types.Int], types.Typ[types.Byte])},
				{"Chan", types.NewChan(types.SendRecv, types.Typ[types.Int])},
				{"PointerArray", types.NewArray(types.Typ[types.Uintptr], 3)},
				{"CallbackArray", types.NewArray(NoArgsNoRet, 2)},
				{"EmptyCallbackArray", types.NewArray(NoArgsNoRet, 0)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					fields := []*types.Var{types.NewField(0, pkg, "Prefix", types.Typ[types.Byte], false), types.NewField(0, pkg, "Value", tc.typ, false)}
					n := native(tc.name, types.NewStruct(fields, nil))
					prog.SetTypeCallConv(pkg.Path()+"."+tc.name, CDecl, fields...)
					raw := prog.Type(n, GoDecl)
					if got := sizes.Sizeof(n); uint64(got) != prog.SizeOf(raw) {
						t.Fatalf("source/LLVM size = %d/%d", got, prog.SizeOf(raw))
					}
					if got := sizes.Alignof(n); uint64(got) != prog.AlignOf(raw) {
						t.Fatalf("source/LLVM alignment = %d/%d", got, prog.AlignOf(raw))
					}
					if got := sizes.Offsetsof(fields)[1]; uint64(got) != prog.OffsetOf(raw, 1) {
						t.Fatalf("source/LLVM offset = %d/%d", got, prog.OffsetOf(raw, 1))
					}
					if got := prog.abi.PtrBytes(n); uint64(got) > prog.SizeOf(raw) {
						t.Fatalf("pointer prefix %d exceeds storage %d", got, prog.SizeOf(raw))
					}
				})
			}
			for _, tc := range []struct {
				name string
				elem types.Type
			}{
				{"Callbacks", NoArgsNoRet},
				{"NestedCallbacks", types.NewArray(NoArgsNoRet, 2)},
				{"Integers", types.Typ[types.Uint64]},
				{"Uintptrs", types.Typ[types.Uintptr]},
			} {
				n := native(tc.name, types.NewArray(tc.elem, 3))
				raw := prog.Type(n, GoDecl)
				if got := sizes.Sizeof(n); uint64(got) != prog.SizeOf(raw) {
					t.Fatalf("%s source/LLVM size = %d/%d", tc.name, got, prog.SizeOf(raw))
				}
				if got := sizes.Alignof(n); uint64(got) != prog.AlignOf(raw) {
					t.Fatalf("%s source/LLVM alignment = %d/%d", tc.name, got, prog.AlignOf(raw))
				}
				for _, view := range []Type{raw, prog.Type(types.NewPointer(n), GoDecl)} {
					if stride := prog.SizeOf(prog.Index(view)); 3*stride != prog.SizeOf(raw) {
						t.Fatalf("%s array element stride/storage = %d/%d", tc.name, stride, prog.SizeOf(raw))
					}
				}
			}
		})
	}
}

func TestNativeConversionStorageGraphs(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", "amd64"))
	if err := prog.CheckNativeConversion(types.Typ[types.Int], types.Typ[types.Int]); err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("graphs", "graphs")
	native := func(name string, underlying types.Type) *types.Named {
		n := types.NewNamed(types.NewTypeName(0, pkg, name, nil), underlying, nil)
		prog.SetTypeCallConv(pkg.Path()+"."+name, CDecl)
		return n
	}
	field := func(name string, typ types.Type) *types.Var { return types.NewField(0, pkg, name, typ, false) }
	a, b := native("A", nil), native("B", nil)
	a.SetUnderlying(types.NewStruct([]*types.Var{field("Next", types.NewPointer(a)), field("Fn", NoArgsNoRet)}, nil))
	b.SetUnderlying(types.NewStruct([]*types.Var{field("Next", types.NewPointer(b)), field("Fn", NoArgsNoRet)}, nil))
	for _, pair := range [][2]types.Type{
		{a, b},
		{native("ArrayA", types.NewArray(a, 2)), native("ArrayB", types.NewArray(b, 2))},
	} {
		if err := prog.CheckNativeConversion(types.NewPointer(pair[0]), types.NewPointer(pair[1])); err != nil {
			t.Fatalf("equivalent recursive storage rejected: %v", err)
		}
	}
	for _, pair := range [][2]types.Type{
		{native("EmptyTwo", types.NewArray(types.NewStruct(nil, nil), 2)), native("EmptyThree", types.NewArray(types.NewStruct(nil, nil), 3))},
		{
			native("OneField", types.NewStruct([]*types.Var{field("Words", types.NewArray(types.Typ[types.Uint32], 2))}, nil)),
			native("TwoFields", types.NewStruct([]*types.Var{field("Lo", types.Typ[types.Uint32]), field("Hi", types.Typ[types.Uint32])}, nil)),
		},
	} {
		if err := prog.CheckNativeConversion(types.NewPointer(pair[0]), types.NewPointer(pair[1])); err == nil {
			t.Fatal("equal byte sizes hid incompatible array/field shapes")
		}
	}
	if err := prog.CheckNativeConversion(types.Typ[types.Int], types.Typ[types.Int]); err != nil {
		t.Fatal(err)
	}
}
