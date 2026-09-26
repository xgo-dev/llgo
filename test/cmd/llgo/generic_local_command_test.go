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
	"testing"
)

const genericNestedLocalCommandLineProbe = `package main

import (
	"fmt"
	"reflect"
)

type intish interface{ ~int }

func nested[A intish]() (reflect.Type, reflect.Type) {
	type Int int
	type T[B intish] struct{}
	type U[_ any] int
	return reflect.TypeOf(T[int]{}), reflect.TypeOf(T[U[int]]{})
}

func main() {
	direct, nested := nested[int]()
	fmt.Println(direct)
	fmt.Println(nested)
}
`

func TestGenericNestedLocalRuntimeTypeNamesForCommandLineMain(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(genericNestedLocalCommandLineProbe), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := runCompiler(t, dir, "run", file)
	if err != nil {
		t.Fatalf("%s command-line main probe failed: %v\n%s", toolCompilerName, err, output)
	}
	const want = "main.T[int;int]\nmain.T[int;main.U[int;int]·3]\n"
	if output != want {
		t.Fatalf("%s probe output = %q, want %q", toolCompilerName, output, want)
	}
}
