//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"testing"
	"unsafe"
)

type resizeInteger interface {
	int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64
}

// Compare all result lanes, including zero-filled lanes, against scalar Go.
// Exhaust byte/word inputs; larger lanes include every power-of-two boundary
// and a deterministic random stream. Inputs outside the selected half differ
// so using the wrong half cannot accidentally pass.
func testSIMDIntegerResize[S, D resizeInteger](t *testing.T, mode string, apply func([]S, []D)) {
	t.Helper()
	var s S
	var d D
	sbits, dbits := int(unsafe.Sizeof(s)*8), int(unsafe.Sizeof(d)*8)
	n, m := 128/sbits, 128/dbits
	input := make([]S, n)
	if mode == "concat" {
		input = make([]S, 2*n)
	}
	out := make([]D, m)
	var values []uint64
	if sbits <= 16 {
		for i := 0; i < 1<<sbits; i++ {
			values = append(values, uint64(i))
		}
	} else {
		values = append(values, 0, ^uint64(0))
		for bit := 0; bit < 64; bit++ {
			v := uint64(1) << bit
			values = append(values, v-1, v, v+1, -v-1, -v, -v+1)
		}
		state := uint64(0x9876543210abcdef)
		for i := 0; i < 512; i++ {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			values = append(values, state)
		}
	}
	step := len(input)
	if mode == "lo" || mode == "hi" {
		step = m
	}
	for start := 0; start < len(values); start += step {
		for i := range input {
			input[i] = S(values[(start+i)%len(values)])
		}
		for i := range out {
			out[i] = ^D(0)
		}
		apply(input, out)
		for i, got := range out {
			var want D
			index := i
			if mode == "hi" {
				index += n / 2
			}
			if mode == "lo" || mode == "hi" {
				want = D(input[index])
			} else if i < len(input) {
				x := input[i]
				upper := uint64(1)<<dbits - 1
				if ^D(0) < 0 {
					upper >>= 1
				}
				if ^S(0) < 0 && x < 0 {
					lower := int64(0)
					if ^D(0) < 0 {
						lower = -int64(upper) - 1
					}
					v := int64(x)
					if v < lower {
						v = lower
					}
					want = D(v)
				} else {
					v := uint64(x)
					if v > upper {
						v = upper
					}
					want = D(v)
				}
			}
			if got != want {
				t.Fatalf("%s input=%v lane=%d: got %v, want %v", mode, input, i, got, want)
			}
		}
	}
}
