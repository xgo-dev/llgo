/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package preprocessor

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xgo-dev/llgo/xtool/clang/pathutil"
)

const (
	DbgFlagExecCmd = 1 << iota
	DbgFlagAll     = DbgFlagExecCmd
)

var (
	debugExecCmd bool
)

func SetDebug(flags int) {
	debugExecCmd = (flags & DbgFlagExecCmd) != 0
}

// -----------------------------------------------------------------------------

type Config struct {
	Compiler    string // default: clang
	PPFlag      string // default: -E
	BaseDir     string // base of include searching directory, should be absolute path
	IncludeDirs []string
	Defines     []string
	Flags       []string
}

func Do(infile, outfile string, conf *Config) (err error) {
	if infile, err = filepath.Abs(infile); err != nil {
		return
	}
	if outfile, err = filepath.Abs(outfile); err != nil {
		return
	}
	if conf == nil {
		conf = new(Config)
	}
	base := conf.BaseDir
	if base == "" {
		if base, err = os.Getwd(); err != nil {
			return
		}
	}
	compiler := conf.Compiler
	if compiler == "" {
		compiler = "clang"
	}
	ppflag := conf.PPFlag
	if ppflag == "" {
		ppflag = "-E"
	}
	n := 4 + len(conf.Flags) + len(conf.IncludeDirs) + len(conf.Defines)
	args := make([]string, 3, n)
	args[0] = ppflag
	args[1], args[2] = "-o", outfile
	args = append(args, conf.Flags...)
	for _, def := range conf.Defines {
		args = append(args, "-D"+def)
	}
	for _, inc := range conf.IncludeDirs {
		args = append(args, "-I"+pathutil.Canonical(base, inc))
	}
	args = append(args, infile)
	if debugExecCmd {
		log.Println("==> runCmd:", compiler, args)
	}
	cmd := exec.Command(compiler, args...)
	cmd.Dir = filepath.Dir(infile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// -----------------------------------------------------------------------------
