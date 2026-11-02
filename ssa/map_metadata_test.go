package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func mapMetadataPointer(v llvm.Value) llvm.Value {
	// Hosted Memory32 pads Go pointers to a full Go word in type descriptors.
	if v.Type().TypeKind() == llvm.StructTypeKind {
		return v.Operand(0)
	}
	return v
}

func TestWasmMapMetadata(t *testing.T) {
	runtimePkg, err := importer.For("source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	field := func(name string, typ types.Type) *types.Var {
		return types.NewField(token.NoPos, nil, name, typ, false)
	}
	namedString := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("keys", "keys"), "Text", nil), types.Typ[types.String], nil)
	keys := []struct {
		typ  types.Type
		hash string
	}{
		{types.Typ[types.Bool], ".memhash8"},
		{types.Typ[types.Uint16], ".memhash16"},
		{types.Typ[types.Int32], ".memhash32"},
		{types.Typ[types.Uint64], ".memhash64"},
		{types.Typ[types.Uintptr], ".memhash64"},
		{types.Typ[types.UnsafePointer], ".memhash64"},
		{types.NewPointer(types.Typ[types.Byte]), ".memhash64"},
		{types.NewChan(types.SendRecv, types.Typ[types.Int]), ".memhash64"},
		{types.Typ[types.String], ".strhash"},
		{namedString, ".strhash"},
		{types.NewAlias(types.NewTypeName(token.NoPos, nil, "TextAlias", nil), namedString), ".strhash"},
		{types.Typ[types.Float32], ".f32hash"},
		{types.Typ[types.Float64], ".f64hash"},
		{types.Typ[types.Complex64], ".c64hash"},
		{types.Typ[types.Complex128], ".c128hash"},
		{types.NewArray(types.Typ[types.Byte], 16), ".memhash128"},
		{types.NewArray(types.Typ[types.Byte], 3), "_llgo_memhash.3"},
		{types.NewArray(types.Typ[types.Byte], 0), "_llgo_memhash.0"},
		{types.NewStruct(nil, nil), "_llgo_memhash.0"},
		{types.NewStruct([]*types.Var{field("A", types.Typ[types.Int64]), field("B", types.Typ[types.Uint64])}, nil), ".memhash128"},
		{types.NewStruct([]*types.Var{field("A", types.Typ[types.Byte]), field("B", types.Typ[types.Int64])}, nil), ".typehash"},
		{types.NewStruct([]*types.Var{field("_", types.Typ[types.Int64])}, nil), ".typehash"},
		{types.NewArray(types.Typ[types.String], 2), ".typehash"},
		{types.NewInterfaceType(nil, nil).Complete(), ".typehash"},
	}
	for _, tc := range []struct {
		name, goos, arch, triple, profile string
		dce                               bool
	}{
		{"w32", "wasip1", "wasm", "wasm32-unknown-wasip1", "w32", true},
		{"j32", "js", "wasm", "wasm32-unknown-emscripten", "j32", true},
		{"j64", "js", "wasm", "wasm64-unknown-emscripten", "j64", true},
		{"legacy32", "wasip1", "wasm", "wasm32-unknown-wasip1", "", true},
		{"no-dce", "wasip1", "wasm", "wasm32-unknown-wasip1", "w32", false},
		{"native", "linux", "amd64", "x86_64-unknown-linux", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: tc.goos, GOARCH: tc.arch, LLVMTarget: tc.triple, WasmProfile: tc.profile})
			defer prog.Dispose()
			prog.TypeSizes(&types.StdSizes{WordSize: 8, MaxAlign: 8})
			prog.SetRuntime(func() *types.Package { return runtimePkg })
			prog.EnableDeadcodeDrop(tc.dce)
			prog.EnableGoGlobalDCE(true)
			pkg := prog.NewPackage("main", "main")
			fn := pkg.NewFunc("use", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			specialized := tc.dce && tc.arch == "wasm"
			for _, key := range keys {
				mt := types.NewMap(key.typ, types.Typ[types.Int])
				b.abiType(mt)
				name, _ := prog.abi.TypeName(mt)
				global := pkg.VarOf(name).impl
				hasher := global.Initializer().Operand(4)
				want := key.hash
				if tc.name == "legacy32" {
					switch key.typ.(type) {
					case *types.Pointer, *types.Chan:
						want = ".memhash32"
					}
					if key.typ == types.Typ[types.Uintptr] || key.typ == types.Typ[types.UnsafePointer] {
						want = ".memhash32"
					}
				}
				if !specialized {
					want = ".typehash"
				}
				if !strings.HasSuffix(mapMetadataPointer(hasher.Operand(0)).Name(), want) {
					t.Fatalf("%s hasher = %s, want %s", mt, hasher, want)
				}
				if hasher.Operand(1).IsNull() != (want != ".typehash") {
					t.Fatalf("%s has incorrect hasher environment: %s", mt, hasher)
				}
				if !containsLLVMValueNameSuffix(pkg.abiTypeFakeUseCache[global], want) {
					t.Fatalf("%s lost its hasher reachability edge", mt)
				}
				if specialized {
					bucket := prog.abi.MapBucket(mt)
					private := mapMetadataPointer(global.Initializer().Operand(3))
					init := private.Initializer()
					if init.Operand(0).ZExtValue() != uint64(prog.abi.Size(bucket)) ||
						init.Operand(1).ZExtValue() != uint64(prog.abi.PtrBytes(bucket)) ||
						init.Operand(4).ZExtValue() != uint64(prog.abi.Align(bucket)) {
						t.Fatalf("%s bucket lost its allocation layout: %s", mt, init)
					}
					if !init.Operand(7).IsNull() || !init.Operand(10).IsNull() {
						t.Fatalf("private bucket retained equality or pointer metadata: %s", init)
					}
					if again := b.abiMapBucket(mt, bucket); again.impl != private {
						t.Fatal("private bucket descriptor was not reused")
					}
					for _, info := range prog.AbiTypes() {
						if info.Name == private.Name() {
							t.Fatal("private bucket leaked into the reflection type list")
						}
					}
				}
			}
			if specialized {
				for name, size := range map[string]uint64{"_llgo_memhash.0": 0, "_llgo_memhash.3": 3} {
					wrapper := pkg.FuncOf(name)
					if wrapper == nil {
						t.Fatal("missing fixed-size hash wrapper", name)
					}
					found := false
					for _, block := range wrapper.impl.BasicBlocks() {
						for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
							if call := inst.IsACallInst(); !call.IsNil() && strings.HasSuffix(call.CalledValue().Name(), ".memhash") {
								found = call.Operand(2).ZExtValue() == size
							}
						}
					}
					if !found {
						t.Fatalf("%s must call generic memhash with size %d", name, size)
					}
				}
				// A real struct with a bucket's layout still needs comparison and
				// reflection metadata; creating it must not reuse the private type.
				mt := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
				bucket := prog.abi.MapBucket(mt)
				b.abiType(bucket)
				name, _ := prog.abi.TypeName(bucket)
				ordinary := pkg.VarOf(name).impl.Initializer()
				if ordinary.Operand(0).Operand(7).IsNull() {
					t.Fatal("ordinary struct equality was stripped with private bucket metadata")
				}
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
