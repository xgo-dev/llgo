package plan9asm

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
)

type importedGlobalTestImporter struct{ owner *types.Package }

func (i importedGlobalTestImporter) Import(path string) (*types.Package, error) {
	if i.owner != nil && i.owner.Path() == path {
		return i.owner, nil
	}
	return nil, fmt.Errorf("unexpected fixture import %q", path)
}

func importedGlobalTestPackage(t *testing.T, arch string) *packages.Package {
	t.Helper()
	owner := mustTestPackage(t, "example.com/flags", "package flags\nvar Enabled bool\n")
	source := "package consumer\nimport \"example.com/flags\"\nvar _ = flags.Enabled\nfunc Read() uint32\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "consumer.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	sizes := types.SizesFor("gc", arch)
	typed, err := (&types.Config{Importer: importedGlobalTestImporter{owner.Types}, Sizes: sizes}).Check(
		"example.com/consumer", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &packages.Package{PkgPath: typed.Path(), Name: typed.Name(), Types: typed, TypesSizes: sizes,
		Fset: fset, Syntax: []*ast.File{file}, Imports: map[string]*packages.Package{owner.PkgPath: owner}}
}

const importedGlobalTestAssembly = `TEXT ·Read(SB),4,$0-4
	CMPB example·com∕flags·Enabled(SB),$0
	JNE enabled
	MOVL $0,AX
	JMP result
enabled:
	MOVL $1,AX
result:
	MOVL AX,X0
	PEXTRD $0,X0,AX
	MOVL AX,ret+0(FP)
	RET
`

func TestTranslateLoadedImportedGlobal(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "386"}, {"linux", "amd64"}, {"darwin", "amd64"}, {"windows", "amd64"},
	} {
		for _, preprocessed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/preprocessed=%t", target.goos, target.goarch, preprocessed), func(t *testing.T) {
				pkg := importedGlobalTestPackage(t, target.goarch)
				tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "read.s", []byte(importedGlobalTestAssembly),
					target.goos, target.goarch, TranslateOptions{SourcePreprocessed: preprocessed})
				if err != nil {
					t.Fatal("loaded dependency syntax was not supplied to the translator", err)
				}
				defer tr.Module.Dispose()
				if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				ir := tr.Module.String()
				if !strings.Contains(ir, "load volatile i8") || !strings.Contains(ir, "example.com/flags.Enabled") {
					t.Fatal("missing actual mutable imported read", ir)
				}
			})
		}
	}
}

func TestTranslateLoadedImportedGlobalRequiresCompleteSource(t *testing.T) {
	for _, mutation := range []string{"missing-syntax", "different-type", "different-owner", "missing-types", "nil-import"} {
		t.Run(mutation, func(t *testing.T) {
			pkg := importedGlobalTestPackage(t, "amd64")
			owner := pkg.Imports["example.com/flags"]
			switch mutation {
			case "missing-syntax":
				owner.Syntax = nil
			case "different-type":
				owner.Syntax = mustTestPackage(t, owner.PkgPath, "package flags\nvar Enabled *byte\n").Syntax
			case "different-owner":
				owner.Types = mustTestPackage(t, owner.PkgPath, "package flags\nvar Enabled bool\n").Types
			case "missing-types":
				owner.Types = nil
			case "nil-import":
				pkg.Imports["example.com/flags"] = nil
			}
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "read.s", []byte(importedGlobalTestAssembly),
				"linux", "amd64", TranslateOptions{SourcePreprocessed: true})
			if tr != nil {
				tr.Module.Dispose()
			}
			if tr != nil || err == nil {
				t.Fatal("incomplete or mismatched import acquired numerical storage", err)
			}
		})
	}
}
