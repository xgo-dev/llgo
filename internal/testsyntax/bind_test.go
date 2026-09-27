package testsyntax

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/gcexportdata"
)

func parseSource(t *testing.T, source string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return fset, file
}

func checkRecordSource(t *testing.T, fset *token.FileSet, file *ast.File) (*types.Package, *types.Info) {
	t.Helper()
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, info
}

func TestBindScopeReceiverAliasesAndGenerics(t *testing.T) {
	fset, file := parseSource(t, `package p
type T struct{}
type Alias = T
type Ptr = *Alias
func (T) A() {}
//go:nointerface
func (*Alias) M() {}
func (Ptr) P() {}
type Generic[X any] struct{}
//go:noinline
func (Generic[X]) Value() {}
type Pair[X, Y any] struct{}
//llgo:env
func (*Pair[X, Y]) Pointer() {}
`)
	pkg, info := checkRecordSource(t, fset, file)
	p, err := directive.Collect(fset, new(directive.Index).Files([]*ast.File{file}), false, false)
	if err != nil {
		t.Fatal(err)
	}
	BindScope(p, fset, pkg)
	for decl, r := range p.Functions {
		if got := p.Objects[info.Defs[decl.Name]]; got != r {
			t.Errorf("method %s not bound: %v", r.Name, got)
		}
	}
	if !p.Names["(*T).M"].(*directive.FunctionDecl).NoInterface || !p.Names["Generic.Value"].(*directive.FunctionDecl).NoInline || !p.Names["(*Pair).Pointer"].(*directive.FunctionDecl).ClosureEnv {
		t.Fatal("receiver source properties lost")
	}
	// A different checked package at different source positions must not bind.
	other, err := parser.ParseFile(fset, "other.go", "package p\n\ntype T struct{}\nfunc (*T) M() {}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	otherPkg, _ := checkRecordSource(t, fset, other)
	unmatched, err := directive.Collect(fset, new(directive.Index).Files([]*ast.File{file}), false, false)
	if err != nil {
		t.Fatal(err)
	}
	BindScope(unmatched, fset, otherPkg)
	if len(unmatched.Objects) != 0 {
		t.Fatalf("unrelated declarations bound: %v", unmatched.Objects)
	}
}

func TestBindExportDataToReparsedSyntax(t *testing.T) {
	const source = `package p
//go:noinline
func F() {}
var V int
//llgo:type C
type T int
type hiddenAlias = *T
//go:nointerface
func (hiddenAlias) M() {}
`
	fset, file := parseSource(t, source)
	original, _ := checkRecordSource(t, fset, file)
	var data bytes.Buffer
	if err := gcexportdata.Write(&data, fset, original); err != nil {
		t.Fatal(err)
	}
	imported, err := gcexportdata.Read(&data, fset, make(map[string]*types.Package), original.Path())
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := parser.ParseFile(fset, "p.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	records, err := directive.Collect(fset, new(directive.Index).Files([]*ast.File{reparsed}), false, false)
	if err != nil {
		t.Fatal(err)
	}
	BindScope(records, fset, imported)
	method := imported.Scope().Lookup("T").Type().(*types.Named).Method(0)
	for name, obj := range map[string]types.Object{
		"F":      imported.Scope().Lookup("F"),
		"V":      imported.Scope().Lookup("V"),
		"T":      imported.Scope().Lookup("T"),
		"(*T).M": method,
	} {
		if got := records.Objects[obj]; got != records.Names[name] {
			t.Errorf("%s export object not bound: %v", name, got)
		}
	}
	if !records.Objects[imported.Scope().Lookup("F")].(*directive.FunctionDecl).NoInline || !records.Objects[method].(*directive.FunctionDecl).NoInterface {
		t.Fatal("imported function properties lost")
	}
}
