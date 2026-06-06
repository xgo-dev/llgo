//go:build !llgo
// +build !llgo

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

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestPackageUsesRuntimeMemProfile(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "no runtime import",
			src: `package foo
func f() {}
`,
		},
		{
			name: "runtime mem profile call",
			src: `package foo
import "runtime"
func f() {
	runtime.MemProfile(nil, false)
}
`,
			want: true,
		},
		{
			name: "runtime mem profile rate",
			src: `package foo
import "runtime"
var _ = runtime.MemProfileRate
`,
			want: true,
		},
		{
			name: "renamed runtime import",
			src: `package foo
import rt "runtime"
var _ = rt.MemProfileRate
`,
			want: true,
		},
		{
			name: "blank runtime import",
			src: `package foo
import _ "runtime"
func f() {}
`,
		},
		{
			name: "dot runtime import",
			src: `package foo
import . "runtime"
var _ = MemProfileRate
`,
		},
		{
			name: "other runtime selector",
			src: `package foo
import "runtime"
var _ = runtime.GOOS
`,
		},
		{
			name: "selector on non runtime value",
			src: `package foo
import "runtime"
var _ = runtime.GOOS
type profiler struct{}
var p profiler
var _ = p.MemProfile
`,
		},
		{
			name: "selector base is not identifier",
			src: `package foo
import "runtime"
type profiler struct{}
var p profiler
var _ = (p).MemProfile
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "foo.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := packageUsesRuntimeMemProfile([]*ast.File{file}); got != tt.want {
				t.Fatalf("packageUsesRuntimeMemProfile() = %v, want %v", got, tt.want)
			}
		})
	}

	badImport := &ast.File{
		Imports: []*ast.ImportSpec{{
			Path: &ast.BasicLit{Kind: token.STRING, Value: "runtime"},
		}},
	}
	if packageUsesRuntimeMemProfile([]*ast.File{badImport}) {
		t.Fatal("bad import literal should not enable memprofile instrumentation")
	}
}

func TestMemProfileFunctionName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "command-line-arguments.main", want: "main.main"},
		{name: "command-line-arguments.profiledAlloc", want: "main.profiledAlloc"},
		{name: "example.com/mod.profiledAlloc", want: "example.com/mod.profiledAlloc"},
	}
	for _, tt := range tests {
		if got := memProfileFunctionName(tt.name); got != tt.want {
			t.Fatalf("memProfileFunctionName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCompileRuntimeMemProfileInstrumentation(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `package foo
import "runtime"

var _ = runtime.MemProfileRate

func sample() {
}
`)
	fn := mustNamedFunction(t, m, "foo.sample")
	fnIR := fn.String()
	for _, want := range []string{"MemProfileEnter", "MemProfileExit"} {
		if !strings.Contains(fnIR, want) {
			t.Fatalf("compiled function missing %s instrumentation:\n%s", want, fnIR)
		}
	}
	ir := m.String()
	for _, want := range []string{"noinline", "disable-tail-calls"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled module missing %s attribute:\n%s", want, ir)
		}
	}
}
