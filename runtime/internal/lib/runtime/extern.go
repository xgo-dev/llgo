// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
	rtdebug "github.com/goplus/llgo/runtime/internal/runtime"
)

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	if frame, ok := rtdebug.Caller(skip); ok {
		file = frame.File
		line = frame.Line
		if file == "" {
			file = "???"
		}
		if line == 0 {
			line = 1
		}
		return frame.PC, file, line, true
	}
	var pcs [1]uintptr
	if Callers(skip+2, pcs[:]) < 1 {
		return 0, "", 0, false
	}
	sym := frameSymbol(pcs[0])
	file, line = sym.file, sym.line
	if file == "" {
		file = "???"
	}
	if line == 0 {
		line = 1
	}
	return pcs[0], file, line, true
}

func Callers(skip int, pc []uintptr) int {
	if n := rtdebug.Callers(skip, pc); n > 0 {
		return n
	}
	return callers(skip+1, pc)
}

func callers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	n := 0
	clitedebug.StackTrace(skip, func(fr *clitedebug.Frame) bool {
		if n >= len(pc) {
			return false
		}
		pc[n] = fr.PC
		recordFrameSymbol(fr.PC, fr.Offset, fr.Name)
		rtdebug.BindCallerLocation(fr.PC, fr.Name)
		n++
		return true
	})
	return n
}
