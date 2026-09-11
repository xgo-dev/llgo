//go:build wasm

package runtime

// The native hook walks frame pointers. WebAssembly records compiler caller
// frames, which must be captured before longjmp discards deferred-call frames.
// Install this in the core so an unrecovered panic has a Go traceback even
// when the program never imports the public runtime package.
func init() {
	PanicPCSnapshot = captureWasmPanicPCs
	PanicTraceback = printWasmPanicTraceback
}

func captureWasmPanicPCs() {
	p := panicPCStoreForG()
	n := Callers(1, p.pcs[:])
	StorePanicPCs(p.pcs[:n])
}

func printWasmPanicTraceback(_ int) bool {
	printed := false
	for _, pc := range PanicPCs() {
		frame, ok := FrameForPC(pc)
		if !ok || frame.Function == "" || frame.Function == "runtime.main" || frame.Function == "runtime.goexit" {
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
