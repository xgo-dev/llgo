package ssa

import (
	"go/token"
	"go/types"
	"testing"
)

func TestFunctionDeclarationsKeepSourceIdentity(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("example.com/p", "p")
	variant := types.NewPackage("example.com/p", "p")
	fset := token.NewFileSet()
	first := prog.DeclareFunction(pkg, fset, "example.com/p.Entry", 7)
	first.SetLinkname("shared")
	first.SetExport("entry")
	first.SetExplicitEnv(true)
	first.SetNoInterface(true)
	first.SetWasmImport("host", "entry")

	for _, test := range []struct {
		name string
		pkg  *types.Package
		fset *token.FileSet
		pos  token.Pos
	}{
		{"example.com/p.Alias", pkg, fset, 7},
		{"example.com/p.Entry", pkg, fset, 8},
		{"example.com/p.Entry", pkg, token.NewFileSet(), 7},
		{"example.com/p.Entry", variant, fset, 7},
	} {
		other := prog.DeclareFunction(test.pkg, test.fset, test.name, test.pos)
		other.SetLinkname("shared")
		if other == first || other.HasExplicitEnv() || other.NoInterface() {
			t.Fatal("distinct declarations shared source properties")
		}
		if _, ok := other.Export(); ok {
			t.Fatal("export leaked to another declaration")
		}
		if _, _, ok := other.WasmImport(); ok {
			t.Fatal("wasm import leaked to another declaration")
		}
		if got := prog.SourceFunctionDeclaration(test.pkg, test.fset, test.name, test.pos); got != other {
			t.Fatal("source lookup did not preserve declaration identity")
		}
	}
	if got := prog.DeclareFunction(pkg, fset, first.Name(), 7); got != first {
		t.Fatal("reloading one declaration replaced its object")
	}
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	if got := backend.SourceFunctionDeclaration(pkg, fset, first.Name(), 7); got != first {
		t.Fatal("backend did not retain the prepared declaration")
	}
	if !prog.HasLinknameTarget("shared") {
		t.Fatal("declaration link lost from symbol lookup")
	}
}

func TestFunctionDeclarationAdoptsPendingSymbolInformation(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.SetLinkname("p.Entry", "entry")
	prog.SetPackageExport("p.Entry", "entry")
	prog.SetLinkname("p.variable", "variable")
	fn := prog.DeclareFunction(nil, nil, "p.Entry", token.NoPos)
	if link, ok := fn.Linkname(); !ok || link != "entry" {
		t.Fatal("lost pending link")
	}
	if name, ok := fn.Export(); !ok || name != "entry" {
		t.Fatal("lost pending export")
	}
	if _, ok := prog.packageSyntax.linknames["p.Entry"]; ok {
		t.Fatal("function link remains in side table")
	}
	if _, ok := prog.packageSyntax.exports["p.Entry"]; ok {
		t.Fatal("function export remains in side table")
	}
	if link, ok := prog.Linkname("p.variable"); !ok || link != "variable" {
		t.Fatal("variable link changed")
	}
	prog.SetLinkname("p.Entry", "renamed")
	if link, _ := fn.Linkname(); link != "renamed" {
		t.Fatal("symbol setter replaced declaration metadata")
	}
}

func TestFunctionDeclarationMethodPackageVariants(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	for _, hidden := range []bool{true, false} {
		pkg := types.NewPackage("p", "p")
		named := types.NewNamed(types.NewTypeName(0, pkg, "T", nil), types.NewStruct(nil, nil), nil)
		recv := types.NewVar(0, pkg, "", named)
		fn := types.NewFunc(7, pkg, "M", types.NewSignatureType(recv, nil, nil, nil, nil, false))
		named.AddMethod(fn)
		decl := prog.DeclareFunction(pkg, nil, "p.T.M", 7)
		decl.SetNoInterface(hidden)
		if prog.FunctionDeclarationOf(fn) != decl || prog.isNoInterfaceMethod(fn) != hidden {
			t.Fatal("method metadata crossed package variants")
		}
	}
}

func TestFunctionDeclarationBindsPendingMethod(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.SetNoInterfaceMethod("p.T.M")
	pending := prog.NamedFunctionDeclaration("p.T.M")
	pkg := types.NewPackage("p", "p")
	fset := token.NewFileSet()
	decl := prog.DeclareFunction(pkg, fset, "p.T.M", 7)
	if decl != pending || !decl.NoInterface() {
		t.Fatal("loading syntax lost the pending method declaration")
	}
	if prog.SourceFunctionDeclaration(pkg, fset, "p.T.M", 7) != pending {
		t.Fatal("source identity did not bind to the existing declaration")
	}
}
