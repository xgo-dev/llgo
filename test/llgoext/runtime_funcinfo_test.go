//go:build llgo

package llgoext

import (
	"runtime"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname runtimeInfoRenamedPC github.com/xgo-dev/llgo/test/llgoext.runtimeInfoRenamedPCSymbol
//go:noinline
func runtimeInfoRenamedPC() uintptr {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		panic("missing renamed pc")
	}
	return pc
}

func TestRuntimeFuncInfoKeepsSourceName(t *testing.T) {
	fn := runtime.FuncForPC(runtimeInfoRenamedPC())
	if fn == nil || !strings.HasSuffix(fn.Name(), ".runtimeInfoRenamedPC") {
		name := "<nil>"
		if fn != nil {
			name = fn.Name()
		}
		t.Fatalf("renamed function = %q, want source name suffix .runtimeInfoRenamedPC", name)
	}
}

func TestRuntimeFuncInfoFramePCStatementLine(t *testing.T) {
	checkRuntimeFuncInfoFramePCStatementLine(t)
}

//go:noinline
func checkRuntimeFuncInfoFramePCStatementLine(t *testing.T) {
	var pcs [8]uintptr
	_, wantFile, wantLine, ok := runtime.Caller(0)
	n := runtime.Callers(0, pcs[:]) // CALLERS_PC_MARK
	if !ok {
		t.Fatal("current source position is unavailable")
	}
	// The Callers statement immediately follows the Caller statement, so this
	// remains independent of access to the source tree at run time.
	wantLine++
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".checkRuntimeFuncInfoFramePCStatementLine") {
			fn := runtime.FuncForPC(frame.PC - 1)
			if fn == nil {
				t.Fatal("FuncForPC(pc-1) returned nil")
			}
			file, line := fn.FileLine(frame.PC - 1)
			if file != wantFile || line != wantLine {
				t.Fatalf("Func.FileLine(pc-1) = %s:%d, want %s:%d", file, line, wantFile, wantLine)
			}
			return
		}
		if !more {
			break
		}
	}
	t.Fatal("CallersFrames is missing the current function")
}
