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
type Callback func(int32) int32
type CallbackAlias = func(int32) int32
type Native struct { _ structs.HostLayout; Value Callback; Tail uint32 }
type NativeAlias struct { _ structs.HostLayout; Value CallbackAlias; Tail uint32 }
type NativeArray struct { _ structs.HostLayout; Value [2]Callback; Tail uint32 }
type Wire struct { _ structs.HostLayout; Addr uint32; Len uint32 }
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("storage.Callback", CDecl)
			prog.SetTypeCallConv("storage.CallbackAlias", CDecl)
			pkg, err := (&types.Config{Importer: importer.Default(), Sizes: sizes}).Check("storage", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name  string
				width int64
			}{
				{"Word", 8}, {"Pointer", 8}, {"Unsafe", 8}, {"Map", 8}, {"Chan", 8},
				{"Words", 16}, {"Pointers", 16}, {"Matrix", 32},
				{"Native", 8}, {"NativeAlias", 8}, {"NativeArray", 16},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(tc.name).Type()
					checkHostLayout(t, prog, sizes, raw, tc.width+8, 8, []int64{0, 0, tc.width})
					lowered := prog.Type(raw, GoDecl)
					field := prog.Field(lowered, 1)
					if got := prog.SizeOf(field); got != uint64(tc.width) {
						t.Fatalf("field storage size = %d, want %d", got, tc.width)
					}
					if prog.isNativeStorage(field) {
						t.Fatal("ordinary Go field acquired native storage")
					}
				})
			}
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup("Wire").Type(), 8, 4, []int64{0, 0, 4})
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
