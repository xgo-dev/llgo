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

	c "github.com/goplus/llgo/runtime/internal/clite"
)

// -----------------------------------------------------------------------------

// String is the runtime representation of a string.
// It cannot be used safely or portably and its representation may
// change in a later release.
//
// Unlike reflect.StringHeader, its Data field is sufficient to guarantee the
// data it references will not be garbage collected.
type String struct {
	data unsafe.Pointer
	len  int
}

// StringCat concatenates two strings.
func StringCat(a, b String) String {
	n := a.len + b.len
	dest := AllocU(uintptr(n))
	c.Memcpy(dest, a.data, uintptr(a.len))
	c.Memcpy(c.Advance(dest, a.len), b.data, uintptr(b.len))
	return String{dest, n}
}

// -----------------------------------------------------------------------------

// CStrCopy copies a Go string to a C string buffer and returns it.
func CStrCopy(dest unsafe.Pointer, s String) *int8 {
	n := s.len
	c.Memcpy(dest, s.data, uintptr(n))
	*(*int8)(c.Advance(dest, n)) = 0
	return (*int8)(dest)
}

func CStrDup(s String) *int8 {
	dest := AllocU(uintptr(s.len + 1))
	return CStrCopy(dest, s)
}

func StringSlice(base String, i, j int) String {
	return StringSlice2(base, int64(i), int64(j), true, true)
}

func StringSlice2(base String, i, j int64, iSigned, jSigned bool) String {
	if boundsOutOfRange(j, jSigned, base.len, true) {
		panicBounds(j, jSigned, base.len, boundsSliceAlen)
	}
	if boundsAbove(i, iSigned, j) {
		panicBounds(i, iSigned, int(j), boundsSliceB)
	}
	ii := int(i)
	jj := int(j)
	if ii < base.len {
		return String{c.Advance(base.data, ii), jj - ii}
	}
	// Keep the source base for empty suffix slices to avoid advancing past
	// the underlying allocation while still preserving a stable non-nil base.
	return String{base.data, 0}
}

type StringIter struct {
	s   string
	pos int
}

func NewStringIter(s string) *StringIter {
	return &StringIter{s, 0}
}

func StringIterNext(it *StringIter) (ok bool, k int, v rune) {
	if it.pos >= len(it.s) {
		return false, 0, 0
	}
	k = it.pos
	if c := it.s[it.pos]; c < runeSelf {
		it.pos++
		v = rune(c)
	} else {
		v, it.pos = decoderune(it.s, it.pos)
	}
	ok = true
	return
}

func StringToBytes(s String) []byte {
	if s.len == 0 {
		return []byte{}
	}
	data := make([]byte, s.len)
	c.Memcpy(unsafe.Pointer(&data[0]), s.data, uintptr(s.len))
	return data
}

func StringToRunes(s string) []rune {
	if len(s) == 0 {
		return []rune{}
	}
	data := make([]rune, len(s))
	var index uint
	for i := 0; i < len(s); {
		if c := s[i]; c < runeSelf {
			data[index] = rune(c)
			i++
		} else {
			data[index], i = decoderune(s, i)
		}
		index++
	}
	return data[:index:index]
}

func StringFromCStr(cstr *int8) (s String) {
	if cstr == nil {
		return
	}
	return StringFrom(unsafe.Pointer(cstr), int(c.Strlen(cstr)))
}

func StringFromBytes(b Slice) (s String) {
	return StringFrom(b.data, b.len)
}

func StringFrom(data unsafe.Pointer, n int) (s String) {
	if n == 0 {
		return
	}
	s.len = n
	s.data = AllocU(uintptr(n))
	c.Memcpy(s.data, data, uintptr(n))
	return
}

func StringFromRunes(rs []rune) (s String) {
	if len(rs) == 0 {
		return
	}
	data := make([]byte, len(rs)*4)
	var index int
	for _, r := range rs {
		n := encoderune(data[index:], r)
		index += n
	}
	s.len = index
	s.data = unsafe.Pointer(&data[0])
	return
}

func StringFromRune(r rune) (s String) {
	var buf [4]byte
	n := encoderune(buf[:], r)
	s.len = n
	s.data = unsafe.Pointer(&buf[0])
	return
}

func StringFromInt64(r int64) String {
	if r < 0 || r > maxRune {
		return StringFromRune(runeError)
	}
	// StringFromRune handles surrogate code points by emitting runeError.
	return StringFromRune(rune(r))
}

func StringFromUint64(r uint64) String {
	if r > maxRune {
		return StringFromRune(runeError)
	}
	// StringFromRune handles surrogate code points by emitting runeError.
	return StringFromRune(rune(r))
}

func StringEqual(x, y String) bool {
	if x.len != y.len {
		return false
	}
	if x.data != y.data {
		for i := 0; i < x.len; i++ {
			if *(*byte)(c.Advance(x.data, i)) != *(*byte)(c.Advance(y.data, i)) {
				return false
			}
		}
	}
	return true
}

func StringLess(x, y String) bool {
	n := x.len
	if n > y.len {
		n = y.len
	}
	for i := 0; i < n; i++ {
		ix := *(*byte)(c.Advance(x.data, i))
		iy := *(*byte)(c.Advance(y.data, i))
		if ix < iy {
			return true
		} else if ix > iy {
			return false
		}
	}
	return x.len < y.len
}

// -----------------------------------------------------------------------------
