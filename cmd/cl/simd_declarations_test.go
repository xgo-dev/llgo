package cl

import (
	"go/ast"
	"go/types"
	"os"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Check every intrinsic declaration on the listed targets, including methods that upstream tests
// reach only through architecture or CPU feature predicates.
func TestSIMDCompleteDeclarations(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"linux", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.arch, func(t *testing.T) {
			pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Env: append(os.Environ(), "GOOS="+target.os, "GOARCH="+target.arch, "GOEXPERIMENT=simd")}, "simd/archsimd")
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || packages.PrintErrors(pkgs) != 0 {
				t.Fatal("could not load archsimd")
			}
			prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
			prog.Build()
			count := 0
			for _, file := range pkgs[0].Syntax {
				for _, node := range file.Decls {
					decl, ok := node.(*ast.FuncDecl)
					if !ok || decl.Body != nil {
						continue
					}
					obj := pkgs[0].TypesInfo.Defs[decl.Name].(*types.Func)
					fn := prog.FuncValue(obj)
					desc, ok := lookupSIMD(fn, target.arch)
					if !ok || desc.op == llvmgen.SIMDUnimplemented {
						t.Errorf("unsupported intrinsic %s", obj.FullName())
					}
					count++
				}
			}
			if count == 0 {
				t.Fatal("no intrinsic declarations found")
			}
		})
	}
}
