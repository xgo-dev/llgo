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

const (
	callerPCMask     = uintptr(3)
	callerPCValue    = uintptr(1)
	callersPCValue   = uintptr(3)
	callerPCRingSize = 1024
)

type callerPCStore struct {
	next   uintptr
	frames [callerPCRingSize]CallerFrame
}

var (
	callerFrameTLS      = tls.Alloc[[]CallerFrame](nil)
	callerPCStoreTLS    = tls.Alloc[*callerPCStore](nil)
	callerLookupTLS     = tls.Alloc[bool](nil)
	panicCallerFrameTLS = tls.Alloc[[]CallerFrame](nil)
)

var (
	runtimeCallersFrame = CallerFrame{Function: "runtime.Callers"}
	runtimeMainFrame    = CallerFrame{Function: "runtime.main"}
	runtimeGoexitFrame  = CallerFrame{Function: "runtime.goexit"}
)

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

func SetCallerLookupLine(line int) {
	SetCallerLine(line)
	callerLookupTLS.Set(true)
}

func PopCallerFrame(mark int) {
	frames := callerFrameTLS.Get()
	oldLen := len(frames)
	if mark < 0 || mark > oldLen {
		return
	}
	var zero CallerFrame
	for i := mark; i < oldLen; i++ {
		frames[i] = zero
	}
	callerFrameTLS.Set(frames[:mark])

	panicFrames := panicCallerFrameTLS.Get()
	if len(panicFrames) > 0 && oldLen >= len(panicFrames) && mark <= len(panicFrames) {
		for i := range panicFrames {
			panicFrames[i] = zero
		}
		panicCallerFrameTLS.Clear()
	}
}

func SavePanicCallerFrames() {
	frames := callerFrameTLS.Get()
	if len(frames) == 0 {
		panicCallerFrameTLS.Clear()
		return
	}
	panicFrames := panicCallerFrameTLS.Get()
	if cap(panicFrames) < len(frames) {
		panicFrames = make([]CallerFrame, len(frames))
	} else {
		panicFrames = panicFrames[:len(frames)]
	}
	copy(panicFrames, frames)
	panicCallerFrameTLS.Set(panicFrames)
}

func Caller(skip int) (CallerFrame, bool) {
	if !takeCallerLookup() {
		return CallerFrame{}, false
	}
	if skip < 0 {
		return CallerFrame{}, false
	}
	frames := callerFrameTLS.Get()
	panicFrames := panicCallerFrameTLS.Get()
	if len(frames) == 0 {
		if skip < len(panicFrames) {
			return captureFrame(panicFrames[len(panicFrames)-1-skip], callerPCValue), true
		}
		return CallerFrame{}, false
	}
	if skip < len(frames) {
		return captureFrame(frames[len(frames)-1-skip], callerPCValue), true
	}
	if len(panicFrames) > len(frames) {
		idx := len(panicFrames) - 1 - skip
		if idx >= 0 {
			return captureFrame(panicFrames[idx], callerPCValue), true
		}
	}
	switch skip - len(frames) {
	case 0:
		return captureFrame(runtimeMainFrame, callerPCValue), true
	case 1:
		return captureFrame(runtimeGoexitFrame, callerPCValue), true
	default:
		return CallerFrame{}, false
	}
}

func Callers(skip int, pcs []uintptr) int {
	if !takeCallerLookup() {
		return 0
	}
	if skip < 0 {
		skip = 0
	}
	frames := callerFrameTLS.Get()
	if len(frames) == 0 {
		frames = panicCallerFrameTLS.Get()
	}
	if len(frames) == 0 {
		return 0
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
		pcs[n] = captureFrame(frame, callersPCValue).PC
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

func takeCallerLookup() bool {
	if !callerLookupTLS.Get() {
		return false
	}
	callerLookupTLS.Set(false)
	return true
}

func FrameForPC(pc uintptr) (CallerFrame, bool) {
	if pc&callerPCMask == 0 {
		return CallerFrame{}, false
	}
	store := callerPCStoreTLS.Get()
	if store == nil {
		return CallerFrame{}, false
	}
	addr := pc &^ callerPCMask
	if !store.contains(addr) {
		return CallerFrame{}, false
	}
	frame := *(*CallerFrame)(unsafe.Pointer(addr))
	return frame, true
}

func callerPCStoreForThread() *callerPCStore {
	store := callerPCStoreTLS.Get()
	if store == nil {
		store = new(callerPCStore)
		callerPCStoreTLS.Set(store)
	}
	return store
}

func captureFrame(frame CallerFrame, pcValue uintptr) CallerFrame {
	store := callerPCStoreForThread()
	idx := store.next & (callerPCRingSize - 1)
	store.next++
	store.frames[idx] = frame
	rec := &store.frames[idx]
	pc := uintptr(unsafe.Pointer(rec)) | pcValue
	rec.PC = pc
	if rec.Entry == 0 {
		rec.Entry = pc
	}
	return *rec
}

func (s *callerPCStore) contains(addr uintptr) bool {
	start := uintptr(unsafe.Pointer(&s.frames[0]))
	size := unsafe.Sizeof(s.frames)
	end := start + size
	if addr < start || addr >= end {
		return false
	}
	return (addr-start)%unsafe.Sizeof(s.frames[0]) == 0
}
