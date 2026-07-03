// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
	cliteos "github.com/goplus/llgo/runtime/internal/clite/os"
	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
	rtdebug "github.com/goplus/llgo/runtime/internal/runtime"
)

// Frames may be used to get function/file/line information for a
// slice of PC values returned by Callers.
type Frames struct {
	// callers is a slice of PCs that have not yet been expanded to frames.
	callers []uintptr

	nextPC uintptr

	// frames is a slice of Frames that have yet to be returned.
	frames     []Frame
	frameStore [2]Frame
}

// Frame is the information returned by Frames for each call frame.
type Frame struct {
	// PC is the program counter for the location in this frame.
	// For a frame that calls another frame, this will be the
	// program counter of a call instruction. Because of inlining,
	// multiple frames may have the same PC value, but different
	// symbolic information.
	PC uintptr

	// Func is the Func value of this call frame. This may be nil
	// for non-Go code or fully inlined functions.
	Func *Func

	// Function is the package path-qualified function name of
	// this call frame. If non-empty, this string uniquely
	// identifies a single function in the program.
	// This may be the empty string if not known.
	// If Func is not nil then Function == Func.Name().
	Function string

	// File and Line are the file name and line number of the
	// location in this frame. For non-leaf frames, this will be
	// the location of a call. These may be the empty string and
	// zero, respectively, if not known.
	File string
	Line int

	// startLine is the line number of the beginning of the function in
	// this frame. Specifically, it is the line number of the func keyword
	// for Go functions. Note that //line directives can change the
	// filename and/or line number arbitrarily within a function, meaning
	// that the Line - startLine offset is not always meaningful.
	//
	// This may be zero if not known.
	startLine int

	// Entry point program counter for the function; may be zero
	// if not known. If Func is not nil then Entry ==
	// Func.Entry().
	Entry uintptr

	// The runtime's internal view of the function. This field
	// is set (funcInfo.valid() returns true) only for Go functions,
	// not for C functions.
	funcInfo funcInfo
}

func safeGoString(s *c.Char, defaultStr string) string {
	if s == nil {
		return defaultStr
	}
	return c.GoString(s)
}

func uintptrHex(v uintptr) string {
	const hexdigits = "0123456789abcdef"
	var digits [16]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = hexdigits[v&0xf]
		v >>= 4
	}
	if i == len(digits) {
		i--
		digits[i] = '0'
	}
	out := make([]byte, 2+len(digits)-i)
	out[0] = '0'
	out[1] = 'x'
	copy(out[2:], digits[i:])
	return string(out)
}

func unknownFunctionName(pc uintptr) string {
	// Use a stable PC-based placeholder instead of a constant string.
	// Some stdlib code (e.g. testing cleanup stack mapping) compares function
	// names and can loop if too many frames share the same placeholder.
	return "pc=" + uintptrHex(pc)
}

type pcSymbol struct {
	pc        uintptr
	entry     uintptr
	function  string
	file      string
	line      int
	startLine int
	ok        bool
}

type frameSymbolCacheEntry struct {
	pc     uintptr
	offset uintptr
	name   string
}

// Sized for whole-stack re-walks: CallersFrames over a deep stack touches
// one entry per distinct return pc, and a 128-entry table thrashed on
// 32-frame stacks (adjacent return pcs share high bits, and repeated walks
// paid a pcline search per frame per walk).
const frameSymbolCacheSize = 4096

var frameSymbolCache [frameSymbolCacheSize]frameSymbolCacheEntry

func recordFrameSymbol(pc, offset uintptr, name string) {
	if pc == 0 || name == "" || isPCSiteSymbol(name) {
		return
	}
	i := (pc >> 4) & (frameSymbolCacheSize - 1)
	frameSymbolCache[i] = frameSymbolCacheEntry{pc: pc, offset: offset, name: name}
}

type runtimeFuncInfoRecord struct {
	symbolPkg  uint16
	symbolName uint16
	namePkg    uint16
	nameName   uint16
	fileRoot   uint16
	fileName   uint16
	line       uint32
}

//go:linkname runtimeFuncInfoTable __llgo_funcinfo_table
var runtimeFuncInfoTable *runtimeFuncInfoRecord

//go:linkname runtimeFuncInfoStrings __llgo_funcinfo_strings
var runtimeFuncInfoStrings *c.Char

//go:linkname runtimeFuncInfoStringOffsets __llgo_funcinfo_string_offsets
var runtimeFuncInfoStringOffsets *uint32

//go:linkname runtimeFuncInfoStringCount __llgo_funcinfo_string_count
var runtimeFuncInfoStringCount uintptr

//go:linkname runtimeFuncInfoHash __llgo_funcinfo_hash
var runtimeFuncInfoHash *uint16

//go:linkname runtimeFuncInfoCount __llgo_funcinfo_count
var runtimeFuncInfoCount uintptr

//go:linkname runtimeFuncInfoHashMask __llgo_funcinfo_hash_mask
var runtimeFuncInfoHashMask uintptr

type runtimeFuncInfoSymbolIndexRecord struct {
	symbolID  uint64
	funcIndex uint32
}

//go:linkname runtimeFuncInfoSymbolIndex __llgo_funcinfo_symbol_index
var runtimeFuncInfoSymbolIndex *runtimeFuncInfoSymbolIndexRecord

//go:linkname runtimeFuncInfoSymbolIndexCount __llgo_funcinfo_symbol_index_count
var runtimeFuncInfoSymbolIndexCount uintptr

//go:linkname runtimeFuncInfoStubIndexes __llgo_funcinfo_stub_indexes
var runtimeFuncInfoStubIndexes *uint32

//go:linkname runtimeFuncInfoStubCount __llgo_funcinfo_stub_count
var runtimeFuncInfoStubCount uintptr

type runtimeFuncInfoEntryRecord struct {
	pc       uintptr
	symbolID uint64
}

//go:linkname runtimeFuncInfoEntryStart __llgo_funcinfo_entry_start
var runtimeFuncInfoEntryStart *runtimeFuncInfoEntryRecord

//go:linkname runtimeFuncInfoEntryEnd __llgo_funcinfo_entry_end
var runtimeFuncInfoEntryEnd *runtimeFuncInfoEntryRecord

type runtimeFuncInfoStubSiteRecord struct {
	pc       uintptr
	symbolID uint64
}

//go:linkname runtimeFuncInfoStubSiteStart __llgo_funcinfo_stubsite_start
var runtimeFuncInfoStubSiteStart *runtimeFuncInfoStubSiteRecord

//go:linkname runtimeFuncInfoStubSiteEnd __llgo_funcinfo_stubsite_end
var runtimeFuncInfoStubSiteEnd *runtimeFuncInfoStubSiteRecord

type runtimePCLineRecord struct {
	id        uint64
	funcIndex uint32
	file      uint32
	line      uint32
}

//go:linkname runtimePCLineTable __llgo_pcline_table
var runtimePCLineTable *runtimePCLineRecord

//go:linkname runtimePCLineCount __llgo_pcline_count
var runtimePCLineCount uintptr

type runtimePCSiteRecord struct {
	pc uintptr
	id uint64
}

//go:linkname runtimePCSiteStart __llgo_pcsite_start
var runtimePCSiteStart *runtimePCSiteRecord

//go:linkname runtimePCSiteEnd __llgo_pcsite_end
var runtimePCSiteEnd *runtimePCSiteRecord

type runtimePCLineFrame struct {
	pc        uintptr
	entry     uintptr
	function  string
	file      string
	line      int
	startLine int
}

var runtimePCLineInitState uint32
var runtimePCLineFrames []runtimePCLineFrame
var runtimePCLineIndex runtimePCFindIndex

type runtimeFuncPCFrame struct {
	entry     uintptr
	funcIndex uint32
}

type runtimePCFindBucket struct {
	idx        uint32
	subbuckets [runtimePCFindSubbucket]uint16
}

type runtimePCFindIndex struct {
	base    uintptr
	buckets []runtimePCFindBucket
}

const (
	// Keep the lookup geometry aligned with Go's pclntab findfunc table:
	// 4096-byte buckets and 16 subbuckets. Go stores one-byte subbucket
	// deltas because its linker guarantees a 16-byte minimum function size;
	// LLGo has no minimum size for function entries and indexes call-site
	// records that can sit a few bytes apart, so it stores two-byte deltas.
	// A delta counts distinct PCs inside one 4096-byte bucket and therefore
	// can never exceed 4096, which makes uint16 overflow impossible and the
	// index unconditional. LLGo builds the index at first use after reading
	// DCE-safe entry PC sections, because the LLVM IR stage does not yet own
	// final text addresses the way cmd/link does for Go.
	runtimePCMinFuncSize    = uintptr(16)
	runtimePCFindBucketSize = uintptr(256) * runtimePCMinFuncSize
	runtimePCFindSubbucket  = 16
	runtimeFuncPCEntrySlack = 64
)

var runtimeFuncPCInitState uint32
var runtimeFuncPCFrames []runtimeFuncPCFrame
var runtimeFuncPCEntries []uintptr
var runtimeFuncPCIndex runtimePCFindIndex

const (
	runtimeFuncInfoInitUninit uint32 = iota
	runtimeFuncInfoInitDone
	runtimeFuncInfoInitBusy
	runtimeClosureStubPrefix       = "__llgo_stub."
	runtimePublicClosureStubPrefix = "_llgo_stub."
)

func hasStringPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func isPCSiteSymbol(name string) bool {
	for i := 0; i < len(name) && name[i] == '_'; i++ {
		if hasStringPrefix(name[i:], "__llgo_pcsite_") {
			return true
		}
	}
	return false
}

func publicFunctionName(name string) string {
	const commandLineArguments = "command-line-arguments."
	if hasStringPrefix(name, commandLineArguments) {
		return "main." + name[len(commandLineArguments):]
	}
	if len(name) > 0 && name[0] == '_' {
		name = name[1:]
	}
	return name
}

