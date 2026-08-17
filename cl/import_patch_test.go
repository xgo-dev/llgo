//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/gogen/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildGlobal(t *testing.T, pkgPath, pkgName, src, name string) (*types.Package, *gossa.Global) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, pkgName+".go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage(pkgPath, pkgName)
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatalf("BuildPackage failed: %v", err)
	}
	g, ok := ssapkg.Members[name].(*gossa.Global)
	if !ok {
		t.Fatalf("missing global %q", name)
	}
	return pkg, g
}

func TestExtractTrampolineCName(t *testing.T) {
	cases := map[string]string{
		"libc_close_trampoline": "close",
		"open_trampoline":       "open",
		"open":                  "",
		"libc_open":             "",
	}
	for in, want := range cases {
		if got := extractTrampolineCName(in); got != want {
			t.Fatalf("extractTrampolineCName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVarNameGoLinkRuntimeAndNonRuntime(t *testing.T) {
	prog := llssa.NewProgram(nil)
	c := &context{prog: prog}

	rtPkg, rtGlobal := buildGlobal(t, "runtime", "runtime", "package runtime\nvar G int\n", "G")
	rtKey := llssa.FullName(rtPkg, rtGlobal.Name())
	prog.SetLinkname(rtKey, "go:runtime.G")
	name, vtyp, define := c.varName(rtPkg, rtGlobal)
	if name != "go:runtime.G" || vtyp != goVar || !define {
		t.Fatalf("runtime varName = (%q,%d,%v), want (%q,%d,%v)", name, vtyp, define, "go:runtime.G", goVar, true)
	}

	pkg, global := buildGlobal(t, "example.com/p", "p", "package p\nvar G int\n", "G")
	key := llssa.FullName(pkg, global.Name())
	prog.SetLinkname(key, "go:example.com/p.G")
	name, vtyp, define = c.varName(pkg, global)
	if name != "go:example.com/p.G" || vtyp != goVar || define {
		t.Fatalf("non-runtime varName = (%q,%d,%v), want (%q,%d,%v)", name, vtyp, define, "go:example.com/p.G", goVar, false)
	}
}
