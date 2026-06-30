//go:build darwin || linux

package runtime

import (
	"unsafe"

	llrt "github.com/goplus/llgo/runtime/internal/runtime"
)

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

const funcForPCCacheSize = 1024

type funcForPCCacheEntry struct {
	pc uintptr
	fn *Func
}

var funcForPCCache [funcForPCCacheSize]funcForPCCacheEntry
var funcForPCLast funcForPCCacheEntry

func FuncForPC(pc uintptr) *Func {
	if fn := funcForPCLast.fn; fn != nil && funcForPCLast.pc == pc {
		return fn
	}
	entry := (*funcForPCCacheEntry)(unsafe.Add(
		unsafe.Pointer(&funcForPCCache[0]),
		funcForPCCacheIndex(pc)*unsafe.Sizeof(funcForPCCacheEntry{}),
	))
	if fn := entry.fn; fn != nil && entry.pc == pc {
		funcForPCLast = funcForPCCacheEntry{pc: pc, fn: fn}
		return fn
	}
	return funcForPCSlow(pc)
}

func funcForPCSlow(pc uintptr) *Func {
	if sym, ok := funcPCFrameForPC(pc); ok {
		fn := newFuncForPC(pc, sym)
		cacheFuncForPC(pc, fn)
		return fn
	}
	sym := frameSymbol(pc)
	fn := newFuncForPC(pc, sym)
	cacheFuncForPC(pc, fn)
	return fn
}

func newFuncForPC(pc uintptr, sym pcSymbol) *Func {
	if !sym.ok && sym.function == "" {
		return &Func{entry: pc, name: unknownFunctionName(pc), pc: pc}
	}
	name := sym.function
	if name == "" {
		name = unknownFunctionName(pc)
	}
	entry := sym.entry
	if entry == 0 {
		entry = pc
	}
	return &Func{
		entry: entry,
		name:  name,
		pc:    pc,
		file:  sym.file,
		line:  sym.line,
	}
}

func cacheFuncForPC(pc uintptr, fn *Func) {
	entry := (*funcForPCCacheEntry)(unsafe.Add(
		unsafe.Pointer(&funcForPCCache[0]),
		funcForPCCacheIndex(pc)*unsafe.Sizeof(funcForPCCacheEntry{}),
	))
	entry.fn = fn
	entry.pc = pc
	funcForPCLast = funcForPCCacheEntry{pc: pc, fn: fn}
}

func funcForPCCacheIndex(pc uintptr) uintptr {
	return (pc >> 4) & (funcForPCCacheSize - 1)
}
