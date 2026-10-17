package main

import (
	"go/build"
	"go/importer"

	"github.com/xgo-dev/llgo/internal/build/testdata/runtimecompiler/identity"
)

func main() {
	if build.Default.Compiler != "gc" {
		panic("go/build lost the selected Go toolchain's compiler")
	}
	pkg, err := build.Import("fmt", "", build.FindOnly)
	if err != nil {
		panic(err)
	}
	if pkg.ImportPath != "fmt" || !pkg.Goroot {
		panic("go/build did not resolve the standard library")
	}
	imp := importer.Default()
	if imp == nil {
		panic("go/importer lost its default gc backend")
	}
	if pkg, err := imp.Import("fmt"); err != nil || pkg.Path() != "fmt" {
		panic("go/importer did not resolve the standard library")
	}
	println(identity.Name, identity.Description, len(identity.NameBytes{}), identity.Branch())
}
