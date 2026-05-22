// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
	llrt "github.com/goplus/llgo/runtime/internal/runtime"
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
		if fn, line, ok := llrt.MemProfileSyntheticFrame(pc); ok {
			ci.frames = append(ci.frames, Frame{
				PC:        pc,
				Function:  fn,
				Line:      line,
				startLine: line,
				Entry:     pc,
			})
			continue
		}
		info := &clitedebug.Info{}
		if clitedebug.Addrinfo(unsafe.Pointer(pc), info) == 0 {
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
		fn := safeGoString(info.Sname, "")
		if fn == "" {
			fn = unknownFunctionName(pc)
		}
		ci.frames = append(ci.frames, Frame{
			PC:        pc,
			Function:  fn,
			File:      "",
			Line:      0,
			startLine: 0,
			Entry:     uintptr(info.Saddr),
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
	opaque struct{} // unexported field to disallow conversions
}

func (f *Func) Name() string {
	panic("todo")
}

func (f *Func) FileLine(pc uintptr) (file string, line int) {
	var info clitedebug.Info
	if pc == 0 || clitedebug.Addrinfo(unsafe.Pointer(pc), &info) == 0 {
		return "", 0
	}
	return safeGoString(info.Fname, ""), 0
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
