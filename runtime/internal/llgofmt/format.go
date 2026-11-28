// Package llgofmt provides shared formatting operations for compiler-generated
// typed formatters. Each call site supplies concrete values and formatters.
package llgofmt

import (
	"math"
	"strconv"
)

func AppendInt(dst []byte, value int64) []byte {
	return strconv.AppendInt(dst, value, 10)
}

func AppendUint(dst []byte, value uint64) []byte {
	return strconv.AppendUint(dst, value, 10)
}

func AppendFloat(dst []byte, value float64, bits int) []byte {
	return strconv.AppendFloat(dst, value, 'g', -1, bits)
}

func AppendComplex(dst []byte, value complex128, bits int) []byte {
	dst = append(dst, '(')
	dst = AppendFloat(dst, real(value), bits/2)
	imaginary := imag(value)
	if math.IsNaN(imaginary) || !math.Signbit(imaginary) && !math.IsInf(imaginary, 1) {
		dst = append(dst, '+')
	}
	dst = AppendFloat(dst, imaginary, bits/2)
	return append(dst, 'i', ')')
}

func AppendPointer(dst []byte, value uintptr) []byte {
	if value == 0 {
		return append(dst, "<nil>"...)
	}
	return strconv.AppendUint(append(dst, "0x"...), uint64(value), 16)
}

type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

func Compare[T ordered](a, b T) int {
	if a < b || a != a && b == b {
		return -1
	}
	if a > b || a == a && b != b {
		return 1
	}
	return 0
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// AppendMap snapshots keys and values before sorting, including entries that
// cannot be retrieved by a second lookup, such as NaN keys.
func AppendMap[K comparable, V any](dst []byte, value map[K]V,
	compare func(K, K) int, appendKey func([]byte, K) []byte, appendElem func([]byte, V) []byte,
) []byte {
	entries := make([]entry[K, V], 0, len(value))
	for key, elem := range value {
		entries = append(entries, entry[K, V]{key, elem})
	}
	sortEntries(entries, compare)
	dst = append(dst, "map["...)
	for i, pair := range entries {
		if i != 0 {
			dst = append(dst, ' ')
		}
		dst = appendKey(dst, pair.key)
		dst = append(dst, ':')
		dst = appendElem(dst, pair.value)
	}
	return append(dst, ']')
}
