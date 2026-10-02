//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

//go:noinline
func vectorDirectRecover(x archsimd.Float32x4, fail bool) (result archsimd.Float32x4) {
	live := x.Add(archsimd.BroadcastFloat32x4(3))
	defer func() {
		if r := recover(); r != nil {
			if r != "vector call" {
				panic(r)
			}
			result = live
		}
	}()
	return vectorcall.Echo(x, fail)
}

//go:noinline
func vectorIndirectRecover(f func(archsimd.Float32x4, bool) archsimd.Float32x4, x archsimd.Float32x4, fail bool) (result archsimd.Float32x4) {
	live := x.Add(archsimd.BroadcastFloat32x4(4))
	defer func() {
		if r := recover(); r != nil {
			if r != "vector call" {
				panic(r)
			}
			result = live
		}
	}()
	return f(x, fail)
}

func TestSIMDRecoverAcrossPackageCalls(t *testing.T) {
	x := archsimd.BroadcastFloat32x4(10)
	for _, tc := range []struct {
		value archsimd.Float32x4
		want  float32
	}{
		{vectorDirectRecover(x, false), 12},
		{vectorDirectRecover(x, true), 13},
		{vectorIndirectRecover(vectorcall.Echo, x, false), 12},
		{vectorIndirectRecover(vectorcall.Echo, x, true), 14},
	} {
		var got [4]float32
		tc.value.StoreArray(&got)
		for _, v := range got {
			if v != tc.want {
				t.Fatalf("got %v want %g", got, tc.want)
			}
		}
	}
}

func TestSIMDLiveAcrossScheduling(t *testing.T) {
	x := archsimd.BroadcastFloat32x4(7)
	done := make(chan archsimd.Float32x4, 1)
	go func(v archsimd.Float32x4) {
		runtime.Gosched()
		done <- vectorcall.Echo(v, false)
	}(x)
	runtime.Gosched()
	got := (<-done).Add(x)
	var lanes [4]float32
	got.StoreArray(&lanes)
	for _, v := range lanes {
		if v != 16 {
			t.Fatalf("vector across scheduling: %v", lanes)
		}
	}
}
