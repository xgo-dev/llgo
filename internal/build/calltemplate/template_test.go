package calltemplate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
)

func checkedSource(t *testing.T, text string) *packages.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", text, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return &packages.Package{PkgPath: "p", Name: pkg.Name(), Fset: fset, Types: pkg,
		TypesInfo: info, Syntax: []*ast.File{file}}
}

func TestTemplateSelectsSpecificCandidate(t *testing.T) {
	pkg := checkedSource(t, `package p
//llgo:template first both
func F(a, b any) {}
func first(a int, b any) {}
func both(a, b int) { F(a, b) }

//llgo:template same
func G(a, b any) {}
func same[T comparable](a, b []T) {}

//llgo:template lhs rhs
func H(a, b any) {}
func lhs(a int, b any) {}
func rhs(a any, b int) {}

type Named int
func use(v any) {
	F(1,2)
	F(1,v)
	F(Named(1),2)
	G([]int{1}, []int{2})
	G([]int{1}, []string{"mismatch"})
	G([][]int{{1}}, [][]int{{2}})
	H(1,2)
}`)
	got, err := Rewrite([]*packages.Package{pkg}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	selected := got[pkg.Types]
	if len(selected) != 3 {
		t.Fatalf("selected %v, want three calls", selected)
	}
	for _, want := range []string{"p.first", "p.both", "p.same"} {
		seen := false
		for _, item := range selected {
			seen = seen || strings.HasSuffix(item, ":"+want)
		}
		if !seen {
			t.Errorf("selected %v, missing %s", selected, want)
		}
	}
	for _, decl := range pkg.Syntax[0].Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "both" {
			call := fn.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
			if calledFunc(pkg.TypesInfo, call.Fun).Name() != "F" {
				t.Fatal("a candidate's fallback became recursive")
			}
		}
	}
}

func TestTemplateRejectsInvalidDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name, src, diagnostic string
	}{
		{"missing", "//llgo:template missing\nfunc F(a any) {}", "invalid llgo:template candidate"},
		{"result", "//llgo:template G\nfunc F(a any) {}\nfunc G(a int) int{return a}", "incompatible signature"},
		{"fixed", "//llgo:template G\nfunc F(a bool) {}\nfunc G(a int) {}", "changes non-interface parameter"},
		{"condition", "//llgo:template G if unknown == 1\nfunc F(a any) {}\nfunc G(a int) {}", "compare a parameter"},
		{"multiple conditional", "//llgo:template G G if a == 1\nfunc F(a any) {}\nfunc G(a int) {}", "requires one candidate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := checkedSource(t, "package p\n"+tc.src)
			_, err := Rewrite([]*packages.Package{p}, nil, false)
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("error = %v, want %q", err, tc.diagnostic)
			}
		})
	}
}

func TestCFmtNamedFormatSlot(t *testing.T) {
	p := checkedSource(t, `package main
//go:linkname str llgo.cstr
func str(s string) *int8

//llgointernal:template:cfmt format=pattern args=__llgo_va_list
//go:linkname F C.snprintf
func F(dst *int8, n uintptr, pattern *int8, __llgo_va_list ...any) int32

func main() {
	F(nil, 0, str("%s:%d %% %d"), str("x%"), int32(5), dynamic)
	F(nil, 0, str("%m %d"), int32(5))
	F(nil, 0, str("%d"), 1)
	F(nil, 0, str("%d:"), 1, 2)
	F(nil, 0, str("left %s %04d"), str(shared), dynamic)
	F(nil, 0, str("right %s %04d"), str(shared), dynamic)
}
var dynamic int32
const shared = "one literal to keep shared"`)
	got, err := Rewrite([]*packages.Package{p}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[p.Types]) != 1 {
		t.Fatalf("selected %v, want one call", got[p.Types])
	}
	main := p.Syntax[0].Decls[2].(*ast.FuncDecl)
	call := main.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
	if len(call.Args) != 4 {
		t.Fatalf("specialized call has %d args, want 4", len(call.Args))
	}
	text, ok := constantString(p.TypesInfo, call.Args[2].(*ast.CallExpr).Args[0])
	if !ok || text != "x%%:5 %% %d" {
		t.Fatalf("specialized format = %q", text)
	}
}

func TestGoFmtNamedFormatSlot(t *testing.T) {
	p := checkedSource(t, `package fmt
func Appendf(dst []byte, pattern string, values ...any) []byte
func Fprintf(dst any, pattern int, values ...any) (int, error)
func Printf(pattern string, values ...string) (int, error)
func Unknown(pattern string, values ...any) string`)
	for _, tc := range []struct {
		name, roles, diagnostic string
		trusted                 bool
	}{
		{"Appendf", "format=pattern args=values", "", true},
		{"Appendf", "format=pattern args=values bogus=dst", "invalid format template binding", true},
		{"Appendf", "format=pattern", "variadic argument parameter", true},
		{"Appendf", "format=pattern args=values", "standard fmt entry", false},
		{"Fprintf", "format=pattern args=values", "string format parameter", true},
		{"Printf", "format=pattern args=values", "requires ...any", true},
		{"Unknown", "format=pattern args=values", "unsupported gofmt output", true},
	} {
		t.Run(tc.name+"/"+tc.diagnostic, func(t *testing.T) {
			fn := p.Types.Scope().Lookup(tc.name).(*types.Func)
			rule, err := bindFormat(fn, directive.Directive{Name: "llgointernal:template:gofmt", Args: tc.roles}, nil, tc.trusted)
			if tc.diagnostic != "" {
				if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
					t.Fatalf("error = %v, want %q", err, tc.diagnostic)
				}
				return
			}
			if err != nil || rule.format != 1 || rule.args != 2 || rule.result != "append" {
				t.Fatalf("bound Appendf rule = %+v, %v", rule, err)
			}
		})
	}
}
