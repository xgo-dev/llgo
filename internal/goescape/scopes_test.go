package goescape

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func sourceFunction(t *testing.T, source string) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package test\n"+source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, types.NewPackage("test", "test"), []*ast.File{file}, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Func("f")
}

func TestNestedIterationScopes(t *testing.T) {
	fn := sourceFunction(t, `var sink *int
func f(n int) {
 entry := new(int); sink=entry
 for i:=0;i<n;i++ {
  outer := new(int); sink=outer
  for j:=0;j<n;j++ { inner:=new(int);sink=inner }
 }
}`)
	s := functionScopes(fn)
	depths := make(map[int]int)
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if alloc, ok := instr.(*ssa.Alloc); ok {
				depth := 0
				for region := s.valueScope(alloc); region != s.invocation; region = region.parent {
					depth++
				}
				depths[depth]++
			}
		}
	}
	if len(depths) != 3 || depths[0] != 1 || depths[1] != 1 || depths[2] != 1 {
		t.Fatalf("allocation depths=%v", depths)
	}
	if len(s.uncertain) != 0 {
		t.Fatal("ordinary loops marked irreducible")
	}
}

func TestLoopCarriedPhiOutlivesIteration(t *testing.T) {
	fn := sourceFunction(t, `func f(n int) *int {
 var previous *int
 for i:=0;i<n;i++ { current:=new(int);if previous==current {panic("identity")};previous=current }
 return previous
}`)
	s := functionScopes(fn)
	found := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				continue
			}
			if _, pointer := phi.Type().Underlying().(*types.Pointer); pointer && s.headers[s.blocks[block]] == block {
				found = true
				if s.valueScope(phi) != s.blocks[block].parent {
					t.Fatal("loop-carried phi has iteration lifetime")
				}
			}
		}
	}
	if !found {
		t.Fatal("missing loop-carried pointer phi")
	}
}

func TestIrreducibleCycleRetainsConservativePolicy(t *testing.T) {
	fn := sourceFunction(t, `var sink *int
func f(n int, enter bool) {
 if enter {goto B}
A:
 sink=new(int);n--;if n==0 {return}
B:
 sink=new(int);n--;if n!=0 {goto A}
}`)
	s := functionScopes(fn)
	allocations := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if _, ok := instr.(*ssa.Alloc); ok {
				allocations++
				if !s.uncertain[block] {
					t.Fatal("irreducible allocation accepted")
				}
			}
		}
	}
	if allocations != 2 {
		t.Fatalf("allocations=%d", allocations)
	}
}
