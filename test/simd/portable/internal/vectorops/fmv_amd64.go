//go:build goexperiment.simd && amd64

package vectorops

import (
	"simd"
	"simd/archsimd"
)

// FMVBlock exercises arrays nested in aggregates, together with scalar fields.
type FMVBlock struct {
	Values [2]simd.Uint32s
	Count  int
	Label  string
}

//go:noinline
func FMVLeaf(b FMVBlock) (FMVBlock, int) {
	b.Values[0] = b.Values[0].Add(b.Values[1])
	b.Count++
	return b, b.Count
}

//go:noinline
func FMVGuarded(b FMVBlock, mutate func()) (FMVBlock, int) {
	mutate()
	if archsimd.X86.AVX2() {
		b, _ = FMVLeaf(b)
		b.Count += 100
	} else {
		b.Values[0] = b.Values[0].Sub(b.Values[1])
	}
	if archsimd.X86.AVX() {
		b.Count += 10
	}
	if archsimd.X86.FMA() {
		b.Count++
	}
	return b, b.Count
}

//go:noinline
func FMVArray(v [2]simd.Uint32s) [2]simd.Uint32s {
	if archsimd.X86.AVX2() {
		v[0] = v[0].Add(v[1])
	} else {
		v[0] = v[0].Sub(v[1])
	}
	return v
}

func (b FMVBlock) Apply() (FMVBlock, int) { return FMVGuarded(b, func() {}) }

func FMVClosure(b FMVBlock) func() (FMVBlock, int) {
	return func() (FMVBlock, int) { return FMVGuarded(b, func() {}) }
}

func FMVGeneric[T ~int](b FMVBlock, n T) (FMVBlock, T) {
	b, count := FMVGuarded(b, func() {})
	return b, n + T(count)
}
