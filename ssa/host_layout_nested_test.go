package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestHostLayoutNestedNativeStorage(t *testing.T) {
	Initialize(InitAll)
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "arm"})
	defer prog.Dispose()
	sizes := prog.TypeSizes(types.SizesFor("gc", "arm"))
	setTestRuntime(t, prog)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "probe.go", `package probe
import "structs"
type Inner struct { F float64; B byte }
type FuncFirst struct { F func(); D float64 }
type HostOuter struct { B byte; H HostNested; F float64 }
type HostTail struct { _ structs.HostLayout; I struct { F float64; T struct{} }; B byte }
type HostNested struct { _ structs.HostLayout; Inner Inner; B byte }
type HostFuncFirst struct { _ structs.HostLayout; X FuncFirst; B byte }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default(), Sizes: sizes}).Check("probe", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		size    int64
		offsets []int64
	}{
		{"HostNested", 24, []int64{0, 0, 16}},
		{"HostOuter", 40, []int64{0, 8, 32}},
		{"HostTail", 16, []int64{0, 0, 8}},
		{"HostFuncFirst", 24, []int64{0, 0, 16}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkHostLayout(t, prog, sizes, pkg.Scope().Lookup(tc.name).Type(), tc.size, 8, tc.offsets)
		})
	}
}
