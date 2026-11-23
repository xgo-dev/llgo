package calltemplate

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestStructuralTemplateFallback(t *testing.T) {
	provider := checkedSource(t, `package p
func Count(a any, b any) int { return 0 }
func count[T any](a []T, b any) int { return Count(a, b) }
func Other() {}

//llgointernal:template:bind Count any helper=count body=countBody
//llgointernal:template:bind Other any fallback

const countBody = `+"\x60"+`
{{- .RequireBuiltins "int" "len" -}}
{{- if not (isInteger (elem (.Arg 0))) }}{{ reject }}{{ end }}
return len(a0)
`+"\x60")
	bindings := templateRegistry(t, provider)
	for _, tc := range []struct {
		name, source string
		count        int
	}{
		{"partial", "func f(v any) { Count([]int{1}, v); Count([]int{2}, nil) }", 2},
		{"interface argument", "func f(v any) { Count([]int{1}, nil); Count(v, nil) }", 0},
		{"type parameter", "func f[T any](v []T) { Count(v, nil) }", 0},
		{"rejected type", "func f() { Count([]int{1}, nil); Count([]string{}, nil) }", 0},
		{"shadowed builtin", "var len int; func f() { Count([]int{}, nil) }", 0},
		{"function value", "var g = Count; func f() { Count([]int{}, nil) }", 0},
		{"fallback entry", "func f() { Count([]int{}, nil); Other() }", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := checkedSourceWithImports(t, "package main\nimport . \"p\"\n"+tc.source, typesynth.Importer{"p": provider.Types})
			pkg.TypesSizes = types.SizesFor("gc", "amd64")
			pkg.Imports = map[string]*packages.Package{"p": provider}
			count := pkg.Types.Scope().NumChildren()
			selected, err := bindings.RewriteStructural([]*packages.Package{pkg}, nil)
			if err != nil || len(selected.Changes) != tc.count || selected.Providers["p"] != (tc.count != 0) {
				t.Fatalf("selected %+v, error %v, want %d calls", selected, err, tc.count)
			}
			if tc.count == 0 {
				if len(pkg.Syntax) != 1 || pkg.Types.Scope().NumChildren() != count {
					t.Fatal("rejected plan changed the caller")
				}
				return
			}
			var first *types.Func
			ast.Inspect(pkg.Syntax[0], func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				fn := calledFunc(pkg.TypesInfo, call.Fun)
				if fn == nil || fn.Pkg() != pkg.Types || !strings.HasPrefix(fn.Name(), "__llgo_p_") {
					t.Fatalf("call did not use a caller-local helper: %v", fn)
				}
				if first != nil && first != fn {
					t.Fatal("identical typed calls use distinct helper functions")
				}
				first = fn
				return true
			})
		})
	}
}
