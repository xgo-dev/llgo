// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
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
	if pc == 0 || name == "" {
		return
	}
	i := (pc >> 4) & (frameSymbolCacheSize - 1)
	frameSymbolCache[i] = frameSymbolCacheEntry{pc: pc, offset: offset, name: name}
}

type runtimeFuncInfoRecord struct {
	symbol uint32
	name   uint32
	file   uint32
	line   uint32
	column uint32
}

//go:linkname runtimeFuncInfoTable __llgo_funcinfo_table
var runtimeFuncInfoTable *runtimeFuncInfoRecord

//go:linkname runtimeFuncInfoStrings __llgo_funcinfo_strings
var runtimeFuncInfoStrings *c.Char

//go:linkname runtimeFuncInfoHash __llgo_funcinfo_hash
var runtimeFuncInfoHash *uint32

//go:linkname runtimeFuncInfoCount __llgo_funcinfo_count
var runtimeFuncInfoCount uintptr

//go:linkname runtimeFuncInfoHashMask __llgo_funcinfo_hash_mask
var runtimeFuncInfoHashMask uintptr

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

func cStringEqual(cstr *c.Char, s string) bool {
	return cStringCompare(cstr, s) == 0
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

func funcInfoCString(off uint32) *c.Char {
	if runtimeFuncInfoStrings == nil {
		return nil
	}
	return (*c.Char)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStrings), uintptr(off)))
}

func funcInfoAt(i uintptr) *runtimeFuncInfoRecord {
	size := unsafe.Sizeof(*runtimeFuncInfoTable)
	return (*runtimeFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), i*size))
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

func funcInfoForSymbol(symbol string) *runtimeFuncInfoRecord {
	if symbol == "" || runtimeFuncInfoTable == nil || runtimeFuncInfoCount == 0 {
		return nil
	}
	if runtimeFuncInfoHash != nil && runtimeFuncInfoHashMask != 0 {
		slot := funcInfoHashString(symbol) & runtimeFuncInfoHashMask
		for probes := uintptr(0); probes <= runtimeFuncInfoHashMask; probes++ {
			idx := *(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoHash), slot*unsafe.Sizeof(*runtimeFuncInfoHash)))
			if idx == 0 {
				return nil
			}
			if uintptr(idx) <= runtimeFuncInfoCount {
				rec := funcInfoAt(uintptr(idx) - 1)
				if cStringEqual(funcInfoCString(rec.symbol), symbol) {
					return rec
				}
			}
			slot = (slot + 1) & runtimeFuncInfoHashMask
		}
		return nil
	}
	for i := uintptr(0); i < runtimeFuncInfoCount; i++ {
		rec := funcInfoAt(i)
		if cStringEqual(funcInfoCString(rec.symbol), symbol) {
			return rec
		}
	}
	return nil
}

func applyFuncInfo(sym *pcSymbol, rawFunction string) {
	rec := funcInfoForSymbol(rawFunction)
	if rec == nil {
		public := publicFunctionName(rawFunction)
		if public != rawFunction {
			rec = funcInfoForSymbol(public)
		}
	}
	if rec == nil {
		return
	}
	if name := safeGoString(funcInfoCString(rec.name), ""); name != "" {
		sym.function = publicFunctionName(name)
	}
	if file := safeGoString(funcInfoCString(rec.file), ""); file != "" {
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

func frameSymbol(pc uintptr) pcSymbol {
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
	sym := addrInfoSymbol(pc)
	if pc == 0 {
		return sym
	}
	if sym.entry == 0 || pc > sym.entry {
		if callSym := addrInfoSymbol(pc - 1); callSym.ok {
			callSym.pc = pc
			return callSym
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
			f = &Func{entry: sym.entry, name: fn}
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
