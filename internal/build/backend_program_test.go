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

package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
)

func TestBackendProgramTemplateCreatesIsolatedSessions(t *testing.T) {
	conf := &Config{Goos: "linux", Goarch: "amd64", RewriteMainPrefix: true}
	template := newBackendProgramTemplate(
		&llssa.Target{GOOS: conf.Goos, GOARCH: conf.Goarch, RewriteMainPrefix: conf.RewriteMainPrefix},
		conf,
		true,
		true,
	)
	first, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	defer first.prog.Dispose()
	second, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	defer second.prog.Dispose()
	if first.transformer == nil || second.transformer == nil {
		t.Fatal("backend session missing C ABI transformer")
	}
	if !first.prog.FuncInfoMetadataEnabled() || !first.prog.FuncInfoSitesEnabled() {
		t.Fatal("backend template did not preserve funcinfo configuration")
	}
	if !first.prog.Target().RewriteMainPrefix || !second.prog.Target().RewriteMainPrefix {
		t.Fatal("backend template did not preserve invocation-local symbol naming")
	}
	firstModule := first.prog.NewPackage("first", "example.com/first").Module()
	secondModule := second.prog.NewPackage("second", "example.com/second").Module()
	if firstModule.Context().C == secondModule.Context().C {
		t.Fatal("backend sessions share an LLVM context")
	}
}

func TestBackendProgramTemplateReplaysCPackageExports(t *testing.T) {
	const src = `package C
func Xadd(a, b int) int { return a + b }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "c.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("example.com/c", "C")
	template := backendProgramTemplate{
		inputs: []backendProgramInput{{
			fset:        fset,
			pkg:         pkg,
			files:       []*ast.File{file},
			parseSyntax: true,
		}},
	}
	session, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.prog.Dispose()
	const fullName = "example.com/c.Xadd"
	if got, ok := session.prog.Linkname(fullName); !ok || got != "add" {
		t.Fatalf("replayed Linkname(%q) = (%q, %v), want (%q, true)", fullName, got, ok, "add")
	}
}
