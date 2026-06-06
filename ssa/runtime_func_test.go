//go:build !llgo
// +build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/gogen/packages"
)

func TestRuntimeFuncMarksPackageRuntimeUse(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		imp := packages.NewImporter(token.NewFileSet())
		pkg, err := imp.Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("foo", "example.com/foo")
	fn := pkg.RuntimeFunc("PushCallerFrame")
	if fn.IsNil() {
		t.Fatal("RuntimeFunc returned nil expression")
	}
	if !pkg.NeedRuntime {
		t.Fatal("RuntimeFunc should mark the package as needing runtime")
	}
}
