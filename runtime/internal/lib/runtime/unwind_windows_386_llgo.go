//go:build windows && 386

package runtime

import (
	"unsafe"

	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func platformFaultCallers(_ unsafe.Pointer, fp uintptr, pc []uintptr) int {
	return windowsFPWalkFrom(fp, pc)
}

// callerFramePointer must observe recoverMark as a distinct frame.
//
//go:noinline
func recoverMark() {
	fp := callerFramePointer()
	if fp == 0 {
		return
	}
	rtdebug.MarkPanicRecoverFPs(fp, 0)
}

func panicSplicePCs() []uintptr {
	pcs := rtdebug.PanicPCs()
	if len(pcs) == 0 {
		return nil
	}
	if rtdebug.PanicActive() {
		return trimWindowsFaultPCs(pcs)
	}
	mark, _ := rtdebug.PanicRecoverFPs()
	if mark == 0 {
		return nil
	}
	fp := callerFramePointer()
	for i := 0; fp != 0 && i < maxPanicSpliceFrames; i++ {
		if !memReadable(fp) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		if fp <= mark && (prev > mark || prev == 0) {
			return trimWindowsFaultPCs(pcs)
		}
		if prev <= fp || prev-fp > maxFPStride || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	return nil
}

//go:noinline
func fpCallers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	initRuntimeFuncPCFrames()
	fp := uintptr(c_framepointer())
	n, lastGo := 0, 0
	var callee pcSymbol
	const maxFrames = maxPanicSpliceFrames
	wordSize := unsafe.Sizeof(uintptr(0))
	for i := 0; fp != 0 && n < len(pc) && i < maxFrames; i++ {
		if fp&(wordSize-1) != 0 || !memReadable(fp) || !memReadable(fp+wordSize) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + wordSize))
		if ret < minLegalPC {
			break
		}
		caller := frameSymbol(ret - 1)
		validPrev := prev > fp && prev-fp <= maxFPStride && prev&(wordSize-1) == 0
		if pcSymbolIsWrapper(caller) && elideWrapperCalling(callee.function) {
			callee = caller
			if !validPrev {
				break
			}
			fp = prev
			continue
		}
		if skip > 0 {
			skip--
		} else {
			pc[n] = ret
			n++
			if prebuiltTextContains(ret - 1) {
				lastGo = n
			}
		}
		callee = caller
		if !validPrev {
			break
		}
		fp = prev
	}
	if n < len(pc) {
		return lastGo
	}
	return n
}
