// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

//go:build arm

package bytealg

//llgo:skip IndexByte IndexByteString Compare Count CountString

// LLGo cannot translate this architecture's bytealg assembly. Replace the
// assembly-backed declarations with plain Go implementations; the generic
// Index/IndexString files already cover this build configuration.
func IndexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func IndexByteString(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func Compare(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func Count(b []byte, c byte) int {
	return countGeneric(b, c)
}

func CountString(s string, c byte) int {
	return countGenericString(s, c)
}
