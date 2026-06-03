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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCgoMallocWrapperSymbols(t *testing.T) {
	if strings.TrimSpace(runGoCmd(t, "", "env", "CGO_ENABLED")) != "1" {
		t.Skip("cgo is disabled")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang is unavailable")
	}

	dir := t.TempDir()
	src := `package main

/*
#include <stdlib.h>
*/
import "C"
import "unsafe"

func main() {
	p := C.malloc(8)
	if p == nil {
		panic("C.malloc returned nil")
	}
	C.free(unsafe.Pointer(p))
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

func runGoCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		t.Fatalf("go %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func findLLGoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/goplus/llgo") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find llgo repository root")
		}
		dir = parent
	}
}
