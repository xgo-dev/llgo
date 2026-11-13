//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package reflect

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// wasm32 stores a Go pointer as {ptr, i32}. A struct whose only non-zero
// field is such a pointer is a two-element aggregate that LLVM returns via
// sret. The first element's alignment is 8 so multi-value FFI layouts keep
// Go's 8-byte alignment. Walk the single-field chain to the leaf: native
// cdecl/stdcall storage keeps the original FFI description.
var ffiTypeWideGoPointerWord = ffi.StructOf(
	&ffi.Type{Size: 4, Alignment: 8, Type: ffi.TypeUint32.Type},
	ffi.TypeUint32,
)

func ffiWidenGoPointerStruct(ft *ffi.Type, typ *abi.Type) *ffi.Type {
	if !ffiIsGoWidePointerLeaf(typ) {
		return ft
	}
	return ffiTypeWideGoPointerWord
}

func ffiIsGoWidePointerLeaf(typ *abi.Type) bool {
	for typ != nil {
		if typ.HasNativeLayout() {
			return false
		}
		switch typ.Kind() {
		case abi.Pointer, abi.UnsafePointer, abi.Chan, abi.Map, abi.Func:
			return true
		case abi.Array:
			at := typ.ArrayType()
			if at == nil || at.Len != 1 {
				return false
			}
			typ = at.Elem
		case abi.Struct:
			st := typ.StructType()
			if st == nil {
				return false
			}
			var leaf *abi.Type
			for i := range st.Fields {
				ft := st.Fields[i].Typ
				if ft.Size_ == 0 {
					continue
				}
				if leaf != nil {
					return false
				}
				leaf = ft
			}
			if leaf == nil {
				return false
			}
			typ = leaf
		default:
			return false
		}
	}
	return false
}
