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

package llgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goplus/llgo/internal/build"
)

func GenFrom(fileOrPkg string) string {
	pkg, err := genFrom(fileOrPkg, 0)
	check(err)
	return pkg.LPkg.String()
}

func genFrom(pkgPath string, abiMode build.AbiMode) (build.Package, error) {
	oldDbg := os.Getenv("LLGO_DEBUG")
	oldDbgSyms := os.Getenv("LLGO_DEBUG_SYMBOLS")
	dbg := isDbgSymEnabled(filepath.Join(pkgPath, "flags.txt"))
	if dbg {
		os.Setenv("LLGO_DEBUG", "1")
		os.Setenv("LLGO_DEBUG_SYMBOLS", "1")
	}
	defer func() {
		os.Setenv("LLGO_DEBUG", oldDbg)
		os.Setenv("LLGO_DEBUG_SYMBOLS", oldDbgSyms)
	}()

	conf := &build.Config{
		Mode:    build.ModeGen,
		AbiMode: abiMode,
		GenLL:   true,
	}
	pkgs, err := build.Do([]string{pkgPath}, conf)
	if err != nil {
		return nil, err
	}
	return pkgs[0], nil
}

func DoFile(fileOrPkg, outFile string) {
	ret := GenFrom(fileOrPkg)
	err := os.WriteFile(outFile, []byte(ret), 0644)
	check(err)
}

func isDbgSymEnabled(flagsFile string) bool {
	data, err := os.ReadFile(flagsFile)
	if err != nil {
		return false
	}
	toks := strings.Split(strings.Join(strings.Split(string(data), "\n"), " "), " ")
	for _, tok := range toks {
		if tok == "-dbg" {
			return true
		}
	}
	return false
}

func SmartDoFile(pkgPath string) {
	SmartDoFileEx(pkgPath, 0)
}

func SmartDoFileEx(pkgPath string, abiMode build.AbiMode) {
	pkg, err := genFrom(pkgPath, abiMode)
	check(err)

	const autgenFile = "llgo_autogen.ll"
	dir, _ := filepath.Split(pkg.GoFiles[0])
	absDir, _ := filepath.Abs(dir)
	absDir = filepath.ToSlash(absDir)
	fname := autgenFile
	if inCompilerDir(absDir) {
		fname = "out.ll"
	}
	outFile := dir + fname

	b, err := os.ReadFile(outFile)
	if err == nil && len(b) == 1 && b[0] == ';' {
		return // skip to gen
	}

	if err = os.WriteFile(outFile, []byte(pkg.LPkg.String()), 0644); err != nil {
		panic(err)
	}
	if false && fname == autgenFile {
		genZip(absDir, "llgo_autogen.lla", autgenFile)
	}
}

func genZip(dir string, outFile, inFile string) {
	cmd := exec.Command("zip", outFile, inFile)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func inCompilerDir(dir string) bool {
	return strings.Contains(dir, "/cl/_test")
}
