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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const runtimeLineInfoProbe = `package main

import (
	"reflect"
	"strconv"
	"runtime"
	"runtime/debug"
	"strings"
	_ "unsafe"
)

func main() {
	checkCaller()
	checkCallerSkip()
	checkFrames() // FRAMES_MAIN_MARK
	checkFuncForPC()
	checkFuncForPCFunctionValue()
	checkFuncInfoRename()
	checkRuntimeStack()
	checkPanicStack()
}

//go:noinline
func checkCaller() {
	_, file, line, ok := runtime.Caller(0) // CALLER_MARK
	if !ok || !strings.HasSuffix(file, "main.go") || line != CALLER_LINE {
		panic("bad caller: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func checkCallerSkip() {
	helperCallerSkip() // CALLER_SKIP_MARK
}

//go:noinline
func helperCallerSkip() {
	_, file, line, ok := runtime.Caller(1)
	if !ok || !strings.HasSuffix(file, "main.go") || line != CALLER_SKIP_LINE {
		panic("bad caller skip: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func checkFrames() {
	var pcs [8]uintptr
	n := runtime.Callers(0, pcs[:]) // FRAMES_CHECK_MARK
	frames := runtime.CallersFrames(pcs[:n])
	seenCheckFrames := false
	seenMain := false
	for {
		frame, more := frames.Next()
		if frame.Function == "main.checkFrames" {
			if !strings.HasSuffix(frame.File, "main.go") || frame.Line != FRAMES_CHECK_LINE {
				panic("bad checkFrames frame: " + frame.File + ":" + strconv.Itoa(frame.Line))
			}
			seenCheckFrames = true
		}
		if frame.Function == "main.main" {
			if !strings.HasSuffix(frame.File, "main.go") || frame.Line != FRAMES_MAIN_LINE {
				panic("bad main frame: " + frame.File + ":" + strconv.Itoa(frame.Line))
			}
			seenMain = true
		}
		if seenCheckFrames && seenMain {
			return
		}
		if !more {
			break
		}
	}
	panic("missing frame")
}

//go:noinline
func checkFuncForPC() {
	pc, _, _, ok := runtime.Caller(0) // FUNC_FILELINE_MARK
	if !ok {
		panic("missing pc")
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		panic("missing func")
	}
	if name := fn.Name(); name != "main.checkFuncForPC" {
		panic("bad func: " + name)
	}
	if entry := fn.Entry(); entry == 0 {
		panic("missing func entry")
	}
	file, line := fn.FileLine(pc)
	if !strings.HasSuffix(file, "main.go") || line != FUNC_FILELINE_LINE {
		panic("bad func fileline: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func entryPCTarget() int {
	return 7 // FUNC_ENTRY_TARGET_MARK
}

//go:noinline
func checkFuncForPCFunctionValue() {
	if entryPCTarget() != 7 {
		panic("bad target")
	}
	pc := reflect.ValueOf(entryPCTarget).Pointer()
	if pc == 0 {
		panic("missing function value pc")
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		panic("missing function value func")
	}
	if name := fn.Name(); name != "main.entryPCTarget" {
		panic("bad function value func: " + name)
	}
	if entry := fn.Entry(); entry == 0 {
		panic("missing function value entry")
	}
	file, line := fn.FileLine(pc)
	if !strings.HasSuffix(file, "main.go") || line != FUNC_ENTRY_TARGET_LINE {
		panic("bad function value fileline: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func checkFuncInfoRename() {
	pc := renamedPC()
	if name := runtime.FuncForPC(pc).Name(); name != "main.renamedPC" {
		panic("bad renamed func: " + name)
	}
}

//go:linkname renamedPC main.renamedPCSymbol
//go:noinline
func renamedPC() uintptr {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		panic("missing renamed pc")
	}
	return pc
}

//go:noinline
func checkRuntimeStack() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false) // RUNTIME_STACK_MARK
	stack := string(buf[:n])
	if !strings.Contains(stack, "main.checkRuntimeStack") || !strings.Contains(stack, "main.go:RUNTIME_STACK_LINE") {
		panic("bad runtime stack: " + stack)
	}
}

//go:noinline
func checkPanicStack() {
	defer func() { // DEBUG_STACK_MARK
		if recover() == nil {
			panic("missing panic")
		}
		stack := string(debug.Stack()) // DEBUG_STACK_CALL_MARK
		if !strings.Contains(stack, "main.checkPanicStack") || !strings.Contains(stack, "main.go:DEBUG_STACK_LINE") {
			panic("bad stack: " + stack)
		}
	}()
	s := []int{1, 2, 3}
	_ = s[3]
}
`

