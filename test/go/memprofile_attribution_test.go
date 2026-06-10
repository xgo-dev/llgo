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
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeMemProfileAttribution(t *testing.T) {
	dir := t.TempDir()
	src := `package main

import "runtime"

var sink *[1024]byte

//go:noinline
func profiledAlloc(n int) {
	for i := 0; i < n; i++ {
		sink = new([1024]byte)
		runtime.Gosched()
	}
}

//go:noinline
func profiledAllocCaller(n int) {
	profiledAlloc(n)
}

func main() {
	runtime.MemProfileRate = 1
	runtime.Gosched()
	profiledAllocCaller(25)
	runtime.GC()
	runtime.GC()

	var records []runtime.MemProfileRecord
	for tries := 0; tries < 5; tries++ {
		n, ok := runtime.MemProfile(records, true)
		if ok {
			records = records[:n]
			break
		}
		if n == 0 {
			panic("empty memory profile")
		}
		records = make([]runtime.MemProfileRecord, n+8)
	}
	if records == nil {
		panic("profile kept growing")
	}

	for _, r := range records {
		if r.AllocObjects < 25 || r.AllocBytes < 25*1024 {
			continue
		}
		frames := runtime.CallersFrames(r.Stack())
		firstLine := 0
		seenAlloc := false
		seenCaller := false
		for {
			frame, more := frames.Next()
			if firstLine == 0 {
				firstLine = frame.Line
			}
			if frame.Function == "main.profiledAlloc" {
				seenAlloc = true
			}
			if frame.Function == "main.profiledAllocCaller" {
				seenCaller = true
			}
			if seenAlloc && seenCaller {
				break
			}
			if !more {
				break
			}
		}
		if seenAlloc && seenCaller {
			if firstLine == 0 {
				panic("profiled allocation has no source line")
			}
			return
		}
	}
	for _, r := range records {
		println("record", r.AllocObjects, r.AllocBytes)
		frames := runtime.CallersFrames(r.Stack())
		for {
			frame, more := frames.Next()
			println("frame", frame.Function, frame.Line)
			if !more {
				break
			}
		}
	}
	panic("profiledAlloc not found in memory profile")
}
`
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	root := findLLGoRoot(t)
	out := runGoCmd(t, root, "run", "./cmd/llgo", "run", mainFile)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("unexpected output: %q", out)
	}
}
