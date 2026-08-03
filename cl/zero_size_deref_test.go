//go:build !llgo
// +build !llgo

package cl

import (
	"strings"
	"testing"
)

func TestZeroSizedFieldDerefEmitsBaseNilGuard(t *testing.T) {
	const src = `package zeroderef
type T struct {
	n int
	z struct{}
}
func Eq(p, q *T) bool {
	return p.z == q.z
}
`
	ir := compileWithRewrites(t, src, nil)
	if got := strings.Count(ir, "AssertNilDeref"); got < 4 {
		t.Fatalf("zero-sized field comparison should guard field bases and loads, got %d guards:\n%s", got, ir)
	}
}

func TestUnusedDerefEmitsNilGuard(t *testing.T) {
	const src = `package unusedderef
func LoadArrayElement() {
	var values [2]*int
	_ = *values[1]
}
func LoadPointer(p *int) {
	_ = *p
}
`
	ir := compileWithRewrites(t, src, nil)
	arrayLoad := llvmFunction(t, ir, "unusedderef.LoadArrayElement")
	if !strings.Contains(arrayLoad, "AssertNilDeref") {
		t.Fatalf("unused array-element dereference should retain a nil guard:\n%s", arrayLoad)
	}
	directLoad := llvmFunction(t, ir, "unusedderef.LoadPointer")
	if !strings.Contains(directLoad, "AssertNilDeref") {
		t.Fatalf("unused direct dereference should retain a nil guard:\n%s", directLoad)
	}
}

func TestUnusedDerefAvoidsRedundantNilGuards(t *testing.T) {
	const src = `package unusedderef
func LocalAlloc() {
	p := new(int)
	_ = *p
}
//go:noinline
func ArrayPointer() *[2]int { return nil }
func EffectfulArray() {
	_ = *ArrayPointer()
}
`
	ir := compileWithRewrites(t, src, nil)
	localAlloc := llvmFunction(t, ir, "unusedderef.LocalAlloc")
	if got := strings.Count(localAlloc, "AssertNilDeref"); got != 0 {
		t.Fatalf("known-non-nil local allocation emitted %d nil guards, want 0:\n%s", got, localAlloc)
	}
	effectfulArray := llvmFunction(t, ir, "unusedderef.EffectfulArray")
	if got := strings.Count(effectfulArray, "AssertNilDeref"); got != 1 {
		t.Fatalf("effectful array-pointer dereference emitted %d nil guards, want 1:\n%s", got, effectfulArray)
	}
}
