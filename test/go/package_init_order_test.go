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
	"sync"
	"testing"
)

var (
	packageInitLLGoOnce sync.Once
	packageInitLLGoBin  string
	packageInitLLGoErr  string
)

func packageInitLLGo(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	if llgo := configuredLLGo(t); llgo != "" {
		return llgo
	}
	packageInitLLGoOnce.Do(func() {
		dir, err := os.MkdirTemp("", "llgo-package-init-bin")
		if err != nil {
			packageInitLLGoErr = err.Error()
			return
		}
		packageInitLLGoBin = filepath.Join(dir, "llgo")
		cmd := exec.Command("go", "build", "-tags=dev", "-o", packageInitLLGoBin, "./cmd/llgo")
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			packageInitLLGoErr = err.Error() + "\n" + string(out)
		}
	})
	if packageInitLLGoErr != "" {
		t.Fatalf("building llgo failed: %s", packageInitLLGoErr)
	}
	return packageInitLLGoBin
}

func TestPackageInitializationUsesLexicalReadyOrder(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module initorder\n\ngo 1.21\n",
		"tracker/tracker.go": `package tracker

var Order []string

func Add(name string) { Order = append(Order, name) }
`,
		"z/z.go": `package z

import "initorder/tracker"

func init() { tracker.Add("z") }
`,
		"a/a.go": `package a

import (
	_ "initorder/z"
	"initorder/tracker"
)

func init() { tracker.Add("a") }
`,
		"b/b.go": `package b

import "initorder/tracker"

func init() { tracker.Add("b") }
`,
		"main.go": `package main

import (
	_ "initorder/a"
	_ "initorder/b"
	"initorder/tracker"
	"strings"
)

func main() {
	if got, want := strings.Join(tracker.Order, ","), "b,z,a"; got != want {
		panic("package init order: got " + got + " want " + want)
	}
}
`,
	}
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	llgo := packageInitLLGo(t)
	cmd := exec.Command(llgo, "run", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("llgo package-init probe failed: %v\n%s", err, out)
	}
}
