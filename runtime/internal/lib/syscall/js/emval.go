//go:build js && wasm
// +build js,wasm

package js

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "github.com/xgo-dev/llgo/runtime/internal/embind"
)

var (
	valueGlobal         = emval_get_global(nil)
	objectConstructor   = emval_get_global(c.Str("Object"))
	stringConstructor   = emval_get_global(c.Str("String"))
	arrayConstructor    = emval_get_global(c.Str("Array"))
	functionConstructor = emval_get_global(c.Str("Function"))
)

var (
	valueUndefined = Value{2}
	valueNull      = Value{4}
	valueTrue      = Value{6}
	valueFalse     = Value{8}
	valueNaN       = emval_get_global(c.Str("NaN"))
	valueZero      = emval_new_double(0)
)

//go:linkname emval_get_global C.llgo_emval_get_global
func emval_get_global(name *c.Char) Value

//go:linkname emval_new_double C.llgo_emval_new_double
func emval_new_double(v float64) Value

//go:linkname emval_new_string C.llgo_emval_new_string
func emval_new_string(str *c.Char) Value

//go:linkname emval_new_object C.llgo_emval_new_object
func emval_new_object() Value

//go:linkname emval_new_array C.llgo_emval_new_array
func emval_new_array() Value

//go:linkname emval_set_property C.llgo_emval_set_property
func emval_set_property(object Value, key Value, value Value)

//go:linkname emval_get_property C.llgo_emval_get_property
func emval_get_property(object Value, key Value) Value

//go:linkname emval_delete C.llgo_emval_delete
func emval_delete(object Value, property Value) bool

//go:linkname emval_is_number C.llgo_emval_is_number
func emval_is_number(object Value) bool

//go:linkname emval_is_string C.llgo_emval_is_string
func emval_is_string(object Value) bool

//go:linkname emval_in C.llgo_emval_in
func emval_in(item Value, object Value) bool

//go:linkname emval_typeof C.llgo_emval_typeof
func emval_typeof(value Value) Value

//go:linkname emval_instanceof C.llgo_emval_instanceof
func emval_instanceof(object Value, constructor Value) bool

//go:linkname emval_as_double C.llgo_emval_as_double
func emval_as_double(v Value) float64

//go:linkname emval_as_string C.llgo_emval_as_string
func emval_as_string(v Value) string

//go:linkname emval_equals C.llgo_emval_equals
func emval_equals(first Value, second Value) bool

//go:linkname emval_method_call C.llgo_emval_method_call
func emval_method_call(object Value, name *c.Char, args *Value, nargs c.Int, err *c.Int) Value

// emval_call kind: FUNCTION = 0, CONSTRUCTOR = 1
//
//go:linkname emval_call C.llgo_emval_call
func emval_call(fn Value, args *Value, nargs c.Int, kind c.Int, err *c.Int) Value

//go:linkname emval_memory_view_uint8 C.llgo_emval_memory_view_uint8
func emval_memory_view_uint8(length c.SizeT, data *c.Uint8T) Value

//go:linkname emval_dump C.llgo_emval_dump
func emval_dump(v Value)

//export llgo_export_string_from
func llgo_export_string_from(data *c.Char, size c.Int) string {
	return c.GoString(data, size)
}