func cStringCompare(cstr *c.Char, s string) int {
	if cstr == nil {
		if s == "" {
			return 0
		}
		return -1
	}
	ptr := unsafe.Pointer(cstr)
	for i := 0; ; i++ {
		c := *(*byte)(unsafe.Add(ptr, i))
		if i == len(s) {
			if c == 0 {
				return 0
			}
			return 1
		}
		if c == 0 {
			return -1
		}
		if c < s[i] {
			return -1
		}
		if c > s[i] {
			return 1
		}
	}
}

func cStringLen(cstr *c.Char) int {
	if cstr == nil {
		return 0
	}
	ptr := unsafe.Pointer(cstr)
	for i := 0; ; i++ {
		if *(*byte)(unsafe.Add(ptr, i)) == 0 {
			return i
		}
	}
}

func cStringAppend(dst []byte, cstr *c.Char) []byte {
	if cstr == nil {
		return dst
	}
	ptr := unsafe.Pointer(cstr)
	for i := 0; ; i++ {
		c := *(*byte)(unsafe.Add(ptr, i))
		if c == 0 {
			return dst
		}
		dst = append(dst, c)
	}
}

func funcInfoCString(id uint16) *c.Char {
	if runtimeFuncInfoStrings == nil || runtimeFuncInfoStringOffsets == nil ||
		uintptr(id) >= runtimeFuncInfoStringCount {
		return nil
	}
	off := *(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStringOffsets), uintptr(id)*unsafe.Sizeof(*runtimeFuncInfoStringOffsets)))
	return (*c.Char)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStrings), uintptr(off)))
}

func funcInfoAt(i uintptr) *runtimeFuncInfoRecord {
	size := unsafe.Sizeof(*runtimeFuncInfoTable)
	return (*runtimeFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), i*size))
}

func pcLineAt(i uintptr) *runtimePCLineRecord {
	size := unsafe.Sizeof(*runtimePCLineTable)
	return (*runtimePCLineRecord)(unsafe.Add(unsafe.Pointer(runtimePCLineTable), i*size))
}

func funcInfoStubIndexAt(i uintptr) uint32 {
	size := unsafe.Sizeof(*runtimeFuncInfoStubIndexes)
	return *(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStubIndexes), i*size))
}

func funcInfoHashString(s string) uintptr {
	const (
		offset = uint32(2166136261)
		prime  = uint32(16777619)
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return uintptr(h)
}

func funcInfoSymbolEqual(rec *runtimeFuncInfoRecord, symbol string) bool {
	pkg := funcInfoCString(rec.symbolPkg)
	name := funcInfoCString(rec.symbolName)
	pkgLen := cStringLen(pkg)
	nameLen := cStringLen(name)
	if pkgLen == 0 {
		return cStringCompare(name, symbol) == 0
	}
	if len(symbol) != pkgLen+1+nameLen {
		return false
	}
	if cStringCompare(pkg, symbol[:pkgLen]) != 0 || symbol[pkgLen] != '.' {
		return false
	}
	return cStringCompare(name, symbol[pkgLen+1:]) == 0
}

func funcInfoJoinName(pkgID, nameID uint16) string {
	pkg := funcInfoCString(pkgID)
	name := funcInfoCString(nameID)
	pkgLen := cStringLen(pkg)
	nameLen := cStringLen(name)
	if pkgLen == 0 {
		return safeGoString(name, "")
	}
	if nameLen == 0 {
		return safeGoString(pkg, "")
	}
	buf := make([]byte, 0, pkgLen+1+nameLen)
	buf = cStringAppend(buf, pkg)
	buf = append(buf, '.')
	buf = cStringAppend(buf, name)
	return string(buf)
}

func funcInfoNameLen(pkgID, nameID uint16) int {
	pkgLen := cStringLen(funcInfoCString(pkgID))
	nameLen := cStringLen(funcInfoCString(nameID))
	if pkgLen == 0 {
		return nameLen
	}
	if nameLen == 0 {
		return pkgLen
	}
	return pkgLen + 1 + nameLen
}

func appendFuncInfoName(dst []byte, pkgID, nameID uint16) []byte {
	pkg := funcInfoCString(pkgID)
	name := funcInfoCString(nameID)
	pkgLen := cStringLen(pkg)
	nameLen := cStringLen(name)
	if pkgLen == 0 {
		return cStringAppend(dst, name)
	}
	if nameLen == 0 {
		return cStringAppend(dst, pkg)
	}
	dst = cStringAppend(dst, pkg)
	dst = append(dst, '.')
	return cStringAppend(dst, name)
}

func funcInfoJoinFile(rootID, nameID uint16) string {
	root := funcInfoCString(rootID)
	name := funcInfoCString(nameID)
	rootLen := cStringLen(root)
	nameLen := cStringLen(name)
	if rootLen == 0 {
		return safeGoString(name, "")
	}
	if nameLen == 0 {
		return safeGoString(root, "")
	}
	buf := make([]byte, 0, rootLen+nameLen)
	buf = cStringAppend(buf, root)
	buf = cStringAppend(buf, name)
	return string(buf)
}

func funcInfoPackedFile(file uint32) string {
	return funcInfoJoinFile(uint16(file>>16), uint16(file))
}

func maxFuncInfoSymbolLen() int {
	maxLen := 0
	for i := uintptr(0); i < runtimeFuncInfoCount; i++ {
		fn := funcInfoAt(i)
		if n := funcInfoNameLen(fn.symbolPkg, fn.symbolName); n > maxLen {
			maxLen = n
		}
	}
	return maxLen
}

func symbolPCBytes(name []byte) uintptr {
	if len(name) == 0 {
		return 0
	}
	name = append(name, 0)
	return uintptr(clitedebug.Symbol((*c.Char)(unsafe.Pointer(&name[0]))))
}

func symbolPCFuncInfoName(buf []byte, pkgID, nameID uint16) uintptr {
	name := appendFuncInfoName(buf[:0], pkgID, nameID)
	return symbolPCBytes(name)
}

func symbolPCPrefixedFuncInfoName(buf []byte, prefix string, pkgID, nameID uint16) uintptr {
	name := append(buf[:0], prefix...)
	name = appendFuncInfoName(name, pkgID, nameID)
	return symbolPCBytes(name)
}

func funcInfoFunctionName(fn *runtimeFuncInfoRecord) string {
	if fn == nil {
		return ""
	}
	if function := publicFunctionName(funcInfoJoinName(fn.namePkg, fn.nameName)); function != "" {
		return function
	}
	return publicFunctionName(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
}

func funcInfoFileName(fn *runtimeFuncInfoRecord) string {
	if fn == nil {
		return ""
	}
	return funcInfoJoinFile(fn.fileRoot, fn.fileName)
}

func funcInfoForSymbol(symbol string) *runtimeFuncInfoRecord {
	if symbol == "" || runtimeFuncInfoTable == nil || runtimeFuncInfoCount == 0 {
		return nil
	}
	if runtimeFuncInfoStrings == nil || runtimeFuncInfoStringOffsets == nil || runtimeFuncInfoCount > 1<<20 || runtimeFuncInfoHashMask > 1<<22 {
		return nil
	}
	if runtimeFuncInfoHash != nil && runtimeFuncInfoHashMask != 0 {
		slot := funcInfoHashString(symbol) & runtimeFuncInfoHashMask
		for probes := uintptr(0); probes <= runtimeFuncInfoHashMask; probes++ {
			idx := *(*uint16)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoHash), slot*unsafe.Sizeof(*runtimeFuncInfoHash)))
			if idx == 0 {
				return nil
			}
			if uintptr(idx) <= runtimeFuncInfoCount {
				rec := funcInfoAt(uintptr(idx) - 1)
				if funcInfoSymbolEqual(rec, symbol) {
					return rec
				}
			}
			slot = (slot + 1) & runtimeFuncInfoHashMask
		}
		return nil
	}
	for i := uintptr(0); i < runtimeFuncInfoCount; i++ {
		rec := funcInfoAt(i)
		if funcInfoSymbolEqual(rec, symbol) {
			return rec
		}
	}
	return nil
}

func funcInfoForRuntimeSymbol(symbol string) *runtimeFuncInfoRecord {
	if rec := funcInfoForSymbol(symbol); rec != nil {
		return rec
	}
	if hasStringPrefix(symbol, runtimeClosureStubPrefix) {
		return funcInfoForSymbol(symbol[len(runtimeClosureStubPrefix):])
	}
	if hasStringPrefix(symbol, runtimePublicClosureStubPrefix) {
		return funcInfoForSymbol(symbol[len(runtimePublicClosureStubPrefix):])
	}
	return nil
}

func applyFuncInfo(sym *pcSymbol, rawFunction string) {
	rec := funcInfoForRuntimeSymbol(rawFunction)
	if rec == nil {
		public := publicFunctionName(rawFunction)
		if public != rawFunction {
			rec = funcInfoForRuntimeSymbol(public)
		}
	}
	if rec == nil {
		return
	}
	if name := funcInfoJoinName(rec.namePkg, rec.nameName); name != "" {
		sym.function = publicFunctionName(name)
	}
	if file := funcInfoJoinFile(rec.fileRoot, rec.fileName); file != "" {
		if sym.file == "" {
			sym.file = file
		}
	}
	if rec.line != 0 {
		sym.startLine = int(rec.line)
		if sym.line == 0 {
			sym.line = int(rec.line)
		}
	}
	sym.ok = sym.ok || sym.function != "" || sym.file != ""
}

