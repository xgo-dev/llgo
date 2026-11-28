//go:build goexperiment.simd && amd64

package simd_test

import (
	"os"
	"simd/archsimd"
	"testing"
	"unsafe"
)

//go:noinline
func storeFixedSIMD512(dst *archsimd.Int32x16, x archsimd.Int32x16) {
	*dst = x
}

//go:noinline
func conditionalFixedSIMD512(take bool, dst, src *archsimd.Int32x16) {
	if take {
		if src == nil {
			panic("missing fixed-call input")
		}
		storeFixedSIMD512(dst, *src)
	}
}

func TestSIMD512FixedRegisterCall(t *testing.T) {
	// This path must work even on a machine with no ZMM register support.
	conditionalFixedSIMD512(false, nil, nil)
	t.Run("panic before the transfer", func(t *testing.T) {
		defer func() {
			if got := recover(); got != "missing fixed-call input" {
				t.Fatalf("recovered %v", got)
			}
		}()
		conditionalFixedSIMD512(true, nil, nil)
	})
	t.Run("taken", func(t *testing.T) {
		available := archsimd.X86.AVX512()
		if os.Getenv("LLGO_TEST_MASKED_ZMM") == "1" {
			// The runner has established hardware and OS support independently.
			if available {
				t.Fatal("expected the Go AVX512 query to be disabled")
			}
		} else if !available {
			t.Skip("requires ZMM register support")
		}

		// Initialize and inspect all lanes without invoking an archsimd operation
		// that may have its own effective-feature requirement.
		var src, dst archsimd.Int32x16
		in := (*[16]int32)(unsafe.Pointer(&src))
		for i := range in {
			in[i] = int32(7*i + 1)
		}
		conditionalFixedSIMD512(true, &dst, &src)
		if got := *(*[16]int32)(unsafe.Pointer(&dst)); got != *in {
			t.Fatalf("transferred lanes = %v, want %v", got, *in)
		}
	})
}

type fixedCombiner interface {
	combine(archsimd.Int32x8, archsimd.Int32x8) (archsimd.Int32x8, int32)
}

type fixedBias int32

//go:noinline
func (bias fixedBias) combine(x, y archsimd.Int32x8) (archsimd.Int32x8, int32) {
	// Check the transfers even when Go's effective AVX predicates are off.
	a := (*[8]int32)(unsafe.Pointer(&x))
	b := (*[8]int32)(unsafe.Pointer(&y))
	for i := range a {
		a[i] += b[i] + int32(bias)
	}
	return x, int32(bias)
}

type fixedSubtract struct{ n fixedBias }

func (s *fixedSubtract) combine(x, y archsimd.Int32x8) (archsimd.Int32x8, int32) {
	return (-s.n).combine(x, y)
}

//go:noinline
func fixedFuncCall(dst, x, y *archsimd.Int32x8, fn func(archsimd.Int32x8, archsimd.Int32x8) archsimd.Int32x8) {
	*dst = fn(*x, *y)
}

//go:noinline
func fixedInterfaceCall(dst, x, y *archsimd.Int32x8, op fixedCombiner) (tag int32) {
	*dst, tag = op.combine(*x, *y)
	return
}

func TestSIMD256FixedRegisterIndirect(t *testing.T) {
	if os.Getenv("LLGO_TEST_MASKED_YMM") == "1" {
		if archsimd.X86.AVX() {
			t.Fatal("masked test requires the AVX query to be false")
		}
	} else if !archsimd.X86.AVX() {
		t.Skip("requires YMM register support")
	}
	var x, y, dst archsimd.Int32x8
	a := (*[8]int32)(unsafe.Pointer(&x))
	b := (*[8]int32)(unsafe.Pointer(&y))
	for i := range a {
		a[i], b[i] = int32(3*i+1), int32(11*i-91)
	}
	check := func(want int32) {
		t.Helper()
		for i, v := range (*[8]int32)(unsafe.Pointer(&dst)) {
			if w := a[i] + b[i] + want; v != w {
				t.Fatalf("lane %d = %d, want %d", i, v, w)
			}
		}
	}
	for _, tc := range []struct {
		bias fixedBias
		op   fixedCombiner
	}{{17, fixedBias(17)}, {-3, &fixedSubtract{3}}} {
		bias := tc.bias
		fixedFuncCall(&dst, &x, &y, func(x, y archsimd.Int32x8) archsimd.Int32x8 {
			result, _ := bias.combine(x, y)
			return result
		})
		check(int32(bias))
		if tag := fixedInterfaceCall(&dst, &x, &y, tc.op); tag != int32(bias) {
			t.Fatalf("scalar result = %d, want %d", tag, bias)
		}
		check(int32(bias))
	}
}
