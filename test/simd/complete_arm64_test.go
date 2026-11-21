//go:build goexperiment.simd && arm64

package simd_test

import (
	"math/bits"
	"simd/archsimd"
	"testing"
)

func TestSIMDLeadingBitsInt8x16(t *testing.T) {
	var input, zeros, sign [16]int8
	for offset := 0; offset < 256; offset += 16 {
		for i := range input {
			input[i] = int8(i + offset)
		}
		x := archsimd.LoadInt8x16Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint8(lane)
			z := bits.LeadingZeros8(u)
			if int8(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros8(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}

// Sweep the high byte, then every single-bit value and its complement.
func leadingBitsInput(index, width int) uint32 {
	if index < 256 {
		return uint32(index) << (width - 8)
	}
	bit := uint32(1) << ((index - 256) / 2)
	if index%2 != 0 {
		return ^bit
	}
	return bit
}

func TestSIMDLeadingBitsInt16x8(t *testing.T) {
	var input, zeros, sign [8]int16
	for offset := 0; offset < 256+2*16; offset += 8 {
		for i := range input {
			input[i] = int16(leadingBitsInput(offset+i, 16))
		}
		x := archsimd.LoadInt16x8Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint16(lane)
			z := bits.LeadingZeros16(u)
			if int16(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros16(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}
func TestSIMDLeadingBitsInt32x4(t *testing.T) {
	var input, zeros, sign [4]int32
	for offset := 0; offset < 256+2*32; offset += 4 {
		for i := range input {
			input[i] = int32(leadingBitsInput(offset+i, 32))
		}
		x := archsimd.LoadInt32x4Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint32(lane)
			z := bits.LeadingZeros32(u)
			if int32(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros32(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}
func TestSIMDLeadingBitsUint8x16(t *testing.T) {
	var input, zeros, sign [16]uint8
	for offset := 0; offset < 256; offset += 16 {
		for i := range input {
			input[i] = uint8(i + offset)
		}
		x := archsimd.LoadUint8x16Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint8(lane)
			z := bits.LeadingZeros8(u)
			if int8(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros8(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}
func TestSIMDLeadingBitsUint16x8(t *testing.T) {
	var input, zeros, sign [8]uint16
	for offset := 0; offset < 256+2*16; offset += 8 {
		for i := range input {
			input[i] = uint16(leadingBitsInput(offset+i, 16))
		}
		x := archsimd.LoadUint16x8Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint16(lane)
			z := bits.LeadingZeros16(u)
			if int16(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros16(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}
func TestSIMDLeadingBitsUint32x4(t *testing.T) {
	var input, zeros, sign [4]uint32
	for offset := 0; offset < 256+2*32; offset += 4 {
		for i := range input {
			input[i] = uint32(leadingBitsInput(offset+i, 32))
		}
		x := archsimd.LoadUint32x4Array(&input)
		x.LeadingZeros().StoreArray(&zeros)
		x.LeadingSignBits().StoreArray(&sign)
		for i, lane := range input {
			u := uint32(lane)
			z := bits.LeadingZeros32(u)
			if int32(u) < 0 {
				u = ^u
			}
			cls := bits.LeadingZeros32(u) - 1
			if int(zeros[i]) != z || int(sign[i]) != cls {
				t.Fatalf("input %v: got %d/%d want %d/%d", lane, zeros[i], sign[i], z, cls)
			}
		}
	}
}
