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

	clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"
	"github.com/goplus/llgo/runtime/internal/clite/tls"
)

type CallerFrame struct {
	PC        uintptr
	Entry     uintptr
	Function  string
	File      string
	Line      int
	StartLine int
	// captured memoizes the interned synthetic PC base (seq << 2) for this
	// exact frame content. It is cleared whenever the frame's line info
	// changes, so repeated Caller/Callers walks over an unchanged stack skip
	// the intern hash probe entirely. Only meaningful inside shadow-stack
	// slots; ignored by frame comparison and hashing.
	captured uintptr
}

const callerLocationLimit = 4096

const (
	callerPCMask     = uintptr(3)
	callerPCValue    = uintptr(1)
	callersPCValue   = uintptr(3)
	callerPCHashInit = 64
)

type callerLocationStore struct {
	frames        []CallerFrame
	stack         []CallerFrame
	synthetic     []CallerFrame
	syntheticHash []uintptr
	// Memoized synthetic PC bases for the static frames emitted around every
	// Callers walk. Per-store because synthetic sequences are per-store.
	callersPCBase uintptr
	mainPCBase    uintptr
	goexitPCBase  uintptr
}

var callerLocationTLS = tls.Alloc[*callerLocationStore](nil)

func PushCallerLocationFrame(entry uintptr, name, file string, startLine int) int {
	store := callerLocationStoreForThread()
	mark := len(store.stack)
	store.stack = append(store.stack, CallerFrame{
		PC:        entry,
		Entry:     entry,
		Function:  name,
		File:      file,
		Line:      startLine,
		StartLine: startLine,
	})
	return mark
}

func PopCallerLocationFrame(mark int) {
	store := callerLocationTLS.Get()
	if store == nil {
		return
	}
	oldLen := len(store.stack)
	if mark < 0 || mark > oldLen {
		return
	}
	var zero CallerFrame
	for i := mark; i < oldLen; i++ {
		store.stack[i] = zero
	}
	store.stack = store.stack[:mark]
}

func RecordCallerLocation(entry uintptr, name, file string, line int) {
	if entry == 0 || line <= 0 {
		return
	}
	updateCurrentFrame(entry, name, file, line)
	recordPCLocation(0, entry, name, file, line)
}

func RecordPanicLocation(entry uintptr, name, file string, line int) {
	if entry == 0 || line <= 0 {
		return
	}
	updateCurrentFrame(entry, name, file, line)
	recordPCLocation(0, entry, name, file, line)
}

func updateCurrentFrame(entry uintptr, name, file string, line int) {
	store := callerLocationTLS.Get()
	if store == nil {
		return
	}
	for i := len(store.stack) - 1; i >= 0; i-- {
		frame := &store.stack[i]
		if frame.Entry == entry {
			frame.Function = name
			frame.File = file
			// For one entry the instrumented name/file operands are
			// constants; only the line changes between call sites. Comparing
			// just the line keeps this per-call path free of string
			// comparisons while still invalidating the capture memo whenever
			// the frame content can differ.
			if frame.Line != line {
				frame.Line = line
				frame.captured = 0
			}
			return
		}
	}
}

func recordPCLocation(pc, entry uintptr, name, file string, line int) {
	store := callerLocationStoreForThread()
	for i := range store.frames {
		frame := &store.frames[i]
		if (pc != 0 && frame.PC == pc) || (pc == 0 && frame.PC == 0 && frame.Entry == entry) {
			frame.PC = pc
			frame.Entry = entry
			frame.Function = name
			frame.File = file
			frame.Line = line
			return
		}
	}
	if len(store.frames) >= callerLocationLimit {
		copy(store.frames, store.frames[1:])
		store.frames[len(store.frames)-1] = CallerFrame{}
		store.frames = store.frames[:len(store.frames)-1]
	}
	store.frames = append(store.frames, CallerFrame{
		PC:       pc,
		Entry:    entry,
		Function: name,
		File:     file,
		Line:     line,
	})
}

func Caller(skip int) (CallerFrame, bool) {
	if skip < 0 {
		return CallerFrame{}, false
	}
	store := callerLocationTLS.Get()
	if store == nil || len(store.stack) == 0 {
		return CallerFrame{}, false
	}
	if skip < len(store.stack) {
		return store.captureFrameAt(&store.stack[len(store.stack)-1-skip], callerPCValue), true
	}
	switch skip - len(store.stack) {
	case 0:
		return store.captureFrame(runtimeMainFrame, callerPCValue), true
	case 1:
		return store.captureFrame(runtimeGoexitFrame, callerPCValue), true
	default:
		return CallerFrame{}, false
	}
}

