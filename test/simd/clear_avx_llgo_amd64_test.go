//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// Exercise the callable entry even when CPU overrides disable AVX.
func TestSIMDClearAVXUpperBits(t *testing.T) {
	clear := archsimd.ClearAVXUpperBits
	invokeClearAVX(clear)
}

//go:noinline
func invokeClearAVX(clear func()) { clear() }
