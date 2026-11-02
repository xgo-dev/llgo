//go:build llgo && !wasm

package cgo

/*
#include "linkname_vars.h"
*/
import "C"

import "unsafe"

//go:linkname linkedScalar C.llgo_linkvar_scalar
var linkedScalar int32

//llgo:link linkedScalarAlias C.llgo_linkvar_scalar
var linkedScalarAlias int32

//go:linkname linkedArray C.llgo_linkvar_array
var linkedArray [3]int32

//go:linkname linkedPointer C.llgo_linkvar_pointer
var linkedPointer *int32

//go:linkname linkedCallback C.llgo_linkvar_callback
var linkedCallback func(int32) int32

//go:linkname linkedCallbacks C.llgo_linkvar_callbacks
var linkedCallbacks [2]func(int32) int32

func resetLinkedCallbacks() {
	C.llgo_linkvar_reset_callbacks()
}

//go:linkname linkedCallbackBox C.llgo_linkvar_callback_box
var linkedCallbackBox struct {
	Callback func(int32) int32
	Guard    uintptr
}

func linkedVarAddressesMatch() bool {
	return C.llgo_linkvar_addresses(
		unsafe.Pointer(&linkedScalar), unsafe.Pointer(&linkedArray),
		unsafe.Pointer(&linkedPointer), unsafe.Pointer(&linkedCallback)) != 0
}

func linkedVarReadScalar() int32 {
	return int32(C.llgo_linkvar_read_scalar())
}

func linkedVarWriteArray(value int32) {
	C.llgo_linkvar_write_array(C.int32_t(value))
}

func linkedVarCall(value int32) int32 {
	return int32(C.llgo_linkvar_call(C.int32_t(value)))
}
