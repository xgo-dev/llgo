package runtime

import "github.com/xgo-dev/llgo/runtime/internal/traceback"

const maxTracebackFrames = traceback.MaxFrames

func appendTracebackHeader(out []byte) []byte {
	out = append(out, "goroutine "...)
	out = appendInt(out, int(goid()))
	return append(out, " [running]:\n"...)
}

func tracebackFrame(frame Frame) traceback.Frame {
	return traceback.Frame{
		PC: frame.PC, Entry: frame.Entry, Function: frame.Function,
		File: frame.File, Line: frame.Line,
	}
}

func visibleTracebackFrame(name string, system bool) bool {
	return traceback.Visible(name, system, true)
}
