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

package gotest

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	callerPanicChild   = "LLGO_TEST_CALLER_PANIC"
	callerRepanicChild = "LLGO_TEST_CALLER_REPANIC"
)

var (
	callerInitFile string
	callerInitLine int
)

func init() {
	_, callerInitFile, callerInitLine, _ = runtime.Caller(0)
	if os.Getenv(callerPanicChild) == "1" {
		callerPanicCaller() // PANIC_INIT_MARK
	}
}

//go:noinline
func callerPanicBoom() {
	panic("acceptance-boom") // PANIC_MARK
}

//go:noinline
func callerPanicCaller() {
	callerPanicBoom() // PANIC_CALLER_MARK
}

func TestCallerPanicTraceback(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), callerPanicChild+"=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("panic child unexpectedly succeeded:\n%s", output)
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("current source file is unavailable")
	}
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"panic: acceptance-boom",
		"goroutine 1 [running]:",
		"callerPanicBoom",
		"caller_runtime_test.go:" + strconv.Itoa(markerLine(string(source), "PANIC_MARK")),
		"callerPanicCaller",
		"caller_runtime_test.go:" + strconv.Itoa(markerLine(string(source), "PANIC_CALLER_MARK")),
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("panic traceback is missing %q:\n%s", want, output)
		}
	}
}

//go:noinline
func callerRepanicOrigin() {
	var pointer *int
	_ = *pointer // REPANIC_ORIGIN_MARK
}

//go:noinline
func callerReplacementPanic() {
	defer func() {
		_ = recover()
		panic("replacement panic") // REPLACEMENT_PANIC_MARK
	}()
	panic("original panic")
}

//go:noinline
func callerLaterSameValuePanic() {
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		panic("earlier panic")
	}()
	panic(recovered) // LATER_SAME_VALUE_PANIC_MARK
}

type callerRepanicReceiver struct{}

//go:noinline
func (callerRepanicReceiver) repanic() {
	if v := recover(); v != nil {
		panic(v)
	}
}

//go:noinline
func callerWrappedRepanic(indirect bool) {
	// A pointer implementing this value-receiver method needs a transparent
	// wrapper. Only the direct deferred invocation is allowed to recover.
	var receiver interface{ repanic() } = &callerRepanicReceiver{}
	if indirect {
		defer func() {
			v := recover()
			receiver.repanic()
			panic(v)
		}()
	} else {
		defer receiver.repanic()
	}
	callerRepanicOrigin()
}

//go:noinline
func callerSliceRepanic() {
	defer func() { panic(recover()) }()
	panic([]int{1}) // SLICE_REPANIC_ORIGIN_MARK
}

func TestCallerRepanicTraceback(t *testing.T) {
	mode := os.Getenv(callerRepanicChild)
	switch mode {
	case "same":
		// Match llcppg: both nested tRunner repanics must retain the nil site.
		t.Run("origin", func(t *testing.T) { callerRepanicOrigin() })
		return
	case "different":
		callerReplacementPanic()
		return
	case "later":
		callerLaterSameValuePanic()
		return
	case "wrapper", "indirect-wrapper":
		callerWrappedRepanic(mode == "indirect-wrapper")
		return
	case "slice":
		callerSliceRepanic()
		return
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("current source file is unavailable")
	}
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}

	run := func(mode string) string {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestCallerRepanicTraceback$")
		cmd.Env = append(os.Environ(), callerRepanicChild+"="+mode)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("%s repanic child unexpectedly succeeded:\n%s", mode, output)
		}
		return string(output)
	}

	for _, mode := range []string{"same", "wrapper", "indirect-wrapper", "slice"} {
		name, marker := "callerRepanicOrigin", "REPANIC_ORIGIN_MARK"
		if mode == "slice" {
			name, marker = "callerSliceRepanic", "SLICE_REPANIC_ORIGIN_MARK"
		}
		same := run(mode)
		for _, want := range []string{
			name,
			"caller_runtime_test.go:" + strconv.Itoa(markerLine(string(source), marker)),
		} {
			if !strings.Contains(same, want) {
				t.Fatalf("%s repanic traceback is missing %q:\n%s", mode, want, same)
			}
		}
	}

	different := run("different")
	for _, want := range []string{
		"panic: replacement panic",
		"caller_runtime_test.go:" + strconv.Itoa(markerLine(string(source), "REPLACEMENT_PANIC_MARK")),
	} {
		if !strings.Contains(different, want) {
			t.Fatalf("replacement panic traceback is missing %q:\n%s", want, different)
		}
	}
	later := run("later")
	wantLater := "caller_runtime_test.go:" + strconv.Itoa(markerLine(string(source), "LATER_SAME_VALUE_PANIC_MARK"))
	if !strings.Contains(later, wantLater) {
		t.Fatalf("same value panicked after its recover activation returned is missing %q:\n%s", wantLater, later)
	}
}