func Callers(skip int, pcs []uintptr) int {
	if len(pcs) == 0 {
		return 0
	}
	if skip < 0 {
		skip = 0
	}
	store := callerLocationTLS.Get()
	if store == nil || len(store.stack) == 0 {
		return 0
	}
	// Unrolled emit sequence: no closure so nothing escapes, and frames the
	// skip count drops are never captured at all.
	n := 0
	if skip > 0 {
		skip--
	} else {
		pcs[n] = store.staticPC(runtimeCallersFrame, &store.callersPCBase, callersPCValue)
		n++
	}
	for i := len(store.stack) - 1; i >= 0; i-- {
		if skip > 0 {
			skip--
			continue
		}
		if n >= len(pcs) {
			return n
		}
		pcs[n] = store.capturePC(&store.stack[i], callersPCValue)
		n++
	}
	if skip > 0 {
		skip--
	} else {
		if n >= len(pcs) {
			return n
		}
		pcs[n] = store.staticPC(runtimeMainFrame, &store.mainPCBase, callersPCValue)
		n++
	}
	if skip <= 0 && n < len(pcs) {
		pcs[n] = store.staticPC(runtimeGoexitFrame, &store.goexitPCBase, callersPCValue)
		n++
	}
	return n
}

func SavePanicCallerFrames() {
}

func BindCallerLocation(pc uintptr, rawName string) {
	store := callerLocationTLS.Get()
	if store == nil || pc == 0 {
		return
	}
	if frame, ok := callerLocationByName(store, rawName); ok {
		bindCallerLocationPC(pc, frame)
		return
	}
}

var (
	runtimeCallersFrame = CallerFrame{Function: "runtime.Callers"}
	runtimeMainFrame    = CallerFrame{Function: "runtime.main"}
	runtimeGoexitFrame  = CallerFrame{Function: "runtime.goexit"}
)

func callerLocationByName(store *callerLocationStore, rawName string) (CallerFrame, bool) {
	if rawName == "" {
		return CallerFrame{}, false
	}
	name := normalizeRuntimeFuncName(rawName)
	for i := len(store.frames) - 1; i >= 0; i-- {
		frame := store.frames[i]
		if frame.PC == 0 && frame.Function == name && frame.Line != 0 {
			return frame, true
		}
	}
	return CallerFrame{}, false
}

func bindCallerLocationPC(pc uintptr, frame CallerFrame) {
	recordPCLocation(pc, frame.Entry, frame.Function, frame.File, frame.Line)
	if pc > 0 {
		recordPCLocation(pc-1, frame.Entry, frame.Function, frame.File, frame.Line)
	}
}

func FrameForPC(pc uintptr) (CallerFrame, bool) {
	if pc&callerPCMask != 0 {
		if frame, ok := syntheticFrameForPC(pc); ok {
			return frame, true
		}
	}
	store := callerLocationTLS.Get()
	if store == nil || pc == 0 {
		return CallerFrame{}, false
	}
	for i := len(store.frames) - 1; i >= 0; i-- {
		frame := store.frames[i]
		if frame.PC == pc {
			return frame, true
		}
	}
	entry := entryForPC(pc)
	if entry == 0 {
		return CallerFrame{}, false
	}
	var best CallerFrame
	for _, frame := range store.frames {
		if frame.PC == 0 || frame.PC > pc || frame.Entry != entry {
			continue
		}
		if best.PC == 0 || frame.PC > best.PC {
			best = frame
		}
	}
	if best.PC != 0 {
		best.PC = pc
		return best, true
	}
	for i := len(store.frames) - 1; i >= 0; i-- {
		frame := store.frames[i]
		if frame.PC == 0 && frame.Entry == entry {
			frame.PC = pc
			return frame, true
		}
	}
	return CallerFrame{}, false
}

func syntheticFrameForPC(pc uintptr) (CallerFrame, bool) {
	store := callerLocationTLS.Get()
	if store == nil {
		return CallerFrame{}, false
	}
	seq := pc >> 2
	if seq == 0 || seq > uintptr(len(store.synthetic)) {
		return CallerFrame{}, false
	}
	frame := store.synthetic[seq-1]
	if frame.PC>>2 != seq {
		return CallerFrame{}, false
	}
	frame.PC = pc
	if frame.Entry == 0 {
		frame.Entry = pc
	}
	return frame, true
}

func callerLocationStoreForThread() *callerLocationStore {
	store := callerLocationTLS.Get()
	if store == nil {
		store = new(callerLocationStore)
		callerLocationTLS.Set(store)
	}
	return store
}

func (s *callerLocationStore) captureFrame(frame CallerFrame, pcValue uintptr) CallerFrame {
	idx := s.internSyntheticFrame(frame)
	rec := s.synthetic[idx]
	seq := uintptr(idx + 1)
	rec.PC = (seq << 2) | pcValue
	if rec.Entry == 0 {
		rec.Entry = rec.PC
	}
	return rec
}

