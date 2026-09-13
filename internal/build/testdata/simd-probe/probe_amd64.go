//go:build goexperiment.simd && amd64

package simdprobe

import "simd/archsimd"

type Vec256 = archsimd.Float32x8
type Vec512 = archsimd.Float32x16
type Nested256 struct {
	Prefix byte
	Value  Vec256
	Suffix byte
}
type Nested512 struct {
	Prefix byte
	Value  Vec512
	Suffix byte
}

var Vector256 Vec256
var Vector512 Vec512
var Container256 Nested256
var Container512 Nested512

//go:noinline
func Identity256(v Vec256) Vec256 { return v }

//go:noinline
func Identity512(v Vec512) Vec512 { return v }

//go:noinline
func IdentityNested256(v Nested256) Nested256 { return v }

//go:noinline
func IdentityNested512(v Nested512) Nested512 { return v }

//go:noinline
func Load256(p *Vec256) Vec256 { return *p }

//go:noinline
func Load512(p *Vec512) Vec512 { return *p }

//go:noinline
func Store256(p *Vec256, v Vec256) { *p = v }

//go:noinline
func Store512(p *Vec512, v Vec512) { *p = v }

//go:noinline
func Add256(x, y Vec256) Vec256 { return x.Add(y) }

//go:noinline
func Add512(x, y Vec512) Vec512 { return x.Add(y) }
