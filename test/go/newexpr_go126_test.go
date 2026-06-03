//go:build go1.26
// +build go1.26

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

const newExpressionProbe = `package main

var (
	globalNewExpr = new(0)
	globalAlias   = globalNewExpr
)

func check(name string, ok bool) {
	if !ok {
		panic(name)
	}
}

func main() {
	{
		p := new(123)
		check("untyped constant", *p == 123)
	}
	{
		x := 42
		p := new(x)
		check("non-constant", *p == x)
	}
	{
		x := [2]int{123, 456}
		p := new(x)
		check("composite value", *p == x)
	}
	{
		var i int
		p := new(i > 0)
		check("untyped bool expression", *p == false)
	}
	check("global initializer", globalAlias == globalNewExpr && *globalNewExpr == 0)
}
`

func TestNewExpressionInitializesAllocatedValue(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(newExpressionProbe), 0644); err != nil {
		t.Fatal(err)
	}

	repoRoot := findStringConversionRepoRoot(t)
	runStringConversionProbe(t, repoRoot, "go", "run", file)
	t.Setenv("LLGO_ROOT", repoRoot)
	runStringConversionProbe(t, repoRoot, "go", "run", "./cmd/llgo", "run", file)
}
