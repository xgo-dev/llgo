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
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	gossa "golang.org/x/tools/go/ssa"
)

func TestPreparedNoInlineCallerTracking(t *testing.T) {
	for _, precompute := range []bool{false, true} {
		pkg, files := buildCallerFrameSSAPackage(t, "example.com/p", `package p
//go:noinline
func Pinned() {}
func Plain() {}
`)
		prog := ssatest.NewProgram(t, nil)
		if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
			t.Fatal(err)
		}
		pkg.Func("Pinned").Syntax().(*ast.FuncDecl).Doc = nil
		pkg.Func("Plain").Syntax().(*ast.FuncDecl).Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}
		tracking := NewCallerTrackingWithSyntax(prog)
		if precompute {
			tracking.Precompute([]*gossa.Package{pkg})
		}
		frames := runtimeCallerFuncSet(tracking, pkg)
		if !frames[pkg.Func("Pinned")] || frames[pkg.Func("Plain")] {
			t.Fatalf("precompute=%v: caller tracking ignored prepared noinline snapshot", precompute)
		}
	}
}

func TestPreparedNoInlineKeepsSourceIdentity(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	const source = `package p
//go:noinline
func F() {}
func G() {}
`
	first, files := buildCallerFrameSSAPackage(t, "example.com/p", source)
	second, secondFiles := buildCallerFrameSSAPackage(t, "example.com/p", source)
	if err := ParsePkgSyntax(prog, first.Prog.Fset, first.Pkg, files); err != nil {
		t.Fatal(err)
	}
	assertMissingFunctionDirectives(t, func() { prog.FunctionDirectives(second.Func("F")) })
	second.Func("F").Syntax().(*ast.FuncDecl).Doc = nil
	if err := ParsePkgSyntax(prog, second.Prog.Fset, second.Pkg, secondFiles); err != nil {
		t.Fatal(err)
	}
	if !prog.FunctionDirectives(first.Func("F")).NoInline || prog.FunctionDirectives(second.Func("F")).NoInline {
		t.Fatal("same-path packages shared noinline records")
	}
	// A shared linker target cannot transfer a declaration attribute.
	prog.SetLinkname("example.com/p.F", "shared")
	prog.SetLinkname("example.com/p.G", "shared")
	if prog.FunctionDirectives(first.Func("G")).NoInline {
		t.Fatal("linkname alias inherited noinline")
	}
	if prog.FunctionDirectives(nil).NoInline {
		t.Fatal("nil function has noinline")
	}
}

func TestPreparedNoInlinePreservesOwnDeclarationRules(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/p", `package p
//go:noinline
func Generic[T any](x T) T { return x }
type T int
//go:noinline
func (x T) M() int { return int(x) }
//go:noinline
func Outer() func() int { return func() int { return 1 } }
func Use() (int, func() int, func(T) int) { var x T; return Generic(1), x.M, T.M }
//go:noinline
func init() {}
func init() {}
`)
	prog := ssatest.NewProgram(t, nil)
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	// Compare source ownership on every reachable SSA function, including generic
	// instances, bound-method wrappers, method expressions and closures.
	functions := make(map[*gossa.Function]bool)
	var visit func(*gossa.Function)
	visit = func(fn *gossa.Function) {
		if fn == nil || functions[fn] {
			return
		}
		functions[fn] = true
		for _, child := range fn.AnonFuncs {
			visit(child)
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				for _, operand := range instr.Operands(nil) {
					if operand != nil {
						if target, ok := (*operand).(*gossa.Function); ok {
							visit(target)
						}
					}
				}
			}
		}
	}
	for _, member := range pkg.Members {
		if fn, ok := member.(*gossa.Function); ok {
			visit(fn)
		}
	}
	expected := make(map[*gossa.Function]bool)
	for fn := range functions {
		expected[fn] = directive.ReadFunctionPragmas(fn).NoInline
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				fn.Doc = nil
			}
		}
	}
	for fn, want := range expected {
		if got := prog.FunctionDirectives(fn).NoInline; got != want {
			t.Errorf("%s (synthetic=%q): prepared noinline=%v, want %v", fn, fn.Synthetic, got, want)
		}
	}
}

func TestFailedPreparationDoesNotPublishNoInline(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/p", `package p
//go:noinline
func F() {}
//llgo:cdecl
//llgo:stdcall
type T int
`)
	prog := ssatest.NewProgram(t, nil)
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err == nil {
		t.Fatal("expected conflicting calling conventions")
	}
	assertMissingFunctionDirectives(t, func() { prog.FunctionDirectives(pkg.Func("F")) })
	for _, decl := range files[0].Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			decl.Doc = nil
		case *ast.GenDecl:
			decl.Doc = nil
		}
	}
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	if prog.FunctionDirectives(pkg.Func("F")).NoInline {
		t.Fatal("retry inherited noinline from failed preparation")
	}
}

func assertMissingFunctionDirectives(t *testing.T, query func()) {
	t.Helper()
	defer func() {
		got := recover()
		if message, ok := got.(string); !ok || !strings.Contains(message, "function directives were not prepared") {
			t.Fatalf("missing preparation panic = %v", got)
		}
	}()
	query()
}
