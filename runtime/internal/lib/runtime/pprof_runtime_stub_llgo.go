//go:build darwin || linux

package runtime

import llrt "github.com/goplus/llgo/runtime/internal/runtime"

type StackRecord struct {
	Stack []uintptr
}

type MemProfileRecord struct {
	AllocBytes, FreeBytes     int64
	AllocObjects, FreeObjects int64
	Stack0                    [32]uintptr
}

func (r *MemProfileRecord) InUseBytes() int64 {
	return r.AllocBytes - r.FreeBytes
}

func (r *MemProfileRecord) InUseObjects() int64 {
	return r.AllocObjects - r.FreeObjects
}

func (r *MemProfileRecord) Stack() []uintptr {
	for i, pc := range r.Stack0 {
		if pc == 0 {
			return r.Stack0[:i]
		}
	}
	return r.Stack0[:]
}

// BlockProfileRecord is a minimal placeholder for runtime/pprof.
type BlockProfileRecord struct {
	Count  int64
	Cycles int64
	Stack  []uintptr
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	n, _ = llrt.MemProfile(nil, inuseZero)
	if len(p) < n {
		return n, false
	}
	if n == 0 {
		return 0, true
	}
	var records [64]llrt.MemProfileRecord
	if n > len(records) {
		return n, false
	}
	n, ok = llrt.MemProfile(records[:n], inuseZero)
	if !ok {
		return n, false
	}
	for i := 0; i < n; i++ {
		p[i] = MemProfileRecord{
			AllocBytes:   records[i].AllocBytes,
			FreeBytes:    records[i].FreeBytes,
			AllocObjects: records[i].AllocObjects,
			FreeObjects:  records[i].FreeObjects,
			Stack0:       records[i].Stack0,
		}
	}
	return n, true
}

func BlockProfile(p []BlockProfileRecord) (n int, ok bool) {
	return 0, false
}

func MutexProfile(p []BlockProfileRecord) (n int, ok bool) {
	return 0, false
}

func ThreadCreateProfile(p []StackRecord) (n int, ok bool) {
	return 0, false
}

func NumGoroutine() int {
	return 1
}

func SetCPUProfileRate(hz int) {}

func FuncForPC(pc uintptr) *Func {
	return funcForPC(pc)
}