func cachedFrameSymbol(pc uintptr) pcSymbol {
	i := (pc >> 4) & (frameSymbolCacheSize - 1)
	entry := frameSymbolCache[i]
	if entry.pc != pc || entry.name == "" {
		return pcSymbol{pc: pc}
	}
	rawFn := entry.name
	if isPCSiteSymbol(rawFn) {
		return pcSymbol{pc: pc}
	}
	fn := publicFunctionName(rawFn)
	sym := pcSymbol{
		pc:       pc,
		entry:    pc - entry.offset,
		function: fn,
		ok:       fn != "" || entry.offset != 0,
	}
	applyFuncInfo(&sym, rawFn)
	return sym
}

func addrInfoSymbol(pc uintptr) pcSymbol {
	var info clitedebug.Info
	if clitedebug.Addrinfo(unsafe.Pointer(pc), &info) == 0 {
		return cachedFrameSymbol(pc)
	}
	rawFn := safeGoString(info.Sname, "")
	if isPCSiteSymbol(rawFn) {
		return pcSymbol{pc: pc}
	}
	if rawFn == "" {
		if sym := cachedFrameSymbol(pc); sym.ok {
			return sym
		}
	}
	fn := publicFunctionName(rawFn)
	sym := pcSymbol{
		pc:       pc,
		entry:    uintptr(info.Saddr),
		function: fn,
		ok:       fn != "" || info.Saddr != nil,
	}
	applyFuncInfo(&sym, rawFn)
	return sym
}

func initRuntimeFuncPCFrames() {
	if latomic.LoadUint32(&runtimeFuncPCInitState) == runtimeFuncInfoInitDone {
		return
	}
	initRuntimeFuncPCFramesSlow()
}

// runtimeFuncPCFramesBuilt reports whether the entry frame table has already
// been constructed, without triggering its construction.
func runtimeFuncPCFramesBuilt() bool {
	return latomic.LoadUint32(&runtimeFuncPCInitState) == runtimeFuncInfoInitDone
}

// Set LLGO_FUNCINFO_DEBUG=1 to print one line per lazily built runtime
// metadata table. This is how benchmarks and bug reports can tell whether a
// lookup used the compact find index or a degraded full-table fallback.
var runtimeFuncInfoDebugState uint32

var runtimeFuncPCFramesFromSites bool
var runtimeFuncPCStubsFromSites bool

func runtimeFuncInfoDebugEnabled() bool {
	state := latomic.LoadUint32(&runtimeFuncInfoDebugState)
	if state == 0 {
		state = 1
		if p := cliteos.Getenv(c.AllocaCStr("LLGO_FUNCINFO_DEBUG")); p != nil {
			if v := c.GoString(p); v != "" && v != "0" {
				state = 2
			}
		}
		latomic.StoreUint32(&runtimeFuncInfoDebugState, state)
	}
	return state == 2
}

func runtimeFuncInfoDebugSource(fromSites bool) string {
	if fromSites {
		return "sites"
	}
	return "dlsym"
}

func runtimeFuncInfoDebugIndex(index runtimePCFindIndex) string {
	if len(index.buckets) != 0 {
		return "built"
	}
	return "fallback"
}

func reportRuntimeFuncPCDebug() {
	if !runtimeFuncInfoDebugEnabled() {
		return
	}
	entrySrc := runtimeFuncInfoDebugSource(runtimeFuncPCFramesFromSites)
	stubSrc := runtimeFuncInfoDebugSource(runtimeFuncPCStubsFromSites)
	if runtimeFuncPCFramesPrebuilt {
		entrySrc = "prebuilt"
		stubSrc = "prebuilt"
	}
	frameCount := len(runtimeFuncPCFrames)
	if runtimeFuncPCFramesPrebuilt {
		frameCount = prebuiltFrameCount()
	}
	println("llgo funcinfo: func table frames=", frameCount,
		" buckets=", len(runtimeFuncPCIndex.buckets),
		" index=", runtimeFuncInfoDebugIndex(runtimeFuncPCIndex),
		" entries=", entrySrc,
		" stubs=", stubSrc)
}

func reportRuntimePCLineDebug() {
	if !runtimeFuncInfoDebugEnabled() {
		return
	}
	println("llgo funcinfo: pcline table frames=", len(runtimePCLineFrames),
		" buckets=", len(runtimePCLineIndex.buckets),
		" index=", runtimeFuncInfoDebugIndex(runtimePCLineIndex))
}

func initRuntimeFuncPCFramesSlow() {
	for {
		state := latomic.LoadUint32(&runtimeFuncPCInitState)
		switch state {
		case runtimeFuncInfoInitDone:
			return
		case runtimeFuncInfoInitUninit:
			if latomic.CompareAndSwapUint32(&runtimeFuncPCInitState, runtimeFuncInfoInitUninit, runtimeFuncInfoInitBusy) {
				initRuntimeFuncPCFramesOnce()
				latomic.StoreUint32(&runtimeFuncPCInitState, runtimeFuncInfoInitDone)
				reportRuntimeFuncPCDebug()
				return
			}
		}
		c.Usleep(1)
	}
}

// Prebuilt table format written into the entry-site section by the
// link-phase tool (chore/pclnpost -write). Layout, all little-endian,
// 8-byte aligned at the section start:
//
//	u64 magic          "LLGOFTB1"
//	u64 linkSectAddr   link-time vmaddr of this section (informational)
//	u64 base           runtime PC of the first table entry
//	u32 count          ftab entries incl. trailing sentinel
//	u32 bucketCount    findfunctab buckets (runtime uint16 layout)
//	count × {u32 entryOff /* relative to base */, u32 funcIndex}
//	bucketCount × {u32 idx; 16 × u16 subbuckets}
//
// The base slot is a live relocation: on Mach-O the rewriter splices it back
// into the dyld chained-fixup page chain (so dyld both pre-touches the
// table's pages at load and writes the slid address), and on non-PIE ELF the
// link-time value already equals the runtime address. Either way the slot
// holds a runtime PC — no slide arithmetic here.
//
// The tool sorts, deduplicates LTO inline copies against the symbol table,
// and normalizes entries to true symbol starts, so adopting the table also
// retires first-use sorting and the dlsym/stub fallbacks.
const runtimePrebuiltMagic = uint64(0x314254464F474C4C) // "LLGOFTB1" little-endian
// "LLGOFTB2": the entry section holds only a 32-byte redirect whose third
// word is the runtime address of the real blob, written into the (larger)
// stub section when the table outgrew the entry section.
const runtimePrebuiltRedirectMagic = uint64(0x324254464F474C4C)
const runtimePrebuiltHeaderSize = 8 + 8 + 8 + 4 + 4

type runtimePrebuiltFtabEntry struct {
	entryOff  uint32
	funcIndex uint32
}

var runtimeFuncPCFramesPrebuilt bool

// Zero-copy view of the prebuilt table: lookups binary-search the on-disk
// ftab directly; nothing is materialized at adoption time.
var runtimePrebuiltBase uintptr
var runtimePrebuiltFtab []runtimePrebuiltFtabEntry
var runtimePrebuiltEntriesOnce uint32

// runtimePrebuiltFuncs caches one *Func per ftab row for exact-entry
// lookups. The set-associative pc cache in FuncForPC thrashes once the live
// pc population outgrows it (thousands of distinct functions queried in a
// loop); this cache is keyed by table row, so batch workloads stay O(search)
// after the first pass regardless of scale. Same benign-race model as the
// pc cache: word-sized pointer stores of identical values.
var runtimePrebuiltFuncs []unsafe.Pointer

func prebuiltFuncCacheLoad(idx int) unsafe.Pointer {
	if idx < 0 || idx >= len(runtimePrebuiltFuncs) {
		return nil
	}
	return runtimePrebuiltFuncs[idx]
}

func prebuiltFuncCacheStore(idx int, fn unsafe.Pointer) {
	if idx < 0 || idx >= len(runtimePrebuiltFuncs) {
		return
	}
	runtimePrebuiltFuncs[idx] = fn
}

// prebuiltFrameIndexForEntry returns the ftab row whose entry is exactly pc,
// or -1.
func prebuiltFrameIndexForEntry(pc uintptr) int {
	idx := prebuiltFrameIndex(pc)
	if idx < 0 || prebuiltFrame(idx).entry != pc {
		return -1
	}
	return idx
}

// adoptPrebuiltFuncPCTable installs a zero-copy view over the prebuilt table
// if the entry section carries the magic header. Returns false to fall back
// to first-use construction.
func adoptPrebuiltFuncPCTable() bool {
	if runtimeFuncInfoEntryStart == nil || runtimeFuncInfoEntryEnd == nil {
		return false
	}
	start := uintptr(unsafe.Pointer(runtimeFuncInfoEntryStart))
	end := uintptr(unsafe.Pointer(runtimeFuncInfoEntryEnd))
	if end < start+runtimePrebuiltHeaderSize {
		return false
	}
	if *(*uint64)(unsafe.Pointer(start)) == runtimePrebuiltRedirectMagic {
		// Blob spilled into the stub section; the pointer slot is a live
		// relocation, so it already holds the runtime address.
		blob := uintptr(*(*uint64)(unsafe.Pointer(start + 16)))
		if blob == 0 || runtimeFuncInfoStubSiteStart == nil || runtimeFuncInfoStubSiteEnd == nil {
			return false
		}
		stubStart := uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteStart))
		stubEnd := uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteEnd))
		if blob != stubStart || stubEnd < stubStart {
			return false
		}
		start, end = blob, stubEnd
	}
	if *(*uint64)(unsafe.Pointer(start)) != runtimePrebuiltMagic {
		return false
	}
	base := uintptr(*(*uint64)(unsafe.Pointer(start + 16)))
	count := *(*uint32)(unsafe.Pointer(start + 24))
	bucketCount := *(*uint32)(unsafe.Pointer(start + 28))
	need := uintptr(runtimePrebuiltHeaderSize) + uintptr(count)*8 +
		uintptr(bucketCount)*unsafe.Sizeof(runtimePCFindBucket{})
	if count < 2 || end < start+need || uintptr(count) > runtimeFuncInfoCount*16+1 {
		return false
	}
	runtimePrebuiltBase = base
	runtimePrebuiltFtab = unsafe.Slice((*runtimePrebuiltFtabEntry)(unsafe.Pointer(start+runtimePrebuiltHeaderSize)), count)
	runtimeFuncPCIndex = runtimePCFindIndex{
		base:    base &^ (runtimePCFindBucketSize - 1),
		buckets: unsafe.Slice((*runtimePCFindBucket)(unsafe.Pointer(start+runtimePrebuiltHeaderSize+uintptr(count)*8)), bucketCount),
	}
	runtimeFuncPCFramesPrebuilt = true
	runtimeFuncPCFramesFromSites = true
	runtimeFuncPCStubsFromSites = true
	runtimePrebuiltFuncs = make([]unsafe.Pointer, count)
	return true
}

