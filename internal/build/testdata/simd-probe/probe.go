//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simdprobe

import (
	"simd/archsimd"
	"unsafe"
)

type Vec128 = archsimd.Float32x4
type Array128 [4]float32
type Nested128 struct {
	Prefix byte
	Value  Vec128
	Suffix byte
}

var Vector128 Vec128
var Scalar128 Array128
var Container128 Nested128

//go:noinline
func Identity128(v Vec128) Vec128 { return v }

//go:noinline
func IdentityArray128(v Array128) Array128 { return v }

//go:noinline
func IdentityNested128(v Nested128) Nested128 { return v }

//go:noinline
func Load128(p *Vec128) Vec128 { return *p }

//go:noinline
func Store128(p *Vec128, v Vec128) { *p = v }

//go:noinline
func Add128(x, y Vec128) Vec128 { return x.Add(y) }

//go:noinline
func RoundTrip128(p *Vec128, y Vec128) Vec128 {
	v := Identity128(Add128(Load128(p), y))
	Store128(p, v)
	return v
}

func Layout128() (uintptr, uintptr, uintptr, uintptr) {
	return unsafe.Sizeof(Vector128), unsafe.Alignof(Vector128),
		unsafe.Offsetof(Container128.Value), unsafe.Sizeof(Container128)
}
