package main

import _ "runtime"

func copyABI0(dst, src *byte, n uintptr) uint64

func main() {
	source := [105]byte{}
	for i := range source {
		source[i] = byte(i*17 + 3)
	}
	for _, count := range []uintptr{0, 1, 7, 16, 63, 99} {
		destination := [105]byte{}
		for i := range destination {
			destination[i] = 0xa5
		}
		if copyABI0(&destination[1], &source[1], count) != uint64(count)+37 {
			panic("saved spill/result")
		}
		for i, value := range destination {
			want := byte(0xa5)
			if i >= 1 && uintptr(i) <= count {
				want = source[i]
			}
			if value != want {
				panic("ABI0 copy/canary")
			}
		}
	}
	println("ARM64 ABI0 memmove verified")
}
