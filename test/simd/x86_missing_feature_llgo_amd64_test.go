//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"fmt"
	"simd/archsimd"
	"testing"
)

func TestSIMDMissingInstructionFeature(t *testing.T) {
	if archsimd.X86.AVX512() {
		t.Skip("requires AVX512 disabled or unavailable")
	}
	defer func() {
		if got := fmt.Sprint(recover()); got != "SIMD instruction requires an unavailable x86 CPU feature" {
			t.Fatalf("unexpected panic: %s", got)
		}
	}()
	var output [4]float32
	archsimd.LoadFloat32x4Array(&output).RoundScaled(2).StoreArray(&output)
	t.Error("unsupported instruction returned")
}
