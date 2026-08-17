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

package parser

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/xgo-dev/llgo/xtool/clang/ast"
)

type Mode uint

// -----------------------------------------------------------------------------

type ParseError struct {
	Err    error
	Stderr []byte
}

func (p *ParseError) Error() string {
	if len(p.Stderr) > 0 {
		return string(p.Stderr)
	}
	return p.Err.Error()
}

// -----------------------------------------------------------------------------

type Config struct {
	Json   *[]byte
	Flags  []string
	Stderr bool
}

func DumpAST(filename string, conf *Config) (result []byte, warning []byte, err error) {
	if conf == nil {
		conf = new(Config)
	}
	skiperr := strings.HasSuffix(filename, "vfprintf.c.i")
	stdout := NewPagedWriter()
	stderr := new(bytes.Buffer)
	args := []string{"-Xclang", "-ast-dump=json", "-fsyntax-only", filename}
	if len(conf.Flags) != 0 {
		args = append(conf.Flags, args...)
	}
	cmd := exec.Command("clang", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	if conf.Stderr && !skiperr {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = stderr
	}
	err = cmd.Run()
	errmsg := stderr.Bytes()
	if err != nil && !skiperr {
		return nil, nil, &ParseError{Err: err, Stderr: errmsg}
	}
	return stdout.Bytes(), errmsg, nil
}

// -----------------------------------------------------------------------------

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func ParseFileEx(filename string, mode Mode, conf *Config) (file *ast.Node, warning []byte, err error) {
	out, warning, err := DumpAST(filename, conf)
	if err != nil {
		return
	}
	if conf != nil && conf.Json != nil {
		*conf.Json = out
	}
	file = new(ast.Node)
	err = json.Unmarshal(out, file)
	if err != nil {
		err = &ParseError{Err: err}
	}
	return
}

func ParseFile(filename string, mode Mode) (file *ast.Node, warning []byte, err error) {
	return ParseFileEx(filename, mode, nil)
}

// -----------------------------------------------------------------------------
