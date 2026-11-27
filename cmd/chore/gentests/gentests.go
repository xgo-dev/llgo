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

	"github.com/goplus/mod"
	"github.com/xgo-dev/llgo/cmd/compiler/cl/cltest"
	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/xtool/env/llvm"
)

func main() {
	llvm.SetupPath()
	dir, _, err := mod.FindGoMod(".")
	check(err)

	genMetaDir(dir, dir + "/compiler/cl/_testmeta")

	genExpects(dir)
}

func genExpects(root string) {
	runExpectDir(root, "compiler/cl/_testlibc")
	runExpectDir(root, "compiler/cl/_testrt")
	runExpectDir(root, "compiler/cl/_testgo")
	runExpectDir(root, "compiler/cl/_testlto", func(conf *build.Config) {
		conf.LTO = lto.Full
	})
	runExpectDir(root, "compiler/cl/_testpy")
	runExpectDir(root, "compiler/cl/_testdata")
}

func genMetaDir(root, dir string) {
	fis, err := os.ReadDir(dir)
	check(err)
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		testDir := filepath.Join(dir, name)
		if _, err := os.Stat(filepath.Join(testDir, "in.go")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			check(err)
		}
		relPath, err := filepath.Rel(root, testDir)
		check(err)
		relPath = filepath.ToSlash(relPath)
		fmt.Fprintln(os.Stderr, "meta", relPath)
		meta, err := cltest.CaptureMeta("./"+relPath, testDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", relPath, err)
			continue
		}
		check(os.WriteFile(filepath.Join(testDir, "meta-expect.txt"), []byte(meta), 0644))
	}
}

func runExpectDir(root, relDir string, configure ...func(*build.Config)) {
	dir := filepath.Join(root, relDir)
	fis, err := os.ReadDir(dir)
	check(err)
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		relPath := filepath.ToSlash(filepath.Join(relDir, name))
		testDir := filepath.Join(dir, name)
		expectFile, expect, err := cltest.ReadGolden(filepath.Join(testDir, "expect.txt"))
		if err != nil || cltest.IsOutputMarker(expect) {
			fmt.Fprintln(os.Stderr, "skip", relPath, "(no output golden)")
			continue
		}
		fmt.Fprintln(os.Stderr, "expect", relPath)
		pkgPath := "./" + relPath
		var output []byte
		if len(configure) == 0 {
			output, err = cltest.RunAndCapture(pkgPath, testDir)
		} else {
			conf := build.NewDefaultConf(build.ModeRun)
			for _, configure := range configure {
				configure(conf)
			}
			output, err = cltest.RunAndCaptureWithConf(pkgPath, testDir, conf)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", relPath, err)
			continue
		}
		check(os.WriteFile(expectFile, output, 0644))
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
