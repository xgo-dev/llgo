/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"testing"
)

const stringConversionProbe = `package main

func runesFromInt64(v int64) []rune {
	var out []rune
	for _, r := range string(v) {
		out = append(out, r)
	}
	return out
}

func runesFromUint64(v uint64) []rune {
	var out []rune
	for _, r := range string(v) {
		out = append(out, r)
	}
	return out
}

func check(name string, got []rune, want rune) {
	if len(got) != 1 || got[0] != want {
		panic(name)
	}
}

func main() {
	check("int64", runesFromInt64(0x1F642), '\U0001F642')
	check("int64-out-of-range", runesFromInt64(0x110000), '\uFFFD')
	check("uint64", runesFromUint64(0x1F642), '\U0001F642')
	check("uint64-out-of-range", runesFromUint64(0x110000), '\uFFFD')
}
`

const emptyStringConversionProbe = `package main

type mystring string
type mybytes []byte
type myrunes []rune

func checkBytes(s []byte, name string) {
	if len(s) != 0 {
		panic("len(" + name + ") != 0")
	}
	if s == nil {
		panic(name + " == nil")
	}
}

func checkRunes(s []rune, name string) {
	if len(s) != 0 {
		panic("len(" + name + ") != 0")
	}
	if s == nil {
		panic(name + " == nil")
	}
}

func main() {
	checkBytes([]byte(""), "[]byte(\"\")")
	checkBytes([]byte(mystring("")), "[]byte(mystring(\"\"))")
	checkBytes(mybytes(""), "mybytes(\"\")")
	checkBytes(mybytes(mystring("")), "mybytes(mystring(\"\"))")

	checkRunes([]rune(""), "[]rune(\"\")")
	checkRunes([]rune(mystring("")), "[]rune(mystring(\"\"))")
	checkRunes(myrunes(""), "myrunes(\"\")")
	checkRunes(myrunes(mystring("")), "myrunes(mystring(\"\"))")
}
`

func TestStringConversionFromWideIntegers(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(stringConversionProbe), 0644); err != nil {
		t.Fatal(err)
	}
	repoRoot := findStringConversionRepoRoot(t)
	runStringConversionProbe(t, repoRoot, "go", "run", file)
	t.Setenv("LLGO_ROOT", repoRoot)
	runStringConversionProbe(t, repoRoot, "go", "run", "./cmd/llgo", "run", file)
}

func TestEmptyStringToByteRuneSlicesNonNil(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(emptyStringConversionProbe), 0644); err != nil {
		t.Fatal(err)
	}
	repoRoot := findStringConversionRepoRoot(t)
	runStringConversionProbe(t, repoRoot, "go", "run", file)
	t.Setenv("LLGO_ROOT", repoRoot)
	runStringConversionProbe(t, repoRoot, "go", "run", "./cmd/llgo", "run", file)
}

func runStringConversionProbe(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func findStringConversionRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
