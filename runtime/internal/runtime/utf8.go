// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import _ "unsafe"

// This declaration-only view lets cmd/go load package metadata. LLGo replaces
// the entire file with the selected GOROOT runtime/utf8.go before type checking.
// See doc/runtime-source-reuse.md.

// Numbers fundamental to the encoding.
const (
	runeError = '\uFFFD'     // the "error" Rune or "Unicode replacement character"
	runeSelf  = 0x80         // characters below runeSelf are represented as themselves in a single byte.
	maxRune   = '\U0010FFFF' // Maximum valid Unicode code point.
)

// Code points in the surrogate range are not valid for UTF-8.
const (
	surrogateMin = 0xD800
	surrogateMax = 0xDFFF
)

const (
	t1 = 0x00 // 0000 0000
	tx = 0x80 // 1000 0000
	t2 = 0xC0 // 1100 0000
	t3 = 0xE0 // 1110 0000
	t4 = 0xF0 // 1111 0000
	t5 = 0xF8 // 1111 1000

	maskx = 0x3F // 0011 1111
	mask2 = 0x1F // 0001 1111
	mask3 = 0x0F // 0000 1111
	mask4 = 0x07 // 0000 0111

	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1

	// The default lowest and highest continuation byte.
	locb = 0x80 // 1000 0000
	hicb = 0xBF // 1011 1111
)

//go:linkname countrunes
func countrunes(s string) int

//go:linkname encoderune
func encoderune(p []byte, r rune) int
