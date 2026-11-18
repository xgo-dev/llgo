package main

import "simd/archsimd"

func add256(x, y archsimd.Uint32x8) archsimd.Uint32x8 { return x.Add(y) }
