//go:build llgo && windows

package llgoext

import _ "unsafe"

//go:linkname stdcallEntry stdcall.llgo_ext_std_entry
func stdcallEntry(v int32) int32 { return v + 7 }

//go:linkname stdcallCallback stdcall.llgo_ext_std_callback
var stdcallCallback func(int32) int32

//go:linkname stdcallCounter stdcall.llgo_ext_counter
var stdcallCounter int32

//go:linkname invokeStdcallEntry C.llgo_ext_invoke_std
func invokeStdcallEntry(int32) int32

//llgo:link stdcallPlaceholder stdcall.llgo_ext_std_twice
func stdcallPlaceholder(int32) int32 { panic("stdcall placeholder body") }
