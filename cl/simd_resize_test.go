package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDIntegerResizeSignatures(t *testing.T) {
	const declarations = `package archsimd
 type v128 struct { _ [0]func() }
 type Int8x16 struct { tag v128; vals [16]int8 }
 type Uint8x16 struct { tag v128; vals [16]uint8 }
 type Int16x8 struct { tag v128; vals [8]int16 }
 type Uint16x8 struct { tag v128; vals [8]uint16 }
 type Int32x4 struct { tag v128; vals [4]int32 }
 type Uint32x4 struct { tag v128; vals [4]uint32 }
 type Int64x2 struct { tag v128; vals [2]int64 }
 type Uint64x2 struct { tag v128; vals [2]uint64 }
 type Float32x4 struct { tag v128; vals [4]float32 }
 type Mask16x8 struct { tag v128; vals [8]int16 }
 type Uint16x16 struct { tag v128; vals [16]uint16 }
 `
	for _, tc := range []struct {
		name, recv, method string
		want               llssa.SIMDOp
	}{
		{"zero extend", "Uint8x16", "ExtendLo8ToUint16() Uint16x8", llssa.SIMDExtendLo},
		{"sign extend", "Int8x16", "ExtendLo2ToInt64() Int64x2", llssa.SIMDExtendLo},
		{"high half", "Int16x8", "ExtendHi4ToInt32() Int32x4", llssa.SIMDExtendHi},
		{"signed saturation", "Int32x4", "SaturateToInt8() Int8x16", llssa.SIMDSaturate},
		{"unsigned saturation", "Uint64x2", "SaturateToUint8() Uint8x16", llssa.SIMDSaturate},
		{"signed to unsigned", "Int16x8", "SaturateToUint8() Uint8x16", llssa.SIMDSaturate},
		{"concat", "Int32x4", "SaturateToUint16Concat(Int32x4) Uint16x8", llssa.SIMDSaturateConcat},
		{"missing result", "Int8x16", "ExtendLo8ToInt16()", llssa.SIMDUnimplemented},
		{"scalar result", "Int8x16", "ExtendLo8ToInt16() int16", llssa.SIMDUnimplemented},
		{"multiple results", "Int8x16", "ExtendLo8ToInt16() (Int16x8, int)", llssa.SIMDUnimplemented},
		{"mask result", "Int8x16", "ExtendLo8ToInt16() Mask16x8", llssa.SIMDUnimplemented},
		{"floating result", "Int8x16", "ExtendLo4ToInt32() Float32x4", llssa.SIMDUnimplemented},
		{"floating input", "Float32x4", "SaturateToInt16() Int16x8", llssa.SIMDUnimplemented},
		{"unchanged width", "Int16x8", "ExtendLo8ToInt16() Int16x8", llssa.SIMDUnimplemented},
		{"narrow extension", "Int32x4", "ExtendLo8ToInt16() Int16x8", llssa.SIMDUnimplemented},
		{"wrong signedness", "Int8x16", "ExtendLo8ToUint16() Uint16x8", llssa.SIMDUnimplemented},
		{"wrong lane spelling", "Int8x16", "ExtendLo4ToInt32() Int64x2", llssa.SIMDUnimplemented},
		{"high quarter", "Int8x16", "ExtendHi2ToInt64() Int64x2", llssa.SIMDUnimplemented},
		{"extra argument", "Int8x16", "ExtendLo8ToInt16(int) Int16x8", llssa.SIMDUnimplemented},
		{"variadic", "Int8x16", "ExtendLo8ToInt16(...int) Int16x8", llssa.SIMDUnimplemented},
		{"wide result", "Uint8x16", "ExtendLo8ToUint16() Uint16x16", llssa.SIMDUnimplemented},
		{"wrong narrowing spelling", "Int32x4", "SaturateToUint8() Int8x16", llssa.SIMDUnimplemented},
		{"unsigned to signed", "Uint16x8", "SaturateToInt8() Int8x16", llssa.SIMDUnimplemented},
		{"widening saturation", "Int8x16", "SaturateToInt16() Int16x8", llssa.SIMDUnimplemented},
		{"unchanged saturation", "Int16x8", "SaturateToInt16() Int16x8", llssa.SIMDUnimplemented},
		{"unary argument", "Int16x8", "SaturateToInt8(Int16x8) Int8x16", llssa.SIMDUnimplemented},
		{"concat missing operand", "Int32x4", "SaturateToInt16Concat() Int16x8", llssa.SIMDUnimplemented},
		{"concat wrong operand", "Int32x4", "SaturateToInt16Concat(Uint32x4) Int16x8", llssa.SIMDUnimplemented},
		{"concat wrong width", "Int64x2", "SaturateToInt16Concat(Int64x2) Int16x8", llssa.SIMDUnimplemented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "resize.go", declarations+"func (x "+tc.recv+") "+tc.method, 0)
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
			if wantOK := tc.want != llssa.SIMDUnimplemented; ok != wantOK || ok && desc.op != tc.want {
				t.Fatalf("lookup = %v (%t), want %v", desc.op, ok, tc.want)
			}
		})
	}
}

// Verify the complete 128-bit declaration family against the selected Go
// toolchain, including architecture-specific methods and standalone bodies.
func TestSIMDIntegerResizeDeclarations(t *testing.T) {
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
				if _, ok := llssa.SIMDNumericShape(fn.Signature.Recv().Type()); !ok {
					continue
				}
				if fn.Signature.Results().Len() != 1 {
					continue
				}
				if _, ok := llssa.SIMDNumericShape(fn.Signature.Results().At(0).Type()); !ok {
					continue
				}
				var want llssa.SIMDOp
				switch name := obj.Name(); {
				case strings.HasPrefix(name, "ExtendLo"):
					want = llssa.SIMDExtendLo
				case strings.HasPrefix(name, "ExtendHi"):
					want = llssa.SIMDExtendHi
				case strings.HasPrefix(name, "SaturateTo"):
					want = llssa.SIMDSaturate
					if strings.HasSuffix(name, "Concat") {
						want = llssa.SIMDSaturateConcat
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
