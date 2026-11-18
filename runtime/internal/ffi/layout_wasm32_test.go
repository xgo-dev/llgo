//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi_test

import (
	"reflect"
	"testing"
	"unsafe"

	native "github.com/xgo-dev/llgo/runtime/internal/clite/ffi"
)

func TestWasm32FFIHostLayout(t *testing.T) {
	// ffi_type and ffi_cif from the vendored wasm32 libffi headers. Cif.Extra
	// overlays nfixedargs. Check actual array stride as well as source/reflect.
	var types [2]native.Type
	if unsafe.Sizeof(types[0]) != 12 || reflect.TypeOf(types[0]).Size() != 12 ||
		uintptr(unsafe.Pointer(&types[1]))-uintptr(unsafe.Pointer(&types[0])) != 12 ||
		unsafe.Offsetof(types[0].Alignment) != 4 || unsafe.Offsetof(types[0].Type) != 6 || unsafe.Offsetof(types[0].Elements) != 8 {
		t.Fatal("ffi_type does not match wasm32 C storage")
	}
	var cifs [2]native.Cif
	if unsafe.Sizeof(cifs[0]) != 28 || reflect.TypeOf(cifs[0]).Size() != 28 ||
		uintptr(unsafe.Pointer(&cifs[1]))-uintptr(unsafe.Pointer(&cifs[0])) != 28 ||
		unsafe.Offsetof(cifs[0].ArgTypes) != 8 || unsafe.Offsetof(cifs[0].RType) != 12 || unsafe.Offsetof(cifs[0].Extra) != 24 {
		t.Fatal("ffi_cif does not match wasm32 C storage")
	}
}

func TestWasm32FFITypeSizeOverflow(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("oversized native size_t was truncated")
		}
	}()
	native.NewType(uintptr(1)<<32, 8, native.Struct, nil)
}
