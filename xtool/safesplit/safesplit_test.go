//go:build !llgo
// +build !llgo

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

package safesplit

import (
	"strings"
	"testing"
)

func TestSplitPkgConfigFlags(t *testing.T) {
	ftest := func(s, want string) {
		t.Helper() // for better error message
		got := toString(SplitPkgConfigFlags(s))
		if got != want {
			t.Errorf("\nSplitPkgConfigFlags(%q) =\n got %v\nwant %v", s, got, want)
		}
	}

	t.Run("basic", func(t *testing.T) {
		ftest("-I/usr/include -L/usr/lib", `["-I/usr/include" "-L/usr/lib"]`)
		ftest("-I /usr/include -L /usr/lib", `["-I/usr/include" "-L/usr/lib"]`)
		ftest("-L/opt/homebrew/Cellar/bdw-gc/8.2.8/lib -lgc",
			`["-L/opt/homebrew/Cellar/bdw-gc/8.2.8/lib" "-lgc"]`)
	})

	t.Run("spaces_in_path", func(t *testing.T) {
		ftest("-I/usr/local/include directory -L/usr/local/lib path",
			`["-I/usr/local/include directory" "-L/usr/local/lib path"]`)
	})

	t.Run("multiple_spaces", func(t *testing.T) {
		ftest("  -I  /usr/include   -L   /usr/lib  ", `["-I/usr/include" "-L/usr/lib"]`)
	})

	t.Run("consecutive_flags", func(t *testing.T) {
		ftest("-I -L", `["-I" "-L"]`)
		ftest("-I -L /usr/lib", `["-I" "-L/usr/lib"]`)
	})

	t.Run("edge_cases", func(t *testing.T) {
		ftest("", "[]")
		ftest("   ", "[]")
		ftest("-", `["-"]`)
		ftest("-I", `["-I"]`)
		ftest("-I -", `["-I" "-"]`)
	})

	t.Run("escaped_spaces", func(t *testing.T) {
		ftest(`-I/path\ with\ spaces -L/lib`, `["-I/path with spaces" "-L/lib"]`)
		ftest(`-I /first\ path -L /second\ long path`, `["-I/first path" "-L/second long path"]`)
	})

	t.Run("macro_flags", func(t *testing.T) {
		ftest("-DMACRO -I/usr/include", `["-DMACRO" "-I/usr/include"]`)
		ftest("-D MACRO -I/usr/include", `["-DMACRO" "-I/usr/include"]`)
		ftest("-DMACRO=value -I/usr/include", `["-DMACRO=value" "-I/usr/include"]`)
		ftest("-D MACRO=value -I/usr/include", `["-DMACRO=value" "-I/usr/include"]`)
		ftest("-D_DEBUG -D_UNICODE -DWIN32", `["-D_DEBUG" "-D_UNICODE" "-DWIN32"]`)
		ftest("-D _DEBUG -D _UNICODE -D WIN32", `["-D_DEBUG" "-D_UNICODE" "-DWIN32"]`)
		ftest("-DVERSION=2.1 -DDEBUG=1", `["-DVERSION=2.1" "-DDEBUG=1"]`)
		ftest("-D VERSION=2.1 -D DEBUG=1", `["-DVERSION=2.1" "-DDEBUG=1"]`)
	})

	// case for https://github.com/xgo-dev/llgo/issues/1244
	t.Run("w_pipe", func(t *testing.T) {
		ftest("-w -pipe", `["-w" "-pipe"]`)
		ftest("-Os -w -pipe", `["-Os" "-w" "-pipe"]`)
	})
}

func toString(ss []string) string {
	if ss == nil {
		return "[]"
	}
	s := "["
	for i, v := range ss {
		if i > 0 {
			s += " "
		}
		v = strings.ReplaceAll(v, `"`, `\"`)
		s += `"` + v + `"`
	}
	return s + "]"
}