// prebuiltFrame returns the ftab row as a runtimeFuncPCFrame view.
func prebuiltFrame(i int) runtimeFuncPCFrame {
	e := runtimePrebuiltFtab[i]
	return runtimeFuncPCFrame{entry: runtimePrebuiltBase + uintptr(e.entryOff), funcIndex: e.funcIndex}
}

// prebuiltFrameCount excludes the trailing sentinel.
func prebuiltFrameCount() int {
	return len(runtimePrebuiltFtab) - 1
}

// materializePrebuiltEntries lazily builds the funcIndex -> entry map that
// only the pcline initializer consumes; FuncForPC lookups never pay for it.
// Two-phase (busy/done) so concurrent losers wait for the winner's store.
func materializePrebuiltEntries() {
	for {
		switch latomic.LoadUint32(&runtimePrebuiltEntriesOnce) {
		case 2:
			return
		case 0:
			if !latomic.CompareAndSwapUint32(&runtimePrebuiltEntriesOnce, 0, 1) {
				continue
			}
			entries := make([]uintptr, runtimeFuncInfoCount+1)
			for _, e := range runtimePrebuiltFtab[:prebuiltFrameCount()] {
				if e.funcIndex == 0 || uintptr(e.funcIndex) > runtimeFuncInfoCount {
					continue
				}
				pc := runtimePrebuiltBase + uintptr(e.entryOff)
				if entries[e.funcIndex] == 0 || pc < entries[e.funcIndex] {
					entries[e.funcIndex] = pc
				}
			}
			runtimeFuncPCEntries = entries
			latomic.StoreUint32(&runtimePrebuiltEntriesOnce, 2)
			return
		default:
			c.Usleep(1)
		}
	}
}

func initRuntimeFuncPCFramesOnce() {
	if runtimeFuncInfoTable == nil ||
		runtimeFuncInfoCount == 0 ||
		runtimeFuncInfoStrings == nil ||
		runtimeFuncInfoStringOffsets == nil {
		return
	}
	if runtimeFuncInfoCount > 1<<20 {
		return
	}
	if adoptPrebuiltFuncPCTable() {
		return
	}
	frames := make([]runtimeFuncPCFrame, 0, runtimeFuncInfoCount)
	entries := make([]uintptr, runtimeFuncInfoCount+1)
	frames, usedEntrySites := appendRuntimeFuncInfoEntryFrames(frames, entries)
	symbolBuf := []byte(nil)
	if !usedEntrySites {
		symbolBuf = make([]byte, 0, maxFuncInfoSymbolLen()+len(runtimeClosureStubPrefix)+1)
		for i := uintptr(0); i < runtimeFuncInfoCount; i++ {
			fn := funcInfoAt(i)
			pc := symbolPCFuncInfoName(symbolBuf, fn.symbolPkg, fn.symbolName)
			if pc == 0 {
				continue
			}
			index := uint32(i + 1)
			frames = append(frames, runtimeFuncPCFrame{
				entry:     pc,
				funcIndex: index,
			})
			if entries[index] == 0 || pc < entries[index] {
				entries[index] = pc
			}
		}
	}
	frames, usedStubSites := appendRuntimeFuncInfoStubSiteFrames(frames)
	// Closure stubs are an ABI adapter and may go away in a future closure
	// lowering. Keep the fallback compatibility table light: it stores only
	// target funcinfo record indexes. When the stub-site section is present it
	// is authoritative (linkers do not expose local stubs through dlsym), and
	// skipping the dlsym loop below matters: each dlsym is a dynamic-loader
	// query, and one query per stub used to dominate first-use latency.
	if !usedStubSites && runtimeFuncInfoStubIndexes != nil && runtimeFuncInfoStubCount != 0 && runtimeFuncInfoStubCount <= runtimeFuncInfoCount {
		if symbolBuf == nil {
			symbolBuf = make([]byte, 0, maxFuncInfoSymbolLen()+len(runtimeClosureStubPrefix)+1)
		}
		for i := uintptr(0); i < runtimeFuncInfoStubCount; i++ {
			index := funcInfoStubIndexAt(i)
			if index == 0 || uintptr(index) > runtimeFuncInfoCount {
				continue
			}
			fn := funcInfoAt(uintptr(index) - 1)
			pc := symbolPCPrefixedFuncInfoName(symbolBuf, runtimeClosureStubPrefix, fn.symbolPkg, fn.symbolName)
			if pc == 0 {
				continue
			}
			frames = append(frames, runtimeFuncPCFrame{
				entry:     pc,
				funcIndex: index,
			})
		}
	}
	sortRuntimeFuncPCFrames(frames)
	frames = uniqueRuntimeFuncPCFrames(frames)
	runtimeFuncPCFrames = frames
	runtimeFuncPCEntries = entries
	runtimeFuncPCIndex = buildRuntimeFuncPCIndex(frames)
	runtimeFuncPCFramesFromSites = usedEntrySites
	runtimeFuncPCStubsFromSites = usedStubSites
}

func appendRuntimeFuncInfoEntryFrames(frames []runtimeFuncPCFrame, entries []uintptr) ([]runtimeFuncPCFrame, bool) {
	if runtimeFuncInfoEntryStart == nil || runtimeFuncInfoEntryEnd == nil {
		return frames, false
	}
	start := uintptr(unsafe.Pointer(runtimeFuncInfoEntryStart))
	end := uintptr(unsafe.Pointer(runtimeFuncInfoEntryEnd))
	size := unsafe.Sizeof(*runtimeFuncInfoEntryStart)
	if end <= start || size == 0 || (end-start)%size != 0 {
		return frames, false
	}
	nsite := (end - start) / size
	if nsite > runtimeFuncInfoCount*16 || nsite > 1<<20 {
		return frames, false
	}
	used := false
	for i := uintptr(0); i < nsite; i++ {
		site := (*runtimeFuncInfoEntryRecord)(unsafe.Pointer(start + i*size))
		if site == nil || site.pc == 0 || site.symbolID == 0 {
			continue
		}
		funcIndex := funcInfoIndexForSymbolID(site.symbolID)
		if funcIndex == 0 || uintptr(funcIndex) > runtimeFuncInfoCount {
			continue
		}
		frames = append(frames, runtimeFuncPCFrame{
			entry:     site.pc,
			funcIndex: funcIndex,
		})
		if entries[funcIndex] == 0 || site.pc < entries[funcIndex] {
			entries[funcIndex] = site.pc
		}
		used = true
	}
	return frames, used
}

func appendRuntimeFuncInfoStubSiteFrames(frames []runtimeFuncPCFrame) ([]runtimeFuncPCFrame, bool) {
	if runtimeFuncInfoStubSiteStart == nil || runtimeFuncInfoStubSiteEnd == nil {
		return frames, false
	}
	start := uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteStart))
	end := uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteEnd))
	size := unsafe.Sizeof(*runtimeFuncInfoStubSiteStart)
	if end <= start || size == 0 || (end-start)%size != 0 {
		return frames, false
	}
	nsite := (end - start) / size
	if nsite > runtimeFuncInfoCount*16 || nsite > 1<<20 {
		return frames, false
	}
	used := false
	for i := uintptr(0); i < nsite; i++ {
		site := (*runtimeFuncInfoStubSiteRecord)(unsafe.Pointer(start + i*size))
		if site == nil || site.pc == 0 || site.symbolID == 0 {
			continue
		}
		funcIndex := funcInfoIndexForSymbolID(site.symbolID)
		if funcIndex == 0 || uintptr(funcIndex) > runtimeFuncInfoCount {
			continue
		}
		frames = append(frames, runtimeFuncPCFrame{
			entry:     site.pc,
			funcIndex: funcIndex,
		})
		used = true
	}
	return frames, used
}

func funcInfoIndexForSymbolID(symbolID uint64) uint32 {
	if symbolID == 0 || runtimeFuncInfoSymbolIndex == nil || runtimeFuncInfoSymbolIndexCount == 0 {
		return 0
	}
	if runtimeFuncInfoSymbolIndexCount > runtimeFuncInfoCount || runtimeFuncInfoSymbolIndexCount > 1<<20 {
		return 0
	}
	lo, hi := uintptr(0), runtimeFuncInfoSymbolIndexCount
	size := unsafe.Sizeof(*runtimeFuncInfoSymbolIndex)
	for lo < hi {
		mid := (lo + hi) >> 1
		rec := (*runtimeFuncInfoSymbolIndexRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoSymbolIndex), mid*size))
		if rec.symbolID >= symbolID {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo >= runtimeFuncInfoSymbolIndexCount {
		return 0
	}
	rec := (*runtimeFuncInfoSymbolIndexRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoSymbolIndex), lo*size))
	if rec.symbolID != symbolID || rec.funcIndex == 0 || uintptr(rec.funcIndex) > runtimeFuncInfoCount {
		return 0
	}
	return rec.funcIndex
}

func sortRuntimeFuncPCFrames(frames []runtimeFuncPCFrame) {
	if len(frames) < 2 {
		return
	}
	quickSortRuntimeFuncPCFrames(frames, 0, len(frames)-1)
}

