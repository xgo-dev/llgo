//go:build !llgo

package ssa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDScalarPermuteIR(t *testing.T) {
	// Independent literal masks pin group boundaries and the preserved word
	// half. Dynamic cases must keep lane selection in vector operations.
	for _, tc := range []struct {
		name, elem  string
		bits, lanes int
		op          SIMDOp
		mask        string
	}{
		{"Int32x4", "int32", 128, 4, SIMDPermuteScalars, "<i32 2, i32 3, i32 0, i32 1>"},
		{"Uint32x8", "uint32", 256, 8, SIMDPermuteScalarsGrouped, "<i32 2, i32 3, i32 0, i32 1, i32 6, i32 7, i32 4, i32 5>"},
		{"Int16x8", "int16", 128, 8, SIMDPermuteScalarsHi, "<i32 0, i32 1, i32 2, i32 3, i32 6, i32 7, i32 4, i32 5>"},
		{"Uint16x8", "uint16", 128, 8, SIMDPermuteScalarsLo, "<i32 2, i32 3, i32 0, i32 1, i32 4, i32 5, i32 6, i32 7>"},
		{"Int16x16", "int16", 256, 16, SIMDPermuteScalarsHiGrouped, "<i32 0, i32 1, i32 2, i32 3, i32 6, i32 7, i32 4, i32 5, i32 8, i32 9, i32 10, i32 11, i32 14, i32 15, i32 12, i32 13>"},
		{"Uint16x16", "uint16", 256, 16, SIMDPermuteScalarsLoGrouped, "<i32 2, i32 3, i32 0, i32 1, i32 4, i32 5, i32 6, i32 7, i32 10, i32 11, i32 8, i32 9, i32 12, i32 13, i32 14, i32 15>"},
	} {
		for _, dynamic := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dynamic=%v", tc.name, dynamic), func(t *testing.T) {
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
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", raw), types.NewParam(token.NoPos, nil, "selector", types.Typ[types.Uint8])), types.NewTuple(types.NewParam(token.NoPos, nil, "", raw)), false)
				fn := pkg.NewFunc("p.permute", sig, GoDecl)
				b := fn.MakeBody(1)
				defer b.Dispose()
				selector := fn.Param(1)
				if !dynamic {
					selector = prog.IntVal(0x4e, selector.Type)
				}
				b.Return(b.SIMD(tc.op, fn.Param(0).Type, fn.Param(0), selector))
				b.EndBuild()
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				body := fn.impl.String()
				if strings.Contains(body, "extractelement") || strings.Contains(body, "PanicSIMD") {
					t.Fatalf("unexpected scalar extraction or panic:\n%s", body)
				}
				if dynamic {
					if !strings.Contains(body, fmt.Sprintf("select <%d x i1>", tc.lanes)) {
						t.Fatalf("missing vector selection:\n%s", body)
					}
				} else if !strings.Contains(body, tc.mask) || strings.Count(body, "shufflevector") != 1 || strings.Contains(body, "select ") {
					t.Fatalf("expected single constant shuffle %s:\n%s", tc.mask, body)
				}
			})
		}
	}
}
