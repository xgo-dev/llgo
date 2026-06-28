/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	"github.com/goplus/llgo/runtime/internal/clite/tls"
)

type CallerFrame struct {
	PC        uintptr
	Entry     uintptr
	Function  string
	File      string
	Line      int
	StartLine int
}

var (
	callerFrameTLS = tls.Alloc[[]CallerFrame](nil)

	pcFramesMu sync.Mutex
	pcFrames   []*CallerFrame
)

var (
	runtimeCallersFrame = CallerFrame{Function: "runtime.Callers"}
	runtimeMainFrame    = CallerFrame{Function: "runtime.main"}
	runtimeGoexitFrame  = CallerFrame{Function: "runtime.goexit"}
)

func init() {
	pcFramesMu.Init(nil)
}

func PushCallerFrame(entry uintptr, name, file string, startLine int) int {
	frames := callerFrameTLS.Get()
	mark := len(frames)
	frames = append(frames, CallerFrame{
		PC:        entry,
		Entry:     entry,
		Function:  name,
		File:      file,
		Line:      startLine,
		StartLine: startLine,
	})
	callerFrameTLS.Set(frames)
	return mark
}

func SetCallerLine(line int) {
	frames := callerFrameTLS.Get()
	if line <= 0 || len(frames) == 0 {
		return
	}
	frames[len(frames)-1].Line = line
	callerFrameTLS.Set(frames)
}

func PopCallerFrame(mark int) {
	frames := callerFrameTLS.Get()
	if mark < 0 || mark > len(frames) {
		return
	}
	var zero CallerFrame
	for i := mark; i < len(frames); i++ {
		frames[i] = zero
	}
	callerFrameTLS.Set(frames[:mark])
}

func Caller(skip int) (CallerFrame, bool) {
	if skip < 0 {
		return CallerFrame{}, false
	}
	frames := callerFrameTLS.Get()
	if skip < len(frames) {
		return captureFrame(frames[len(frames)-1-skip], false), true
	}
	switch skip - len(frames) {
	case 0:
		return captureFrame(runtimeMainFrame, false), true
	case 1:
		return captureFrame(runtimeGoexitFrame, false), true
	default:
		return CallerFrame{}, false
	}
}

func Callers(skip int, pcs []uintptr) int {
	if skip < 0 {
		skip = 0
	}
	frames := callerFrameTLS.Get()
	n := 0
	add := func(frame CallerFrame) bool {
		if skip > 0 {
			skip--
			return true
		}
		if n >= len(pcs) {
			return false
		}
		pcs[n] = captureFrame(frame, true).PC
		n++
		return true
	}
	if !add(runtimeCallersFrame) {
		return n
	}
	for i := len(frames) - 1; i >= 0; i-- {
		if !add(frames[i]) {
			return n
		}
	}
	_ = add(runtimeMainFrame)
	_ = add(runtimeGoexitFrame)
	return n
}

func FrameForPC(pc uintptr) (CallerFrame, bool) {
	if pc == 0 {
		return CallerFrame{}, false
	}
	pcFramesMu.Lock()
	for _, frame := range pcFrames {
		if uintptr(unsafe.Pointer(frame)) == pc || uintptr(unsafe.Pointer(frame))+1 == pc {
			ret := *frame
			pcFramesMu.Unlock()
			return ret, true
		}
	}
	pcFramesMu.Unlock()
	return CallerFrame{}, false
}

func captureFrame(frame CallerFrame, callersPC bool) CallerFrame {
	rec := new(CallerFrame)
	*rec = frame
	pc := uintptr(unsafe.Pointer(rec))
	if callersPC {
		pc++
	}
	rec.PC = pc
	if rec.Entry == 0 {
		rec.Entry = pc
	}
	pcFramesMu.Lock()
	pcFrames = append(pcFrames, rec)
	pcFramesMu.Unlock()
	return *rec
}
