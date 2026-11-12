package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestWasmNativeSourceLayoutMatchesStorage(t *testing.T) {
	for _, tc := range []struct{ goos, profile string }{{"js", "j32"}, {"wasip1", "w32"}} {
		t.Run(tc.profile, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: tc.goos, GOARCH: "wasm", WasmProfile: tc.profile})
			defer prog.Dispose()
			setTestRuntime(t, prog)
			sizes := prog.TypeSizes(types.SizesFor("gc", "wasm"))
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "native.go", `package native
type Inner struct { P *byte; W uintptr }
type Record struct { P *byte; W uintptr; Fn func(); Map map[int]int; Chan chan int; Words [2]uintptr; Inner Inner }
type Wrapper struct { B byte; Record Record; Tail struct{} }
type Words [2]uintptr
type Pointers [2]*byte
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("native.Record", CDecl)
			prog.SetTypeCallConv("native.Words", CDecl)
			prog.SetTypeCallConv("native.Pointers", CDecl)
			for _, decl := range file.Decls {
				spec := decl.(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
				if spec.Name.Name == "Record" {
					var positions []token.Pos
					for _, f := range spec.Type.(*ast.StructType).Fields.List {
						positions = append(positions, f.Names[0].Pos())
					}
					prog.SetNativeStructFields(positions)
				}
			}
			pkg, err := (&types.Config{Sizes: sizes, Importer: importer.Default()}).Check("native", fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name    string
				size    int64
				offsets []int64
			}{
				{"Inner", 16, []int64{0, 8}},
				{"Record", 48, []int64{0, 4, 8, 12, 16, 20, 32}},
				{"Wrapper", 64, []int64{0, 8, 56}},
			} {
				raw := pkg.Scope().Lookup(tc.name).Type()
				checkHostLayout(t, prog, sizes, raw, tc.size, 8, tc.offsets)
				checkHostLayout(t, prog, sizes, types.NewArray(raw, 2), 2*tc.size, 8, nil)
			}
			for _, name := range []string{"Words", "Pointers"} {
				raw := pkg.Scope().Lookup(name).Type()
				checkHostLayout(t, prog, sizes, raw, 8, 4, nil)
				checkHostLayout(t, prog, sizes, types.NewArray(raw, 2), 16, 4, nil)
				if size := prog.SizeOf(prog.Type(raw, CDecl)); size != 8 {
					t.Fatalf("%s C storage size = %d, want 8", name, size)
				}
			}
		})
	}
}

func TestWasmNativeSourceLayoutDuringTypeCompletion(t *testing.T) {
	for _, profile := range []string{"j32", "w32"} {
		t.Run(profile, func(t *testing.T) {
			goos := "js"
			if profile == "w32" {
				goos = "wasip1"
			}
			prog := NewProgram(&Target{GOOS: goos, GOARCH: "wasm", WasmProfile: profile})
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", "wasm"))
			pkg := types.NewPackage("native", "native")
			record := types.NewNamed(types.NewTypeName(0, pkg, "Record", nil), types.NewStruct(nil, nil), nil)
			prog.SetTypeCallConv("native.Record", CDecl)
			fields := []*types.Var{
				types.NewField(0, pkg, "B", types.Typ[types.Byte], false),
				types.NewField(0, pkg, "Record", record, false),
				types.NewField(0, pkg, "Tail", types.Typ[types.Byte], false),
			}
			wrapper := types.NewStruct(fields, nil)
			if got := sizes.Sizeof(wrapper); got != 2 {
				t.Fatalf("incomplete wrapper size = %d, want 2", got)
			}
			record.SetUnderlying(types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "P", types.NewPointer(types.Typ[types.Byte]), false),
				types.NewField(0, pkg, "W", types.Typ[types.Uintptr], false),
			}, nil))
			if got := sizes.Sizeof(wrapper); got != 16 {
				t.Fatalf("completed wrapper size = %d, want 16", got)
			}
			if got := sizes.Alignof(wrapper); got != 4 {
				t.Fatalf("completed wrapper alignment = %d, want 4", got)
			}
			for i, got := range sizes.Offsetsof(fields) {
				if want := []int64{0, 4, 12}[i]; got != want {
					t.Fatalf("completed field %d offset = %d, want %d", i, got, want)
				}
			}
		})
	}
}
