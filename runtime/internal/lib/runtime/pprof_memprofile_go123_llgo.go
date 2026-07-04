//go:build (darwin || linux) && go1.23

package runtime

import _ "unsafe"

type pprofMemProfileRecord struct {
	AllocBytes, FreeBytes     int64
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
	// Size dynamically with slack and retry: a fixed cap makes pprof's
	// retry-until-ok loop spin forever once the bucket set outgrows it.
	records := make([]MemProfileRecord, n+n/4+16)
	for attempt := 0; ; attempt++ {
		n, ok = MemProfile(records, inuseZero)
		if ok || attempt >= 3 {
			break
		}
		records = make([]MemProfileRecord, n+n/4+16)
	}
	if !ok || len(p) < n {
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