type callerReceiver struct{}

var callerValueSink, callerPointerSink, callerClosureSink, callerGenericSink int

//go:noinline
func (callerReceiver) value() uintptr {
	callerValueSink++
	pc, _, _, _ := runtime.Caller(0)
	return pc
}

//go:noinline
func (*callerReceiver) pointer() uintptr {
	callerPointerSink++
	pc, _, _, _ := runtime.Caller(0)
	return pc
}

//go:noinline
func callerGeneric[T any](v T) uintptr {
	callerGenericSink++
	pc, _, _, _ := runtime.Caller(0)
	return pc
}

type callerStackError struct {
	msg string
	pcs [8]uintptr
	n   int
}

func (e *callerStackError) Error() string { return e.msg }

//go:noinline
func newCallerStackError(msg string) *callerStackError {
	err := &callerStackError{msg: msg}
	err.n = runtime.Callers(1, err.pcs[:])
	return err
}

func TestCallerIntrospection(t *testing.T) {
	if !strings.HasSuffix(callerInitFile, "caller_runtime_test.go") || callerInitLine == 0 {
		t.Fatalf("init caller = %s:%d", callerInitFile, callerInitLine)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var goroutineFile string
	var goroutineLine int
	go func() {
		defer wg.Done()
		_, goroutineFile, goroutineLine, _ = runtime.Caller(0)
	}()
	wg.Wait()
	if !strings.HasSuffix(goroutineFile, "caller_runtime_test.go") || goroutineLine == 0 {
		t.Fatalf("goroutine caller = %s:%d", goroutineFile, goroutineLine)
	}

	var deferredFile string
	var deferredLine int
	func() {
		defer func() {
			_, deferredFile, deferredLine, _ = runtime.Caller(0)
		}()
	}()
	if !strings.HasSuffix(deferredFile, "caller_runtime_test.go") || deferredLine == 0 {
		t.Fatalf("deferred caller = %s:%d", deferredFile, deferredLine)
	}

	var receiver callerReceiver
	checkCallerFunctionSuffix(t, receiver.value(), ".callerReceiver.value")
	checkCallerFunctionSuffix(t, (&receiver).pointer(), ".(*callerReceiver).pointer")
	closure := func() uintptr {
		callerClosureSink++
		pc, _, _, _ := runtime.Caller(0)
		return pc
	}
	closureName := runtime.FuncForPC(closure()).Name()
	if !strings.Contains(closureName, "TestCallerIntrospection.func") && !strings.Contains(closureName, "TestCallerIntrospection$") {
		t.Fatalf("closure name = %q", closureName)
	}
	genericName := runtime.FuncForPC(callerGeneric(0)).Name()
	if !strings.Contains(genericName, ".callerGeneric") {
		t.Fatalf("generic function name = %q", genericName)
	}

	_, _, callLine, _ := runtime.Caller(0)
	err := newCallerStackError("wrapped")
	frames := runtime.CallersFrames(err.pcs[:err.n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".TestCallerIntrospection") {
			if frame.Line != callLine+1 {
				t.Fatalf("stack capture line = %d, want %d", frame.Line, callLine+1)
			}
			return
		}
		if !more {
			break
		}
	}
	t.Fatal("TestCallerIntrospection frame is missing")
}

func checkCallerFunctionSuffix(t *testing.T, pc uintptr, suffix string) {
	t.Helper()
	fn := runtime.FuncForPC(pc)
	if fn == nil || !strings.HasSuffix(fn.Name(), suffix) {
		name := "<nil>"
		if fn != nil {
			name = fn.Name()
		}
		t.Fatalf("function for pc = %q, want suffix %q", name, suffix)
	}
}
