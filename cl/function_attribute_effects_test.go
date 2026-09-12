package cl

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestSourceContractRejectsUnmodelledImplicitRuntimeProtocols(t *testing.T) {
	for _, test := range []struct {
		name, source, reason string
		options              Options
	}{
		{
			name: "closure allocation",
			source: `package effects
//llgo:attr memory(none)
//go:noinline
func F(x int) func() int { return func() int { return x } }
`,
			reason: "compiler-generated heap allocation for closure or defer state",
		},
		{
			name: "defer protocol",
			source: `package effects
func cleanup() {}
//llgo:attr memory(none)
//go:noinline
func F() { defer cleanup() }
`,
			reason: "compiler-generated defer runtime protocol",
		},
		{
			name: "local context entry",
			source: `package effects
//llgointernal:gls
var pointer *int
//export F
//llgo:attr memory(read)
//go:noinline
func F() *int { return nil }
`,
			reason: "compiler-generated local context entry",
		},
		{
			name: "local context entry with attribute before export",
			source: `package effects
//llgointernal:gls
var pointer *int
// llgo:attr memory(read)
//export F
//go:noinline
func F() *int { return nil }
`,
			reason: "compiler-generated local context entry",
		},
		{
			name: "local package storage",
			source: `package effects
//llgointernal:gls
var pointer *int
//llgo:attr memory(read)
//go:noinline
func F() *int { return pointer }
`,
			reason: "compiler-generated local package storage lookup",
		},
		{
			name: "local package initialization",
			source: `package effects
func initialValue() int { return 42 }
//llgointernal:tls
var counter = initialValue()
//llgo:attr memory(read)
//go:noinline
func F() int { return counter }
`,
			reason: "compiler-generated local package initialization",
		},
		{
			name: "shadow stack",
			source: `package effects
import "runtime"
//llgo:attr memory(none)
//go:noinline
func F() uintptr { pc, _, _, _ := runtime.Caller(0); return pc }
`,
			reason:  "compiler-generated shadow stack",
			options: Options{ShadowStack: true},
		},
		{
			name: "tracing",
			source: `package effects
//llgo:attr memory(none)
//go:noinline
func F() int { return 42 }
`,
			reason:  "compiler-generated function tracing",
			options: Options{Trace: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := compileImplicitEffectDiagnostic(t, test.source, test.options)
			if err == nil || !strings.Contains(err.Error(), "implicit_effects.go:") || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("error = %v, want source-located %q diagnostic", err, test.reason)
			}
		})
	}
}

func compileImplicitEffectDiagnostic(t *testing.T, source string, options Options) (err error) {
	t.Helper()
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("%v", value)
		}
	}()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "implicit_effects.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := newLocalityTypeInfo()
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("example.com/effects", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	options.AllowInternalDirectives = true
	if err = ParsePkgSyntaxWithOptions(prog, fset, pkg, files, options); err != nil {
		return err
	}
	if err = PrepareLocalVariables(prog, fset, pkg, info, files); err != nil {
		return err
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	imports := make(map[*types.Package]bool)
	var addImport func(*types.Package)
	addImport = func(dependency *types.Package) {
		if imports[dependency] {
			return
		}
		imports[dependency] = true
		for _, child := range dependency.Imports() {
			addImport(child)
		}
		goProg.CreatePackage(dependency, nil, nil, true)
	}
	for _, dependency := range pkg.Imports() {
		addImport(dependency)
	}
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()
	_, _, err = NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssaPkg, files, nil, false, options)
	return err
}
