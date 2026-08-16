// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unique

import (
	"internal/abi"
	"unsafe"
)

// clone makes a copy of value. A top-level string is cloned so a substring
// does not keep a larger backing string alive. Composite values remain shallow
// copies because LLGo's composite type metadata has a runtime-specific layout.
func clone[T comparable](value T, seq *cloneSeq) T {
	for _, offset := range seq.stringOffsets {
		ps := (*string)(unsafe.Pointer(uintptr(unsafe.Pointer(&value)) + offset))
		*ps = stringClone(*ps)
	}
	return value
}

func stringClone(s string) string {
	if len(s) == 0 {
		return ""
	}
	b := make([]byte, len(s))
	copy(b, s)
	return unsafe.String(&b[0], len(b))
}

// singleStringClone describes how to clone a single string.
var singleStringClone = cloneSeq{stringOffsets: []uintptr{0}}

// cloneSeq describes how to clone a value of a particular type.
type cloneSeq struct {
	stringOffsets []uintptr
}

// makeCloneSeq creates a cloneSeq for a type.
func makeCloneSeq(typ *abi.Type) cloneSeq {
	if typ == nil {
		return cloneSeq{}
	}
	if typ.Kind() == abi.String {
		return singleStringClone
	}
	return cloneSeq{}
}
