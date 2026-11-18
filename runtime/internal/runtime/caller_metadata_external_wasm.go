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

func resolveCachedCallerFrame(frame *CallerFrame) CallerFrame {
	if frame.locationID != 0 && frame.Function == "" {
		// Leave unresolved entries retryable until the optional map is ready.
		// Successful names/files then live as long as the interned frame.
		*frame = resolveCallerFrame(*frame)
	}
	return *frame
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
	recordCallerLocation(entry, "", "", line, callerLocationMetadata{locationID: id})
}

//go:noinline
func RecordPanicLocationID(entry uintptr, id uint64, line int) {
	RecordCallerLocationID(entry, id, line)
}

func hasCallerLocation(frame CallerFrame) bool { return frame.locationID != 0 }

func callerLocationID(frame CallerFrame) uint64 { return frame.locationID }
