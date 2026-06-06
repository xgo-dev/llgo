package ssa

import (
	"go/importer"
	"go/types"
	"strings"
	"testing"
)

func TestRuntimeFuncWrapper(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("foo", "foo")

	expr := pkg.RuntimeFunc("SetCallerLine")
	if expr.IsNil() {
		t.Fatal("RuntimeFunc returned nil")
	}
	if !pkg.NeedRuntime {
		t.Fatal("RuntimeFunc should mark package as needing runtime")
	}
	if name := expr.Name(); !strings.HasSuffix(name, ".SetCallerLine") {
		t.Fatalf("RuntimeFunc name = %q, want SetCallerLine suffix", name)
	}
}
