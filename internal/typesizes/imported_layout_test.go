package typesizes

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

type layoutImports map[string]*types.Package

func (p layoutImports) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if pkg := p[path]; pkg != nil {
		return pkg, nil
	}
	return nil, fmt.Errorf("missing import %q", path)
}

func TestImportedLayoutInheritance(t *testing.T) {
	imports := layoutImports{}
	check := func(t *testing.T, path, arch string, sources ...string) (*types.Package, *LLGoSizes) {
		t.Helper()
		fset := token.NewFileSet()
		files := make([]*ast.File, len(sources))
		for i, source := range sources {
			file, err := parser.ParseFile(fset, fmt.Sprintf("layout%d.go", i), source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			files[i] = file
		}
		pkg := types.NewPackage(path, files[0].Name.Name)
		s := LLGoSizesFor(arch)
		s.Layout = new(LLGoLayout)
		if err := s.Layout.ParseSyntax(fset, pkg, files, nil); err != nil {
			t.Fatal(err)
		}
		conf := types.Config{Sizes: s, Importer: imports}
		if err := types.NewChecker(&conf, fset, pkg, nil).Files(files); err != nil {
			t.Fatal(err)
		}
		s.Layout.RecordPackage(pkg)
		return pkg, s
	}
	// Equal package names are valid with renaming or separate source files.
	// Each consumer uses a fresh layout, so inheritance reads export metadata.
	imports["example/a"], _ = check(t, "example/a", "amd64", "package native\n//llgo:cdecl\ntype NativeA func()")
	imports["example/b"], _ = check(t, "example/b", "amd64", "package native\n//llgo:stdcall\ntype NativeB func()")
	for _, arch := range []string{"amd64", "386", "wasm"} {
		for name, sources := range map[string][]string{
			"dot": {`package p
import . "example/a"
import . "example/b"
import "unsafe"
type CopyA NativeA
type CopyB NativeB
const SizeA = unsafe.Sizeof(CopyA(nil))
const SizeB = unsafe.Sizeof(CopyB(nil))
`},
			"renamed": {`package p
import renamed "example/a"
import "example/b"
import "unsafe"
type CopyA renamed.NativeA
type CopyB native.NativeB
const SizeA = unsafe.Sizeof(CopyA(nil))
const SizeB = unsafe.Sizeof(CopyB(nil))
`},
			"repeated_named": {`package p
import first "example/a"
import second "example/a"
import "example/b"
import "unsafe"
type CopyA first.NativeA
type CopyB native.NativeB
var _ second.NativeA
const SizeA = unsafe.Sizeof(CopyA(nil))
const SizeB = unsafe.Sizeof(CopyB(nil))
`},
			"dot_and_named": {`package p
import . "example/a"
import renamedA "example/a"
import renamedB "example/b"
import . "example/b"
import "unsafe"
type CopyA NativeA
type CopyB renamedB.NativeB
var _ renamedA.NativeA
var _ NativeB
const SizeA = unsafe.Sizeof(CopyA(nil))
const SizeB = unsafe.Sizeof(CopyB(nil))
`},
			"separate_files": {`package p
import "example/a"
import "unsafe"
type CopyA native.NativeA
const SizeA = unsafe.Sizeof(CopyA(nil))
`, `package p
import "example/b"
import "unsafe"
type CopyB native.NativeB
const SizeB = unsafe.Sizeof(CopyB(nil))
`},
		} {
			t.Run(arch+"/"+name, func(t *testing.T) {
				pkg, s := check(t, "p", arch, sources...)
				for suffix, convention := range map[string]string{"A": "C", "B": "stdcall"} {
					typ := pkg.Scope().Lookup("Copy" + suffix).Type()
					if attrs := s.Layout.TypeAttrs(typ); !attrs.Native || attrs.CallConv != convention {
						t.Errorf("Copy%s attributes = %+v, want native %s", suffix, attrs, convention)
					}
					if size, align := s.Sizeof(typ), s.Alignof(typ); size != s.PointerSize || align != s.PointerSize {
						t.Errorf("Copy%s size/alignment = %d/%d, want %d", suffix, size, align, s.PointerSize)
					}
					size, ok := constant.Int64Val(pkg.Scope().Lookup("Size" + suffix).(*types.Const).Val())
					if !ok || size != s.PointerSize {
						t.Errorf("Size%s constant = %d, want %d", suffix, size, s.PointerSize)
					}
				}
			})
		}
	}
}