func quickSortRuntimeFuncPCFrames(frames []runtimeFuncPCFrame, lo, hi int) {
	for hi-lo > 16 {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].entry < frames[lo].entry {
			frames[mid], frames[lo] = frames[lo], frames[mid]
		}
		if frames[hi].entry < frames[mid].entry {
			frames[hi], frames[mid] = frames[mid], frames[hi]
		}
		if frames[mid].entry < frames[lo].entry {
			frames[mid], frames[lo] = frames[lo], frames[mid]
		}
		pivot := frames[mid].entry
		i, j := lo, hi
		for {
			for frames[i].entry < pivot {
				i++
			}
			for frames[j].entry > pivot {
				j--
			}
			if i >= j {
				break
			}
			frames[i], frames[j] = frames[j], frames[i]
			i++
			j--
		}
		if j-lo < hi-i {
			quickSortRuntimeFuncPCFrames(frames, lo, j)
			lo = i
		} else {
			quickSortRuntimeFuncPCFrames(frames, i, hi)
			hi = j
		}
	}
	for i := lo + 1; i <= hi; i++ {
		x := frames[i]
		j := i - 1
		for j >= lo && frames[j].entry > x.entry {
			frames[j+1] = frames[j]
			j--
		}
		frames[j+1] = x
	}
}

func uniqueRuntimeFuncPCFrames(frames []runtimeFuncPCFrame) []runtimeFuncPCFrame {
	if len(frames) < 2 {
		return frames
	}
	out := frames[:1]
	for i := 1; i < len(frames); i++ {
		if frames[i].entry == out[len(out)-1].entry {
			out[len(out)-1] = frames[i]
			continue
		}
		out = append(out, frames[i])
	}
	return out
}

// buildRuntimeFuncPCIndex is the runtime counterpart of Go's linker-built
// findfunctab. The table shape and lookup behavior are Go-style; the build time
// differs because LLGo's final function PCs are discovered from associated
// sections after link/load instead of being sorted directly by cmd/link.
func buildRuntimeFuncPCIndex(frames []runtimeFuncPCFrame) runtimePCFindIndex {
	if len(frames) == 0 {
		return runtimePCFindIndex{}
	}
	if uintptr(len(frames)) > ^uintptr(0)>>1 {
		return runtimePCFindIndex{}
	}
	base := frames[0].entry &^ (runtimePCFindBucketSize - 1)
	last := frames[len(frames)-1].entry
	if last < base {
		return runtimePCFindIndex{}
	}
	nbuckets := (last-base)/runtimePCFindBucketSize + 1
	if nbuckets > 1<<20 && nbuckets > uintptr(len(frames))*64 {
		return runtimePCFindIndex{}
	}
	buckets := make([]runtimePCFindBucket, nbuckets)
	subSize := runtimePCFindBucketSize / runtimePCFindSubbucket
	for b := range buckets {
		bucketStart := base + uintptr(b)*runtimePCFindBucketSize
		baseIdx := runtimeFuncPCFrameIndexBinary(frames, bucketStart)
		if baseIdx < 0 {
			baseIdx = 0
		}
		if baseIdx > len(frames)-1 {
			baseIdx = len(frames) - 1
		}
		buckets[b].idx = uint32(baseIdx)
		for s := 0; s < runtimePCFindSubbucket; s++ {
			pc := bucketStart + uintptr(s)*subSize
			subIdx := runtimeFuncPCFrameIndexBinary(frames, pc)
			if subIdx < 0 {
				subIdx = 0
			}
			if subIdx > len(frames)-1 {
				subIdx = len(frames) - 1
			}
			// delta counts deduplicated PCs inside one bucket, so it is
			// bounded by the bucket size and always fits in uint16.
			delta := subIdx - baseIdx
			if delta < 0 || delta > 0xffff {
				return runtimePCFindIndex{}
			}
			buckets[b].subbuckets[s] = uint16(delta)
		}
	}
	return runtimePCFindIndex{base: base, buckets: buckets}
}

func runtimePCFindRange(index runtimePCFindIndex, n int, pc uintptr) (int, int, bool) {
	if n == 0 || len(index.buckets) == 0 || pc < index.base {
		return 0, 0, false
	}
	off := pc - index.base
	bucket := off / runtimePCFindBucketSize
	if bucket >= uintptr(len(index.buckets)) {
		return 0, 0, false
	}
	subSize := runtimePCFindBucketSize / runtimePCFindSubbucket
	sub := (off % runtimePCFindBucketSize) / subSize
	b := index.buckets[bucket]
	lo := int(b.idx) + int(b.subbuckets[sub])
	hi := n
	if sub+1 < runtimePCFindSubbucket {
		hi = int(b.idx) + int(b.subbuckets[sub+1])
	} else if bucket+1 < uintptr(len(index.buckets)) {
		hi = int(index.buckets[bucket+1].idx)
	}
	if lo > 0 {
		lo--
	}
	if hi < lo {
		hi = lo
	}
	hi += 2
	if hi > n {
		hi = n
	}
	if lo > n {
		lo = n
	}
	return lo, hi, true
}

