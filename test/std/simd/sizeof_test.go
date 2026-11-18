// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.27 && (amd64 || arm64 || wasm) && goexperiment.simd

package simd_test

import (
	"github.com/xgo-dev/llgo/test/std/simd/internal/pkg"
	"simd"
	"testing"
	"unsafe"
)

var v simd.Float32s

func TestSizeof(t *testing.T) {
	var f float32
	sv0 := int(unsafe.Sizeof(v))
	sv1 := v.Len() * int(unsafe.Sizeof(f))
	sV := int(unsafe.Sizeof(pkg.V))
	sF := int(unsafe.Sizeof(pkg.F()))
	if sv0 != sv1 {
		t.Errorf("sv0=%d and sv1=%d should be equal but are not", sv0, sv1)
	}
	if sF != sv1 {
		t.Errorf("sF=%d and sv1=%d should be equal but are not", sF, sv1)
	}
	if sV != sv1 {
		t.Errorf("sV=%d and sv1=%d should be equal but are not", sV, sv1)
	}
}
