//go:build (darwin || linux) && go1.23

package runtime

import (
	llrt "github.com/goplus/llgo/runtime/internal/runtime"
	_ "unsafe"
)

type pprofMemProfileRecord struct {
	AllocBytes, FreeBytes     int64
	AllocObjects, FreeObjects int64
	Stack                     []uintptr
}

//go:linkname pprof_memProfileInternal runtime.pprof_memProfileInternal
func pprof_memProfileInternal(p []pprofMemProfileRecord, inuseZero bool) (n int, ok bool) {
	suppressed := llrt.MemProfileSuppress()
	defer llrt.MemProfileRestoreSuppressed(suppressed)

	n, _ = MemProfile(nil, inuseZero)
	if len(p) < n {
		return n, false
	}
	if n == 0 {
		return 0, true
	}
	var records [64]MemProfileRecord
	if n > len(records) {
		return n, false
	}
	n, ok = MemProfile(records[:n], inuseZero)
	if !ok {
		return n, false
	}
	for i := 0; i < n; i++ {
		p[i] = pprofMemProfileRecord{
			AllocBytes:   records[i].AllocBytes,
			FreeBytes:    records[i].FreeBytes,
			AllocObjects: records[i].AllocObjects,
			FreeObjects:  records[i].FreeObjects,
			Stack:        pprofMemProfileStack(&records[i]),
		}
	}
	return n, true
}

func pprofMemProfileStack(r *MemProfileRecord) []uintptr {
	stack := r.Stack()
	if len(stack) == 0 {
		return nil
	}
	out := make([]uintptr, len(stack))
	copy(out, stack)
	return out
}
