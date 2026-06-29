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

import "unsafe"

type CallerFrame struct {
	PC        uintptr
	Entry     uintptr
	Function  string
	File      string
	Line      int
	StartLine int
}

var (
	callerFrames []CallerFrame
	pcFrames     []*CallerFrame

	panicCallerFrames []CallerFrame
)

var (
	runtimeCallersFrame = CallerFrame{Function: "runtime.Callers"}
	runtimeMainFrame    = CallerFrame{Function: "runtime.main"}
	runtimeGoexitFrame  = CallerFrame{Function: "runtime.goexit"}
)

func PushCallerFrame(entry uintptr, name, file string, startLine int) int {
	mark := len(callerFrames)
	callerFrames = append(callerFrames, CallerFrame{
		PC:        entry,
		Entry:     entry,
		Function:  name,
		File:      file,
		Line:      startLine,
		StartLine: startLine,
	})
	return mark
}

func SetCallerLine(line int) {
	if line <= 0 || len(callerFrames) == 0 {
		return
	}
	callerFrames[len(callerFrames)-1].Line = line
}

func PopCallerFrame(mark int) {
	oldLen := len(callerFrames)
	if mark < 0 || mark > len(callerFrames) {
		return
	}
	callerFrames = callerFrames[:mark]
	if len(panicCallerFrames) > 0 && oldLen > len(panicCallerFrames) && len(callerFrames) <= len(panicCallerFrames) {
		panicCallerFrames = nil
	}
}

func Caller(skip int) (CallerFrame, bool) {
	if skip < 0 {
		return CallerFrame{}, false
	}
	if skip >= 2 && len(panicCallerFrames) > 0 {
		idx := len(panicCallerFrames) - 1 - (skip - 2)
		if idx >= 0 {
			return captureFrame(panicCallerFrames[idx], false), true
		}
	}
	if skip < len(callerFrames) {
		return captureFrame(callerFrames[len(callerFrames)-1-skip], false), true
	}
	switch skip - len(callerFrames) {
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
	for i := len(callerFrames) - 1; i >= 0; i-- {
		if !add(callerFrames[i]) {
			return n
		}
	}
	_ = add(runtimeMainFrame)
	_ = add(runtimeGoexitFrame)
	return n
}

func SavePanicCallerFrames() {
	panicCallerFrames = append(panicCallerFrames[:0], callerFrames...)
}

func FrameForPC(pc uintptr) (CallerFrame, bool) {
	if pc == 0 {
		return CallerFrame{}, false
	}
	for _, frame := range pcFrames {
		if uintptr(unsafe.Pointer(frame)) == pc || uintptr(unsafe.Pointer(frame))+1 == pc {
			return *frame, true
		}
	}
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
	pcFrames = append(pcFrames, rec)
	return *rec
}
