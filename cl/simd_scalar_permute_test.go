package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDScalarPermuteDeclarations(t *testing.T) {
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
		"permuteScalars":          llvmgen.SIMDPermuteScalars,
		"permuteScalarsGrouped":   llvmgen.SIMDPermuteScalarsGrouped,
		"permuteScalarsHi":        llvmgen.SIMDPermuteScalarsHi,
		"permuteScalarsLo":        llvmgen.SIMDPermuteScalarsLo,
		"permuteScalarsHiGrouped": llvmgen.SIMDPermuteScalarsHiGrouped,
		"permuteScalarsLoGrouped": llvmgen.SIMDPermuteScalarsLoGrouped,
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
	for name := range wants {
		want := 2
		if strings.HasSuffix(name, "Grouped") {
			want = 4
		}
		if counts[name] != want {
			t.Errorf("%s: checked %d declarations, want %d", name, counts[name], want)
		}
	}
}

func TestSIMDScalarPermuteShapes(t *testing.T) {
	const declarations = `package archsimd
    type v128 struct { _ [0]func() }; type v256 struct { _ [0]func() }
    type Int16x8 struct { tag v128; vals [8]int16 }
    type Int16x16 struct { tag v256; vals [16]int16 }
    type Int32x4 struct { tag v128; vals [4]int32 }
    type Int32x8 struct { tag v256; vals [8]int32 }
    type Uint32x4 struct { tag v128; vals [4]uint32 }
    type Float32x4 struct { tag v128; vals [4]float32 }
    type Int64x2 struct { tag v128; vals [2]int64 }
    `
	for _, tc := range []struct {
		recv, method string
		want         llvmgen.SIMDOp
	}{
		{"Int32x4", "permuteScalars(uint8) Int32x4", llvmgen.SIMDPermuteScalars},
		{"Int32x8", "permuteScalarsGrouped(uint8) Int32x8", llvmgen.SIMDPermuteScalarsGrouped},
		{"Int16x8", "permuteScalarsHi(uint8) Int16x8", llvmgen.SIMDPermuteScalarsHi},
		{"Int16x8", "permuteScalarsLo(uint8) Int16x8", llvmgen.SIMDPermuteScalarsLo},
		{"Int16x16", "permuteScalarsHiGrouped(uint8) Int16x16", llvmgen.SIMDPermuteScalarsHiGrouped},
		{"Int16x16", "permuteScalarsLoGrouped(uint8) Int16x16", llvmgen.SIMDPermuteScalarsLoGrouped},
		{"Int32x8", "permuteScalars(uint8) Int32x8", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalarsGrouped(uint8) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int16x16", "permuteScalarsHi(uint8) Int16x16", llvmgen.SIMDUnimplemented},
		{"Int16x16", "permuteScalarsLo(uint8) Int16x16", llvmgen.SIMDUnimplemented},
		{"Int16x8", "permuteScalarsHiGrouped(uint8) Int16x8", llvmgen.SIMDUnimplemented},
		{"Int16x8", "permuteScalarsLoGrouped(uint8) Int16x8", llvmgen.SIMDUnimplemented},
		{"Int16x8", "permuteScalars(uint8) Int16x8", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalarsHi(uint8) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int64x2", "permuteScalars(uint8) Int64x2", llvmgen.SIMDUnimplemented},
		{"Float32x4", "permuteScalars(uint8) Float32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(uint32) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(int8) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(uint8, uint8) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(...uint8) Int32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(uint8) Uint32x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "permuteScalars(uint8) Int32x8", llvmgen.SIMDUnimplemented},
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
