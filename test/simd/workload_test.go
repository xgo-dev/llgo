//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"simd/archsimd"
	"testing"
)

func encodeHexSIMD(dst, src []byte) {
	tableBytes := [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
	table := archsimd.LoadUint8x16Array(&tableBytes)
	mask := archsimd.BroadcastUint8x16(15)
	n := len(src) &^ 15
	var lows, highs [16]byte
	for pos := 0; pos < n; pos += 16 {
		x := archsimd.LoadUint8x16(src[pos:])
		lo := lookupHexDigits(table, x.And(mask))
		// Shifting pairs of bytes and masking each byte also works on amd64,
		// whose official API does not expose a byte-lane ShiftAllRight.
		hi := lookupHexDigits(table, x.ReshapeToUint16s().ShiftAllRight(4).ReshapeToUint8s().And(mask))
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
	// Every tail length, empty input, unaligned starts, and every byte value,
	// plus page-sized and bulk buffers with vector loops and a partial tail.
	lengths := []int{4095, 4096, 4097, 1<<20 + 7}
	for n := 0; n <= 273; n++ {
		lengths = append(lengths, n)
	}
	for offset := 0; offset < 4; offset++ {
		for _, n := range lengths {
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
				for i := range out {
					if out[i] != want[i] {
						t.Fatalf("offset=%d length=%d byte=%d: %02x want %02x", offset, n, i, out[i], want[i])
					}
				}
			}
			if backingOut[0] != 0xa5 || backingOut[len(backingOut)-1] != 0xa5 || backingOut[len(backingOut)-2] != 0xa5 || backingOut[len(backingOut)-3] != 0xa5 {
				t.Fatalf("output boundary overwritten: length=%d", n)
			}
		}
	}
}

func BenchmarkSIMDHex(b *testing.B) {
	for _, n := range []int{64, 4096, 1<<20 + 7} {
		src := make([]byte, n+1)[1:]
		for i := range src {
			src[i] = byte(i * 73)
		}
		want := make([]byte, hex.EncodedLen(n))
		hex.Encode(want, src)
		for _, impl := range []struct {
			name   string
			encode func([]byte, []byte)
		}{
			{"scalar", func(dst, src []byte) { hex.Encode(dst, src) }},
			{"simd", encodeHexSIMD},
		} {
			b.Run(fmt.Sprintf("%s/%d", impl.name, n), func(b *testing.B) {
				dst := make([]byte, len(want))
				b.SetBytes(int64(n))
				b.ReportAllocs()
				for b.Loop() {
					impl.encode(dst, src)
				}
				if !bytes.Equal(dst, want) {
					b.Fatal("hex output differs from encoding/hex")
				}
			})
		}
	}
}
