package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// cdecl gives the struct the native C layout, so unsafe.Sizeof reports the
// target's 4-byte pointer width even where Go widens stored pointers to
// 8-byte slots (Memory32). Pairing it with unsafe.Sizeof(uintptr(0)) drives
// wideGoHeaderSlots below and the low/offset math in clearTypedHeaderRange.
// It is the only cdecl on a struct in the tree — do not remove it, or the
// widened-slot detection silently stops firing.
//
//llgo:cdecl
type nativePointerSlot struct{ ptr unsafe.Pointer }

// Hosted Memory32 widens the Go data slot to 8 bytes around a 4-byte pointer.
// The constant also folds away the typed-copy work on native and Memory64.
const wideGoHeaderSlots = unsafe.Sizeof(uintptr(0)) > unsafe.Sizeof(nativePointerSlot{})

// clearTypedHeaderRange uses the existing runtime element description when a
// dynamically typed operation copies memory. Each recursion visits only inline
// fields. It reports whether it found a header, so that an array of other
// values only requires one visit to its element type.
func clearTypedHeaderRange(typ *Type, dst unsafe.Pointer, count, stride uintptr) bool {
	// Regular-memory types can be compared bytewise, so they contain no
	// inline string or slice headers even when they contain pointers.
	if count == 0 || !typ.Pointers() || typ.TFlag&abi.TFlagRegularMemory != 0 {
		return false
	}
	switch typ.Kind() {
	case abi.String, abi.Slice:
		const low = unsafe.Sizeof(nativePointerSlot{})
		const padding = unsafe.Sizeof(uintptr(0)) - low
		for i := uintptr(0); i < count; i++ {
			*(*[padding]byte)(unsafe.Add(dst, i*stride+low)) = [padding]byte{}
		}
		return true
	case abi.Struct:
		fields := typ.StructType().Fields
		found := false
		for i := range fields {
			field := &fields[i]
			if clearTypedHeaderRange(field.Typ, unsafe.Add(dst, field.Offset), count, stride) {
				found = true
			}
		}
		return found
	case abi.Array:
		arr := typ.ArrayType()
		if count == 1 || stride == typ.Size_ {
			return clearTypedHeaderRange(arr.Elem, dst, count*arr.Len, arr.Elem.Size_)
		}
		if arr.Len == 0 || !clearTypedHeaderRange(arr.Elem, dst, count, stride) {
			return false
		}
		for i := uintptr(1); i < arr.Len; i++ {
			clearTypedHeaderRange(arr.Elem, unsafe.Add(dst, i*arr.Elem.Size_), count, stride)
		}
		return true
	}
	return false
}
