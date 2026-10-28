package main

import (
	"fmt"

	kcrc32 "github.com/klauspost/crc32"
)

func main() {
	requireAssemblyCPU()
	message := []byte("Hello world")
	const custom = 0xD5828281
	check(message, custom)

	// Exercise the scalar, alignment, SSE triple-stream and CLMUL paths, with
	// tails on either side of the library's 64, 504 and 4032 byte boundaries.
	data := make([]byte, 8192+8)
	for i := range data {
		data[i] = byte(i*37 + i/7)
	}
	for _, poly := range []uint32{kcrc32.IEEE, kcrc32.Castagnoli, custom} {
		for offset := 0; offset < 8; offset++ {
			for _, size := range []int{0, 1, 7, 8, 15, 16, 63, 64, 65, 503, 504, 505, 1023, 1024, 4031, 4032, 4033, 8192} {
				check(data[offset:offset+size], poly)
			}
		}
	}
	fmt.Println("crc32 assembly verified")
}

func check(data []byte, poly uint32) {
	got := kcrc32.Checksum(data, kcrc32.MakeTable(poly))
	want := referenceCRC(data, poly)
	if got != want {
		panic(fmt.Sprintf("poly=%08x length=%d: got=%08x want=%08x", poly, len(data), got, want))
	}
}

// Keep the oracle independent of both libraries' assembly implementations.
// In particular, stdlib hash/crc32 can use the same translated instructions.
func referenceCRC(data []byte, poly uint32) uint32 {
	crc := ^uint32(0)
	for _, b := range data {
		crc ^= uint32(b)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ poly
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}
