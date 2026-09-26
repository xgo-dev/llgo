//go:build !baremetal && !wasm

package runtime

import (
	"unsafe"

	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

func init() { rtdebug.TracebackCreator = tracebackCreator }

func tracebackCreator() uintptr {
	if !fpUnwindAvailable() {
		return 0
	}
	var pcs [32]uintptr
	n := fpCallers(0, pcs[:])
	for _, pc := range pcs[:n] {
		name := frameSymbol(pc - 1).function
		if name != "" && visibleTracebackFrame(name, false) {
			return pc
		}
	}
	return 0
}

func appendCreatedBy(out []byte, parent uint64, pc uintptr) []byte {
	if parent == 0 || pc == 0 {
		return out
	}
	frame := frameSymbol(pc - 1)
	if frame.function == "" {
		return out
	}
	out = append(out, "created by "...)
	out = append(out, frame.function...)
	out = append(out, " in goroutine "...)
	out = appendInt(out, int(parent))
	out = append(out, '\n', '\t')
	out = append(out, frame.file...)
	out = append(out, ':')
	out = appendInt(out, frame.line)
	if frame.entry != 0 && pc >= frame.entry {
		out = append(out, " +0x"...)
		out = appendHexUint(out, pc-frame.entry)
	}
	return append(out, '\n')
}

func appendCurrentCreatedBy(out []byte) []byte {
	parent, pc := rtdebug.TracebackParent()
	return appendCreatedBy(out, parent, pc)
}

func appendOtherTracebacks(out []byte, system bool, limit int) []byte {
	snapshots := stacktrace.Capture(goid())
	defer stacktrace.Free(snapshots)
	for s := snapshots; s != nil; s = stacktrace.Next(s) {
		if limit > 0 && len(out) >= limit {
			break
		}
		var id, parent uint64
		var created, count uintptr
		var state uint32
		ptr := stacktrace.Info(s, &id, &parent, &created, &count, &state)
		pcs := unsafe.Slice(ptr, count)
		status := "running"
		if state == 1 {
			status = "runnable"
		} else if state == 4 {
			status = "waiting"
		}
		// Native blocking primitives do not use Go's scheduler waitreason
		// enumeration. Their saved Go call chain supplies the public reason.
		// They are near the innermost end; avoid symbolizing a deep stack
		// twice just to determine the status.
		statusPCs := pcs
		if len(statusPCs) > 32 {
			statusPCs = statusPCs[:32]
		}
		for _, pc := range statusPCs {
			switch frameSymbol(pc - 1).function {
			case "runtime.timeSleep":
				status = "sleep"
			case "github.com/xgo-dev/llgo/runtime/internal/runtime.ChanRecv":
				if status != "sleep" {
					status = "chan receive"
				}
			case "github.com/xgo-dev/llgo/runtime/internal/runtime.ChanSend":
				status = "chan send"
			case "runtime.semacquire", "runtime.runtime_SemacquireMutex":
				status = "semacquire"
			}
		}
		start := len(out)
		out = append(out, "\ngoroutine "...)
		out = appendInt(out, int(id))
		out = append(out, " ["...)
		out = append(out, status...)
		out = append(out, "]:\n"...)
		var window traceback.Window
		visible := false
		if len(pcs) == 0 {
			out = append(out, "\tgoroutine running on other thread; stack unavailable\n"...)
		} else {
			frames := CallersFrames(pcs)
			for {
				frame, more := frames.Next()
				if traceback.Visible(frame.Function, system, !visible) && (system || frame.File != "") {
					visible = true
					out = window.Append(out, tracebackFrame(frame))
				}
				if !more {
					break
				}
			}
		}
		if !visible && len(pcs) != 0 {
			if !system && created == 0 {
				out = out[:start]
				continue
			}
			out = append(out, "\tgoroutine running on other thread; stack unavailable\n"...)
		}
		out = window.Finish(out)
		out = appendCreatedBy(out, parent, created)
	}
	return out
}
