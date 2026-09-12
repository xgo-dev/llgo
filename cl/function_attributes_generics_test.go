package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestImportedGenericSourceContractsValidateConcreteInstance(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
		want  string
	}{
		{"pointer", "*int", ""},
		{"integer", "int", "nonnull requires a pointer, got int"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			check := func(path, filename, source string, imp types.Importer) (*types.Package, *types.Info, *ast.File) {
				t.Helper()
				file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				info := newLocalityTypeInfo()
				pkg, err := (&types.Config{Importer: imp}).Check(path, fset, []*ast.File{file}, info)
				if err != nil {
					t.Fatal(err)
				}
				return pkg, info, file
			}
			dep, depInfo, depFile := check("example.com/contractdep", "contractdep.go", `package contractdep
//llgo:attribute param(v).field(P) nonnull
//llgo:attribute result(0).field(P) same_as(param(v).field(P))
func Identity[T any](v T) T { return v }
`, nil)
			root, rootInfo, rootFile := check("example.com/contractuser", "contractuser.go", `package contractuser
import "example.com/contractdep"
type Box struct { P `+test.field+` }
func Use(v Box) Box { return contractdep.Identity(v) }
`, importerFunc(func(path string) (*types.Package, error) {
				if path == dep.Path() {
					return dep, nil
				}
				return nil, types.Error{Msg: "unexpected import " + path}
			}))
			goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
			goProg.CreatePackage(dep, []*ast.File{depFile}, depInfo, true)
			rootSSA := goProg.CreatePackage(root, []*ast.File{rootFile}, rootInfo, true)
			goProg.Build()
			coordinator := newLLSSAProg(t)
			defer coordinator.Dispose()
			// Source T has no fields until instantiated. Preload must accept
			// the declaration, then the caller's separate backend must retain
			// its origin and check the selected concrete field.
			for _, input := range []struct {
				pkg  *types.Package
				file *ast.File
			}{{dep, depFile}, {root, rootFile}} {
				if err := ParsePkgSyntax(coordinator, fset, input.pkg, []*ast.File{input.file}); err != nil {
					t.Fatalf("generic contract rejected before instantiation: %v", err)
				}
			}
			backend := coordinator.NewBackendProgram()
			defer backend.Dispose()
			compile := func() (ir string, err error) {
				defer func() {
					if value := recover(); value != nil {
						var ok bool
						err, ok = value.(error)
						if !ok {
							panic(value)
						}
					}
				}()
				compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, nil, nil, nil, rootSSA, []*ast.File{rootFile}, nil, false, Options{PreloadedSyntax: true})
				if err != nil {
					return "", err
				}
				if err = llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
					return "", err
				}
				return compiled.String(), nil
			}
			ir, err := compile()
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), "contractdep.go:2:") {
					t.Fatalf("concrete contract diagnostic = %v, want %q at its source declaration", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(ir, "example.com/contractdep.Identity[") || !strings.Contains(ir, "llgo.source.attributes") {
				t.Fatalf("caller did not emit the imported instance and its source contracts:\n%s", ir)
			}
		})
	}
}
