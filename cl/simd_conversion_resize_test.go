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

func TestSIMDConversionResizeDeclarations(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "wasm"} {
		t.Run(arch, func(t *testing.T) {
			goos := "linux"
			if arch == "wasm" {
				goos = "wasip1"
			}
			pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Env: append(os.Environ(), "GOOS="+goos, "GOARCH="+arch, "GOEXPERIMENT=simd")}, "simd/archsimd")
			if err != nil {
				t.Fatal(err)
			}
			if packages.PrintErrors(pkgs) != 0 || len(pkgs) != 1 {
				t.Fatal("could not load archsimd")
			}
			prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
			prog.Build()
			count := 0
			for fn := range ssautil.AllFunctions(prog) {
				decl, ok := fn.Syntax().(*ast.FuncDecl)
				if !ok || decl.Body != nil {
					continue // Go helpers retain their ordinary compilation path.
				}
				obj, ok := fn.Object().(*types.Func)
				if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" || fn.Signature.Recv() == nil {
					continue
				}
				if _, ok := llvmgen.SIMDNumericShape(fn.Signature.Recv().Type()); !ok {
					continue
				}
				if fn.Signature.Results().Len() != 1 {
					continue
				}
				if _, ok := llvmgen.SIMDNumericShape(fn.Signature.Results().At(0).Type()); !ok {
					continue
				}
				if simdVectorBits(fn.Signature.Recv().Type()) > 256 || simdVectorBits(fn.Signature.Results().At(0).Type()) > 256 {
					continue // SIMD512 is a separate stage.
				}
				var want llvmgen.SIMDOp
				switch name := obj.Name(); {
				case strings.HasPrefix(name, "ConvertLo"):
					want = llvmgen.SIMDConvertLo
				case strings.HasPrefix(name, "ConvertTo"):
					want = llvmgen.SIMDConvert
				case name == "Ceil":
					want = llvmgen.SIMDCeil
				case name == "Floor":
					want = llvmgen.SIMDFloor
				case name == "Trunc":
					want = llvmgen.SIMDTrunc
				case name == "Round":
					want = llvmgen.SIMDRound
				case strings.HasPrefix(name, "ExtendTo"):
					want = llvmgen.SIMDExtend
				case strings.HasPrefix(name, "TruncTo"):
					want = llvmgen.SIMDTruncate
				case strings.HasPrefix(name, "ExtendLo"):
					want = llvmgen.SIMDExtendLo
				case strings.HasPrefix(name, "ExtendHi"):
					want = llvmgen.SIMDExtendHi
				case strings.HasPrefix(name, "SaturateTo"):
					want = llvmgen.SIMDSaturate
					if strings.HasSuffix(name, "ConcatGrouped") {
						want = llvmgen.SIMDSaturateConcatGrouped
					} else if strings.HasSuffix(name, "Concat") {
						want = llvmgen.SIMDSaturateConcat
					}
				default:
					continue
				}
				desc, ok := lookupSIMD(fn, arch)
				if !ok || desc.op != want {
					t.Errorf("%s: lookup = %v (%t), want %v", obj.FullName(), desc.op, ok, want)
				}
				count++
			}
			if count == 0 {
				t.Fatal("no integer resize declarations checked")
			}
			t.Logf("checked %d declarations", count)
		})
	}
}

func TestSIMDConversionResizeShapes(t *testing.T) {
	const declarations = `package archsimd
 type v128 struct { _ [0]func() }
 type v256 struct { _ [0]func() }
 type Int8x16 struct { tag v128; vals [16]int8 }
 type Uint8x16 struct { tag v128; vals [16]uint8 }
 type Int16x16 struct { tag v256; vals [16]int16 }
 type Int32x4 struct { tag v128; vals [4]int32 }
 type Int32x8 struct { tag v256; vals [8]int32 }
 type Uint32x8 struct { tag v256; vals [8]uint32 }
 type Int64x4 struct { tag v256; vals [4]int64 }
 type Float32x4 struct { tag v128; vals [4]float32 }
 type Float64x2 struct { tag v128; vals [2]float64 }
 type Float64x4 struct { tag v256; vals [4]float64 }
 `
	for _, tc := range []struct {
		recv, method string
		want         llvmgen.SIMDOp
	}{
		{"Int8x16", "ExtendToInt16() Int16x16", llvmgen.SIMDExtend},
		{"Int8x16", "ExtendLo8ToInt16() Int16x16", llvmgen.SIMDUnimplemented},
		{"Int8x16", "ExtendToInt32() Int32x8", llvmgen.SIMDUnimplemented},
		{"Int8x16", "ExtendLo4ToInt64() Int64x4", llvmgen.SIMDExtendLo},
		{"Int32x8", "SaturateToInt16ConcatGrouped(Int32x8) Int16x16", llvmgen.SIMDSaturateConcatGrouped},
		{"Int32x8", "SaturateToInt16Concat(Int32x8) Int16x16", llvmgen.SIMDUnimplemented},
		{"Int32x8", "SaturateToInt16() Int16x16", llvmgen.SIMDUnimplemented},
		{"Int32x8", "TruncToInt8() Int8x16", llvmgen.SIMDTruncate},
		{"Int32x8", "TruncToUint8() Uint8x16", llvmgen.SIMDUnimplemented},
		{"Uint32x8", "SaturateToInt8() Int8x16", llvmgen.SIMDUnimplemented},
		{"Float32x4", "ConvertLo2ToFloat64() Float64x2", llvmgen.SIMDConvertLo},
		{"Int32x4", "ConvertLo2ToFloat64() Float64x2", llvmgen.SIMDConvertLo},
		{"Float32x4", "ConvertLo2ToFloat64() Float64x4", llvmgen.SIMDUnimplemented},
		{"Float32x4", "ConvertToInt64() Int64x4", llvmgen.SIMDConvert},
		{"Float32x4", "ConvertToFloat64() Int64x4", llvmgen.SIMDUnimplemented},
		{"Int32x4", "ConvertToFloat32() Float64x4", llvmgen.SIMDUnimplemented},
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
