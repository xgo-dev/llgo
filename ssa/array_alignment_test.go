//go:build !llgo

package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestGoArrayAccessAlignment(t *testing.T) {
	Initialize(InitAll)
	markerPkg, err := importer.Default().Import("structs")
	if err != nil {
		t.Fatal(err)
	}
	marker := markerPkg.Scope().Lookup("HostLayout").Type()
	for _, target := range []Target{
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5"}, {GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", LLVMTarget: "wasm32-unknown-emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+target.GOARM, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			pkg := prog.NewPackage("example.com/arrayalignment", "arrayalignment")
			rawPkg := types.NewPackage("example.com/arrayalignment", "arrayalignment")
			field := func(name string, typ types.Type) *types.Var {
				return types.NewField(token.NoPos, rawPkg, name, typ, false)
			}
			named := func(name string, typ types.Type) *types.Named {
				return types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, name, nil), typ, nil)
			}
			intArray := types.NewArray(types.Typ[types.Int64], 2)
			defined := named("Defined", intArray)
			nativeFields := []*types.Var{field("V", types.Typ[types.Int64])}
			native := named("Native", types.NewStruct(nativeFields, nil))
			prog.SetTypeCallConv(rawPkg.Path()+".Native", CDecl, nativeFields...)
			host := types.NewStruct([]*types.Var{field("_", marker), field("V", types.Typ[types.Int64])}, nil)
			_, vector := layoutTestVector()
			for _, tc := range []struct {
				name string
				raw  types.Type
				want int
			}{
				{"int64", intArray, int(sizes.Alignof(intArray))},
				{"float64", types.NewArray(types.Typ[types.Float64], 2), int(sizes.Alignof(types.Typ[types.Float64]))},
				{"complex128", types.NewArray(types.Typ[types.Complex128], 2), int(sizes.Alignof(types.Typ[types.Complex128]))},
				{"named", defined, int(sizes.Alignof(defined))},
				{"nested", types.NewArray(defined, 2), int(sizes.Alignof(defined))},
				{"named_nested", named("Matrix", types.NewArray(defined, 2)), int(sizes.Alignof(defined))},
				{"byte", types.NewArray(types.Typ[types.Byte], 3), 1},
				{"pointer", types.NewArray(types.NewPointer(types.Typ[types.Byte]), 2), prog.PointerSize()},
				{"native", types.NewArray(native, 2), int(prog.AlignOf(prog.Type(native, CDecl)))},
				{"host", types.NewArray(host, 2), int(sizes.Alignof(host))},
				{"vector", types.NewArray(vector, 2), int(prog.simdAlignments[0])},
			} {
				for _, viaField := range []bool{false, true} {
					name := tc.name + "_pointer"
					elem := tc.raw
					if viaField {
						name = tc.name + "_field"
						elem = types.NewStruct([]*types.Var{field("B", types.Typ[types.Byte]), field("V", tc.raw)}, nil)
					}
					t.Run(name, func(t *testing.T) {
						sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewPointer(elem))), types.NewTuple(types.NewVar(token.NoPos, nil, "", tc.raw)), false)
						fn := pkg.NewFunc(name, sig, CDecl)
						b := fn.MakeBody(1)
						defer b.Dispose()
						ptr := fn.Param(0)
						if viaField {
							ptr = b.FieldAddr(ptr, 1)
						}
						typ := prog.Elem(ptr.Type)
						stored := b.Store(ptr, prog.Zero(typ))
						loaded := b.Load(ptr)
						loaded.SetVolatile(true)
						b.Return(loaded)
						b.EndBuild()
						// Array loads may repack native elements into Go storage.
						// Inspect the memory instruction rather than the final value.
						load := loaded.memoryInstruction()
						if load.IsALoadInst().IsNil() || !load.IsVolatile() {
							t.Fatal("array memory load lost volatile access")
						}
						for name, inst := range map[string]llvm.Value{"load": load, "store": stored.impl} {
							if got := inst.Alignment(); got != tc.want {
								t.Errorf("%s alignment = %d, want %d", name, got, tc.want)
							}
						}
					})
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if target.GOARCH == "wasm" {
				// These standalone functions have not undergone Wasm section
				// finalization; validate their memory IR without emitting an object.
				return
			}
			obj, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			obj.Dispose()
		})
	}
}