// capturePC returns the synthetic PC for a shadow-stack slot, memoizing the
// interned base in the slot so an unchanged frame costs two loads instead of
// a hash probe plus frame comparison.
func (s *callerLocationStore) capturePC(frame *CallerFrame, pcValue uintptr) uintptr {
	if frame.captured != 0 {
		return frame.captured | pcValue
	}
	idx := s.internSyntheticFrame(*frame)
	base := uintptr(idx+1) << 2
	frame.captured = base
	return base | pcValue
}

// captureFrameAt is capturePC plus the full frame copy Caller needs.
func (s *callerLocationStore) captureFrameAt(frame *CallerFrame, pcValue uintptr) CallerFrame {
	pc := s.capturePC(frame, pcValue)
	rec := s.synthetic[(pc>>2)-1]
	rec.PC = pc
	if rec.Entry == 0 {
		rec.Entry = rec.PC
	}
	return rec
}

// staticPC memoizes the synthetic PC base of a process-static frame (e.g.
// runtime.main) in the per-store cache slot.
func (s *callerLocationStore) staticPC(frame CallerFrame, cache *uintptr, pcValue uintptr) uintptr {
	if *cache == 0 {
		*cache = uintptr(s.internSyntheticFrame(frame)+1) << 2
	}
	return *cache | pcValue
}

func (s *callerLocationStore) internSyntheticFrame(frame CallerFrame) int {
	frame.captured = 0
	if len(s.syntheticHash) == 0 {
		s.syntheticHash = make([]uintptr, callerPCHashInit)
	}
	if len(s.synthetic)*2 >= len(s.syntheticHash) {
		s.rehashSyntheticFrames(len(s.syntheticHash) * 2)
	}
	slot := s.syntheticSlot(frame)
	for {
		idx := s.syntheticHash[slot]
		if idx == 0 {
			frame.PC = (uintptr(len(s.synthetic)+1) << 2) | callerPCValue
			s.synthetic = append(s.synthetic, frame)
			s.syntheticHash[slot] = uintptr(len(s.synthetic))
			return len(s.synthetic) - 1
		}
		existing := s.synthetic[idx-1]
		if sameSyntheticFrame(existing, frame) {
			return int(idx - 1)
		}
		slot = (slot + 1) & (uintptr(len(s.syntheticHash)) - 1)
	}
}

func (s *callerLocationStore) rehashSyntheticFrames(size int) {
	old := s.syntheticHash
	s.syntheticHash = make([]uintptr, size)
	for _, idx := range old {
		if idx == 0 {
			continue
		}
		frame := s.synthetic[idx-1]
		slot := s.syntheticSlot(frame)
		for s.syntheticHash[slot] != 0 {
			slot = (slot + 1) & (uintptr(len(s.syntheticHash)) - 1)
		}
		s.syntheticHash[slot] = idx
	}
}

func (s *callerLocationStore) syntheticSlot(frame CallerFrame) uintptr {
	h := frame.Entry ^ (uintptr(frame.Line) << 12) ^ (uintptr(frame.StartLine) << 24)
	h ^= uintptr(len(frame.Function)) << 4
	h ^= uintptr(len(frame.File)) << 8
	return h & (uintptr(len(s.syntheticHash)) - 1)
}

func sameSyntheticFrame(a, b CallerFrame) bool {
	return a.Entry == b.Entry &&
		a.Function == b.Function &&
		a.File == b.File &&
		a.Line == b.Line &&
		a.StartLine == b.StartLine
}

func entryForPC(pc uintptr) uintptr {
	var info clitedebug.Info
	if clitedebug.Addrinfo(unsafe.Pointer(pc), &info) == 0 {
		return 0
	}
	return uintptr(info.Saddr)
}

func normalizeRuntimeFuncName(name string) string {
	const commandLineArguments = "command-line-arguments."
	if hasPrefix(name, commandLineArguments) {
		name = "main." + name[len(commandLineArguments):]
	}
	if len(name) > 0 && name[0] == '_' {
		name = name[1:]
	}
	return normalizeRuntimeAnonFuncName(name)
}

func normalizeRuntimeAnonFuncName(name string) string {
	dollar := lastIndexByte(name, '$')
	if dollar < 0 || dollar == len(name)-1 {
		return name
	}
	for i := dollar + 1; i < len(name); i++ {
		if name[i] < '0' || name[i] > '9' {
			return name
		}
	}
	return name[:dollar] + ".func" + name[dollar+1:]
}

func hasPrefix(s, prefix string) bool {
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

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
