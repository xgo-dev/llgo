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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/llgo/cl/cltest"
	"github.com/goplus/llgo/internal/littest"
	"github.com/goplus/llgo/internal/llgen"
	"github.com/goplus/llgo/internal/semmeta"
	"github.com/goplus/llvm"
	"github.com/goplus/mod"
)

func main() {
	dir, _, err := mod.FindGoMod(".")
	check(err)

	llgenDir(dir + "/cl/_testlibc")
	llgenDir(dir + "/cl/_testlibgo")
	llgenDir(dir + "/cl/_testrt")
	llgenDir(dir + "/cl/_testgo")
	llgenDir(dir + "/cl/_testpy")
	llgenDir(dir + "/cl/_testdata")
	genMetaDir(dir + "/cl/_testmeta")

	genExpects(dir)
}

func llgenDir(dir string) {
	forEachTestDir(dir, func(name, testDir string) {
		skip, err := dirHasLITTESTSource(testDir)
		check(err)
		if skip {
			fmt.Fprintln(os.Stderr, "skip llgen", testDir, "(// LITTEST)")
			return
		}
		fmt.Fprintln(os.Stderr, "llgen", testDir)
		check(os.Chdir(testDir))
		outFile := filepath.Join(testDir, "out.ll")
		skipOut := false
		if b, err := os.ReadFile(outFile); err == nil && len(b) == 1 && b[0] == ';' {
			skipOut = true
		}
		mod, err := llgen.GenModuleFrom(testDir)
		check(err)
		func() {
			defer mod.Dispose()
			if !skipOut {
				check(os.WriteFile(outFile, []byte(mod.String()), 0644))
			}
		}()
	})
}

func genMetaDir(dir string) {
	forEachTestDir(dir, func(name, testDir string) {
		check(os.Chdir(testDir))
		mod, err := llgen.GenModuleFrom(testDir)
		check(err)
		func() {
			defer mod.Dispose()
			writeMetaExpect(testDir, mod)
		}()
	})
}

func writeMetaExpect(testDir string, mod llvm.Module) {
	metaFile := filepath.Join(testDir, "meta-expect.txt")
	info := semmeta.Read(mod)
	text := cltest.FormatSemMeta(info)
	check(os.WriteFile(metaFile, []byte(text), 0644))
}

func genExpects(root string) {
	runExpectDir(root, "cl/_testlibc")
	runExpectDir(root, "cl/_testlibgo")
	runExpectDir(root, "cl/_testrt")
	runExpectDir(root, "cl/_testgo")
	runExpectDir(root, "cl/_testpy")
	runExpectDir(root, "cl/_testdata")
}

func runExpectDir(root, relDir string) {
	forEachTestDir(filepath.Join(root, relDir), func(name, testDir string) {
		relPath := filepath.ToSlash(filepath.Join(relDir, name))
		fmt.Fprintln(os.Stderr, "expect", relPath)
		pkgPath := "./" + relPath
		output, err := cltest.RunAndCapture(pkgPath, testDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", relPath, err)
			output = []byte{';'}
		}
		expectFile := filepath.Join(testDir, "expect.txt")
		expect, err := os.ReadFile(expectFile)
		if err != nil || strings.TrimSpace(string(expect)) == ";" {
			fmt.Fprintln(os.Stderr, "skip", relPath, "(expect is ';')")
			return
		}
		check(os.WriteFile(expectFile, output, 0644))
	})
}

func forEachTestDir(dir string, fn func(name, testDir string)) {
	fis, err := os.ReadDir(dir)
	check(err)
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		fn(name, filepath.Join(dir, name))
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func dirHasLITTESTSource(dir string) (bool, error) {
	_, ok, err := littest.FindMarkedSourceFile(dir)
	if err != nil {
		return false, err
	}
	return ok, nil
}
