package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/internal/typepatch"
	"golang.org/x/tools/go/ssa"
)

func TestPreparePatchMergesUpfrontAndIsReadSafe(t *testing.T) {
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

	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	origSSA := prog.CreatePackage(origTypes, []*ast.File{origFile}, origInfo, true)
	altSSA := prog.CreatePackage(altTypes, []*ast.File{altFile}, altInfo, true)
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

	// Concurrent readers inspecting IsPatched and Scope().Lookup must be race-free.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if !typepatch.IsPatched(origTypes) {
					t.Error("IsPatched returned false concurrently")
				}
				if prepared.Types.Scope().Lookup("Keep") == nil || prepared.Types.Scope().Lookup("Replace") == nil {
					t.Error("concurrent Scope().Lookup missed symbol")
				}
			}
		}()
	}
	wg.Wait()
}
