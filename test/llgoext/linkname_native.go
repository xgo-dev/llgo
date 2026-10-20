//go:build llgo && !wasm

package llgoext

import (
	"runtime"
	"unsafe"
)

// Native linkname prefixes are LLGo extensions. Ordinary Go symbol/storage
// semantics are exercised independently in test/go/linkname_storage_test.go.
//
//go:linkname nativeCounter C.llgo_ext_counter
var nativeCounter int32

//llgo:link nativeCounterAlias C.llgo_ext_counter
var nativeCounterAlias int32

//go:linkname nativeArray C.llgo_ext_array
var nativeArray [3]int32

//go:linkname nativePointer C.llgo_ext_pointer
var nativePointer *int32

//go:linkname nativeCallback C.llgo_ext_callback
var nativeCallback func(int32) int32

//go:linkname nativeAddresses C.llgo_ext_addresses
func nativeAddresses(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) int32

//go:linkname invokeNativeEntry C.llgo_ext_invoke
func invokeNativeEntry(int32) int32

//go:linkname invokeNativeThread C.llgo_ext_thread
func invokeNativeThread(int32) int32

//go:linkname invokeNativeCallbackEntry C.llgo_ext_invoke_callback
func invokeNativeCallbackEntry(int32) int32

// A real body defines a native entry with the same runtime boundary as //export.
//
//go:linkname nativeEntry C.llgo_ext_go_entry
func nativeEntry(v int32) int32 {
	runtime.GC()
	return v + 7
}

//go:linkname nativeCallbackEntry C.llgo_ext_go_callback
func nativeCallbackEntry(fn func(int32) int32, v int32) int32 { return fn(v) + 1 }

// The llgo:link directive keeps its existing placeholder-body binding semantics.
//
//llgo:link nativePlaceholder C.llgo_ext_invoke
func nativePlaceholder(int32) int32 { panic("binding placeholder executed") }

//go:linkname nativeGoProvider llgo_ext_plain_go
func nativeGoProvider(v int32) int32 { return v + 9 }

//llgo:link nativeGoPlaceholder llgo_ext_plain_go
func nativeGoPlaceholder(int32) int32 { panic("Go-ABI binding placeholder executed") }
