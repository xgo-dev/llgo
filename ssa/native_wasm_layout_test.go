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
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			prog.SetTypeCallConv("native.Record", CDecl)
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
		})
	}
}
