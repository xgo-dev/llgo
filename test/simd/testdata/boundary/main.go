//go:build goexperiment.simd && (amd64 || arm64 || wasm)

// A small executable also exercises O0 Wasm without the standard testing
// package, whose unoptimized functions exceed engine local-variable limits.
package main

import (
	"simd/archsimd"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

var initial = archsimd.BroadcastFloat32x4(10)

//go:noinline
func recoverVector(f func(archsimd.Float32x4, bool) archsimd.Float32x4, x archsimd.Float32x4, fail bool) (out archsimd.Float32x4) {
	live := x.Add(archsimd.BroadcastFloat32x4(3))
	defer func() {
		if r := recover(); r != nil {
			if r != "vector call" {
				panic(r)
			}
			out = live
		}
	}()
	return f(x, fail)
}

//go:noinline
func check(x archsimd.Float32x4, want float32) {
	var lanes [4]float32
	x.StoreArray(&lanes)
	for _, value := range lanes {
		if value != want {
			panic("vector boundary mismatch")
		}
	}
}

func scalarRecover(fail bool) (out float32) {
	defer func() {
		if r := recover(); r != nil {
			if r != "vector call" {
				panic(r)
			}
			out = 42
		}
	}()
	return vectorcall.Scalar(10, fail)
}

func main() {
	if scalarRecover(false) != 12 || scalarRecover(true) != 42 {
		panic("late inline boundary mismatch")
	}
	check(recoverVector(vectorcall.Echo, initial, false), 12)
	check(recoverVector(vectorcall.Echo, initial, true), 13)
	done := make(chan archsimd.Float32x4)
	go func(x archsimd.Float32x4) { done <- vectorcall.Echo(x, false) }(initial)
	check((<-done).Add(initial), 22)
	println("SIMD call boundaries PASS")
}
