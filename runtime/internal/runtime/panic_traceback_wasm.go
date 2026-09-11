//go:build wasm

package runtime

// The native hook walks frame pointers. WebAssembly instead keeps a logical
// shadow stack for the functions whose frames are observable. Install the
// printer in the core so an unrecovered panic has a Go traceback even when the
// program never imports the public runtime package.
func init() {
	PanicTraceback = printWasmPanicTraceback
}

func printWasmPanicTraceback(_ int) bool {
	store := callerLocationStoreCurrent
	if store == nil {
		return false
	}
	printed := false
	for i := len(store.stack) - 1; i >= 0; i-- {
		frame := store.stack[i]
		if frame.Function == "" || frame.Function == "runtime.main" || frame.Function == "runtime.goexit" {
			continue
		}
		if !printed {
			print("goroutine ", getg().goid, " [running]:\n")
			printed = true
		}
		print(frame.Function, "(...)\n\t")
		if frame.File == "" {
			print("???")
		} else {
			print(frame.File)
		}
		print(":", frame.Line, "\n")
	}
	return printed
}
