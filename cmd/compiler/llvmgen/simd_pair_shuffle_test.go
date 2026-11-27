//go:build !llgo

package llvmgen

import (
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDPairShuffleConstantMasks(t *testing.T) {
	// Pin the lane orders independently of the implementation, especially the
	// per-group selector bits of 64-bit lanes and cross-128-bit half selection.
	for _, tc := range []struct {
		name, elem  string
		bits, lanes int
		op          SIMDOp
		lo, hi      uint64
		mask        string
	}{
		{"Float32x4", "float32", 128, 4, SIMDConcatSelected, 0xd2, 0, "<i32 2, i32 0, i32 5, i32 7>"},
		{"Int32x8", "int32", 256, 8, SIMDConcatSelectedGrouped, 0xd2, 0, "<i32 2, i32 0, i32 9, i32 11, i32 6, i32 4, i32 13, i32 15>"},
		{"Float64x2", "float64", 128, 2, SIMDConcatSelected, 0xfe, 0, "<i32 0, i32 3>"},
		{"Uint64x4", "uint64", 256, 4, SIMDConcatSelectedGrouped, 0xf6, 0, "<i32 0, i32 5, i32 3, i32 6>"},
		{"Float32x8", "float32", 256, 8, SIMDConcatPermute128, 3, 0, "<i32 12, i32 13, i32 14, i32 15, i32 0, i32 1, i32 2, i32 3>"},
		{"Uint64x4", "uint64", 256, 4, SIMDConcatPermute128, 1, 2, "<i32 2, i32 3, i32 4, i32 5>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			source := fmt.Sprintf("package archsimd; type v%d struct {_ [0]func()}; type %s struct {tag v%d; vals [%d]%s}", tc.bits, tc.name, tc.bits, tc.lanes, tc.elem)
			file, err := parser.ParseFile(fs, "types.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			rawPkg, err := new(types.Config).Check("simd/archsimd", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			raw := rawPkg.Scope().Lookup(tc.name).Type()
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "p")
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", raw), types.NewParam(token.NoPos, nil, "y", raw)), types.NewTuple(types.NewParam(token.NoPos, nil, "", raw)), false)
			fn := pkg.NewFunc("p.shuffle", sig, GoDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			selector := prog.IntVal(tc.lo, prog.Type(types.Typ[types.Uint8], GoDecl))
			args := []Expr{fn.Param(0), selector}
			if tc.op == SIMDConcatPermute128 {
				args = append(args, prog.IntVal(tc.hi, selector.Type))
			}
			args = append(args, fn.Param(1))
			b.Return(b.SIMD(tc.op, fn.Param(0).Type, args...))
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := fn.impl.String()
			if !strings.Contains(body, tc.mask) || strings.Contains(body, "PanicSIMDImmediate") {
				t.Fatalf("expected constant shuffle mask %s without a panic check:\n%s", tc.mask, body)
			}
		})
	}
}
