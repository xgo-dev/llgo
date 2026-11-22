//go:build goexperiment.simd && amd64

package regkernel

import "simd/archsimd"

//go:noinline
func Add512(x, y archsimd.Uint32x16) archsimd.Uint32x16 { return x.Add(y) }

//go:noinline
func Add256(x, y archsimd.Uint32x8) archsimd.Uint32x8 { return x.Add(y) }
