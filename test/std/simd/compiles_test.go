// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.27 && (amd64 || arm64 || wasm) && goexperiment.simd

package simd_test

// For testing purposes, this SHOULD compile, because
// the "simd" type whose unsafe.Sizeof is used in a
// constant context, is from some other package whose
// name (but not path) happens to be "simd".

import (
	"github.com/xgo-dev/llgo/test/std/simd/internal/simd"
	"testing"
	"unsafe"
)

var v_for_sizeof [1]simd.HasConstantSize24
var u [unsafe.Sizeof(v_for_sizeof)]byte

func TestCompiles(t *testing.T) {

	if len(u) != 24 {
		t.Errorf("len(u) is %d, instead of expected 24", len(u))
	}
}
