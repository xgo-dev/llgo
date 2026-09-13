//go:build goexperiment.simd

package main

import "simd/archsimd"

type snapshot struct{ pmull, avx, avx2, avx512 bool }

func read() snapshot {
	return snapshot{archsimd.ARM64.PMULL(), archsimd.X86.AVX(), archsimd.X86.AVX2(), archsimd.X86.AVX512()}
}

var atGlobal = read()
var atInit snapshot

func init() { atInit = read() }
func show(phase string, s snapshot) {
	println(phase, "PMULL", s.pmull, "AVX", s.avx, "AVX2", s.avx2, "AVX512", s.avx512)
}
func main() {
	show("global", atGlobal)
	show("init", atInit)
	show("main", read())
}
