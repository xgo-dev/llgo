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

package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const dwarfPCLNProbe = `package main

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func checkCaller() {
	_, file, line, ok := runtime.Caller(0) // CALLER_MARK
	if !ok || !strings.HasSuffix(file, "main.go") || line != CALLER_LINE {
		panic("bad caller: " + file + ":" + strconv.Itoa(line))
	}
}

func checkFrames() {
	var pcs [8]uintptr
	n := runtime.Callers(0, pcs[:]) // FRAMES_MARK
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "main.checkFrames" {
			if !strings.HasSuffix(frame.File, "main.go") || frame.Line != FRAMES_LINE {
				panic("bad frame: " + frame.File + ":" + strconv.Itoa(frame.Line))
			}
			break
		}
		if !more {
			panic("checkFrames frame missing")
		}
	}
}

//go:noinline
func panicSite() {
	panic("dwarf panic")
}

func checkRecoveredPanic() {
	defer func() {
		if recover() != "dwarf panic" {
			panic("bad recovery")
		}
		var pcs [32]uintptr
		n := runtime.Callers(1, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if frame.Function == "main.panicSite" {
				return
			}
			if !more {
				panic("recovered stack lost panicSite")
			}
		}
	}()
	panicSite()
}

func main() {
	var out strings.Builder
	logger := log.New(&out, "", log.Lshortfile)
	logger.Print("site") // LOG_MARK
	want := "main.go:" + strconv.Itoa(LOG_LINE) + ":"
	if !strings.HasPrefix(out.String(), want) {
		panic("bad log site: " + out.String())
	}

	checkCaller()
	checkFrames()
	checkRecoveredPanic()
	os.Stdout.WriteString("PCLN_DWARF_OK\n")
}
`

func TestDWARFPCLNLineSites(t *testing.T) {
	source := dwarfPCLNProbe
	for _, marker := range []string{"CALLER", "FRAMES", "LOG"} {
		source = strings.ReplaceAll(source, marker+"_LINE", strconv.Itoa(dwarfPCLNMarkerLine(source, marker+"_MARK")))
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "probe")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	if _, err := Do([]string{file}, conf); err != nil {
		t.Fatal(err)
	}
	if out := runBinary(t, conf.OutFile); !strings.Contains(out, "PCLN_DWARF_OK") {
		t.Fatalf("unexpected DWARF PCLN probe output: %s", out)
	}
}

func dwarfPCLNMarkerLine(source, marker string) int {
	for i, line := range strings.Split(source, "\n") {
		if strings.Contains(line, marker) {
			return i + 1
		}
	}
	panic("missing marker: " + marker)
}
