package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestHostLayoutAddressableStorage(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", LLVMTarget: "wasm32-unknown-emscripten"},
	} {
		t.Run(target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", "wasm"))
			setTestRuntime(t, prog)
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "storage.go", `package storage
import "structs"
import "unsafe"
type Word struct { _ structs.HostLayout; Value uintptr; Tail uint32 }
type Pointer struct { _ structs.HostLayout; Value *byte; Tail uint32 }
type Unsafe struct { _ structs.HostLayout; Value unsafe.Pointer; Tail uint32 }
type Map struct { _ structs.HostLayout; Value map[int]int; Tail uint32 }
type Chan struct { _ structs.HostLayout; Value chan int; Tail uint32 }
type Words struct { _ structs.HostLayout; Value [2]uintptr; Tail uint32 }
type Pointers struct { _ structs.HostLayout; Value [2]*byte; Tail uint32 }
type Matrix struct { _ structs.HostLayout; Value [2][2]uintptr; Tail uint32 }
type Callback func()
type CallbackAlias = func()
type Native struct { _ structs.HostLayout; Value Callback; Tail uint32 }
type NativeAlias struct { _ structs.HostLayout; Value CallbackAlias; Tail uint32 }
type NativeArray struct { _ structs.HostLayout; Value [2]Callback; Tail uint32 }
type Legacy struct { Value Callback; Tail uint32 }
type Wire struct { _ structs.HostLayout; Addr uint32; Len uint32 }
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("storage.Callback", CDecl)
			prog.SetTypeCallConv("storage.CallbackAlias", CDecl)
			prog.SetTypeCallConv("storage.Legacy", CDecl)
			pkg, err := (&types.Config{Importer: importer.Default(), Sizes: sizes}).Check("storage", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name             string
				width, alignment int64
			}{
				{"Word", 8, 8}, {"Pointer", 8, 8}, {"Unsafe", 8, 8}, {"Map", 8, 8}, {"Chan", 8, 8},
				{"Words", 16, 8}, {"Pointers", 16, 8}, {"Matrix", 32, 8},
				{"Native", 4, 4}, {"NativeAlias", 4, 4}, {"NativeArray", 8, 4},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(tc.name).Type()
					checkHostLayout(t, prog, sizes, raw, tc.width+tc.alignment, tc.alignment, []int64{0, 0, tc.width})
					lowered := prog.Type(raw, GoDecl)
					field := prog.Field(lowered, 1)
					if got := prog.SizeOf(field); got != uint64(tc.width) {
						t.Fatalf("field storage size = %d, want %d", got, tc.width)
					}
					if prog.isNativeStorage(field) {
						t.Fatal("ordinary Go field acquired native storage")
					}
					address := prog.Pointer(field)
					if got := prog.SizeOf(prog.Elem(address)); got != uint64(tc.width) {
						t.Fatalf("escaped field storage size = %d, want %d", got, tc.width)
					}
					if array, ok := field.RawType().Underlying().(*types.Array); ok {
						if got := prog.SizeOf(prog.Index(address)); got != uint64(tc.width/array.Len()) {
							t.Fatalf("array element stride = %d, want %d", got, tc.width/array.Len())
						}
					}
				})
			}
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("Wire").Type(), 8, 4, []int64{0, 0, 4})
			// An explicit C boundary or legacy native struct still uses the
			// physical callback width, matching addressable Go callback
			// variables and arrays.
			callback := pkg.Scope().Lookup("Callback").Type()
			if got := prog.SizeOf(prog.Type(callback, CDecl)); got != 4 {
				t.Fatalf("C callback storage size = %d, want 4", got)
			}
			legacy := prog.Type(pkg.Scope().Lookup("Legacy").Type(), GoDecl)
			if prog.SizeOf(legacy) != 8 || prog.OffsetOf(legacy, 1) != 4 || prog.SizeOf(prog.Field(legacy, 0)) != 4 {
				t.Fatal("legacy native callback layout changed")
			}
		})
	}
}

func TestHostLayoutZeroSizedTail(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"}, {GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			markerPkg, err := importer.Default().Import("structs")
			if err != nil {
				t.Fatal(err)
			}
			marker := markerPkg.Scope().Lookup("HostLayout").Type()
			empty := types.NewStruct(nil, nil)
			field := func(name string, typ types.Type) *types.Var {
				return types.NewField(token.NoPos, nil, name, typ, false)
			}
			alignment := prog.hostBasicLayouts[types.Uint64].align
			for _, tail := range []struct {
				name string
				typ  types.Type
			}{
				{"Tail", empty}, {"_", empty}, {"Tail", types.NewArray(types.Typ[types.Uint64], 0)},
			} {
				raw := types.NewStruct([]*types.Var{field("_", marker), field("Value", types.Typ[types.Uint64]), field(tail.name, tail.typ)}, nil)
				size := align(9, alignment)
				checkHostLayout(t, prog, sizes, raw, size, alignment, []int64{0, 0, 8})
				checkHostLayout(t, prog, sizes, types.NewArray(raw, 2), 2*size, alignment, nil)
			}
			allEmpty := types.NewStruct([]*types.Var{field("_", marker), field("Tail", empty)}, nil)
			checkHostLayout(t, prog, sizes, allEmpty, 0, 1, []int64{0, 0})
			middle := types.NewStruct([]*types.Var{field("_", marker), field("Empty", empty), field("Value", types.Typ[types.Uint64])}, nil)
			checkHostLayout(t, prog, sizes, middle, 8, alignment, []int64{0, 0, 0})
		})
	}
}
