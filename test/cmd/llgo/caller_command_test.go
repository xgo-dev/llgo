//go:build !wasm

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

package llgocmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestTestingFailureLocation(t *testing.T) {
	dir := t.TempDir()
	const source = `package tpkg

import "testing"

func TestFail(t *testing.T) {
	t.Errorf("acceptance failure") // TESTING_MARK
}
`
	writeCommandTestFiles(t, dir, map[string]string{
		"x_test.go": source,
		"go.mod":    "module tpkg\n\ngo 1.21\n",
	})
	output, err := runCompiler(t, dir, "test", ".")
	if err == nil {
		t.Fatalf("%s failing test unexpectedly passed:\n%s", toolCompilerName, output)
	}
	wantLine := markerLine(source, "TESTING_MARK")
	if want := "x_test.go:" + strconv.Itoa(wantLine) + ": acceptance failure"; !strings.Contains(output, want) {
		t.Fatalf("%s testing failure is missing %q:\n%s", toolCompilerName, want, output)
	}
	if !strings.Contains(output, "--- FAIL: TestFail") {
		t.Fatalf("%s testing failure is missing FAIL header:\n%s", toolCompilerName, output)
	}
}

func TestModuleMainRuntimeNaming(t *testing.T) {
	dir := t.TempDir()
	const source = `package main

import (
	"os"
	"runtime"
	"strings"
)

//go:noinline
func here() (string, bool) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	return runtime.FuncForPC(pc).Name(), true
}

func main() {
	name, ok := here()
	if !ok || name != "main.here" {
		panic("bad module-main name: " + name)
	}
	var pcs [8]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(0, pcs[:])])
	foundMain := false
	for {
		frame, more := frames.Next()
		if strings.HasPrefix(frame.Function, "mymainmod.") {
			panic("module path leaked into frame name: " + frame.Function)
		}
		if frame.Function == "main.main" {
			foundMain = true
		}
		if !more {
			break
		}
	}
	if !foundMain {
		panic("main.main frame missing")
	}
	os.Stdout.WriteString("MODMAIN_OK\n")
}
`
	writeCommandTestFiles(t, dir, map[string]string{
		"main.go": source,
		"go.mod":  "module mymainmod\n\ngo 1.21\n",
	})
	output, err := runCompiler(t, dir, "run", ".")
	if err != nil || !strings.Contains(output, "MODMAIN_OK") {
		t.Fatalf("%s module-main probe failed: %v\n%s", toolCompilerName, err, output)
	}
}

func TestLogicalRuntimeCallerTail(t *testing.T) {
	dir := t.TempDir()
	const mainSource = `package main

import (
	"os"
	"runtime"
	_ "caller-tail/probe"
)

func main() {
	want := []string{"main.main", "runtime.main", "runtime.goexit"}
	for skip, name := range want {
		pc, _, _, ok := runtime.Caller(skip)
		got := "<missing>"
		if ok {
			got = runtime.FuncForPC(pc).Name()
		}
		if got != name {
			panic("bad runtime caller tail: got " + got + ", want " + name)
		}
	}
	os.Stdout.WriteString("CALLER_TAIL_OK\n")
}
`
	const probeSource = `package probe

import (
	"runtime"
	"strings"
)

func init() {
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	seenInit := false
	var names []string
	for {
		frame, more := frames.Next()
		names = append(names, frame.Function)
		if strings.Contains(frame.Function, "/probe.init") {
			seenInit = true
		} else if seenInit && strings.HasPrefix(frame.Function, "runtime.") {
			return
		}
		if !more {
			panic("runtime initialization driver missing: " + strings.Join(names, ", "))
		}
	}
}
`
	writeCommandTestFiles(t, dir, map[string]string{
		"main.go":        mainSource,
		"probe/probe.go": probeSource,
		"go.mod":         "module caller-tail\n\ngo 1.21\n",
	})
	output, err := runCompiler(t, dir, "run", ".")
	if err != nil || !strings.Contains(output, "CALLER_TAIL_OK") {
		t.Fatalf("%s caller-tail probe failed: %v\n%s", toolCompilerName, err, output)
	}
}

func writeCommandTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
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
