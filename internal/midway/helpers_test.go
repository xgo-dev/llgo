package midway

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

type testImporter map[string]*types.Package

func (i testImporter) Import(path string) (*types.Package, error) {
	if p := i[path]; p != nil {
		return p, nil
	}
	return importer.Default().Import(path)
}

func testInfo() *types.Info {
	return &types.Info{Scopes: map[ast.Node]*types.Scope{}, Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Implicits: map[ast.Node]types.Object{}, Instances: map[*ast.Ident]types.Instance{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
}

func testCheck(t *testing.T, imports testImporter, path, source string, rewrite bool) (*types.Package, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path+".go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := testInfo()
	cfg := types.Config{Importer: imports, Sizes: types.SizesFor("gc", "amd64")}
	pkg, err := cfg.Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	if rewrite {
		changed, _, err := Rewrite(fset, pkg, info, []*ast.File{file}, new(directive.Index))
		if err != nil {
			t.Fatal(err)
		}
		if !changed {
			t.Fatal("portable source was not rewritten")
		}
		info = testInfo()
		pkg, err = cfg.Check(path, fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
	}
	imports[path] = pkg
	return pkg, info
}

// Deliberately different layouts make accidental reuse of an original type
// visible to the second checker and layout-constant assertions.
func testSIMDImports(t *testing.T) testImporter {
	t.Helper()
	imports := testImporter{}
	testCheck(t, imports, simdPath, `package simd
 type Uint32s struct{}
 func (Uint32s) Len() int { return 4 }
 func BroadcastUint32s(uint32) Uint32s { return Uint32s{} }
 func Emulated() bool {return false}
 func VectorBitSize() int {return 128}
 `, false)
	testCheck(t, imports, bridgePath, `package bridge
 type Uint32s [8]uint32
 type Uint32x4 [4]uint32
 func (Uint32s) Len() int {return 8}
 func (Uint32x4) Len() int {return 4}
 func BroadcastUint32s(uint32) Uint32s {return Uint32s{}}
 func BroadcastUint32x4(uint32) Uint32x4 {return Uint32x4{}}
 func LLGoMidwayUnreachable(){panic("unspecialized")}
 type LLGoMidwayInt = int
 type LLGoMidwayBool = bool
 type LLGoMidwayString = string
 type LLGoMidwayError = error
 `, false)
	return imports
}
