// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
	rtdebug "github.com/goplus/llgo/runtime/internal/runtime"
)

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	frame, ok := rtdebug.Caller(skip)
	if !ok {
		return 0, "", 0, false
	}
	return frame.PC, frame.File, frame.Line, true
}

func Callers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	if n := rtdebug.Callers(skip, pc); n > 0 {
		return n
	}
	n := 0
	clitedebug.StackTrace(skip, func(fr *clitedebug.Frame) bool {
		if n >= len(pc) {
			return false
		}
		pc[n] = fr.PC
		n++
		return true
	})
	return n
}
