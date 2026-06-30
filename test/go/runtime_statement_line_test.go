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

const runtimeStatementLineProbe = `package main

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

type Wrapper struct {
	a []int
}

func (w Wrapper) Get(i int) int {
	return w.a[i]
}

func main() {
	checkCallerStatement()
	checkCallersFramesStatement()
	checkInterfaceIndirectCaller()
	checkClosureIndirectCaller()
	checkAdjacentRuntimeStack()
	checkRecoveredDebugStackBounds()
}

//go:noinline
func checkCallerStatement() {
	_, file, line, ok := runtime.Caller(0) // CALLER_STMT_MARK
	if !ok || !strings.HasSuffix(file, "main.go") || line != CALLER_STMT_LINE {
		panic("bad caller statement: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func checkCallersFramesStatement() {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:]) // CALLERS_STMT_MARK
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "main.checkCallersFramesStatement" {
			if !strings.HasSuffix(frame.File, "main.go") || frame.Line != CALLERS_STMT_LINE {
				panic("bad callers frame: " + frame.File + ":" + strconv.Itoa(frame.Line))
			}
			fn := runtime.FuncForPC(frame.PC - 1)
			if fn == nil || fn.Name() != "main.checkCallersFramesStatement" {
				name := "<nil>"
				if fn != nil {
					name = fn.Name()
				}
				panic("bad FuncForPC(pc-1): " + name)
			}
			file, line := fn.FileLine(frame.PC - 1)
			if !strings.HasSuffix(file, "main.go") || line != CALLERS_STMT_LINE {
				panic("bad Func.FileLine(pc-1): " + file + ":" + strconv.Itoa(line))
			}
			return
		}
		if !more {
			break
		}
	}
	panic("missing callers frame")
}

type indirectCaller interface {
	call()
}

type indirectCallerImpl struct{}

//go:noinline
func checkInterfaceIndirectCaller() {
	var c indirectCaller = indirectCallerImpl{}
	c.call() // INTERFACE_CALL_MARK
}

//go:noinline
func (indirectCallerImpl) call() {
	interfaceMiddle()
}

//go:noinline
func interfaceMiddle() {
	checkCallerLine("interface", 2, INTERFACE_CALL_LINE)
}

//go:noinline
func checkClosureIndirectCaller() {
	f := closureLayer(closureLayer(func() {
		checkCallerLine("closure", 3, CLOSURE_CALL_LINE)
	}))
	f() // CLOSURE_CALL_MARK
}

//go:noinline
func closureLayer(next func()) func() {
	return func() {
		next()
	}
}

//go:noinline
func checkCallerLine(kind string, skip, want int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok || !strings.HasSuffix(file, "main.go") || line != want {
		panic("bad " + kind + " indirect caller line: " + file + ":" + strconv.Itoa(line))
	}
}

//go:noinline
func checkAdjacentRuntimeStack() {
	var buf1, buf2 [4096]byte
	n1 := runtime.Stack(buf1[:], false) // STACK_ONE_MARK
	n2 := runtime.Stack(buf2[:], false) // STACK_TWO_MARK
	line1 := stackLineFor(string(buf1[:n1]), "main.checkAdjacentRuntimeStack")
	line2 := stackLineFor(string(buf2[:n2]), "main.checkAdjacentRuntimeStack")
	if line1 != STACK_ONE_LINE || line2 != STACK_TWO_LINE || line1+1 != line2 {
		panic("bad adjacent stack lines: " + strconv.Itoa(line1) + "," + strconv.Itoa(line2))
	}
}

//go:noinline
func checkRecoveredDebugStackBounds() {
	defer func() {
		if recover() == nil {
			panic("missing bounds panic")
		}
		stack := string(debug.Stack())
		if !strings.Contains(stack, "main.go:BOUNDS_LINE") {
			panic("bad recovered stack: " + stack)
		}
	}()
	foo := Wrapper{a: []int{0, 1, 2}}
	_ = foo.Get(3) // BOUNDS_MARK
}

func stackLineFor(stack, fn string) int {
	lines := strings.Split(stack, "\n")
	for i := 0; i+1 < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == fn+"()" {
			loc := strings.TrimSpace(lines[i+1])
			colon := strings.LastIndexByte(loc, ':')
			if colon < 0 {
				return 0
			}
			rest := loc[colon+1:]
			end := strings.IndexByte(rest, ' ')
			if end >= 0 {
				rest = rest[:end]
			}
			n, _ := strconv.Atoi(rest)
			return n
		}
	}
	return 0
}
`

func TestRuntimeStatementLineInfo(t *testing.T) {
	source := runtimeStatementLineProbe
	source = strings.ReplaceAll(source, "CALLER_STMT_LINE", strconv.Itoa(markerLine(source, "CALLER_STMT_MARK")))
	source = strings.ReplaceAll(source, "CALLERS_STMT_LINE", strconv.Itoa(markerLine(source, "CALLERS_STMT_MARK")))
	source = strings.ReplaceAll(source, "INTERFACE_CALL_LINE", strconv.Itoa(markerLine(source, "INTERFACE_CALL_MARK")))
	source = strings.ReplaceAll(source, "CLOSURE_CALL_LINE", strconv.Itoa(markerLine(source, "CLOSURE_CALL_MARK")))
	source = strings.ReplaceAll(source, "STACK_ONE_LINE", strconv.Itoa(markerLine(source, "STACK_ONE_MARK")))
	source = strings.ReplaceAll(source, "STACK_TWO_LINE", strconv.Itoa(markerLine(source, "STACK_TWO_MARK")))
	source = strings.ReplaceAll(source, "BOUNDS_LINE", strconv.Itoa(markerLine(source, "BOUNDS_MARK")))

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
		t.Fatalf("llgo statement line probe failed: %v\n%s", err, out)
	}
}
