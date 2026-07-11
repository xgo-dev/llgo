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

// trimMemProfileStack drops the allocator/runtime plumbing the physical
// capture recorded above the allocation site (AllocZ, the capture path
// itself) so record stacks start at user code like gc's.
func trimMemProfileStack(stk [32]uintptr) [32]uintptr {
	i := 0
	for i < len(stk) && stk[i] != 0 {
		if !isRuntimePlumbingFrame(stk[i]) {
			break
		}
		i++
	}
	if i == 0 {
		return stk
	}
	var out [32]uintptr
	copy(out[:], stk[i:])
	return out
}

// isRuntimePlumbingFrame reports whether pc belongs to LLGo runtime
// plumbing (allocator, capture hooks — including their __llgo_stub.
// wrappers, which is how a hook held in a function variable is entered).
func isRuntimePlumbingFrame(pc uintptr) bool {
	name := frameSymbol(pc - 1).function
	if name == "" {
		return false
	}
	const stub = "__llgo_stub."
	if hasPrefix(name, stub) {
		name = name[len(stub):]
	}
	return hasPrefix(name, "github.com/goplus/llgo/runtime/internal/") ||
		name == "runtime.captureMemProfileStack"
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	// Size dynamically with slack and retry: sampling between a sizing
	// call and its fill call can grow the bucket set, and a fixed cap
	// would make callers that retry-until-ok (pprof) loop forever.
	records := make([]llrt.MemProfileRecord, 64)
	for attempt := 0; ; attempt++ {
		n, ok = llrt.MemProfile(records, inuseZero)
		if ok || attempt >= 3 {
			break
		}
		records = make([]llrt.MemProfileRecord, n+n/4+16)
	}
	if !ok || len(p) < n {
		return n, false
	}
	if n == 0 {
		return 0, true
	}
	for i := 0; i < n; i++ {
		p[i] = MemProfileRecord{
			AllocBytes:   records[i].AllocBytes,
			FreeBytes:    records[i].FreeBytes,
			AllocObjects: records[i].AllocObjects,
			FreeObjects:  records[i].FreeObjects,
			Stack0:       trimMemProfileStack(records[i].Stack0),
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
	// Exact-entry lookup first, regardless of alignment: arm64 functions are
	// always 4-aligned, but amd64 function and stub entries need not be, and
	// an unaligned function-value pc must not be mistaken for a shadow-stack
	// synthetic marker (a synthetic pc simply misses this cheap search).
	if pc != 0 && runtimeFuncPCFramesBuilt() && runtimeFuncPCFramesPrebuilt {
		if idx := prebuiltFrameIndexForEntry(pc); idx >= 0 {
			if p := prebuiltFuncCacheLoad(idx); p != nil {
				fn := (*Func)(p)
				cacheFuncForPC(pc, fn)
				return fn
			}
			if sym, ok := pcSymbolForFuncInfoIndex(pc, pc, prebuiltFrame(idx).funcIndex); ok {
				// amd64 entries are byte-dense: a ret-1 style query can
				// coincide with another symbol's entry; statement records
				// win via the shared refinement rule (entry queries are
				// unaffected — sites never precede their function's entry).
				sym = refinePCSymbolLine(sym, pc)
				fn := newFuncForPC(pc, sym)
				prebuiltFuncCacheStore(idx, unsafe.Pointer(fn))
				cacheFuncForPC(pc, fn)
				return fn
			}
		}
	}
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
		// Mid-function pcs deserve statement lines, not the declaration
		// line (amd64 return addresses can be 4-aligned and land here).
		sym = refinePCSymbolLine(sym, pc)
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

// frameFuncForPC returns the *Func for a frame PC that Frames.Next already
// symbolized, going through the FuncForPC cache so repeated CallersFrames
// walks over the same PCs stop allocating a Func per frame.
func frameFuncForPC(pc uintptr, sym pcSymbol, name string) *Func {
	if fn := funcForPCLast.fn; fn != nil && funcForPCLast.pc == pc {
		return fn
	}
	set := &funcForPCCache[funcForPCCacheIndex(pc)]
	for i := 0; i < funcForPCCacheWays; i++ {
		if fn := set[i].fn; fn != nil && set[i].pc == pc {
			return fn
		}
	}
	fn := &Func{
		entry: sym.entry,
		name:  name,
		pc:    pc,
		file:  sym.file,
		line:  sym.line,
	}
	// FuncForPC's own constructor falls back to entry == pc; keep frames with
	// an unresolved entry out of the shared cache so a later FuncForPC(pc)
	// does not observe Entry() == 0.
	if sym.entry != 0 {
		cacheFuncForPC(pc, fn)
	}
	return fn
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

func CPUProfile() []byte {
	panic("CPUProfile no longer available")
}

func GoroutineProfile(p []StackRecord) (n int, ok bool) {
	return
}
