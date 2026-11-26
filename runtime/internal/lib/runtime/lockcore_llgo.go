package runtime

import rt "github.com/xgo-dev/llgo/runtime/internal/runtime"

// LockCore pins the execution context that carries the Go scheduler to
// hardware core id.
//
// The meaning of a core - and of success - is target-defined; the runtime
// backends implement the honest variant for each class:
//
//   - esp32s31-idf: the FreeRTOS task hosting the fiber scheduler (the IDF
//     main task every goroutine runs on) is pinned to HP core id (0 or 1).
//     The pin reads back through the kernel's own getter before success is
//     reported. Migration materializes at the next scheduling point.
//   - non-IDF RV32 baremetal: single-hart scheduler on a target with no
//     second scheduler core - always locked, any id succeeds.
//   - everything else (hosted OSes, Wasm, non-RISC-V baremetal): not
//     implemented; LockCore reports an error instead of pretending.
func LockCore(id int) error {
	return rt.LockCore(id)
}