// runtimeFuncPCFrameIndex mirrors runtime.findfunc: use the compact bucket
// table to jump near the containing function, then scan the sorted frame table
// inside that narrow range.
func runtimeFuncPCFrameIndex(pc uintptr) int {
	if runtimeFuncPCFramesPrebuilt {
		return prebuiltFrameIndex(pc)
	}
	frames := runtimeFuncPCFrames
	if len(frames) == 0 {
		return -1
	}
	if lo, hi, ok := runtimePCFindRange(runtimeFuncPCIndex, len(frames), pc); ok {
		for lo < hi {
			mid := int(uint(lo+hi) >> 1)
			if frames[mid].entry > pc {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		idx := lo - 1
		if idx < 0 || frames[idx].entry > pc {
			return -1
		}
		return idx
	}
	return runtimeFuncPCFrameIndexBinary(frames, pc)
}

func runtimeFuncPCFrameIndexBinary(frames []runtimeFuncPCFrame, pc uintptr) int {
	lo, hi := 0, len(frames)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].entry > pc {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	idx := lo - 1
	if idx < 0 {
		return -1
	}
	return idx
}

// prebuiltFrameIndex is runtimeFuncPCFrameIndex over the zero-copy ftab:
// bucket narrowing via the shared find index, then binary search on
// entryOff. Returns the index of the last entry with PC <= pc, or -1.
func prebuiltFrameIndex(pc uintptr) int {
	n := prebuiltFrameCount()
	if n <= 0 || pc < runtimePrebuiltBase {
		return -1
	}
	off := uint32(pc - runtimePrebuiltBase)
	lo, hi := 0, n
	if l, h, ok := runtimePCFindRange(runtimeFuncPCIndex, n, pc); ok {
		lo, hi = l, h
	}
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if runtimePrebuiltFtab[mid].entryOff > off {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	idx := lo - 1
	if idx < 0 || runtimePrebuiltFtab[idx].entryOff > off {
		return -1
	}
	return idx
}

func funcEntryForIndex(index uint32) uintptr {
	if index == 0 {
		return 0
	}
	initRuntimeFuncPCFrames()
	if runtimeFuncPCFramesPrebuilt {
		materializePrebuiltEntries()
	}
	if uintptr(index) >= uintptr(len(runtimeFuncPCEntries)) {
		return 0
	}
	return runtimeFuncPCEntries[index]
}

// coldFuncInfoEntryLookup resolves an exact function-entry PC by scanning the
// raw entry-site and stub-site sections, without building the sorted frame
// table and without any dynamic-loader query. Function values can point at
// either a real function entry or its closure stub, so both sections are
// scanned. The scan is linear, so it is capped: for larger binaries the
// dladdr cold path is cheaper than streaming the whole section.
const coldFuncInfoEntryScanLimit = 4096

// coldFuncInfoScanRange scans one {pc, symbolID} record section for the
// anchor nearest at-or-after pc within the warm path's entry slack (anchors
// are emitted from LLVM IR and land after the backend prologue). It returns
// the matched funcinfo index and delta, or (0, maxDelta) on miss.
func coldFuncInfoScanRange(start, end, size, pc uintptr, bestDelta uintptr) (uint32, uintptr) {
	if start == 0 || end <= start || size == 0 || (end-start)%size != 0 {
		return 0, bestDelta
	}
	nsite := (end - start) / size
	if nsite > coldFuncInfoEntryScanLimit || nsite > runtimeFuncInfoCount*16 {
		return 0, bestDelta
	}
	bestIndex := uint32(0)
	for i := uintptr(0); i < nsite; i++ {
		site := (*runtimeFuncInfoEntryRecord)(unsafe.Pointer(start + i*size))
		if site.symbolID == 0 || site.pc < pc {
			continue
		}
		delta := site.pc - pc
		if delta >= bestDelta {
			continue
		}
		funcIndex := funcInfoIndexForSymbolID(site.symbolID)
		if funcIndex == 0 || uintptr(funcIndex) > runtimeFuncInfoCount {
			continue
		}
		bestDelta = delta
		bestIndex = funcIndex
		if delta == 0 {
			break
		}
	}
	return bestIndex, bestDelta
}

// coldFuncPCLookupBudget grants a small number of table-free cold lookups per
// process; past that, building the sorted table amortizes better than more
// linear scans or dladdr calls.
var coldFuncPCLookupCount uint32

func coldFuncPCLookupBudget() bool {
	if prebuiltFuncPCTablePresent() {
		// The prebuilt table makes first-use initialization cheap; skip the
		// scan/dladdr path entirely and let the caller fall through to it.
		return false
	}
	return latomic.AddUint32(&coldFuncPCLookupCount, 1) <= 8
}

// prebuiltFuncPCTablePresent reports whether the entry section carries the
// link-phase prebuilt table, in which case the cold scan must not interpret
// its bytes as site records — and does not need to: adopting the prebuilt
// table is itself cheap.
func prebuiltFuncPCTablePresent() bool {
	if runtimeFuncInfoEntryStart == nil || runtimeFuncInfoEntryEnd == nil {
		return false
	}
	start := uintptr(unsafe.Pointer(runtimeFuncInfoEntryStart))
	end := uintptr(unsafe.Pointer(runtimeFuncInfoEntryEnd))
	if end < start+8 {
		return false
	}
	m := *(*uint64)(unsafe.Pointer(start))
	return m == runtimePrebuiltMagic || m == runtimePrebuiltRedirectMagic
}

// runtimeFuncInfoWarmSink keeps the warm-up loads observable.
var runtimeFuncInfoWarmSink byte

// init pre-warms the prebuilt function table, mirroring Go: Go's pclntab
// pages are touched by the runtime itself (traceback, GC) long before user
// code queries it, so its "first" FuncForPC never pays page-in. Touching the
// pages the lookup path reads — adopted blob, funcinfo records, string
// offsets, strings — moves first-touch page-in (plus, on darwin,
// code-signature validation) from the first user lookup to process startup
// (tens of µs once, on binaries that carry funcinfo tables). Without a
// prebuilt table everything stays lazy: first-use construction allocates,
// which has no place in init, and programs that never introspect pay
// nothing.
func init() {
	if !prebuiltFuncPCTablePresent() {
		return
	}
	initRuntimeFuncPCFrames() // zero-copy adoption, sub-µs
	if !runtimeFuncPCFramesPrebuilt {
		return
	}
	touch := func(base unsafe.Pointer, n uintptr) {
		if base == nil || n == 0 {
			return
		}
		const pageStep = 4096
		sink := runtimeFuncInfoWarmSink
		p := uintptr(base)
		for off := uintptr(0); off < n; off += pageStep {
			sink += *(*byte)(unsafe.Pointer(p + off))
		}
		sink += *(*byte)(unsafe.Pointer(p + n - 1))
		runtimeFuncInfoWarmSink = sink
	}
	// The adopted blob may live in the entry section or (spilled) in the
	// stub section; derive its range from the adopted views.
	if n := len(runtimePrebuiltFtab); n > 0 {
		touch(unsafe.Pointer(&runtimePrebuiltFtab[0]), uintptr(n)*8)
	}
	if n := len(runtimeFuncPCIndex.buckets); n > 0 {
		touch(unsafe.Pointer(&runtimeFuncPCIndex.buckets[0]),
			uintptr(n)*unsafe.Sizeof(runtimePCFindBucket{}))
	}
	touch(unsafe.Pointer(runtimeFuncInfoTable),
		runtimeFuncInfoCount*unsafe.Sizeof(runtimeFuncInfoRecord{}))
	touch(unsafe.Pointer(runtimeFuncInfoStringOffsets),
		runtimeFuncInfoStringCount*unsafe.Sizeof(uint32(0)))
	if runtimeFuncInfoStrings != nil && runtimeFuncInfoStringCount > 0 {
		last := uintptr(*(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStringOffsets),
			(runtimeFuncInfoStringCount-1)*unsafe.Sizeof(uint32(0)))))
		lastStr := funcInfoCString(uint16(runtimeFuncInfoStringCount - 1))
		touch(unsafe.Pointer(runtimeFuncInfoStrings), last+uintptr(cStringLen(lastStr))+1)
	}
	// The pcline table is on the Caller/CallersFrames path; building it
	// here keeps the first user lookup at steady-state cost (the build cost
	// scales with call-site count and showed up as a 200µs first Caller).
	initRuntimePCLineFrames()
	// One synthetic lookup warms the code paths themselves (allocator size
	// classes, lookup caches), not just the data pages.
	if prebuiltFrameCount() > 0 {
		frame := prebuiltFrame(0)
		if sym, ok := pcSymbolForFuncInfoIndex(frame.entry, frame.entry, frame.funcIndex); ok {
			runtimeFuncInfoWarmSink += byte(len(sym.function))
		}
	}
	// Write-warm the FuncForPC cache: its first stores otherwise take
	// zero-fill write faults, one per page, on the first few lookups.
	for i := 0; i < funcForPCCacheSets; i += 4096 / int(unsafe.Sizeof(funcForPCCache[0])) {
		funcForPCCache[i][0].pc = 0
	}
	funcForPCCache[funcForPCCacheSets-1][0].pc = 0
}

func coldFuncInfoEntryLookup(pc uintptr) (pcSymbol, bool) {
	if pc == 0 || prebuiltFuncPCTablePresent() {
		return pcSymbol{}, false
	}
	bestDelta := uintptr(runtimeFuncPCEntrySlack) + 1
	bestIndex := uint32(0)
	if runtimeFuncInfoEntryStart != nil && runtimeFuncInfoEntryEnd != nil {
		bestIndex, bestDelta = coldFuncInfoScanRange(
			uintptr(unsafe.Pointer(runtimeFuncInfoEntryStart)),
			uintptr(unsafe.Pointer(runtimeFuncInfoEntryEnd)),
			unsafe.Sizeof(*runtimeFuncInfoEntryStart), pc, bestDelta)
	}
	if bestDelta != 0 && runtimeFuncInfoStubSiteStart != nil && runtimeFuncInfoStubSiteEnd != nil {
		if idx, _ := coldFuncInfoScanRange(
			uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteStart)),
			uintptr(unsafe.Pointer(runtimeFuncInfoStubSiteEnd)),
			unsafe.Sizeof(*runtimeFuncInfoStubSiteStart), pc, bestDelta); idx != 0 {
			bestIndex = idx
		}
	}
	if bestIndex == 0 {
		return pcSymbol{}, false
	}
	return pcSymbolForFuncInfoIndex(pc, pc, bestIndex)
}

func funcPCFrameForPC(pc uintptr) (pcSymbol, bool) {
	if pc == 0 {
		return pcSymbol{}, false
	}
	initRuntimeFuncPCFrames()
	idx := runtimeFuncPCFrameIndex(pc)
	if idx < 0 {
		return pcSymbol{}, false
	}
	var frame runtimeFuncPCFrame
	if runtimeFuncPCFramesPrebuilt {
		frame = prebuiltFrame(idx)
	} else {
		frame = runtimeFuncPCFrames[idx]
	}
	return pcSymbolForFuncInfoIndex(pc, frame.entry, frame.funcIndex)
}

func funcPCFrameForEntryPC(pc uintptr) (pcSymbol, bool) {
	if pc == 0 {
		return pcSymbol{}, false
	}
	initRuntimeFuncPCFrames()
	if runtimeFuncPCFramesPrebuilt {
		// Prebuilt entries are true symbol starts (normalized against the
		// symbol table by the link-phase tool), so no slack is needed.
		idx := prebuiltFrameIndex(pc)
		if idx < 0 {
			return pcSymbol{}, false
		}
		frame := prebuiltFrame(idx)
		if frame.entry != pc {
			return pcSymbol{}, false
		}
		return pcSymbolForFuncInfoIndex(pc, pc, frame.funcIndex)
	}
	frames := runtimeFuncPCFrames
	if len(frames) == 0 {
		return pcSymbol{}, false
	}
	lo, hi := 0, len(frames)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].entry >= pc {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo >= len(frames) {
		return pcSymbol{}, false
	}
	frame := frames[lo]
	if frame.entry != pc && frame.entry-pc > runtimeFuncPCEntrySlack {
		return pcSymbol{}, false
	}
	return pcSymbolForFuncInfoIndex(pc, pc, frame.funcIndex)
}

func pcSymbolForFuncInfoIndex(pc, entry uintptr, funcIndex uint32) (pcSymbol, bool) {
	if funcIndex == 0 || uintptr(funcIndex) > runtimeFuncInfoCount {
		return pcSymbol{}, false
	}
	fn := funcInfoAt(uintptr(funcIndex) - 1)
	line := int(fn.line)
	return pcSymbol{
		pc:        pc,
		entry:     entry,
		function:  funcInfoFunctionName(fn),
		file:      funcInfoFileName(fn),
		line:      line,
		startLine: line,
		ok:        true,
	}, true
}

// refinePCSymbolLine upgrades a function-record symbol to statement
// granularity when a same-function pcline record covers pc, or the call
// instruction at pc-1 for return addresses (statement labels can sit
// exactly on a return address). Cross-function records are rejected by
// pcLineFrameForPC's entry check, so exact-entry queries keep their
// declaration line. This is the single place the pc/pc-1 statement rule
// lives; FuncForPC and CallersFrames must agree on it.
func refinePCSymbolLine(sym pcSymbol, pc uintptr) pcSymbol {
	if lineSym, ok := pcLineFrameForPC(pc, sym.entry); ok {
		return mergePCLineSymbol(sym, lineSym)
	}
	if lineSym, ok := pcLineFrameForPC(pc-1, sym.entry); ok {
		lineSym.pc = pc
		return mergePCLineSymbol(sym, lineSym)
	}
	return sym
}

func initRuntimePCLineFrames() {
	if latomic.LoadUint32(&runtimePCLineInitState) == runtimeFuncInfoInitDone {
		return
	}
	initRuntimePCLineFramesSlow()
}

func initRuntimePCLineFramesSlow() {
	for {
		state := latomic.LoadUint32(&runtimePCLineInitState)
		switch state {
		case runtimeFuncInfoInitDone:
			return
		case runtimeFuncInfoInitUninit:
			if latomic.CompareAndSwapUint32(&runtimePCLineInitState, runtimeFuncInfoInitUninit, runtimeFuncInfoInitBusy) {
				initRuntimePCLineFramesOnce()
				latomic.StoreUint32(&runtimePCLineInitState, runtimeFuncInfoInitDone)
				reportRuntimePCLineDebug()
				return
			}
		}
		c.Usleep(1)
	}
}

