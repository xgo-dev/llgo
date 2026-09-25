package directive

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"
)

func TestCollectDeclarationContracts(t *testing.T) {
	_, file := parseSource(t, `package C
//llgo:skip imported
import _ "unsafe"
//export public
func public() {}
func Xautomatic() {}
func Automatic() {}
func private() {}
//llgo:link explicit C.custom
func explicit() {}
//export wrong
func invalid() {}
//llgo:link single attached
var single int
//llgo:link many ignored
var many, other int
//llgo:type C
type Native func()
//llgo:type stdcall
type (Grouped func(); Another func())
//llgo:skip removed
const keep = 1
//llgo:skipall
type Last int
//go:linkname single final.symbol
//go:linkname absent ignored
`)
	p := Collect(new(Index).Files([]*ast.File{file}), true, false)
	for name, target := range map[string]string{"public": "public", "Xautomatic": "automatic", "Automatic": "Automatic", "explicit": "C.custom"} {
		d := p.Names[name].(*FunctionDecl)
		if !d.HasLinkname || d.Linkname != target {
			t.Errorf("%s link = %+v", name, d)
		}
	}
	for _, name := range []string{"public", "Xautomatic", "Automatic"} {
		d := p.Names[name].(*FunctionDecl)
		if d.ExportName != d.Linkname {
			t.Errorf("%s export = %q", name, d.ExportName)
		}
	}
	if d := p.Names["private"].(*FunctionDecl); d.HasLinkname {
		t.Fatal("private function was auto-exported")
	}
	if d := p.Names["invalid"].(*FunctionDecl); d.Err == nil || !strings.Contains(d.Err.Error(), `wrong name "wrong"`) {
		t.Fatalf("deferred export error = %v", d.Err)
	}
	if d := p.Names["single"].(*VariableDecl); d.Linkname != "final.symbol" {
		t.Fatalf("file link did not override attached link: %+v", d)
	}
	if p.Names["many"].(*VariableDecl).HasLinkname || p.Names["other"].(*VariableDecl).HasLinkname {
		t.Fatal("multi-variable declaration consumed an attached link")
	}
	if p.Names["Native"].(*TypeDecl).Background != "C" || p.Names["Grouped"].(*TypeDecl).Background != "" {
		t.Fatal("type background attachment changed")
	}
	if !p.Skip.All || !reflect.DeepEqual(p.Skip.Names, []string{"imported", "removed"}) {
		t.Fatalf("skip = %+v", p.Skip)
	}
	renamed := Collect(new(Index).Files([]*ast.File{file}), false, true).Names["invalid"].(*FunctionDecl)
	if renamed.Err != nil || renamed.ExportName != "wrong" {
		t.Fatalf("renamed export = %+v", renamed)
	}
}

func TestPackageLinksRequireUnsafe(t *testing.T) {
	_, file := parseSource(t, `package p
//llgo:link F first
//llgo:link other unrelated
//llgo:link F last
func F() {}

//go:linkname F file.symbol
//go:linkname malformed
`)
	d := Collect(new(Index).Files([]*ast.File{file}), false, false).Names["F"].(*FunctionDecl)
	if d.Linkname != "last" {
		t.Fatalf("link without unsafe = %q", d.Linkname)
	}
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

func TestBindingUsesEffectiveDeclarationsAndPositions(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(name, source string) *ast.File {
		f, err := parser.ParseFile(fset, name, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	original := parse("original.go", "package p\nfunc F() {}\nvar V int\ntype T int\n")
	replacement := parse("replacement.go", "package p\n//go:noinline\nfunc F() {}\nvar V string\n//llgo:type C\ntype T string\n")
	oldPkg, oldInfo := checkRecordSource(t, fset, original)
	newPkg, newInfo := checkRecordSource(t, fset, replacement)
	files := new(Index).Files([]*ast.File{original, replacement})
	for _, scoped := range []bool{false, true} {
		t.Run(map[bool]string{false: "checker objects", true: "standalone scope"}[scoped], func(t *testing.T) {
			p := Collect(files, false, false)
			p.Bind(nil)
			p.BindScope(nil)
			if scoped {
				p.BindScope(oldPkg)
			} else {
				p.Bind(oldInfo)
			}
			if len(p.Objects) != 0 {
				t.Fatalf("inactive originals bound: %v", p.Objects)
			}
			if scoped {
				p.BindScope(newPkg)
			} else {
				p.Bind(newInfo)
			}
			for _, name := range []string{"F", "V", "T"} {
				if got := p.Objects[newPkg.Scope().Lookup(name)]; got != p.Names[name] {
					t.Errorf("%s binding = %v", name, got)
				}
			}
			if !p.Names["F"].(*FunctionDecl).NoInline || p.Names["T"].(*TypeDecl).Background != "C" {
				t.Fatal("replacement properties lost")
			}
		})
	}
}

func TestBindScopeReceiverAliasesAndGenerics(t *testing.T) {
	fset, file := parseSource(t, `package p
type T struct{}
type Alias = T
func (T) A() {}
//go:nointerface
func (*Alias) M() {}
type Generic[X any] struct{}
//go:noinline
func (Generic[X]) Value() {}
type Pair[X, Y any] struct{}
//llgo:env
func (*Pair[X, Y]) Pointer() {}
`)
	pkg, info := checkRecordSource(t, fset, file)
	p := Collect(new(Index).Files([]*ast.File{file}), false, false)
	p.BindScope(pkg)
	for decl, r := range p.Functions {
		if got := p.Objects[info.Defs[decl.Name]]; got != r {
			t.Errorf("method %s not bound: %v", r.Name, got)
		}
	}
	if !p.Names["(*Alias).M"].(*FunctionDecl).NoInterface || !p.Names["Generic.Value"].(*FunctionDecl).NoInline || !p.Names["(*Pair).Pointer"].(*FunctionDecl).ClosureEnv {
		t.Fatal("receiver source properties lost")
	}
	// A different checked package at different source positions must not bind.
	otherSet, other := parseSource(t, "package p\n\ntype T struct{}\nfunc (*T) M() {}\n")
	otherPkg, _ := checkRecordSource(t, otherSet, other)
	unmatched := Collect(new(Index).Files([]*ast.File{file}), false, false)
	unmatched.BindScope(otherPkg)
	if len(unmatched.Objects) != 0 {
		t.Fatalf("unrelated declarations bound: %v", unmatched.Objects)
	}
}

func TestParenthesizedGenericReceiverSelector(t *testing.T) {
	expr, err := parser.ParseExpr("(Pair[A, B])")
	if err != nil {
		t.Fatal(err)
	}
	fn := &ast.FuncDecl{Name: ast.NewIdent("M"), Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: expr}}}}}
	if got := FuncName(fn); got != "(*Pair).M" {
		t.Fatalf("selector = %q", got)
	}
}
