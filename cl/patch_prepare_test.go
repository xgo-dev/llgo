package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/typepatch"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestPreparePatchMergesUpfrontAndSkipsRedundantMerge(t *testing.T) {
	fset := token.NewFileSet()
	parseAndCheck := func(path, src string) (*types.Package, *ast.File, *types.Info) {
		t.Helper()
		file, err := parser.ParseFile(fset, path+".go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Scopes:     make(map[ast.Node]*types.Scope),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}
		pkg, err := (&types.Config{}).Check(path, fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		return pkg, file, info
	}

	origTypes, origFile, origInfo := parseAndCheck("example.com/demo", `package demo
func Keep() int { return 1 }
func Replace() int { return 2 }
func Drop() int { return 3 }
`)
	altTypes, altFile, altInfo := parseAndCheck("example.com/demo", `package demo
//llgo:skip Drop
type marker int

func Replace() int { return 20 }
func Extra() int { return 40 }
`)
	variantTypes, variantFile, variantInfo := parseAndCheck("example.com/demo", `package demo
func Keep() int { return 1 }
func Replace() int { return 2 }
func VariantOnly() int { return 99 }
`)

	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	origSSA := prog.CreatePackage(origTypes, []*ast.File{origFile}, origInfo, true)
	altSSA := prog.CreatePackage(altTypes, []*ast.File{altFile}, altInfo, true)
	variantSSA := prog.CreatePackage(variantTypes, []*ast.File{variantFile}, variantInfo, true)
	prog.Build()

	patch := Patch{Alt: altSSA, Types: typepatch.Clone(altTypes)}
	prepared := PreparePatch(patch, origSSA, []*ast.File{origFile, altFile})
	if !prepared.Prepared {
		t.Fatal("expected Prepared=true")
	}
	if origSSA.Pkg != prepared.Types || altSSA.Pkg != prepared.Types {
		t.Fatal("PreparePatch did not rebind origSSA.Pkg and altSSA.Pkg to patch.Types")
	}
	if !typepatch.IsPatched(origTypes) {
		t.Fatal("original package was not marked patched")
	}
	if prepared.Types.Scope().Lookup("Keep") == nil {
		t.Fatal("expected non-skipped original symbol Keep in merged scope")
	}
	if prepared.Types.Scope().Lookup("Extra") == nil {
		t.Fatal("expected alternate symbol Extra in merged scope")
	}
	if prepared.Types.Scope().Lookup("Drop") != nil {
		t.Fatal("skipped symbol Drop should not be merged into patch.Types")
	}

	// Idempotent re-invocation should be a no-op.
	again := PreparePatch(prepared, origSSA, []*ast.File{origFile, altFile})
	if again != prepared {
		t.Fatal("second PreparePatch call unexpectedly mutated prepared patch")
	}

	// Lowering the prepared primary package must skip typepatch.Merge:
	// patch.Types itself must not be passed to Merge as oldTypes (sanity check:
	// its Scope pointer also remains intact).
	scopeBefore := prepared.Types.Scope()
	llprog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer llprog.Dispose()
	patches := Patches{"example.com/demo": prepared}
	pkg1, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, patches, nil, origSSA, []*ast.File{origFile, altFile}, nil, false, Options{})
	if err != nil {
		t.Fatalf("NewPackageExWithEmbedMetaOptions(primary) failed: %v", err)
	}
	pkg1.Module().Dispose()
	if prepared.Types.Scope() != scopeBefore {
		t.Fatal("newPackageEx unexpectedly replaced patch.Types.Scope() for an already-prepared patch")
	}
	if typepatch.IsPatched(prepared.Types) {
		t.Fatal("newPackageEx unexpectedly re-ran typepatch.Merge on an already-prepared patch")
	}

	// Lowering a secondary test variant with the same PkgPath (variantSSA.Pkg != patch.Types)
	// must still run typepatch.Merge and bind variantSSA.Pkg to patch.Types.
	llprog2 := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer llprog2.Dispose()
	pkg2, _, err := NewPackageExWithEmbedMetaOptions(llprog2, nil, patches, nil, variantSSA, []*ast.File{variantFile, altFile}, nil, false, Options{})
	if err != nil {
		t.Fatalf("NewPackageExWithEmbedMetaOptions(variant) failed: %v", err)
	}
	pkg2.Module().Dispose()
	if variantSSA.Pkg != prepared.Types {
		t.Fatal("newPackageEx did not rebind secondary variant package to patch.Types")
	}
	if !typepatch.IsPatched(variantTypes) {
		t.Fatal("newPackageEx did not merge secondary variant package")
	}
	if prepared.Types.Scope().Lookup("VariantOnly") == nil {
		t.Fatal("secondary variant symbol VariantOnly missing from merged patch.Types scope")
	}
}
