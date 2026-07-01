// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
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

const frameSymbolCacheSize = 128

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

//go:linkname runtimeFuncInfoStubIndexes __llgo_funcinfo_stub_indexes
var runtimeFuncInfoStubIndexes *uint32

//go:linkname runtimeFuncInfoStubCount __llgo_funcinfo_stub_count
var runtimeFuncInfoStubCount uintptr

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

type runtimeFuncPCFrame struct {
	entry     uintptr
	funcIndex uint32
	function  string
	file      string
	startLine int
}

type runtimePCPageIndex struct {
	base  uintptr
	pages []uint32
}

const runtimeFuncPCPageShift = 12

var runtimeFuncPCInitState uint32
var runtimeFuncPCFrames []runtimeFuncPCFrame
var runtimeFuncPCEntries []uintptr
var runtimeFuncPCIndex runtimePCPageIndex

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
				return
			}
		}
		c.Usleep(1)
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
	frames := make([]runtimeFuncPCFrame, 0, runtimeFuncInfoCount)
	entries := make([]uintptr, runtimeFuncInfoCount+1)
	for i := uintptr(0); i < runtimeFuncInfoCount; i++ {
		fn := funcInfoAt(i)
		pc := symbolPC(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
		if pc == 0 {
			continue
		}
		index := uint32(i + 1)
		function := publicFunctionName(funcInfoJoinName(fn.namePkg, fn.nameName))
		if function == "" {
			function = publicFunctionName(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
		}
		file := funcInfoJoinFile(fn.fileRoot, fn.fileName)
		frames = append(frames, runtimeFuncPCFrame{
			entry:     pc,
			funcIndex: index,
			function:  function,
			file:      file,
			startLine: int(fn.line),
		})
		if entries[index] == 0 || pc < entries[index] {
			entries[index] = pc
		}
	}
	// Closure stubs are an ABI adapter and may go away in a future closure
	// lowering. Keep the compatibility table light: it stores only target
	// funcinfo record indexes, and live stub PCs are resolved lazily here.
	if runtimeFuncInfoStubIndexes != nil && runtimeFuncInfoStubCount != 0 && runtimeFuncInfoStubCount <= runtimeFuncInfoCount {
		for i := uintptr(0); i < runtimeFuncInfoStubCount; i++ {
			index := funcInfoStubIndexAt(i)
			if index == 0 || uintptr(index) > runtimeFuncInfoCount {
				continue
			}
			fn := funcInfoAt(uintptr(index) - 1)
			symbol := funcInfoJoinName(fn.symbolPkg, fn.symbolName)
			if symbol == "" {
				continue
			}
			pc := symbolPC(runtimeClosureStubPrefix + symbol)
			if pc == 0 {
				continue
			}
			function := publicFunctionName(funcInfoJoinName(fn.namePkg, fn.nameName))
			if function == "" {
				function = publicFunctionName(symbol)
			}
			frames = append(frames, runtimeFuncPCFrame{
				entry:     pc,
				funcIndex: index,
				function:  function,
				file:      funcInfoJoinFile(fn.fileRoot, fn.fileName),
				startLine: int(fn.line),
			})
		}
	}
	sortRuntimeFuncPCFrames(frames)
	frames = uniqueRuntimeFuncPCFrames(frames)
	runtimeFuncPCFrames = frames
	runtimeFuncPCEntries = entries
	runtimeFuncPCIndex = buildRuntimeFuncPCIndex(frames)
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

func buildRuntimeFuncPCIndex(frames []runtimeFuncPCFrame) runtimePCPageIndex {
	if len(frames) == 0 {
		return runtimePCPageIndex{}
	}
	base := frames[0].entry >> runtimeFuncPCPageShift
	last := frames[len(frames)-1].entry >> runtimeFuncPCPageShift
	if last < base {
		return runtimePCPageIndex{}
	}
	npages := last - base + 2
	if npages > 1<<20 && npages > uintptr(len(frames))*64 {
		return runtimePCPageIndex{}
	}
	pages := make([]uint32, npages)
	next := 0
	for page := range pages {
		limit := (base + uintptr(page)) << runtimeFuncPCPageShift
		for next < len(frames) && frames[next].entry < limit {
			next++
		}
		pages[page] = uint32(next)
	}
	return runtimePCPageIndex{base: base, pages: pages}
}

func runtimeFuncPCFrameIndex(pc uintptr) int {
	frames := runtimeFuncPCFrames
	if len(frames) == 0 {
		return -1
	}
	lo, hi := 0, len(frames)
	if pages := runtimeFuncPCIndex.pages; len(pages) != 0 {
		page := pc >> runtimeFuncPCPageShift
		if page >= runtimeFuncPCIndex.base {
			off := page - runtimeFuncPCIndex.base
			if off < uintptr(len(pages)) {
				lo = int(pages[off])
				if off+1 < uintptr(len(pages)) {
					hi = int(pages[off+1])
				}
				if lo > 0 {
					lo--
				}
				if hi < len(frames) {
					hi++
				}
			}
		}
	}
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

func funcEntryForIndex(index uint32) uintptr {
	if index == 0 {
		return 0
	}
	initRuntimeFuncPCFrames()
	if uintptr(index) >= uintptr(len(runtimeFuncPCEntries)) {
		return 0
	}
	return runtimeFuncPCEntries[index]
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
	frame := runtimeFuncPCFrames[idx]
	return pcSymbol{
		pc:        pc,
		entry:     frame.entry,
		function:  frame.function,
		file:      frame.file,
		line:      frame.startLine,
		startLine: frame.startLine,
		ok:        true,
	}, true
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
		entry := funcEntryForIndex(rec.funcIndex)
		if entry == 0 {
			entry = symbolPC(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
		}
		if entry == 0 {
			sym := addrInfoSymbol(pc)
			entry = sym.entry
		}
		file := funcInfoPackedFile(rec.file)
		if file == "" {
			file = funcInfoJoinFile(fn.fileRoot, fn.fileName)
		}
		line := int(rec.line)
		if line == 0 {
			line = int(fn.line)
		}
		function := publicFunctionName(funcInfoJoinName(fn.namePkg, fn.nameName))
		if function == "" {
			function = publicFunctionName(funcInfoJoinName(fn.symbolPkg, fn.symbolName))
		}
		frames = append(frames, runtimePCLineFrame{
			pc:        pc,
			entry:     entry,
			function:  function,
			file:      file,
			line:      line,
			startLine: int(fn.line),
		})
	}
	sortRuntimePCLineFrames(frames)
	runtimePCLineFrames = uniqueRuntimePCLineFrames(frames)
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

func pcLineFrameForPC(pc, entry uintptr) (pcSymbol, bool) {
	if pc == 0 {
		return pcSymbol{}, false
	}
	initRuntimePCLineFrames()
	frames := runtimePCLineFrames
	if len(frames) == 0 {
		return pcSymbol{}, false
	}
	lo, hi := 0, len(frames)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].pc > pc {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo == 0 {
		return pcSymbol{}, false
	}
	frame := frames[lo-1]
	if entry != 0 && frame.entry != 0 && frame.entry != entry {
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
	if len(frames) == 0 {
		return pcSymbol{}, false
	}
	lo, hi := 0, len(frames)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if frames[mid].pc >= pc {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo >= len(frames) || frames[lo].pc != pc {
		return pcSymbol{}, false
	}
	frame := frames[lo]
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

func frameSymbol(pc uintptr) pcSymbol {
	if pc&3 != 0 {
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
		sym := frameSymbol(pc)
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
			f = &Func{
				entry: sym.entry,
				name:  fn,
				pc:    pc,
				file:  sym.file,
				line:  sym.line,
			}
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
