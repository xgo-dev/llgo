package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDPairShuffleDeclarations(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Env: append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "GOEXPERIMENT=simd")}, "simd/archsimd")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) != 0 || len(pkgs) != 1 {
		t.Fatal("could not load archsimd")
	}
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	wants := map[string]llvmgen.SIMDOp{
		"concatSelectedConstant":        llvmgen.SIMDConcatSelected,
		"concatSelectedConstantGrouped": llvmgen.SIMDConcatSelectedGrouped,
		"ConcatPermute128Scalars":       llvmgen.SIMDConcatPermute128,
	}
	counts := make(map[string]int)
	for fn := range ssautil.AllFunctions(prog) {
		obj, ok := fn.Object().(*types.Func)
		if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" {
			continue
		}
		want, ok := wants[obj.Name()]
		decl, isDecl := fn.Syntax().(*ast.FuncDecl)
		if !ok || !isDecl || decl.Body != nil || fn.Signature.Recv() == nil {
			continue
		}
		if simdVectorBits(fn.Signature.Recv().Type()) == 0 {
			want = llvmgen.SIMDUnimplemented
		} else {
			counts[obj.Name()]++
		}
		for _, arch := range []string{"amd64", "arm64", "wasm"} {
			expected := want
			if arch != "amd64" {
				expected = llvmgen.SIMDUnimplemented
			}
			desc, ok := lookupSIMD(fn, arch)
			if !ok || desc.op != expected {
				t.Errorf("%s on %s: got %v (%t), want %v", obj.FullName(), arch, desc.op, ok, expected)
			}
		}
	}
	for name, count := range map[string]int{"concatSelectedConstant": 6, "concatSelectedConstantGrouped": 12, "ConcatPermute128Scalars": 10} {
		if counts[name] != count {
			t.Errorf("%s: checked %d declarations, want %d", name, counts[name], count)
		}
	}
}

func TestSIMDPairShuffleShapes(t *testing.T) {
	const declarations = `package archsimd
type v128 struct { _ [0]func() }; type v256 struct { _ [0]func() }
type Int16x8 struct { tag v128; vals [8]int16 }
type Int32x4 struct { tag v128; vals [4]int32 }
type Int32x8 struct { tag v256; vals [8]int32 }
type Uint32x8 struct { tag v256; vals [8]uint32 }
type Float64x2 struct { tag v128; vals [2]float64 }
type Float64x4 struct { tag v256; vals [4]float64 }
`
	for _, tc := range []struct {
		recv, method string
		want         llvmgen.SIMDOp
	}{
		{"Int32x4", "concatSelectedConstant(uint8, Int32x4) Int32x4", llvmgen.SIMDConcatSelected},
		{"Float64x2", "concatSelectedConstant(uint8, Float64x2) Float64x2", llvmgen.SIMDConcatSelected},
		{"Int16x8", "concatSelectedConstant(uint8, Int16x8) Int16x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "concatSelectedConstant(uint8, Int32x8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x4", "concatSelectedConstantGrouped(uint8, Int32x4) Int32x4", llvmgen.SIMDUnimplemented},
		{"Float64x4", "concatSelectedConstantGrouped(uint8, Float64x4) Float64x4", llvmgen.SIMDConcatSelectedGrouped},
		{"Int32x8", "concatSelectedConstantGrouped(uint8, Int32x8) Int32x8", llvmgen.SIMDConcatSelectedGrouped},
		{"Int32x8", "concatSelectedConstantGrouped(uint32, Int32x8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "concatSelectedConstantGrouped(Int32x8, uint8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "concatSelectedConstantGrouped(uint8, Int32x4) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "concatSelectedConstantGrouped(uint8, Int32x8) Uint32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "ConcatPermute128Scalars(uint8, uint8, Int32x8) Int32x8", llvmgen.SIMDConcatPermute128},
		{"Int32x4", "ConcatPermute128Scalars(uint8, uint8, Int32x4) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int32x8", "ConcatPermute128Scalars(uint8, Int32x8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "ConcatPermute128Scalars(uint8, int8, Int32x8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "ConcatPermute128Scalars(uint8, uint8, Uint32x8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x8", "ConcatPermute128Scalars(uint8, uint8, Int32x8) Int32x4", llvmgen.SIMDUnimplemented},
	} {
		t.Run(tc.recv+"."+tc.method, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "shapes.go", declarations+"func (x "+tc.recv+") "+tc.method, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check("simd/archsimd", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssa.NewProgram(fs, 0)
			prog.CreatePackage(pkg, nil, nil, true)
			fn := prog.FuncValue(pkg.Scope().Lookup(tc.recv).Type().(*types.Named).Method(0))
			desc, ok := lookupSIMD(fn, "amd64")
			if wantOK := tc.want != llvmgen.SIMDUnimplemented; ok != wantOK || ok && desc.op != tc.want {
				t.Fatalf("lookup=%v (%t), want %v", desc.op, ok, tc.want)
			}
		})
	}
}
