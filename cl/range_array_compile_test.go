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
	var calls int

	func rangeArray(p *[3]int) {
		for i := range *p {
			sink += i
		}
	}

	func nextArray() *[3]int {
		calls++
		return nil
	}

	func rangeArrayCall() {
		for i := range *nextArray() {
			sink += i
		}
	}

	func explicitDiscard(p *[3]int) {
		_ = *p
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
	if !skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("rangeArrayCall"), token.MUL, true)) {
		t.Fatal("range array call deref should be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("explicitDiscard"), token.MUL, true)) {
		t.Fatal("explicit array deref discard should not be skipped")
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

func TestRangeArrayPointerCallEvaluatesWithoutNilCheck(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

var sink int

func nextArray() *[3]int {
	return nil
}

func rangeArrayCall() {
	for i := range *nextArray() {
		sink += i
	}
}
`)

	ir := mustNamedFunction(t, m, "foo.rangeArrayCall").String()
	if !strings.Contains(ir, "foo.nextArray") {
		t.Fatalf("range over call operand should still evaluate the call:\n%s", ir)
	}
	if strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("range over nil *array call should not nil-check the array pointer:\n%s", ir)
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