func initRuntimePCLineFramesOnce() {
	if runtimePCLineTable == nil ||
		runtimePCLineCount == 0 ||
		runtimePCSiteStart == nil ||
		runtimePCSiteEnd == nil ||
		runtimeFuncInfoTable == nil ||
		runtimeFuncInfoCount == 0 ||
		runtimeFuncInfoStrings == nil ||
		runtimeFuncInfoStringOffsets == nil {
		return
	}
	if runtimePCLineCount > 1<<20 || runtimePCLineCount > runtimeFuncInfoCount*1024 {
		return
	}
	start := uintptr(unsafe.Pointer(runtimePCSiteStart))
	end := uintptr(unsafe.Pointer(runtimePCSiteEnd))
	size := unsafe.Sizeof(*runtimePCSiteStart)
	if end <= start || size == 0 || (end-start)%size != 0 {
		return
	}
	nsite := (end - start) / size
	if nsite > runtimePCLineCount*1024 || nsite > 1<<22 {
		return
	}
	frames := make([]runtimePCLineFrame, 0, nsite)
	symbolBuf := make([]byte, 0, maxFuncInfoSymbolLen()+1)
	// Sites vastly outnumber distinct functions and files, so materialize the
	// per-function strings and entry PCs once and the packed file strings once
	// per file ID. Building them per site used to dominate first-use latency.
	type pcLineFuncInfo struct {
		entry    uintptr
		function string
		file     string
		line     int
		resolved bool
	}
	funcCache := make([]pcLineFuncInfo, runtimeFuncInfoCount+1)
	fileCache := make(map[uint32]string)
	for i := uintptr(0); i < nsite; i++ {
		site := (*runtimePCSiteRecord)(unsafe.Pointer(start + i*size))
		if site == nil || site.id == 0 || site.pc == 0 {
			continue
		}
		rec := pcLineInfoForID(site.id)
		if rec == nil || rec.funcIndex == 0 || uintptr(rec.funcIndex) > runtimeFuncInfoCount {
			continue
		}
		pc := site.pc
		fn := funcInfoAt(uintptr(rec.funcIndex) - 1)
		fc := &funcCache[rec.funcIndex]
		if !fc.resolved {
			fc.entry = funcEntryForIndex(rec.funcIndex)
			if fc.entry == 0 {
				fc.entry = symbolPCFuncInfoName(symbolBuf, fn.symbolPkg, fn.symbolName)
			}
			fc.function = publicFunctionName(funcInfoJoinName(fn.namePkg, fn.nameName))
			if fc.function == "" {
				fc.function = publicFunctionName(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
			}
			fc.file = funcInfoJoinFile(fn.fileRoot, fn.fileName)
			fc.line = int(fn.line)
			fc.resolved = true
		}
		entry := fc.entry
		if entry == 0 {
			sym := addrInfoSymbol(pc)
			entry = sym.entry
		}
		file := ""
		if rec.file != 0 {
			var ok bool
			if file, ok = fileCache[rec.file]; !ok {
				file = funcInfoPackedFile(rec.file)
				fileCache[rec.file] = file
			}
		}
		if file == "" {
			file = fc.file
		}
		line := int(rec.line)
		if line == 0 {
			line = fc.line
		}
		frames = append(frames, runtimePCLineFrame{
			pc:        pc,
			entry:     entry,
			function:  fc.function,
			file:      file,
			line:      line,
			startLine: fc.line,
		})
	}
	sortRuntimePCLineFrames(frames)
	frames = uniqueRuntimePCLineFrames(frames)
	runtimePCLineFrames = frames
	runtimePCLineIndex = buildRuntimePCLineIndex(frames)
}

func pcLineInfoForID(id uint64) *runtimePCLineRecord {
	lo, hi := uintptr(0), runtimePCLineCount
	for lo < hi {
		mid := (lo + hi) >> 1
		rec := pcLineAt(mid)
		if rec.id >= id {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo >= runtimePCLineCount {
		return nil
	}
	rec := pcLineAt(lo)
	if rec.id != id {
		return nil
	}
	return rec
}

func symbolPC(symbol string) uintptr {
	if symbol == "" {
		return 0
	}
	buf := make([]byte, len(symbol)+1)
	copy(buf, symbol)
	return uintptr(clitedebug.Symbol((*c.Char)(unsafe.Pointer(&buf[0]))))
}

func sortRuntimePCLineFrames(frames []runtimePCLineFrame) {
	if len(frames) < 2 {
		return
	}
	quickSortRuntimePCLineFrames(frames, 0, len(frames)-1)
}

func quickSortRuntimePCLineFrames(frames []runtimePCLineFrame, lo, hi int) {
	for hi-lo > 16 {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].pc < frames[lo].pc {
			frames[mid], frames[lo] = frames[lo], frames[mid]
		}
		if frames[hi].pc < frames[mid].pc {
			frames[hi], frames[mid] = frames[mid], frames[hi]
		}
		if frames[mid].pc < frames[lo].pc {
			frames[mid], frames[lo] = frames[lo], frames[mid]
		}
		pivot := frames[mid].pc
		i, j := lo, hi
		for {
			for frames[i].pc < pivot {
				i++
			}
			for frames[j].pc > pivot {
				j--
			}
			if i >= j {
				break
			}
			frames[i], frames[j] = frames[j], frames[i]
			i++
			j--
		}
		if j-lo < hi-i {
			quickSortRuntimePCLineFrames(frames, lo, j)
			lo = i
		} else {
			quickSortRuntimePCLineFrames(frames, i, hi)
			hi = j
		}
	}
	for i := lo + 1; i <= hi; i++ {
		x := frames[i]
		j := i - 1
		for j >= lo && frames[j].pc > x.pc {
			frames[j+1] = frames[j]
			j--
		}
		frames[j+1] = x
	}
}

func uniqueRuntimePCLineFrames(frames []runtimePCLineFrame) []runtimePCLineFrame {
	if len(frames) < 2 {
		return frames
	}
	out := frames[:1]
	for i := 1; i < len(frames); i++ {
		if frames[i].pc == out[len(out)-1].pc {
			out[len(out)-1] = frames[i]
			continue
		}
		out = append(out, frames[i])
	}
	return out
}

// buildRuntimePCLineIndex reuses the same Go-style bucket geometry for
// statement PC-line sites. Go stores dense per-function pcdata; LLGo keeps
// statement sites as a separate sorted table for now, but the hot PC lookup
// follows the same bucket/subbucket narrowing.
func buildRuntimePCLineIndex(frames []runtimePCLineFrame) runtimePCFindIndex {
	if len(frames) == 0 {
		return runtimePCFindIndex{}
	}
	base := frames[0].pc &^ (runtimePCFindBucketSize - 1)
	last := frames[len(frames)-1].pc
	if last < base {
		return runtimePCFindIndex{}
	}
	nbuckets := (last-base)/runtimePCFindBucketSize + 1
	if nbuckets > 1<<20 && nbuckets > uintptr(len(frames))*64 {
		return runtimePCFindIndex{}
	}
	buckets := make([]runtimePCFindBucket, nbuckets)
	subSize := runtimePCFindBucketSize / runtimePCFindSubbucket
	for b := range buckets {
		bucketStart := base + uintptr(b)*runtimePCFindBucketSize
		baseIdx := runtimePCLineFrameIndexBinary(frames, bucketStart, false)
		if baseIdx < 0 {
			baseIdx = 0
		}
		if baseIdx > len(frames)-1 {
			baseIdx = len(frames) - 1
		}
		buckets[b].idx = uint32(baseIdx)
		for s := 0; s < runtimePCFindSubbucket; s++ {
			pc := bucketStart + uintptr(s)*subSize
			subIdx := runtimePCLineFrameIndexBinary(frames, pc, false)
			if subIdx < 0 {
				subIdx = 0
			}
			if subIdx > len(frames)-1 {
				subIdx = len(frames) - 1
			}
			// delta counts deduplicated PCs inside one bucket, so it is
			// bounded by the bucket size and always fits in uint16.
			delta := subIdx - baseIdx
			if delta < 0 || delta > 0xffff {
				return runtimePCFindIndex{}
			}
			buckets[b].subbuckets[s] = uint16(delta)
		}
	}
	return runtimePCFindIndex{base: base, buckets: buckets}
}

func runtimePCLineFrameRange(pc uintptr) (int, int) {
	frames := runtimePCLineFrames
	if lo, hi, ok := runtimePCFindRange(runtimePCLineIndex, len(frames), pc); ok {
		return lo, hi
	}
	return 0, len(frames)
}

func runtimePCLineFrameIndex(pc uintptr, exact bool) int {
	frames := runtimePCLineFrames
	if len(frames) == 0 {
		return -1
	}
	lo, hi := runtimePCLineFrameRange(pc)
	return runtimePCLineFrameIndexInRange(frames, pc, exact, lo, hi)
}

func runtimePCLineFrameIndexBinary(frames []runtimePCLineFrame, pc uintptr, exact bool) int {
	return runtimePCLineFrameIndexInRange(frames, pc, exact, 0, len(frames))
}

func runtimePCLineFrameIndexInRange(frames []runtimePCLineFrame, pc uintptr, exact bool, lo, hi int) int {
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].pc > pc || (exact && frames[mid].pc == pc) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if exact {
		if lo >= len(frames) || frames[lo].pc != pc {
			return -1
		}
		return lo
	}
	idx := lo - 1
	if idx < 0 {
		return -1
	}
	return idx
}

func pcLineFrameForPC(pc, entry uintptr) (pcSymbol, bool) {
	if pc == 0 {
		return pcSymbol{}, false
	}
	initRuntimePCLineFrames()
	frames := runtimePCLineFrames
	idx := runtimePCLineFrameIndex(pc, false)
	if idx < 0 {
		return pcSymbol{}, false
	}
	frame := frames[idx]
	// When the caller knows the function entry, only accept a site from the
	// same function. A site with an unresolved entry cannot prove it belongs
	// to the queried function, so it must be rejected too — otherwise a
	// nearest-below hit from a neighboring function leaks its file/line.
	if entry != 0 && frame.entry != entry {
		return pcSymbol{}, false
	}
	return pcSymbol{
		pc:        pc,
		entry:     frame.entry,
		function:  frame.function,
		file:      frame.file,
		line:      frame.line,
		startLine: frame.startLine,
		ok:        true,
	}, true
}

func pcLineFrameForExactPC(pc uintptr) (pcSymbol, bool) {
	if pc == 0 {
		return pcSymbol{}, false
	}
	initRuntimePCLineFrames()
	frames := runtimePCLineFrames
	idx := runtimePCLineFrameIndex(pc, true)
	if idx < 0 {
		return pcSymbol{}, false
	}
	frame := frames[idx]
	return pcSymbol{
		pc:        pc,
		entry:     frame.entry,
		function:  frame.function,
		file:      frame.file,
		line:      frame.line,
		startLine: frame.startLine,
		ok:        true,
	}, true
}

func mergePCLineSymbol(base, line pcSymbol) pcSymbol {
	if line.entry == 0 {
		line.entry = base.entry
	}
	if line.function == "" {
		line.function = base.function
	}
	if line.file == "" {
		line.file = base.file
	}
	if line.line == 0 {
		line.line = base.line
	}
	if line.startLine == 0 {
		line.startLine = base.startLine
	}
	line.ok = true
	return line
}

// prebuiltTextContains reports whether pc falls inside the text range the
// prebuilt table covers (first entry .. end-of-text sentinel). When the
// link-phase rewrite was skipped (e.g. blob overflow) the first-use frame
// table provides the bounds instead — the pc-1 return-address convention
// and the walk bound must not silently turn off with the fast table, or
// frames get attributed to the next statement (a return address equals the
// following anchor exactly).
func prebuiltTextContains(pc uintptr) bool {
	if n := len(runtimePrebuiltFtab); n > 0 {
		return pc >= runtimePrebuiltBase && pc-runtimePrebuiltBase < uintptr(runtimePrebuiltFtab[n-1].entryOff)
	}
	if frames := runtimeFuncPCFrames; len(frames) > 0 {
		const lastFuncSlack = 1 << 20
		return pc >= frames[0].entry && pc < frames[len(frames)-1].entry+lastFuncSlack
	}
	return false
}

// frameSymbolResultCache memoizes full symbolization results per pc. Deep
// CallersFrames walks re-symbolize the same return addresses on every walk,
// and a miss below costs a dladdr; entries are immutable heap nodes so the
// benign-racy word-sized store never tears.
type frameSymbolResult struct {
	pc  uintptr
	sym pcSymbol
}

const frameSymbolResultCacheSize = 4096

var frameSymbolResultCache [frameSymbolResultCacheSize]*frameSymbolResult

// minLegalPC: nothing below the zero page can be code. Values under it are
// null-ish slots or shadow-stack synthetic markers, never return addresses.
const minLegalPC = 4096

func frameSymbol(pc uintptr) pcSymbol {
	if pc > minLegalPC {
		i := (pc >> 2) & uintptr(len(frameSymbolResultCache)-1)
		if e := frameSymbolResultCache[i]; e != nil && e.pc == pc {
			return e.sym
		}
		sym := frameSymbolUncached(pc)
		frameSymbolResultCache[i] = &frameSymbolResult{pc: pc, sym: sym}
		return sym
	}
	return frameSymbolUncached(pc)
}

func frameSymbolUncached(pc uintptr) pcSymbol {
	if pc&3 != 0 && !prebuiltTextContains(pc+1) {
		// Unaligned pcs outside the text range are shadow-stack synthetic
		// markers. Text-range pcs — return addresses minus one, and on
		// amd64 any instruction pc — flow through the normal lookups:
		// pcline nearest-below is byte-exact, no alignment games (rounding
		// by instruction size was an arm64-only assumption).
		if frame, ok := rtdebug.FrameForPC(pc); ok {
			return pcSymbol{
				pc:        pc,
				entry:     frame.Entry,
				function:  frame.Function,
				file:      frame.File,
				line:      frame.Line,
				startLine: frame.StartLine,
				ok:        true,
			}
		}
	}
	if pc == 0 {
		sym := addrInfoSymbol(pc)
		if frame, ok := rtdebug.FrameForPC(pc); ok {
			return pcSymbol{
				pc:        pc,
				entry:     frame.Entry,
				function:  frame.Function,
				file:      frame.File,
				line:      frame.Line,
				startLine: frame.StartLine,
				ok:        true,
			}
		}
		return sym
	}
	if lineSym, ok := pcLineFrameForExactPC(pc); ok {
		return lineSym
	}
	if lineSym, ok := pcLineFrameForExactPC(pc - 1); ok {
		lineSym.pc = pc
		return lineSym
	}
	// Resolve through the prebuilt table before touching the dynamic
	// loader: frames in packages without pcline records (the runtime
	// itself) otherwise cost a dladdr each on the first full-stack walk.
	if runtimeFuncPCFramesBuilt() {
		if funcSym, ok := funcPCFrameForPC(pc); ok {
			funcSym.pc = pc
			return refinePCSymbolLine(funcSym, pc)
		}
	}
	sym := addrInfoSymbol(pc)
	if lineSym, ok := pcLineFrameForPC(pc, sym.entry); ok {
		return mergePCLineSymbol(sym, lineSym)
	}
	if sym.entry == 0 || pc > sym.entry {
		if callSym := addrInfoSymbol(pc - 1); callSym.ok {
			if lineSym, ok := pcLineFrameForPC(pc-1, callSym.entry); ok {
				lineSym.pc = pc
				return mergePCLineSymbol(callSym, lineSym)
			}
			callSym.pc = pc
			return callSym
		}
	}
	if !sym.ok {
		if funcSym, ok := funcPCFrameForPC(pc); ok {
			return funcSym
		}
	}
	if frame, ok := rtdebug.FrameForPC(pc); ok {
		return pcSymbol{
			pc:        pc,
			entry:     frame.Entry,
			function:  frame.Function,
			file:      frame.File,
			line:      frame.Line,
			startLine: frame.StartLine,
			ok:        true,
		}
	}
	return sym
}

func (ci *Frames) Next() (frame Frame, more bool) {
	for len(ci.frames) < 2 {
		// Find the next frame.
		// We need to look for 2 frames so we know what
		// to return for the "more" result.
		if len(ci.callers) == 0 {
			break
		}
		var pc uintptr
		if ci.nextPC != 0 {
			pc, ci.nextPC = ci.nextPC, 0
		} else {
			pc, ci.callers = ci.callers[0], ci.callers[1:]
		}
		// Physical pcs are return addresses; attribute them to the call
		// instruction (Go's pc-1 convention). Statement labels can land
		// exactly on a return address — the next statement's marker sits
		// right after the call — so a raw-pc nearest-below lookup would
		// report the following line. Synthetic shadow-stack pcs live
		// outside the text range and keep raw-pc semantics.
		lookupPC := pc
		if prebuiltTextContains(pc) {
			lookupPC = pc - 1
		}
		sym := frameSymbol(lookupPC)
		sym.pc = pc
		if !sym.ok {
			ci.frames = append(ci.frames, Frame{
				PC:        pc,
				Function:  unknownFunctionName(pc),
				File:      "",
				Line:      0,
				startLine: 0,
				Entry:     0,
			})
			continue
		}
		fn := sym.function
		if fn == "" {
			fn = unknownFunctionName(pc)
		}
		var f *Func
		if sym.entry != 0 || fn != "" {
			f = frameFuncForPC(pc, sym, fn)
		}
		ci.frames = append(ci.frames, Frame{
			PC:        pc,
			Func:      f,
			Function:  fn,
			File:      sym.file,
			Line:      sym.line,
			startLine: sym.startLine,
			Entry:     sym.entry,
		})
	}

	// Pop one frame from the frame list. Keep the rest.
	// Avoid allocation in the common case, which is 1 or 2 frames.
	switch len(ci.frames) {
	case 0: // In the rare case when there are no frames at all, we return Frame{}.
		return
	case 1:
		frame = ci.frames[0]
		ci.frames = ci.frameStore[:0]
	case 2:
		frame = ci.frames[0]
		ci.frameStore[0] = ci.frames[1]
		ci.frames = ci.frameStore[:1]
	default:
		frame = ci.frames[0]
		ci.frames = ci.frames[1:]
	}
	more = len(ci.frames) > 0
	return
}

// CallersFrames takes a slice of PC values returned by Callers and
// prepares to return function/file/line information.
// Do not change the slice until you are done with the Frames.
func CallersFrames(callers []uintptr) *Frames {
	f := &Frames{callers: callers}
	f.frames = f.frameStore[:0]
	return f
}

// A Func represents a Go function in the running binary.
type Func struct {
	entry uintptr
	name  string
	pc    uintptr
	file  string
	line  int
}

func (f *Func) Name() string {
	if f == nil {
		return ""
	}
	return f.name
}

func (f *Func) Entry() uintptr {
	if f == nil {
		return 0
	}
	return f.entry
}

func (f *Func) FileLine(pc uintptr) (file string, line int) {
	if f != nil && f.pc == pc && (f.file != "" || f.line != 0) {
		return f.file, f.line
	}
	sym := frameSymbol(pc)
	return sym.file, sym.line
}

// moduledata records information about the layout of the executable
// image. It is written by the linker. Any changes here must be
// matched changes to the code in cmd/link/internal/ld/symtab.go:symtab.
// moduledata is stored in statically allocated non-pointer memory;
// none of the pointers here are visible to the garbage collector.
type moduledata struct {
	unused [8]byte
}

type funcInfo struct {
	*_func
	datap *moduledata
}
