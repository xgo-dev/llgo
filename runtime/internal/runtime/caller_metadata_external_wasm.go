//go:build llgo && wasm && llgo_pclntab_external && !llgo.wasm.debugger

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/pcln"

type callerLocationMetadata struct{ locationID uint64 }

func resolveCallerFrame(frame CallerFrame) CallerFrame {
	if frame.locationID != 0 {
		name, file, line, start := pcln.Resolve(frame.locationID)
		if name != "" {
			frame.Function, frame.File, frame.Line, frame.StartLine = name, file, line, start
		}
	}
	return frame
}

func callerLocationHash(frame CallerFrame) uintptr {
	return uintptr(frame.locationID) ^ uintptr(frame.locationID>>32)
}
func sameCallerLocationMetadata(a, b CallerFrame) bool { return a.locationID == b.locationID }

//go:noinline
func PushCallerLocationFrameID(entry uintptr, id uint64, line int) int {
	mark := PushCallerLocationFrame(entry, "", "", line)
	callerLocationStoreCurrent.stack[mark].locationID = id
	return mark
}

//go:noinline
func RecordCallerLocationID(entry uintptr, id uint64, line int) {
	RecordCallerLocation(entry, "", "", line)
	store := callerLocationStoreCurrent
	if store == nil {
		return
	}
	for i := len(store.stack) - 1; i >= 0; i-- {
		frame := &store.stack[i]
		if frame.Entry == entry {
			if frame.locationID != id {
				frame.captured = 0
			}
			frame.locationID = id
			break
		}
	}
	for i := len(store.frames) - 1; i >= 0; i-- {
		frame := &store.frames[i]
		if frame.Entry == entry && frame.PC == 0 {
			frame.locationID = id
			break
		}
	}
}

//go:noinline
func RecordPanicLocationID(entry uintptr, id uint64, line int) {
	RecordCallerLocationID(entry, id, line)
}

func hasCallerLocation(frame CallerFrame) bool { return frame.locationID != 0 }
