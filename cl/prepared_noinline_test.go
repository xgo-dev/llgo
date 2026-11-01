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
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goembed"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
)

func TestPreparedNoInlineFeedsLowering(t *testing.T) {
	for _, entry := range []string{"one-shot", "prepared-one-shot", "backend", "patched-backend"} {
		t.Run(entry, func(t *testing.T) {
			pkg, files := buildCallerFrameSSAPackage(t, "example.com/prepared", `package p
//go:noinline
func Pinned(x int) int { return x + 1 }
func Plain(x int) int { return x + 2 }
type T int
//go:noinline
func (x T) Method() int { return int(x) }
//go:noinline
func Generic[T any](x T) T { return x }
func Call() int { return Generic(3) + T(4).Method() }
//go:noinline
func Closure() func() int { return func() int { return 5 } }
`)
			prog := ssatest.NewProgram(t, nil)
			var patches Patches
			if entry == "patched-backend" {
				// Prepared original and effective package identities differ, but both
				// refer to the same source declarations.
				alt, _ := buildCallerFrameSSAPackage(t, "example.com/prepared/alt", "package p")
				patches = Patches{pkg.Pkg.Path(): {Alt: alt, Types: types.NewPackage(pkg.Pkg.Path(), pkg.Pkg.Name())}}
			}
			if entry != "one-shot" {
				typs := pkg.Pkg
				if patch, ok := patches[pkg.Pkg.Path()]; ok {
					typs = patch.Types
				}
				if err := ParsePkgSyntax(prog, pkg.Prog.Fset, typs, files); err != nil {
					t.Fatal(err)
				}
				for _, file := range files {
					for _, decl := range file.Decls {
						if fn, ok := decl.(*ast.FuncDecl); ok {
							fn.Doc = nil
							if fn.Name.Name == "Plain" {
								fn.Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}
							}
						}
					}
					file.Comments = nil
				}
			}
			backend := prog
			preloaded := strings.Contains(entry, "backend")
			if preloaded {
				backend = prog.NewBackendProgram()
				defer backend.Dispose()
			}
			compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, nil, patches, nil, pkg, files, goembed.VarMap{}, false, Options{PreloadedSyntax: preloaded})
			if err != nil {
				t.Fatal(err)
			}
			// Inspect actual LLVM attributes for functions, methods, a generic
			// instantiation and a closure. Synthetic closure bodies must not inherit.
			found := map[string]bool{}
			for fn := compiled.Module().FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
				name := fn.Name()
				want, check := false, false
				for _, suffix := range []string{".Pinned", ".Plain", ".T.Method", ".Closure", ".Closure$1"} {
					if strings.HasSuffix(name, suffix) {
						check = true
						want = suffix != ".Plain" && suffix != ".Closure$1"
						found[suffix] = true
					}
				}
				if strings.Contains(name, ".Generic[") {
					check = true
					want = true
					found["generic"] = true
				}
				if !check {
					continue
				}
				ir := fn.String()
				noInline := !fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("noinline")).IsNil()
				tail := fn.GetStringAttributeAtIndex(-1, "disable-tail-calls")
				noTail := !tail.IsNil() && tail.GetStringValue() == "true"
				if noInline != want || noTail != want {
					t.Errorf("%s: want source noinline/tail-call attributes %v:\n%s", name, want, ir)
				}
			}
			for _, key := range []string{".Pinned", ".Plain", ".T.Method", ".Closure", ".Closure$1", "generic"} {
				if !found[key] {
					t.Errorf("missing test function %s in:\n%s", key, compiled.String())
				}
			}
		})
	}
}

func TestNoInlineCrossPackageGenericOneShot(t *testing.T) {
	for _, prepared := range []bool{false, true} {
		dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
//go:noinline
func Generic[T any](v T) T { return v }
func Plain[T any](v T) T { return v }
type Box[T any] struct { Value T }
//go:noinline
func (b Box[T]) Get() T { return b.Value }
`, "example.com/root", `package root
import "example.com/dep"
func Use() int { return dep.Generic(1) + dep.Plain(2) + (dep.Box[int]{Value: 3}).Get() }
`)
		files := []*ast.File{{Name: ast.NewIdent("root"), Decls: []ast.Decl{root.Func("Use").Syntax().(*ast.FuncDecl)}}}
		prog := ssatest.NewProgram(t, nil)
		if prepared {
			method := dep.Prog.FuncValue(dep.Type("Box").Type().(*types.Named).Method(0))
			decls := []ast.Decl{dep.Func("Generic").Syntax().(*ast.FuncDecl), dep.Func("Plain").Syntax().(*ast.FuncDecl), method.Syntax().(*ast.FuncDecl)}
			if err := ParsePkgSyntax(prog, dep.Prog.Fset, dep.Pkg, []*ast.File{{Name: ast.NewIdent("dep"), Decls: decls}}); err != nil {
				t.Fatal(err)
			}
			for _, decl := range decls {
				decl.(*ast.FuncDecl).Doc = nil
			}
			dep.Func("Plain").Syntax().(*ast.FuncDecl).Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}
		}
		compiled, err := NewPackage(prog, root, files)
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range map[string]bool{"example.com/dep.Generic[int]": true, "example.com/dep.Plain[int]": false, "example.com/dep.Box[int].Get": true} {
			fn := compiled.Module().NamedFunction(name)
			if fn.IsNil() || !fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("noinline")).IsNil() != want {
				t.Fatalf("prepared=%v: %s noinline mismatch:\n%s", prepared, name, compiled.String())
			}
		}
	}
}
