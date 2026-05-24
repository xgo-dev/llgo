//go:build !llgo

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
	"runtime"
	"testing"
)

func TestMayMoreStackGCFlag(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	tmp := t.TempDir()
	src := filepath.Join(tmp, "main.go")
	out := filepath.Join(tmp, "maymorestack")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	if err := os.WriteFile(src, []byte(mayMoreStackProgram), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "-tags=dev", "./cmd/llgo", "build", "-gcflags=-d=maymorestack=main.mayMoreStack", "-o", out, src)
	cmd.Dir = repoRoot
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("llgo build failed: %v\n%s", err, data)
	}

	cmd = exec.Command(out)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("maymorestack program failed: %v\n%s", err, data)
	}
}

const mayMoreStackProgram = `
package main

var count int

//go:nosplit
func mayMoreStack() {
	count++
}

func main() {
	const want = 8
	anotherFunc(want - 1)
	if count != want {
		println(count, "!=", want)
		panic("wrong number of calls to mayMoreStack")
	}
}

//go:noinline
func anotherFunc(n int) {
	var x [16]byte
	if n > 1 {
		anotherFunc(n - 1)
	}
	_ = x
}
`
