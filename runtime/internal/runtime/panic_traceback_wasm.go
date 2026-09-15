//go:build wasm

package runtime

// The native hook walks frame pointers. WebAssembly instead keeps a logical
// shadow stack for the functions whose frames are observable. Install the
// printer in the core so an unrecovered panic has a Go traceback even when the
// program never imports the public runtime package.
func init() {
	PanicTraceback = printWasmPanicTraceback
}

func capturePanicCallerFrames(v any) {
	gp := getg()
	p := &gp.panicPCs
	// Match interface identity rather than Go equality so uncomparable panic
	// values can reuse the original prefix too. A different panic replaces it.
	value, recovered := efaceOf(&v), efaceOf(&p.recovered.value)
	same := p.recovered.frame != nil && p.recovered.frame == gp.recoverFrame &&
		value._type == recovered._type && value.data == recovered.data
	p.recovered = recoveredPanic{}
	if same {
		return
	}
	store := callerLocationStoreCurrent
	if store == nil {
		return
	}
	store.panicDepth = len(store.stack)
}

func printWasmPanicTraceback(_ int) bool {
	store := callerLocationStoreCurrent
	if store == nil {
		return false
	}
	frames := store.stack
	if snapshot := activePanicCallerFrames(); len(snapshot) != 0 {
		frames = snapshot
	}
	printed := false
	for i := len(frames) - 1; i >= 0; i-- {
		frame := frames[i]
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

func activePanicCallerFrames() []CallerFrame {
	store := callerLocationStoreCurrent
	if store == nil || store.panicDepth == 0 || store.panicDepth > len(store.stack) {
		return nil
	}
	return store.stack[:store.panicDepth]
}

func panicCallerSnapshotAvailable() bool {
	store := callerLocationStoreCurrent
	return store != nil && store.panicDepth != 0 && store.panicDepth <= len(store.stack)
}

func clearPanicCallerSnapshot() {
	if store := callerLocationStoreCurrent; store != nil {
		store.panicDepth = 0
	}
}
