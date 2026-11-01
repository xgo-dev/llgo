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

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestPreparedFunctionDirectivesSharedWithBackends(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", `package p
//go:noinline
func F() {}
func G() {}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("p", "p"), files, gossa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.AddFunctionDirectives(directive.CollectFunctions(files))
	pkg.Func("F").Syntax().(*ast.FuncDecl).Doc = nil
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	if !backend.FunctionDirectives(pkg.Func("F")).NoInline || backend.FunctionDirectives(pkg.Func("G")).NoInline {
		t.Fatal("backend lost prepared source attributes")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if !backend.FunctionDirectives(pkg.Func("F")).NoInline || backend.FunctionDirectives(pkg.Func("G")).NoInline {
				t.Error("concurrent backend read changed prepared attributes")
			}
		})
	}
	wg.Wait()
}
