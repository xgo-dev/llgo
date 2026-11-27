/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// snprintf writes at most 63 bytes plus its terminator. The legacy format
// can insert up to two additional exponent digits in the same buffer.
const floatFormatSize = 64

type floatBuffer [floatFormatSize + 2]byte

func formatFloat(buf *floatBuffer, v float64, imaginary bool) int {
	switch {
	case v != v:
		if imaginary && !legacyFloatFormat {
			return copy(buf[:], "+NaN")
		}
		return copy(buf[:], "NaN")
	case v+v == v && v > 0:
		return copy(buf[:], "+Inf")
	case v+v == v && v < 0:
		return copy(buf[:], "-Inf")
	}
	format := c.Str("%g")
	if legacyFloatFormat {
		format = c.Str("%+.6e")
	} else if imaginary {
		format = c.Str("%+g")
	}
	p := (*c.Char)(unsafe.Pointer(buf))
	c.Snprintf(p, floatFormatSize, format, v)
	n := int(c.Strlen(p))
	if legacyFloatFormat {
		return padLegacyExponent(buf, n)
	}
	return n
}

func padLegacyExponent(buf *floatBuffer, n int) int {
	// The legacy Go format uses three exponent digits. Some C libraries use
	// fewer, so move their digits to the right before inserting the zeroes.
	for i := n - 1; i >= 0; i-- {
		if buf[i] != 'e' && buf[i] != 'E' {
			continue
		}
		exp := i + 2
		digits := n - exp
		if (digits == 1 || digits == 2) && (buf[i+1] == '+' || buf[i+1] == '-') {
			pad := 3 - digits
			copy(buf[exp+pad:], buf[exp:n])
			copy(buf[exp:], "00"[:pad])
			n += pad
		}
		break
	}
	return n
}
