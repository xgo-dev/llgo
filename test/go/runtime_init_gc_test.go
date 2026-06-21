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
	"testing"
)

func TestRuntimeGCStatsDuringInit(t *testing.T) {
	dir := t.TempDir()
	src := `package main

import "runtime"

var initialized bool

func init() {
	c := make(chan struct{})
	go func() {
		c <- struct{}{}
	}()
	<-c

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		panic("NumGC did not advance during init")
	}
	initialized = true
}

func main() {
	if !initialized {
		panic("init did not run")
	}
}
`
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", mainFile)

	root := findLLGoRoot(t)
	runGoCmd(t, root, "run", "./cmd/llgo", "run", mainFile)
}
