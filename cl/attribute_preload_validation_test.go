package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/funcattrs"
)

func TestSourceAttributePreloadDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"//llgo:attr cold(extra)\nfunc F() {}", "invalid arguments"},
		{"//llgo:attr param(p) noalias\nfunc F(p int) {}", "noalias requires a pointer"},
		{"//llgo:attribute cold\nfunc F() {}", "use //llgo:attr"},
		{`import _ "unsafe"
//go:linkname A shared_function
//llgo:attr result(0) range(0,10)
func A() int { return 0 }
//go:linkname B shared_function
//llgo:attr result(0) range(0,20)
func B() int { return 0 }`, "conflicting range"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			err := compileImplicitEffectDiagnostic(t, "package effects\n"+tc.source, Options{})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "implicit_effects.go:") {
				t.Fatalf("source diagnostic = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSourcePreloadRejectsConflictingKnownDeclaration(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "conflict.go", `package p
//llgo:attr memory(read)
func F() {}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{}).Check("p", fs, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	if err = prog.SetFunctionAttributes("p.F", []funcattrs.Attribute{{Target: funcattrs.Target{Scope: funcattrs.Function}, Name: "memory", Args: "none"}}); err != nil {
		t.Fatal(err)
	}
	err = ParsePkgSyntax(prog, fs, pkg, []*ast.File{file})
	if err == nil || !strings.Contains(err.Error(), "conflicting memory") || !strings.Contains(err.Error(), "conflict.go:2:") {
		t.Fatalf("conflicting preloaded declaration accepted: %v", err)
	}
}
