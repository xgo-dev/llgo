//go:build windows

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

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

//go:linkname c_windowsFaultPCBuf C.llgo_windows_fault_pcbuf
func c_windowsFaultPCBuf() unsafe.Pointer

//go:linkname c_memReadable C.llgo_mem_readable
func c_memReadable(p unsafe.Pointer) c.Int

func memReadable(addr uintptr) bool {
	return c_memReadable(unsafe.Pointer(addr)) != 0
}

func init() {
	rtdebug.WindowsFaultSnapshot = storeWindowsFaultSnapshot
}

func storeWindowsFaultSnapshot(context unsafe.Pointer) bool {
	// Faults are recoverable and different goroutines run on different host
	// threads, so capture into the handler's thread-local scratch storage.
	// StoreFaultPCs retains this storage until the recovering activation ends,
	// including after panic's non-local jump unwinds the handler stack.
	pcs := (*[64]uintptr)(c_windowsFaultPCBuf())[:]
	if buf := stacktrace.FaultBuffer(); buf != nil {
		pcs = unsafe.Slice(buf, stacktrace.MaxFrames)
	}
	pc, fp := windowsFaultPCFP(context)
	originPC := pc
	textPC := pc
	if originPC == 0 {
		// A call through a nil function value faults while fetching the target
		// instruction at address zero. Recover the return PC installed by the
		// call instruction so the fault is still attributed to its Go caller.
		// This keeps native faults out without breaking Go's nil-call panic.
		originPC = windowsFaultCallerPC(context)
		if originPC == 0 {
			return false
		}
		textPC = originPC
		// originPC is the instruction after the call. Attribute it to the
		// call itself, including when the return PC is exactly at etext.
		textPC--
	}
	// Do not construct the first-use table in exception context. Normal linked
	// binaries adopt their prebuilt table during runtime init, and external
	// metadata is fully constructed before it is published. If neither has
	// completed, preserve the historical fallback and let the core handle the
	// exception rather than risking an allocation here.
	if runtimeFuncPCFramesBuilt() &&
		(len(runtimePrebuiltFtab) != 0 || len(runtimeFuncPCFrames) != 0) &&
		!prebuiltTextContains(textPC) {
		return false
	}
	n := 0
	if pc != 0 {
		// Stored PCs follow runtime.Callers' return-PC convention. Adding one
		// keeps PC-1 on the instruction that raised the exception.
		pcs[0] = pc + 1
		n = 1
	} else if originPC != 0 {
		// originPC is already a return PC, which is the convention expected by
		// CallersFrames and avoids leaving a recovered nil-call trace empty.
		pcs[0] = originPC
		n = 1
	}
	if fpUnwindAvailable() && n < len(pcs) {
		n += platformFaultCallers(context, fp, pcs[n:])
	}
	rtdebug.StoreFaultPCs(pcs[:n])
	return true
}

func windowsFaultCallerPC(raw unsafe.Pointer) uintptr {
	return (*windowsFaultContext)(raw).faultCallerPC()
}

func windowsFaultStackCallerPC(sp uintptr) uintptr {
	const ptrSize = unsafe.Sizeof(uintptr(0))
	if sp < minLegalPC || sp > ^uintptr(0)-ptrSize ||
		!memReadable(sp) || !memReadable(sp+ptrSize-1) {
		return 0
	}
	pc := *(*uintptr)(unsafe.Pointer(sp))
	if pc < minLegalPC {
		return 0
	}
	return pc
}

func windowsFPWalkFrom(fp uintptr, pcs []uintptr) int {
	n := 0
	const maxFrames = maxTracebackFrames
	wordSize := unsafe.Sizeof(uintptr(0))
	for i := 0; fp != 0 && n < len(pcs) && i < maxFrames; i++ {
		if fp&(wordSize-1) != 0 || !memReadable(fp) || !memReadable(fp+wordSize) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + wordSize))
		if ret < minLegalPC {
			break
		}
		pcs[n] = ret
		n++
		if prev <= fp || prev-fp > maxFPStride || prev&(wordSize-1) != 0 {
			break
		}
		fp = prev
	}
	return n
}

// Windows fault snapshots live in the current G. Keep a recovered snapshot
// available while its deferred frame can still observe runtime.Callers; the
// next panic replaces it. PanicActive supplies the separate in-flight bit.
func faultTracebackActive() bool {
	return rtdebug.PanicPCsAreFault() && rtdebug.PanicActive()
}

func trimWindowsFaultPCs(pcs []uintptr) []uintptr {
	if !rtdebug.PanicPCsAreFault() {
		return pcs
	}
	return trimLogicalGoTail(pcs)
}

func trimLogicalGoTail(pcs []uintptr) []uintptr {
	for i, pc := range pcs {
		if frameSymbol(pc-1).function == "runtime.goexit" {
			return pcs[:i+1]
		}
	}
	return pcs
}

func faultTraceback(skip int) bool {
	pcs := rtdebug.PanicPCs()
	if !rtdebug.PanicPCsAreFault() || len(pcs) == 0 {
		return false
	}
	ready := fpUnwindAvailable()
	if ready {
		initRuntimeFuncPCFrames()
		pcs = trimLogicalGoTail(pcs)
		pcs = trimNativeTracebackTail(pcs)
	}
	system := traceback.Level(rtdebug.TracebackSetting()) > 1
	out := appendTracebackHeader(nil)
	var window traceback.Window
	printed := 0
	for _, pc := range pcs {
		var sym pcSymbol
		if ready {
			sym = frameSymbol(pc - 1)
			if !visibleTracebackFrame(sym.function, system) {
				continue
			}
		}
		out = window.Append(out, traceback.Frame{PC: pc - 1, Entry: sym.entry,
			Function: sym.function, File: sym.file, Line: sym.line})
		printed++
	}
	out = appendCurrentCreatedBy(window.Finish(out))
	print(string(out))
	return printed != 0
}
