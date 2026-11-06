package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

func StringToBytesReadOnly(s String) []byte {
	if s.len == 0 {
		return []byte{}
	}
	return unsafe.Slice((*byte)(s.data), s.len)
}

func StringFromBytesReadOnly(b Slice) String {
	if b.len == 0 {
		return String{}
	}
	return String{b.data, b.len}
}

// These entry points copy into caller-owned storage only when escape analysis
// proves that the result cannot outlive that storage. They preserve conversion
// copy semantics; no mutable source array is shared with a string result.
func StringFromBytesBuffer(b Slice, buffer *[abi.SmallStackBufferSize]byte) String {
	if b.len == 0 {
		return String{}
	}
	if b.len > len(buffer) {
		return StringFromBytes(b)
	}
	c.Memcpy(unsafe.Pointer(buffer), b.data, uintptr(b.len))
	return String{unsafe.Pointer(buffer), b.len}
}

func StringToBytesBuffer(s String, buffer *[abi.SmallStackBufferSize]byte) []byte {
	if s.len > len(buffer) {
		return StringToBytes(s)
	}
	c.Memcpy(unsafe.Pointer(buffer), s.data, uintptr(s.len))
	return buffer[:s.len:s.len]
}

func StringCatBuffer(a, b String, buffer *[abi.SmallStackBufferSize]byte) String {
	n := a.len + b.len
	if uint(n) > uint(len(buffer)) || n < a.len {
		return StringCat(a, b)
	}
	if n == 0 {
		return String{}
	}
	c.Memcpy(unsafe.Pointer(buffer), a.data, uintptr(a.len))
	c.Memcpy(c.Advance(unsafe.Pointer(buffer), a.len), b.data, uintptr(b.len))
	return String{unsafe.Pointer(buffer), n}
}

func StringToRunesBuffer(s string, buffer *[abi.SmallStackBufferSize]rune) []rune {
	// Byte length is an inexpensive upper bound. Count longer UTF-8 inputs
	// before deciding to fall back, so 32 multibyte runes can still fit.
	if len(s) > len(buffer) {
		n := 0
		for i := 0; i < len(s); {
			if s[i] < runeSelf {
				i++
			} else {
				_, i = decodeNonASCII(s, i)
			}
			n++
			if n > len(buffer) {
				return StringToRunes(s)
			}
		}
	}
	n := 0
	for i := 0; i < len(s); {
		if s[i] < runeSelf {
			buffer[n] = rune(s[i])
			i++
		} else {
			buffer[n], i = decodeNonASCII(s, i)
		}
		n++
	}
	return buffer[:n:n]
}

func StringFromRunesBuffer(rs []rune, buffer *[abi.SmallStackBufferSize]byte) String {
	if len(rs) == 0 {
		return String{}
	}
	var scratch [4]byte
	n := 0
	if len(rs) > len(buffer)/4 {
		for _, r := range rs {
			n += encoderune(scratch[:], r)
			if n > len(buffer) {
				return StringFromRunes(rs)
			}
		}
	}
	index := 0
	for _, r := range rs {
		index += encoderune(buffer[index:], r)
	}
	return String{unsafe.Pointer(buffer), index}
}

func StringFromInt64Buffer(r int64, buffer *[abi.SmallStackBufferSize]byte) String {
	if r < 0 || r > maxRune {
		r = runeError
	}
	n := encoderune(buffer[:], rune(r))
	return String{unsafe.Pointer(buffer), n}
}

func StringFromUint64Buffer(r uint64, buffer *[abi.SmallStackBufferSize]byte) String {
	if r > maxRune {
		r = runeError
	}
	n := encoderune(buffer[:], rune(r))
	return String{unsafe.Pointer(buffer), n}
}
