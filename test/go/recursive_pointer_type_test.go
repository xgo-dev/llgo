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

const recursivePointerTypeProbe = `package main

type Link *Link
type Peano *Peano

func box(v Link) *Link {
	p := new(Link)
	*p = v
	return p
}

func unbox(p *Link) Link {
	return *p
}

func makePeano(n int) *Peano {
	if n == 0 {
		return nil
	}
	p := Peano(makePeano(n - 1))
	return &p
}

var countArg Peano
var countResult int

func countPeano() {
	if countArg == nil {
		countResult = 0
		return
	}
	countArg = *countArg
	countPeano()
	countResult++
}

func main() {
	sentinel := Link(new(Link))
	p := box(sentinel)
	if unbox(p) != sentinel {
		panic("recursive pointer type lost value")
	}

	countArg = makePeano(4096)
	countPeano()
	if countResult != 4096 {
		panic("recursive Peano pointer count failed")
	}
}
`

func TestRecursivePointerTypeBuilds(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(recursivePointerTypeProbe), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", mainFile)

	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	runGoCmd(t, root, "run", "./cmd/llgo", "run", mainFile)
}
