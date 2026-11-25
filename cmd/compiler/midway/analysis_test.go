package midway

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func dependencyChain(tb testing.TB, depth int) (*types.Package, *types.Info, []*ast.File) {
	tb.Helper()
	simd := types.NewPackage(simdPath, "simd")
	obj := types.NewTypeName(token.NoPos, simd, "Uint32s", nil)
	types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	simd.Scope().Insert(obj)
	simd.MarkComplete()
	var source strings.Builder
	source.WriteString("package chain\nimport (\"simd\"; \"unsafe\")\n")
	// Reverse source order forces the old full-AST fixed-point scan to make
	// one pass per link. Every constant still needs its own specialized value.
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&source, "const C%d = C%d\n", i, i+1)
	}
	fmt.Fprintf(&source, "const C%d = unsafe.Sizeof(simd.Uint32s{})\n", depth)
	source.WriteString(`type Bytes [C0]byte
func Size() int { return int(C0) }
func Caller() int { return Size() }
const Plain = 7
type Scalar int
`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "chain.go", source.String(), 0)
	if err != nil {
		tb.Fatal(err)
	}
	info := testInfo()
	cfg := types.Config{Importer: testImporter{simdPath: simd}, Sizes: types.SizesFor("gc", "amd64")}
	pkg, err := cfg.Check("chain", fset, []*ast.File{file}, info)
	if err != nil {
		tb.Fatal(err)
	}
	return pkg, info, []*ast.File{file}
}

func TestAnalyzeDependencyChain(t *testing.T) {
	const depth = 256
	pkg, info, files := dependencyChain(t, depth)
	a := analyze(pkg, info, files)
	for i := 0; i <= depth; i++ {
		name := fmt.Sprintf("C%d", i)
		if !a.dependent(pkg.Scope().Lookup(name)) {
			t.Errorf("lost layout dependency for %s", name)
		}
	}
	for _, name := range []string{"Bytes", "Size"} {
		if !a.dependent(pkg.Scope().Lookup(name)) {
			t.Errorf("lost dependency for %s", name)
		}
	}
	for _, name := range []string{"Plain", "Scalar", "Caller"} {
		if a.dependent(pkg.Scope().Lookup(name)) {
			t.Errorf("unrelated scalar declaration %s became dependent", name)
		}
	}
}

func BenchmarkAnalyzeDependencyChain(b *testing.B) {
	for _, depth := range []int{128, 512, 2048} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			pkg, info, files := dependencyChain(b, depth)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				analyze(pkg, info, files)
			}
		})
	}
}
