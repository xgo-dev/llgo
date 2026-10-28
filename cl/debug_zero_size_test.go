package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// GOROOT/test/abi/f_ret_z_not.go crashed in LLVM 22's
// SelectionDAGBuilder::resolveDanglingDebugInfo after optimization promoted an
// empty aggregate snapshot to dbg.value(empty zeroinitializer). Module
// verification alone does not catch it: exercise object emission too.
func TestZeroSizedDebugValuesEmitObjects(t *testing.T) {
	const source = `package zero
type Z struct{}
type NZ struct { x, y int }
//go:noinline
func f(x, y int) (Z, NZ, Z) {
	var z Z
	return z, NZ{x, y}, z
}
//go:noinline
func g() (Z, NZ, Z) {
	a, b, c := f(3, 4)
	return c, b, a
}
func array(z [0]int) ([0]int, int, [0]int) {
	var empty [0]int
	return empty, 7, z
}
func check() int {
	_, b, _ := g()
	return b.x + b.y
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "zero.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{}, fset,
		types.NewPackage("zero", "zero"), []*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct{ name, goos, triple string }{
		{"darwin", "darwin", "aarch64-apple-darwin"},
		{"linux", "linux", "aarch64-unknown-linux-gnu"},
		{"windows-msvc", "windows", "aarch64-pc-windows-msvc"},
		{"windows-mingw", "windows", "aarch64-w64-windows-gnu"},
	} {
		for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2, optlevel.Os} {
			t.Run(target.name+"/"+level.String(), func(t *testing.T) {
				prog := newLLSSAProgForTarget(t, &llssa.Target{
					GOOS: target.goos, GOARCH: "arm64", LLVMTarget: target.triple, OptLevel: level,
				})
				defer prog.Dispose()
				if err := prepareTestSyntax(prog, ssaPkg, []*ast.File{file}, Options{}); err != nil {
					t.Fatal(err)
				}
				prog.Directives().Freeze()
				pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg,
					[]*ast.File{file}, nil, false, Options{Debug: true, DebugSymbols: true})
				if err != nil {
					t.Fatal(err)
				}
				mod := pkg.Module()
				mod.SetDataLayout(prog.DataLayout())
				mod.SetTarget(target.triple)
				options := llvm.NewPassBuilderOptions()
				defer options.Dispose()
				options.SetVerifyEach(true)
				if err := mod.RunPasses("default<"+level.String()+">", prog.TargetMachine(), options); err != nil {
					t.Fatal(err)
				}
				object, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
				if err != nil {
					t.Fatal(err)
				}
				defer object.Dispose()
				if len(object.Bytes()) == 0 {
					t.Fatal("empty object")
				}
			})
		}
	}
}
