package types_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// Type-checking these API fixtures does not require a host Go compiler or its
// export cache. Use a real source-checked fixture package for imported objects;
// go/importer has its own tests for the standard-library import mechanism.
func testImporter(t testing.TB) types.ImporterFrom {
	t.Helper()
	const source = `package fmt
func Sprint(args ...any) string
func Sprintf(format string, args ...any) string
func Println(args ...any) (int, error)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var conf types.Config
	pkg, err := conf.Check("example.org/fixture/fmt", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fixtureImporter{pkg}
}

type fixtureImporter struct{ pkg *types.Package }

func (imp fixtureImporter) Import(path string) (*types.Package, error) {
	return imp.ImportFrom(path, "", 0)
}

func (imp fixtureImporter) ImportFrom(path, _ string, _ types.ImportMode) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if path != imp.pkg.Path() {
		return nil, fmt.Errorf("unexpected fixture import %q", path)
	}
	return imp.pkg, nil
}
