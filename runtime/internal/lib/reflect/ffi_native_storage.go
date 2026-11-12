package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// libffi's size_t slot follows the physical address width, including Memory32
// targets whose ordinary Go uintptr is eight bytes. SizeT is uint32 there.
const ffiNativeUintptrSize = unsafe.Sizeof(ffi.Type{}.Size)

func nativeFFIStorageDiffers(typ *abi.Type) bool {
	if ffiNativeUintptrSize >= unsafe.Sizeof(uintptr(0)) {
		return false
	}
	switch typ.Kind() {
	case abi.Uintptr:
		return true
	case abi.Array:
		return nativeFFIStorageDiffers(typ.ArrayType().Elem)
	}
	return false
}

func nativeFFIStorageSize(typ *abi.Type) uintptr {
	switch typ.Kind() {
	case abi.Uintptr:
		return ffiNativeUintptrSize
	case abi.Array:
		array := typ.ArrayType()
		return array.Len * nativeFFIStorageSize(array.Elem)
	}
	return typ.Size_
}

// Only uintptr and enclosing arrays change storage at this native boundary.
// Struct parameters keep their declared field layouts, as in SSA's CDecl.
func copyNativeFFIValue(dst, src unsafe.Pointer, typ *abi.Type, toNative bool) {
	switch typ.Kind() {
	case abi.Uintptr:
		if toNative {
			n := *(*uintptr)(src)
			if n > uintptr(^uint32(0)) {
				panic("WebAssembly ABI integer conversion out of range")
			}
			*(*uint32)(dst) = uint32(n)
		} else {
			*(*uintptr)(dst) = uintptr(*(*uint32)(src))
		}
	case abi.Array:
		array := typ.ArrayType()
		goStride, nativeStride := array.Elem.Size_, nativeFFIStorageSize(array.Elem)
		for i := uintptr(0); i < array.Len; i++ {
			if toNative {
				copyNativeFFIValue(add(dst, i*nativeStride, ""), add(src, i*goStride, ""), array.Elem, true)
			} else {
				copyNativeFFIValue(add(dst, i*goStride, ""), add(src, i*nativeStride, ""), array.Elem, false)
			}
		}
	default:
		c.Memmove(dst, src, typ.Size_)
	}
}

func makeFuncFFIArgValue(ptr unsafe.Pointer, typ *abi.Type, native bool) Value {
	if !native || !nativeFFIStorageDiffers(typ) {
		return makeFuncArgValue(ptr, typ)
	}
	storage := unsafe_New(typ)
	copyNativeFFIValue(storage, ptr, typ, false)
	return ffiToValue(storage, typ)
}
