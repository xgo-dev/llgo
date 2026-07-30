//go:build !llgo || !wasm || !llgo.wasm_resume || (!js && !wasip1) || (wasip1 && llgo.wasi_threads)

package runtime

import "unsafe"

// goroutineFunc is the target-independent entry ABI between compiler-generated
// goroutine wrappers and the selected stackful scheduler.
//
//llgo:type C
type goroutineFunc func(unsafe.Pointer) unsafe.Pointer
