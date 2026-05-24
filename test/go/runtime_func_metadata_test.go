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

const runtimeFuncMetadataProbe = `package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type T struct {
	a, b int
}

func f(t *T) int {
	if t != nil {
		return t.b
	}
	return 0
}

func g(t *T) int {
	return f(t) + 5
}

func check(label, got, want string) {
	if got != want {
		panic(label + ": got " + got + ", want " + want)
	}
}

func main() {
	check("func f", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), "main.f")
	check("func g", runtime.FuncForPC(reflect.ValueOf(g).Pointer()).Name(), "main.g")
	fmt.Println("ok")
}
`

func TestRuntimeFuncMetadataWithLLGo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(runtimeFuncMetadataProbe), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", file)

	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	if got := strings.TrimSpace(runGoCmd(t, root, "run", "./cmd/llgo", "run", file)); got != "ok" {
		t.Fatalf("llgo probe output = %q, want ok", got)
	}
}
