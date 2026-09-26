//go:build js && wasm
// +build js,wasm

package js

import (
	"runtime"
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "github.com/xgo-dev/llgo/runtime/internal/embind"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Emscripten emval handles belong to the JavaScript realm of the physical
// worker that created them.
//
//llgointernal:tls
var (
	emvalGlobalsReady   bool
	valueGlobal         Value
	objectConstructor   Value
	stringConstructor   Value
	arrayConstructor    Value
	functionConstructor Value
	valueNaN            Value
	valueZero           Value
	uint8Array          Value
	uint8ClampedArray   Value
)

var (
	valueUndefined = Value{ref: 2}
	valueNull      = Value{ref: 4}
	valueTrue      = Value{ref: 6}
	valueFalse     = Value{ref: 8}
)

func ensureEmvalGlobals() {
	llruntime.MarkCurrentJSRealm()
	initEmvalGlobals()
}

func initEmvalGlobals() {
	if emvalGlobalsReady {
		return
	}
	valueGlobal = emval_get_global(nil)
	objectConstructor = emval_get_global(c.Str("Object"))
	stringConstructor = emval_get_global(c.Str("String"))
	arrayConstructor = emval_get_global(c.Str("Array"))
	functionConstructor = emval_get_global(c.Str("Function"))
	valueNaN = emval_get_global(c.Str("NaN"))
	valueZero = emval_new_double(0)
	uint8Array = emval_get_global(c.Str("Uint8Array"))
	uint8ClampedArray = emval_get_global(c.Str("Uint8ClampedArray"))
	emvalGlobalsReady = true
}

func valueFromEmval(handle uintptr) Value {
	if handle == 0 {
		return Value{}
	}
	p := new(ref)
	*p = ref(handle)
	owner := emvalOwner()
	runtime.SetFinalizer(p, func(p *ref) {
		releaseEmval(uintptr(*p), owner)
	})
	return Value{ref: *p, gcPtr: p}
}

func (v Value) emvalHandle() uintptr {
	if v.ref == 0 {
		return uintptr(valueUndefined.ref)
	}
	return uintptr(v.ref)
}

func emval_get_global(name *c.Char) Value {
	return valueFromEmval(cEmvalGetGlobal(name))
}

func emval_get_module_property(name *c.Char) Value {
	return valueFromEmval(cEmvalGetModuleProperty(name))
}

func emval_install_invoke()              { cEmvalInstallInvoke(dispatchSynchronousCallback) }
func emval_has_pending_invoke() bool     { return cEmvalHasPendingInvoke() }
func emval_take_pending_invoke() uintptr { return cEmvalTakePendingInvoke() }

func emval_new_double(v float64) Value { return valueFromEmval(cEmvalNewDouble(v)) }
func emval_new_string(str *c.Char, length c.SizeT) Value {
	return valueFromEmval(cEmvalNewString(str, length))
}
func emval_new_object() Value { return valueFromEmval(cEmvalNewObject()) }
func emval_new_array() Value  { return valueFromEmval(cEmvalNewArray()) }

func emval_set_property(object, key, value Value) {
	cEmvalSetProperty(object.emvalHandle(), key.emvalHandle(), value.emvalHandle())
	runtime.KeepAlive(object)
	runtime.KeepAlive(key)
	runtime.KeepAlive(value)
}

func emval_get_property(object, key Value) Value {
	result := valueFromEmval(cEmvalGetProperty(object.emvalHandle(), key.emvalHandle()))
	runtime.KeepAlive(object)
	runtime.KeepAlive(key)
	return result
}

func emval_delete(object, property Value) bool {
	deleted := cEmvalDelete(object.emvalHandle(), property.emvalHandle())
	runtime.KeepAlive(object)
	runtime.KeepAlive(property)
	return deleted
}

func emval_is_number(object Value) bool {
	result := cEmvalIsNumber(object.emvalHandle())
	runtime.KeepAlive(object)
	return result
}
func emval_is_string(object Value) bool {
	result := cEmvalIsString(object.emvalHandle())
	runtime.KeepAlive(object)
	return result
}
func emval_in(item, object Value) bool {
	result := cEmvalIn(item.emvalHandle(), object.emvalHandle())
	runtime.KeepAlive(item)
	runtime.KeepAlive(object)
	return result
}
func emval_typeof(value Value) Value {
	result := valueFromEmval(cEmvalTypeof(value.emvalHandle()))
	runtime.KeepAlive(value)
	return result
}
func emval_instanceof(object, constructor Value) bool {
	result := cEmvalInstanceof(object.emvalHandle(), constructor.emvalHandle())
	runtime.KeepAlive(object)
	runtime.KeepAlive(constructor)
	return result
}
func emval_length(object Value) int {
	result := int(cEmvalLength(object.emvalHandle()))
	runtime.KeepAlive(object)
	return result
}
func emval_as_double(v Value) float64 {
	result := cEmvalAsDouble(v.emvalHandle())
	runtime.KeepAlive(v)
	return result
}
func emval_as_string(v Value) string {
	result := cEmvalAsString(v.emvalHandle())
	runtime.KeepAlive(v)
	return result
}
func emval_equals(first, second Value) bool {
	result := cEmvalEquals(first.emvalHandle(), second.emvalHandle())
	runtime.KeepAlive(first)
	runtime.KeepAlive(second)
	return result
}

func emvalArgs(args *Value, nargs c.Int) []c.Ulong {
	if nargs == 0 {
		return nil
	}
	values := unsafe.Slice(args, int(nargs))
	// EM_VAL is a physical C pointer. Use the Emscripten C word here rather
	// than Go uintptr: J32 keeps Go uintptr at 64 bits, while C pointer arrays
	// still have four-byte elements.
	handles := make([]c.Ulong, len(values))
	for i := range values {
		handles[i] = c.Ulong(values[i].emvalHandle())
	}
	return handles
}

func emval_method_call(object Value, name *c.Char, nameLength c.SizeT, args *Value, nargs c.Int, err *c.Int) Value {
	handles := emvalArgs(args, nargs)
	var data *c.Ulong
	if len(handles) != 0 {
		data = &handles[0]
	}
	result := valueFromEmval(cEmvalMethodCall(object.emvalHandle(), name, nameLength, data, nargs, err))
	runtime.KeepAlive(object)
	runtime.KeepAlive(unsafe.Slice(args, int(nargs)))
	return result
}

func emval_call(fn Value, args *Value, nargs c.Int, kind c.Int, err *c.Int) Value {
	handles := emvalArgs(args, nargs)
	var data *c.Ulong
	if len(handles) != 0 {
		data = &handles[0]
	}
	result := valueFromEmval(cEmvalCall(fn.emvalHandle(), data, nargs, kind, err))
	runtime.KeepAlive(fn)
	runtime.KeepAlive(unsafe.Slice(args, int(nargs)))
	return result
}

func emval_copy_bytes(data []byte, value Value, toGo bool) (int, bool) {
	var ptr *c.Uint8T
	if len(data) != 0 {
		ptr = (*c.Uint8T)(unsafe.Pointer(&data[0]))
	}
	direction := c.Int(0)
	if toGo {
		direction = 1
	}
	n := int(cEmvalCopyBytes(ptr, c.SizeT(len(data)), value.emvalHandle(), direction))
	runtime.KeepAlive(data)
	runtime.KeepAlive(value)
	return n, n >= 0
}

func emval_dump(v Value) {
	cEmvalDump(v.emvalHandle())
	runtime.KeepAlive(v)
}

//go:linkname cEmvalGetGlobal C.llgo_emval_get_global
func cEmvalGetGlobal(name *c.Char) uintptr

//go:linkname cEmvalGetModuleProperty C.llgo_emval_get_module_property
func cEmvalGetModuleProperty(name *c.Char) uintptr

//go:linkname cEmvalInstallInvoke C.llgo_emval_install_invoke
func cEmvalInstallInvoke(handler emvalCallback)

//llgo:type C
type emvalCallback func(c.Ulong)

//go:linkname cEmvalTakePendingInvoke C.llgo_emval_take_pending_invoke
func cEmvalTakePendingInvoke() uintptr

//go:linkname cEmvalHasPendingInvoke C.llgo_emval_has_pending_invoke
func cEmvalHasPendingInvoke() bool

//go:linkname cEmvalDecref C.llgo_emval_decref
func cEmvalDecref(value uintptr)

//go:linkname cEmvalNewDouble C.llgo_emval_new_double
func cEmvalNewDouble(v float64) uintptr

//go:linkname cEmvalNewString C.llgo_emval_new_string
func cEmvalNewString(str *c.Char, length c.SizeT) uintptr

//go:linkname cEmvalNewObject C.llgo_emval_new_object
func cEmvalNewObject() uintptr

//go:linkname cEmvalNewArray C.llgo_emval_new_array
func cEmvalNewArray() uintptr

//go:linkname cEmvalSetProperty C.llgo_emval_set_property
func cEmvalSetProperty(object, key, value uintptr)

//go:linkname cEmvalGetProperty C.llgo_emval_get_property
func cEmvalGetProperty(object, key uintptr) uintptr

//go:linkname cEmvalDelete C.llgo_emval_delete
func cEmvalDelete(object, property uintptr) bool

//go:linkname cEmvalIsNumber C.llgo_emval_is_number
func cEmvalIsNumber(object uintptr) bool

//go:linkname cEmvalIsString C.llgo_emval_is_string
func cEmvalIsString(object uintptr) bool

//go:linkname cEmvalIn C.llgo_emval_in
func cEmvalIn(item, object uintptr) bool

//go:linkname cEmvalTypeof C.llgo_emval_typeof
func cEmvalTypeof(value uintptr) uintptr

//go:linkname cEmvalInstanceof C.llgo_emval_instanceof
func cEmvalInstanceof(object, constructor uintptr) bool

//go:linkname cEmvalLength C.llgo_emval_length
func cEmvalLength(object uintptr) float64

//go:linkname cEmvalAsDouble C.llgo_emval_as_double
func cEmvalAsDouble(v uintptr) float64

//go:linkname cEmvalAsString C.llgo_emval_as_string
func cEmvalAsString(v uintptr) string

//go:linkname cEmvalEquals C.llgo_emval_equals
func cEmvalEquals(first, second uintptr) bool

//go:linkname cEmvalMethodCall C.llgo_emval_method_call
func cEmvalMethodCall(object uintptr, name *c.Char, nameLength c.SizeT, args *c.Ulong, nargs c.Int, err *c.Int) uintptr

//go:linkname cEmvalCall C.llgo_emval_call
func cEmvalCall(fn uintptr, args *c.Ulong, nargs c.Int, kind c.Int, err *c.Int) uintptr

//go:linkname cEmvalCopyBytes C.llgo_emval_copy_bytes
func cEmvalCopyBytes(data *c.Uint8T, length c.SizeT, value uintptr, toGo c.Int) c.Double

//go:linkname cEmvalDump C.llgo_emval_dump
func cEmvalDump(v uintptr)

//export llgo_export_string_from
func llgo_export_string_from(data *c.Char, size c.Int) string {
	return c.GoString(data, size)
}
