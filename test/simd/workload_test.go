//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"bytes"
	"encoding/hex"
	"simd/archsimd"
	"testing"
)

func encodeHexSIMD(dst, src []byte) {
	tableBytes := [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
	table := archsimd.LoadUint8x16Array(&tableBytes)
	mask := archsimd.BroadcastUint8x16(15)
	n := len(src) &^ 15
	for pos := 0; pos < n; pos += 16 {
		x := archsimd.LoadUint8x16(src[pos:])
		lo := lookupHexDigits(table, x.And(mask))
		// Shifting pairs of bytes and masking each byte also works on amd64,
		// whose official API does not expose a byte-lane ShiftAllRight.
		hi := lookupHexDigits(table, x.ReshapeToUint16s().ShiftAllRight(4).ReshapeToUint8s().And(mask))
		var lows, highs [16]byte
		lo.StoreArray(&lows)
		hi.StoreArray(&highs)
		for i := range lows {
			dst[2*(pos+i)], dst[2*(pos+i)+1] = highs[i], lows[i]
		}
	}
	for i := n; i < len(src); i++ {
		dst[2*i], dst[2*i+1] = tableBytes[src[i]>>4], tableBytes[src[i]&15]
	}
}

func TestSIMDHexWorkload(t *testing.T) {
	// Every tail length, empty input, unaligned starts, and every byte value.
	for offset := 0; offset < 4; offset++ {
		for n := 0; n <= 273; n++ {
			backing := make([]byte, n+offset)
			src := backing[offset:]
			for i := range src {
				src[i] = byte(i*73 + n)
			}
			want := make([]byte, n*2)
			hex.Encode(want, src)
			backingOut := bytes.Repeat([]byte{0xa5}, 2*n+4)
			out := backingOut[1 : 1+2*n]
			encodeHexSIMD(out, src)
			if !bytes.Equal(out, want) {
				t.Fatalf("offset=%d length=%d: %x want %x", offset, n, out, want)
			}
			if backingOut[0] != 0xa5 || backingOut[len(backingOut)-1] != 0xa5 || backingOut[len(backingOut)-2] != 0xa5 || backingOut[len(backingOut)-3] != 0xa5 {
				t.Fatalf("output boundary overwritten: length=%d", n)
			}
		}
	}
}
