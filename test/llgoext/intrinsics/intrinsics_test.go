//go:build llgo

package intrinsics

import (
	"testing"
	_ "unsafe"
)

//go:linkname population llvm.ctpop.i32
func population(uint32) uint32

//llgo:link leading llvm.ctlz
func leading(uint32, bool) uint32 { panic("intrinsic body must be ignored") }

//llgo:link root llvm.sqrt
func root(float64) float64 { panic("intrinsic body must be ignored") }

//llgo:link multiply llvm.umul.with.overflow
func multiply(uint64, uint64) (uint64, bool) { panic("intrinsic body must be ignored") }

//go:linkname legacyMultiply llgo.umulOverflow
func legacyMultiply(uint64, uint64) (uint64, bool)

func multiplyWrapper(a, b uint64) (uint64, bool) { return multiply(a, b) }

//go:noinline
func checkMultiply(t *testing.T, mul func(uint64, uint64) (uint64, bool)) {
	t.Helper()
	max := ^uint64(0)
	for _, a := range []uint64{0, 1, 3, 1 << 32, 1 << 63, max} {
		for _, b := range []uint64{0, 1, 2, 1 << 32, max} {
			wantOverflow := a != 0 && b > max/a
			if product, overflow := mul(a, b); product != a*b || overflow != wantOverflow {
				t.Fatalf("multiply(%d, %d) = (%d, %v), want (%d, %v)", a, b, product, overflow, a*b, wantOverflow)
			}
		}
	}
}

func TestScalarBindings(t *testing.T) {
	if got := population(0xf010); got != 5 {
		t.Fatalf("population = %d", got)
	}
	if got := leading(0, false); got != 32 {
		t.Fatalf("leading(0) = %d", got)
	}
	if got := leading(1, false); got != 31 {
		t.Fatalf("leading(1) = %d", got)
	}
	if got := root(81); got != 9 {
		t.Fatalf("root(81) = %g", got)
	}
}

func TestOverflowAndLegacyCompatibility(t *testing.T) {
	checkMultiply(t, multiplyWrapper)
	// Existing instructions continue to provide their function-value wrappers.
	checkMultiply(t, legacyMultiply)
}
