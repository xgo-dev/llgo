//go:build llgo && wasm

package runtime

import "unsafe"

// Compiler instrumentation passes the parts of immutable string literals as
// scalar arguments. Passing Go strings directly uses indirect C ABI arguments
// and reserves aggregate temporaries in every instrumented caller's stack.
// Construct the strings here instead, so those slots are live only during the
// runtime update and disappear before the instrumented call runs.
// Keep the ABI boundary under LTO too: inlining would move the aggregate
// argument storage back into each instrumented caller.
//
//go:noinline
func PushCallerLocationFrameWasm(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int) int {
	return PushCallerLocationFrame(entry, wasmLiteralLocationString(nameData, nameLen), wasmLiteralLocationString(fileData, fileLen), line)
}

//go:noinline
func RecordCallerLocationWasm(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int) {
	RecordCallerLocation(entry, wasmLiteralLocationString(nameData, nameLen), wasmLiteralLocationString(fileData, fileLen), line)
}

//go:noinline
func RecordPanicLocationWasm(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int) {
	RecordPanicLocation(entry, wasmLiteralLocationString(nameData, nameLen), wasmLiteralLocationString(fileData, fileLen), line)
}

// The compiler supplies a literal's data and length together. The header lives
// on the stack and the resulting string points to the original static data.
func wasmLiteralLocationString(data *byte, length int) string {
	value := String{data: unsafe.Pointer(data), len: length}
	return *(*string)(noescape(unsafe.Pointer(&value)))
}
