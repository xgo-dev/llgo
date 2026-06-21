//go:build !llgo
// +build !llgo

package cl

import (
	"go/token"
	"go/types"
	"strings"
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

func TestZeroLengthSliceToArrayConversionKeepsNilCheck(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

func convert(p *[]byte) {
	_ = [0]byte(*p)
}
`)

	ir := mustNamedFunction(t, m, "foo.convert").String()
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("zero-length slice-to-array conversion should keep operand nil check:\n%s", ir)
	}
}

func TestIsInterfaceCompareDeref(t *testing.T) {
	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

func compareInterfacePtr(p *interface{}, q interface{}) bool {
	return *p == q
}

func derefOnly(p *interface{}) interface{} {
	return *p
}
`)

	if !isInterfaceCompareDeref(findUnOp(t, ssaPkg.Func("compareInterfacePtr"), token.MUL, false)) {
		t.Fatal("interface deref used by comparison should be detected")
	}
	derefOnly := findUnOp(t, ssaPkg.Func("derefOnly"), token.MUL, false)
	if isInterfaceCompareDeref(derefOnly) {
		t.Fatal("interface deref without comparison referrer should not be detected")
	}
	refs := derefOnly.Referrers()
	if refs == nil {
		t.Fatal("derefOnly has no referrer slice")
	}
	oldRefs := *refs
	*refs = nil
	defer func() { *refs = oldRefs }()
	if isInterfaceCompareDeref(derefOnly) {
		t.Fatal("interface deref without referrers should not be detected")
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
