package funcattrs

import (
	"github.com/xgo-dev/llgo/internal/directive"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func parseTest(t *testing.T, source string) ([]Attribute, *types.Signature, error) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "attributes.go", "package p\nimport \"unsafe\"\nvar _ unsafe.Pointer\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	_, err = (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	decl := f.Decls[len(f.Decls)-1].(*ast.FuncDecl)
	record := new(directive.Index).Function(decl).WithPositions(fset)
	attrs, err := record.Values, record.ContractError
	return attrs, info.Defs[decl.Name].Type().(*types.Signature), err
}