func TestRuntimeLineInfoAndStack(t *testing.T) {
	source := runtimeLineInfoProbe
	source = strings.ReplaceAll(source, "CALLER_LINE", strconv.Itoa(markerLine(source, "CALLER_MARK")))
	source = strings.ReplaceAll(source, "CALLER_SKIP_LINE", strconv.Itoa(markerLine(source, "CALLER_SKIP_MARK")))
	source = strings.ReplaceAll(source, "FRAMES_MAIN_LINE", strconv.Itoa(markerLine(source, "FRAMES_MAIN_MARK")))
	source = strings.ReplaceAll(source, "FRAMES_CHECK_LINE", strconv.Itoa(markerLine(source, "FRAMES_CHECK_MARK")))
	source = strings.ReplaceAll(source, "FUNC_FILELINE_LINE", strconv.Itoa(markerLine(source, "FUNC_FILELINE_MARK")))
	source = strings.ReplaceAll(source, "FUNC_ENTRY_TARGET_LINE", strconv.Itoa(markerLine(source, "FUNC_ENTRY_TARGET_MARK")))
	source = strings.ReplaceAll(source, "RUNTIME_STACK_LINE", strconv.Itoa(markerLine(source, "RUNTIME_STACK_MARK")))
	source = strings.ReplaceAll(source, "DEBUG_STACK_LINE", strconv.Itoa(markerLine(source, "DEBUG_STACK_CALL_MARK")))

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	repoRoot := findStringConversionRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	cmd := exec.Command("go", "run", "./cmd/llgo", "run", "-a", file)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("llgo lineinfo probe failed: %v\n%s", err, out)
	}
}

const runtimeFuncInfoConcurrentFirstUseProbe = `package main

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func main() {
	const n = 32
	start := make(chan struct{})
	errc := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errc <- checkRuntimeInfo()
		}()
	}
	close(start)
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != "" {
			panic(err)
		}
	}
}

//go:noinline
func checkRuntimeInfo() string {
	pc, file, line, ok := runtime.Caller(0) // CONCURRENT_CALLER_MARK
	if !ok || !strings.HasSuffix(file, "main.go") || line != CONCURRENT_CALLER_LINE {
		return "bad caller: " + file + ":" + strconv.Itoa(line)
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil || fn.Name() != "main.checkRuntimeInfo" {
		name := "<nil>"
		if fn != nil {
			name = fn.Name()
		}
		return "bad func: " + name
	}
	file, line = fn.FileLine(pc)
	if !strings.HasSuffix(file, "main.go") || line != CONCURRENT_CALLER_LINE {
		return "bad fileline: " + file + ":" + strconv.Itoa(line)
	}
	var pcs [8]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "main.checkRuntimeInfo" {
			if !strings.HasSuffix(frame.File, "main.go") || frame.Line == 0 {
				return "bad frame: " + frame.File + ":" + strconv.Itoa(frame.Line)
			}
			return ""
		}
		if !more {
			return "missing frame"
		}
	}
}
`

func TestRuntimeFuncInfoConcurrentFirstUse(t *testing.T) {
	source := runtimeFuncInfoConcurrentFirstUseProbe
	source = strings.ReplaceAll(source, "CONCURRENT_CALLER_LINE", strconv.Itoa(markerLine(source, "CONCURRENT_CALLER_MARK")))

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	repoRoot := findStringConversionRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	cmd := exec.Command("go", "run", "./cmd/llgo", "run", "-a", file)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("llgo concurrent funcinfo probe failed: %v\n%s", err, out)
	}
}

func markerLine(source, marker string) int {
	line := 1
	for _, part := range strings.SplitAfter(source, "\n") {
		if strings.Contains(part, marker) {
			return line
		}
		line++
	}
	panic("missing marker " + marker)
}
