//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

//go:noinline
func simd256Guard(x, y archsimd.Float32x8) archsimd.Float32x8 {
	if archsimd.X86.AVX2() {
		return vectorcall.WideCompute(x, y)
	}
	return x.Sub(y)
}

var simd256Entry = simd256Guard
var simd256Initial = func() archsimd.Float32x8 {
	if runtime.Compiler == "llgo" || archsimd.X86.AVX2() {
		return simd256Guard(archsimd.BroadcastFloat32x8(2), archsimd.BroadcastFloat32x8(3))
	}
	return archsimd.Float32x8{}
}()

//go:noinline
func simd256Indirect(f func(archsimd.Float32x8, archsimd.Float32x8) archsimd.Float32x8, x, y archsimd.Float32x8) archsimd.Float32x8 {
	return f(x, y)
}

func TestSIMD256Calls(t *testing.T) {
	requireSIMD256(t)
	var a, b [8]float32
	for i := range a {
		a[i] = float32(i) + 2
		b[i] = float32(i)*3 + 1
	}
	x, y := archsimd.LoadFloat32x8Array(&a), archsimd.LoadFloat32x8Array(&b)
	initial := float32(-1)
	if archsimd.X86.AVX2() {
		initial = 8
	}
	checkSIMD256Lanes(t, simd256Initial, func(int) float32 { return initial })
	for _, fn := range []func(archsimd.Float32x8, archsimd.Float32x8) archsimd.Float32x8{simd256Guard, simd256Entry} {
		checkSIMD256Lanes(t, simd256Indirect(fn, x, y), func(i int) float32 {
			if archsimd.X86.AVX2() {
				return a[i]*b[i] + a[i]
			}
			return a[i] - b[i]
		})
	}
	checkSIMD256Lanes(t, simd256Indirect(archsimd.Float32x8.Sub, x, y), func(i int) float32 { return a[i] - b[i] })
	method := x.Sub
	x = archsimd.BroadcastFloat32x8(99)
	checkSIMD256Lanes(t, method(y), func(i int) float32 { return a[i] - b[i] })
	x = archsimd.LoadFloat32x8Array(&a)
	first, second, n := vectorcall.WidePair(x, y, func() { x = archsimd.BroadcastFloat32x8(100) })
	if n != 73 {
		t.Fatal("scalar tuple result")
	}
	checkSIMD256Lanes(t, first, func(i int) float32 { return a[i] - b[i] })
	checkSIMD256Lanes(t, second, func(i int) float32 { return a[i] + b[i] })
	x = archsimd.LoadFloat32x8Array(&a)
	x, second, n = vectorcall.WidePair(x, y, func() {})
	checkSIMD256Lanes(t, x, func(i int) float32 { return a[i] - b[i] })
	checkSIMD256Lanes(t, second, func(i int) float32 { return a[i] + b[i] })
	block := vectorcall.WideBlock{Prefix: 17, Values: [2]archsimd.Float32x8{archsimd.LoadFloat32x8Array(&a), y}, Count: 4}
	updated := vectorcall.WideUpdate(block, func() { block.Count = 99; block.Values[0] = archsimd.BroadcastFloat32x8(101) })
	if updated.Count != 5 || updated.Prefix != 17 || block.Count != 99 {
		t.Fatal("aggregate snapshot")
	}
	checkSIMD256Lanes(t, updated.Values[0], func(i int) float32 { return a[i]*b[i] + a[i] })
	checkSIMD256Lanes(t, updated.Values[1], func(i int) float32 { return b[i] })
	if unsafe.Offsetof(block.Values) != 8 || unsafe.Sizeof(block) != 80 || unsafe.Alignof(block) != 8 {
		t.Fatal("wide vector changed Go storage layout")
	}
	values := [10]archsimd.Float32x8{}
	for j := range values {
		var v [8]float32
		for i := range v {
			v[i] = float32((j+1)*(j+1) + i*j)
		}
		values[j] = archsimd.LoadFloat32x8Array(&v)
	}
	many := vectorcall.WideMany(values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7], values[8], values[9])
	checkSIMD256Lanes(t, many, func(i int) float32 {
		var sum float32
		for j := 0; j < 10; j++ {
			v := float32((j+1)*(j+1) + i*j)
			if j%2 == 0 {
				sum += v
			} else {
				sum -= v
			}
		}
		return sum
	})
}

func TestSIMD256ClosuresAndScheduling(t *testing.T) {
	requireSIMD256(t)
	x, y := archsimd.BroadcastFloat32x8(9), archsimd.BroadcastFloat32x8(2)
	t.Run("closure", func(t *testing.T) {
		closure := func(v archsimd.Float32x8) archsimd.Float32x8 { runtime.Gosched(); return x.Sub(v) }
		checkSIMD256Lanes(t, closure(y), func(int) float32 { return 7 })
	})
	t.Run("interface", func(t *testing.T) {
		var boxed any = x
		checkSIMD256Lanes(t, boxed.(archsimd.Float32x8), func(int) float32 { return 9 })
	})
	t.Run("goroutine", func(t *testing.T) {
		done := make(chan archsimd.Float32x8, 1)
		go func(f func(archsimd.Float32x8) archsimd.Float32x8) { runtime.Gosched(); done <- f(y) }(x.Sub)
		checkSIMD256Lanes(t, <-done, func(int) float32 { return 7 })
	})
}

func checkSIMD256Lanes(t *testing.T, v archsimd.Float32x8, want func(int) float32) {
	t.Helper()
	var lanes [8]float32
	v.StoreArray(&lanes)
	for i, got := range lanes {
		if got != want(i) {
			t.Fatalf("lane %d: got %g want %g", i, got, want(i))
		}
	}
}
