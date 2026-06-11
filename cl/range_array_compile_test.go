//go:build !llgo
// +build !llgo

package cl

import (
	"strings"
	"testing"
)

func TestUnusedDerefNilChecks(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

var sink int

func rangeArray(p *[3]int) {
	for i := range *p {
		sink += i
	}
}

func addressOfDeref(p *int) *int {
	return &*p
}
`)

	rangeIR := mustNamedFunction(t, m, "foo.rangeArray").String()
	if strings.Contains(rangeIR, "AssertNilDeref") {
		t.Fatalf("range over array pointer should not emit nil-deref guard, got:\n%s", rangeIR)
	}

	ir := mustNamedFunction(t, m, "foo.addressOfDeref").String()
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("address-of deref should emit nil-deref guard, got:\n%s", ir)
	}
	if strings.Contains(ir, "load i64, ptr %0") {
		t.Fatalf("address-of deref should not load the pointee, got:\n%s", ir)
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
