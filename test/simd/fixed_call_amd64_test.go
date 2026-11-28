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
		storeFixedSIMD512(dst, *src)
	}
}

func TestSIMD512FixedRegisterCall(t *testing.T) {
	// This path must work even on a machine with no ZMM register support.
	conditionalFixedSIMD512(false, nil, nil)
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
