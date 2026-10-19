//go:build goexperiment.simd && amd64

package vectorcall

import "simd/archsimd"

type WideBlock struct {
	Prefix byte
	Values [2]archsimd.Float32x8
	Count  int
}

//go:noinline
func WidePair(x, y archsimd.Float32x8, mutate func()) (archsimd.Float32x8, archsimd.Float32x8, int) {
	mutate()
	return x.Sub(y), x.Add(y), 73
}

//go:noinline
func WideCompute(x, y archsimd.Float32x8) archsimd.Float32x8 { return x.Mul(y).Add(x) }

//go:noinline
func WideUpdate(x WideBlock, mutate func()) WideBlock {
	mutate()
	x.Values[0] = WideCompute(x.Values[0], x.Values[1])
	x.Count++
	return x
}

//go:noinline
func WideMany(a, b, c, d, e, f, g, h, i, j archsimd.Float32x8) archsimd.Float32x8 {
	return a.Sub(b).Add(c).Sub(d).Add(e).Sub(f).Add(g).Sub(h).Add(i).Sub(j)
}
