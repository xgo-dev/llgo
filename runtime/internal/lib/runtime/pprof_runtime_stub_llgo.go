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

const funcForPCCacheSets = 1024
const funcForPCCacheWays = 4

type funcForPCCacheEntry struct {
	pc uintptr
	fn *Func
}

var funcForPCCache [funcForPCCacheSets][funcForPCCacheWays]funcForPCCacheEntry
var funcForPCCacheNext [funcForPCCacheSets]uint8
var funcForPCLast funcForPCCacheEntry

func FuncForPC(pc uintptr) *Func {
	if fn := funcForPCLast.fn; fn != nil && funcForPCLast.pc == pc {
		return fn
	}
	set := &funcForPCCache[funcForPCCacheIndex(pc)]
	for i := 0; i < funcForPCCacheWays; i++ {
		if fn := set[i].fn; fn != nil && set[i].pc == pc {
			funcForPCLast = funcForPCCacheEntry{pc: pc, fn: fn}
			return fn
		}
	}
	return funcForPCSlow(pc)
}

func funcForPCSlow(pc uintptr) *Func {
	if pc&3 != 0 {
		if sym := frameSymbol(pc); sym.ok {
			fn := newFuncForPC(pc, sym)
			cacheFuncForPC(pc, fn)
			return fn
		}
	} else if pc != 0 {
		// Cold fast path: before the entry frame table has been built, resolve
		// an exact function-entry PC without paying first-use table
		// construction. First a bounded linear scan of the raw entry-site
		// section (compile-time data, no dynamic-loader query), then one
		// dladdr as fallback. Requiring an exact entry match means a
		// stripped-local misattribution (dladdr returning the nearest
		// exported symbol) can never be accepted, so this path only ever
		// answers true function-value PCs. The path is intentionally capped:
		// each cold lookup costs microseconds, so after a handful of them the
		// sorted table is the cheaper answer and we fall through to build it.
		if !runtimeFuncPCFramesBuilt() && coldFuncPCLookupBudget() {
			if sym, ok := coldFuncInfoEntryLookup(pc); ok {
				fn := newFuncForPC(pc, sym)
				cacheFuncForPC(pc, fn)
				return fn
			}
			if sym := addrInfoSymbol(pc); sym.ok && sym.entry == pc && sym.function != "" {
				fn := newFuncForPC(pc, sym)
				cacheFuncForPC(pc, fn)
				return fn
			}
		}
		// Function-value PCs point at the real function entry. ELF funcinfo
		// entry-site anchors are emitted from LLVM IR and can land after the
		// backend prologue, so an exact entry PC may sort before its anchor.
		// Prefer the section table when it can match within the entry slack;
		// native symbol lookup is kept only as a fallback.
		if sym, ok := funcPCFrameForEntryPC(pc); ok {
			fn := newFuncForPC(pc, sym)
			cacheFuncForPC(pc, fn)
			return fn
		}
		if sym := addrInfoSymbol(pc); sym.ok && sym.entry == pc && sym.function != "" {
			fn := newFuncForPC(pc, sym)
			cacheFuncForPC(pc, fn)
			return fn
		}
	}
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
	setIndex := funcForPCCacheIndex(pc)
	set := &funcForPCCache[setIndex]
	for i := 0; i < funcForPCCacheWays; i++ {
		if set[i].fn == nil || set[i].pc == pc {
			set[i] = funcForPCCacheEntry{pc: pc, fn: fn}
			funcForPCLast = set[i]
			return
		}
	}
	way := funcForPCCacheNext[setIndex] & (funcForPCCacheWays - 1)
	funcForPCCacheNext[setIndex] = way + 1
	set[way] = funcForPCCacheEntry{pc: pc, fn: fn}
	funcForPCLast = set[way]
}

func funcForPCCacheIndex(pc uintptr) uintptr {
	return (pc >> 4) & (funcForPCCacheSets - 1)
}
