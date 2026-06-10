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

const runtimeFuncForPCNamesProbe = `package main

import (
	"fmt"
	"reflect"
	"runtime"
)

func f(n int) int {
	return n % 2
}

func g(n int) int {
	return f(n)
}

func name(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func check(label, got, want string) {
	if got != want {
		panic(label + ": got " + got + ", want " + want)
	}
}

func main() {
	check("f", name(f), "main.f")
	check("g", name(g), "main.g")
	fmt.Println("ok")
}
`

func TestRuntimeFuncForPCNamesWithLLGo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(runtimeFuncForPCNamesProbe), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", file)

	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	if got := strings.TrimSpace(runGoCmd(t, root, "run", "./cmd/llgo", "run", file)); got != "ok" {
		t.Fatalf("llgo probe output = %q, want ok", got)
	}
}
