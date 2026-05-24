package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type runtimeCallerSnapshot struct {
	pc   uintptr
	file string
	line int
	ok   bool
}

func TestRuntimeCallerMetadata(t *testing.T) {
	tests := []struct {
		skip       int
		nameSuffix string
		line       int
	}{
		{0, ".runtimeCallerLeaf", 101},
		{1, ".runtimeCallerMid", 111},
		{2, ".runtimeCallerTop", 121},
	}
	for _, tt := range tests {
		got := runtimeCallerTop(tt.skip)
		if !got.ok {
			t.Fatalf("runtime.Caller(%d) failed", tt.skip)
		}
		if !strings.HasSuffix(got.file, "runtime_caller_metadata.go") {
			t.Fatalf("runtime.Caller(%d) file = %q", tt.skip, got.file)
		}
		if got.line != tt.line {
			t.Fatalf("runtime.Caller(%d) line = %d, want %d", tt.skip, got.line, tt.line)
		}
		fn := runtime.FuncForPC(got.pc)
		if fn == nil {
			t.Fatalf("FuncForPC(runtime.Caller(%d) pc) = nil", tt.skip)
		}
		if name := fn.Name(); !strings.HasSuffix(name, tt.nameSuffix) {
			t.Fatalf("FuncForPC(runtime.Caller(%d) pc).Name = %q, want suffix %q", tt.skip, name, tt.nameSuffix)
		}
	}
}

func TestRuntimeCallersFramesMetadata(t *testing.T) {
	frames := runtimeCallersTop(0)
	want := []struct {
		nameSuffix string
		line       int
	}{
		{"runtime.Callers", 0},
		{".runtimeCallersLeaf", 202},
		{".runtimeCallersMid", 211},
		{".runtimeCallersTop", 221},
	}
	if len(frames) < len(want) {
		t.Fatalf("runtime.CallersFrames returned %d frames, want at least %d: %#v", len(frames), len(want), frames)
	}
	for i, tt := range want {
		if name := frames[i].Function; !strings.HasSuffix(name, tt.nameSuffix) {
			t.Fatalf("frame %d function = %q, want suffix %q", i, name, tt.nameSuffix)
		}
		if tt.line != 0 && frames[i].Line != tt.line {
			t.Fatalf("frame %d line = %d, want %d", i, frames[i].Line, tt.line)
		}
		if frames[i].Func == nil {
			continue
		}
		if name := frames[i].Func.Name(); !strings.HasSuffix(name, tt.nameSuffix) {
			t.Fatalf("frame %d Func.Name = %q, want suffix %q", i, name, tt.nameSuffix)
		}
	}
}

type runtimeCallerNilIface interface{ runtimeCallerNilMethod() }

type runtimeCallerNilImpl struct{}

func (*runtimeCallerNilImpl) runtimeCallerNilMethod() {}

func TestRuntimeCallersFramesRecoveredNilPanicMetadata(t *testing.T) {
	tests := []struct {
		nameSuffix string
		line       int
		run        func() (runtime.Frame, bool)
	}{
		{".runtimeRecoveredNilPanicClosureFrame.func1", 303, runtimeRecoveredNilPanicClosureFrame},
		{".runtimeRecoveredNilPanicSplitClosureFrame.func1", 314, runtimeRecoveredNilPanicSplitClosureFrame},
	}
	for _, tt := range tests {
		frame, ok := tt.run()
		if !ok {
			t.Fatalf("CallersFrames did not include %s", tt.nameSuffix)
		}
		if name := frame.Function; !strings.HasSuffix(name, tt.nameSuffix) {
			t.Fatalf("recovered panic frame function = %q, want suffix %q", name, tt.nameSuffix)
		}
		if frame.Func == nil {
			t.Fatalf("recovered panic frame Func = nil for %s", tt.nameSuffix)
		}
		if name := frame.Func.Name(); !strings.HasSuffix(name, tt.nameSuffix) {
			t.Fatalf("recovered panic frame Func.Name = %q, want suffix %q", name, tt.nameSuffix)
		}
		if frame.Line != tt.line {
			t.Fatalf("recovered panic frame line = %d, want %d", frame.Line, tt.line)
		}
	}

	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed after recovered panic")
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		t.Fatal("FuncForPC(runtime.Caller(0) pc) = nil after recovered panic")
	}
	if name := fn.Name(); !strings.HasSuffix(name, ".TestRuntimeCallersFramesRecoveredNilPanicMetadata") {
		t.Fatalf("runtime.Caller(0) after recovered panic = %q, want current test frame", name)
	}
}

func runtimeRecoveredNilPanicFrame(wantSuffix string, f func()) (frame runtime.Frame, ok bool) {
	defer func() {
		if recover() == nil {
			return
		}
		var pcs [32]uintptr
		n := runtime.Callers(0, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			next, more := frames.Next()
			if strings.HasSuffix(next.Function, wantSuffix) {
				frame = next
				ok = true
				return
			}
			if !more {
				return
			}
		}
	}()
	f()
	return
}

//line runtime_callers_recovered_nil_metadata.go:300
func runtimeRecoveredNilPanicClosureFrame() (runtime.Frame, bool) {
	return runtimeRecoveredNilPanicFrame(".runtimeRecoveredNilPanicClosureFrame.func1", func() {
		var v runtimeCallerNilIface
		v.runtimeCallerNilMethod()
	})
}

//line runtime_callers_recovered_nil_metadata.go:310
func runtimeRecoveredNilPanicSplitClosureFrame() (runtime.Frame, bool) {
	return runtimeRecoveredNilPanicFrame(".runtimeRecoveredNilPanicSplitClosureFrame.func1", func() {
		var v runtimeCallerNilIface
		v. // method name is on the following line
			runtimeCallerNilMethod()
	})
}

//line runtime_caller_metadata.go:100
func runtimeCallerLeaf(skip int) runtimeCallerSnapshot {
	pc, file, line, ok := runtime.Caller(skip)
	return runtimeCallerSnapshot{pc: pc, file: file, line: line, ok: ok}
}

//line runtime_caller_metadata.go:110
func runtimeCallerMid(skip int) runtimeCallerSnapshot {
	return runtimeCallerLeaf(skip)
}

//line runtime_caller_metadata.go:120
func runtimeCallerTop(skip int) runtimeCallerSnapshot {
	return runtimeCallerMid(skip)
}

//line runtime_callers_metadata.go:200
func runtimeCallersLeaf(skip int) []runtime.Frame {
	var pcs [16]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var out []runtime.Frame
	for {
		frame, more := frames.Next()
		out = append(out, frame)
		if !more {
			break
		}
	}
	return out
}

//line runtime_callers_metadata.go:210
func runtimeCallersMid(skip int) []runtime.Frame {
	return runtimeCallersLeaf(skip)
}

//line runtime_callers_metadata.go:220
func runtimeCallersTop(skip int) []runtime.Frame {
	return runtimeCallersMid(skip)
}
