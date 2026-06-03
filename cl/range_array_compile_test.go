//go:build !llgo
// +build !llgo

package cl

import (
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestSkipUnusedArrayDeref(t *testing.T) {
	if skipUnusedArrayDeref(&ssa.UnOp{Op: token.SUB}) {
		t.Fatal("non-deref unop should not be skipped")
	}

	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

var sink int

func rangeArray(p *[3]int) {
	for i := range *p {
		sink += i
	}
}

func copyArray(p *[3]int) [3]int {
	return *p
}

func useNonArray(p *int) int {
	return *p
}
`)

	if !skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("rangeArray"), token.MUL, true)) {
		t.Fatal("range array deref should be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("copyArray"), token.MUL, true)) {
		t.Fatal("referenced array deref should not be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("useNonArray"), token.MUL, false)) {
		t.Fatal("non-array deref should not be skipped")
	}
}

func findUnOp(t *testing.T, fn *ssa.Function, op token.Token, wantArray bool) *ssa.UnOp {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			unop, ok := instr.(*ssa.UnOp)
			if !ok || unop.Op != op {
				continue
			}
			_, isArray := unop.Type().Underlying().(*types.Array)
			if isArray == wantArray {
				return unop
			}
		}
	}
	t.Fatalf("missing %s unop in %s", op, fn.Name())
	return nil
}
