//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMDX86MissingInstructionFeature(t *testing.T) {
	if archsimd.X86.AVX512() {
		t.Skip("requires AVX512 disabled or unavailable")
	}
	defer func() {
		value := recover()
		err, ok := value.(runtime.Error)
		if !ok || err.Error() != "runtime error: SIMD instruction requires an unavailable x86 CPU feature" {
			t.Fatalf("unexpected panic: %T %v", value, value)
		}
	}()
	var output [4]float32
	archsimd.LoadFloat32x4Array(&output).RoundScaled(2).StoreArray(&output)
	t.Error("unsupported instruction returned")
}
