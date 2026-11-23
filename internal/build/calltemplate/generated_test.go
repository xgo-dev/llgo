package calltemplate

import (
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestGeneratedTemplateFallback(t *testing.T) {
	provider := checkedSource(t, `package p
//llgo:template count
func Count(a any, b any) int { return 0 }
func count[T any](a []T, b any) int { return Count(a, b) }
func Other() {}

`)
	var planned []GeneratedCall
	bindings, err := LoadBindings([]*packages.Package{provider}, func(*packages.Package) bool { return true }, Generator{
		Package:         provider.PkgPath,
		FallbackEntries: []string{"Other"},
		Apply: func(input Generation) (bool, error) {
			planned = input.Calls
			return true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, source string
		count        int
	}{
		{"partial", "func f(v any) { Count([]int{1}, v); Count([]int{2}, nil) }", 2},
		{"interface argument", "func f(v any) { Count([]int{1}, nil); Count(v, nil) }", 0},
		{"type parameter", "func f[T any](v []T) { Count(v, nil) }", 0},
		{"function value", "var g = Count; func f() { Count([]int{}, nil) }", 0},
		{"fallback entry", "func f() { Count([]int{}, nil); Other() }", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := checkedSourceWithImports(t, "package main\nimport . \"p\"\n"+tc.source, typesynth.Importer{"p": provider.Types})
			pkg.TypesSizes = types.SizesFor("gc", "amd64")
			pkg.Imports = map[string]*packages.Package{"p": provider}
			planned = nil
			if direct, err := bindings.Rewrite([]*packages.Package{pkg}, false); err != nil || len(direct) != 0 {
				t.Fatalf("generator candidates entered direct rewrite: %v, %v", direct, err)
			}
			selected, err := bindings.RewriteGenerated([]*packages.Package{pkg}, nil)
			if err != nil || len(selected.Changes) != tc.count || len(planned) != tc.count || selected.Providers["p"] != (tc.count != 0) {
				t.Fatalf("selected %+v, error %v, want %d calls", selected, err, tc.count)
			}
			for _, c := range planned {
				if c.Candidate.Name() != "count" || c.Candidate.Pkg() != provider.Types {
					t.Fatalf("unexpected candidate %v", c.Candidate)
				}
			}
		})
	}
}
