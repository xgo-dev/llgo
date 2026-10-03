//go:build (darwin || linux || wasm || windows) && go1.27

package runtime

import _ "unsafe"

type pprofMemProfileRecord struct {
	ObjectSize                int64
	AllocObjects, FreeObjects int64
	Stack                     []uintptr
}

//go:linkname pprof_memProfileInternal runtime.pprof_memProfileInternal
func pprof_memProfileInternal(p []pprofMemProfileRecord, inuseZero bool) (n int, ok bool) {
	n, _ = MemProfile(nil, inuseZero)
	if len(p) < n {
		return n, false
	}
	if n == 0 {
		return 0, true
	}
	records := make([]MemProfileRecord, n+n/4+16)
	for attempt := 0; attempt < 4; attempt++ {
		n, ok = MemProfile(records, inuseZero)
		if ok {
			break
		}
		records = make([]MemProfileRecord, n+n/4+16)
	}
	if !ok || len(p) < n {
		return n, false
	}
	for i := 0; i < n; i++ {
		objectSize := int64(0)
		if records[i].AllocObjects != 0 {
			objectSize = records[i].AllocBytes / records[i].AllocObjects
		}
		p[i] = pprofMemProfileRecord{ObjectSize: objectSize, AllocObjects: records[i].AllocObjects, FreeObjects: records[i].FreeObjects, Stack: pprofMemProfileStack(&records[i])}
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
