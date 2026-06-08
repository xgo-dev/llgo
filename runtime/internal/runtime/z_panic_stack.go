//go:build !baremetal && !wasm

package runtime

import clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"

var savedPanicPCs []uintptr

func savePanicStack(skip int) {
	var pcs [64]uintptr
	n := 0
	clitedebug.StackTrace(skip+2, func(fr *clitedebug.Frame) bool {
		if n >= len(pcs) {
			return false
		}
		pcs[n] = fr.PC
		n++
		return true
	})
	if cap(savedPanicPCs) < n {
		savedPanicPCs = make([]uintptr, n)
	} else {
		savedPanicPCs = savedPanicPCs[:n]
	}
	copy(savedPanicPCs, pcs[:n])
}

func savedPanicStack(pc []uintptr) int {
	n := copy(pc, savedPanicPCs)
	return n
}
