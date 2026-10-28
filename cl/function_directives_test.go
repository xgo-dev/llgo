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
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Analysis tests start from SSA fixtures rather than the package loader.
// Prepare their source snapshots before invoking the analyses under test.
func testFunctionProgram(t *testing.T, goProg *ssa.Program, bindings ...map[string]string) llssa.Program {
	t.Helper()
	prog := llssa.NewProgram(nil)
	t.Cleanup(prog.Dispose)
	if goProg == nil {
		return prog
	}
	declarations := make(map[*ast.FuncDecl]*directive.FunctionDirectives)
	names := make(map[*types.Package]map[string]*directive.FunctionDirectives)
	for fn := range ssautil.AllFunctions(goProg) {
		if origin := fn.Origin(); origin != nil {
			fn = origin
		}
		decl, _ := fn.Syntax().(*ast.FuncDecl)
		if decl == nil || declarations[decl] != nil {
			continue
		}
		var pkg *types.Package
		var fullName, inPkgName string
		if obj, ok := fn.Object().(*types.Func); ok {
			pkg = obj.Pkg()
			fullName, inPkgName = directive.FuncName(llssa.PathOf(pkg), obj)
		} else if fn.Pkg != nil {
			pkg = fn.Pkg.Pkg
			fullName, inPkgName = astFuncName(llssa.PathOf(pkg), decl, nil)
		} else {
			continue
		}
		info, err := directive.ParseFunction(goProg.Fset, decl.Doc, inPkgName, true, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, binding := range bindings {
			if link, ok := binding[fullName]; ok {
				info.Linkname = link
			}
		}
		declarations[decl] = &info
		if names[pkg] == nil {
			names[pkg] = make(map[string]*directive.FunctionDirectives)
		}
		names[pkg][fullName] = &info
	}
	for pkg, named := range names {
		prog.FunctionSources().Set(pkg, declarations, named)
	}
	return prog
}

func testCallerTracking(t *testing.T, goProg *ssa.Program) *CallerTracking {
	return NewCallerTracking(NewFunctionSet(testFunctionProgram(t, goProg), nil))
}

func testUintptrEscapesRoots(t *testing.T, fn *ssa.Function) map[ssa.Value]struct{} {
	t.Helper()
	functions := NewFunctionSet(testFunctionProgram(t, fn.Prog), nil)
	return uintptrEscapesRoots(functions.Function(fn), functions)
}

func TestFunctionWrappersUsePreparedSnapshots(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/snapshot", `package snapshot
//go:noinline
//go:nosplit
//go:uintptrescapes
//llgo:env
func marked[T any](p uintptr) {}
func plain() {}
func use() { marked[int](0); marked[string](0) }
`)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	// A consumer must not recover directives by scanning Syntax().Doc.
	for _, node := range files[0].Decls {
		if decl, ok := node.(*ast.FuncDecl); ok {
			decl.Doc = nil
		}
	}
	functions := NewFunctionSet(prog, nil)
	functions.Prepare(pkg.Prog)
	instances := 0
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if fn.Origin() == nil || fn.Origin().Name() != "marked" {
			continue
		}
		instances++
		wrapped := functions.Function(fn)
		if wrapped.SSA != fn || !wrapped.Directives.NoInline || !wrapped.Directives.NoSplit ||
			!wrapped.Directives.UintptrEscapes || !wrapped.Directives.ClosureEnv {
			t.Fatalf("instance %s lost its concrete body or source directives: %+v", fn, wrapped)
		}
		if functions.Function(fn) != wrapped {
			t.Fatal("wrapper identity is not stable")
		}
	}
	if instances != 2 {
		t.Fatalf("got %d generic instances, want 2", instances)
	}
	if got := functions.Function(pkg.Func("plain")).Directives; got != (directive.FunctionDirectives{}) {
		t.Fatalf("plain function acquired directives: %+v", got)
	}
	tracking := NewCallerTracking(functions)
	tracking.Precompute([]*ssa.Package{pkg})
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if fn.Origin() != nil && fn.Origin().Name() == "marked" && !runtimeCallerFuncSet(tracking, pkg)[fn] {
			t.Fatalf("caller analysis did not use prepared noinline for %s", fn)
		}
	}
}

func TestFunctionWrappersKeepPackageVariantsSeparate(t *testing.T) {
	marked, markedFiles := buildCallerFrameSSAPackage(t, "example.com/variant", `package variant
//go:noinline
//go:linkname F C.foreign
func F() {}
`)
	plain, plainFiles := buildCallerFrameSSAPackage(t, "example.com/variant", `package variant
func F() {}
`)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	// Collect the marked view last to make a global string-keyed lookup fail.
	for _, input := range []struct {
		pkg   *ssa.Package
		files []*ast.File
	}{{plain, plainFiles}, {marked, markedFiles}} {
		if err := ParsePkgSyntax(prog, input.pkg.Prog.Fset, input.pkg.Pkg, input.files); err != nil {
			t.Fatal(err)
		}
	}
	functions := NewFunctionSet(prog, nil)
	if got := functions.Function(plain.Func("F")).Directives; got.NoInline || got.Linkname != "" {
		t.Fatalf("plain package view inherited another view: %+v", got)
	}
	if got := functions.Function(marked.Func("F")).Directives; !got.NoInline || got.Linkname != "C.foreign" {
		t.Fatalf("marked package view lost its directives: %+v", got)
	}
}

func TestFunctionWrappersSeparatePatchBindingsFromBody(t *testing.T) {
	for _, replacement := range []string{"//go:linkname F C.replacement\n", ""} {
		t.Run(replacement, func(t *testing.T) {
			original, files := buildCallerFrameSSAPackage(t, "example.com/p", `package p
//go:noinline
//go:nosplit
//llgo:env
//go:linkname F C.original
func F() {}
`)
			alt, altFiles := buildCallerFrameSSAPackage(t, "example.com/p", "package p\n"+replacement+"func F() {}\n")
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			if err := ParsePkgSyntax(prog, original.Prog.Fset, original.Pkg, files); err != nil {
				t.Fatal(err)
			}
			if err := ParsePkgSyntax(prog, alt.Prog.Fset, alt.Pkg, altFiles); err != nil {
				t.Fatal(err)
			}
			functions := NewFunctionSet(prog, Patches{"example.com/p": {Alt: alt, Types: alt.Pkg}})
			got := functions.Function(original.Func("F"))
			wantLink := ""
			if replacement != "" {
				wantLink = "C.replacement"
			}
			if got.SSA != original.Func("F") || got.Directives.Linkname != wantLink ||
				!got.Directives.NoInline || !got.Directives.NoSplit || !got.Directives.ClosureEnv {
				t.Fatalf("patch changed source body attributes or selected the wrong binding: %+v", got)
			}
		})
	}
}

func TestFunctionWrappersInheritOnlyTransparentPointerSemantics(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/adapters", `package adapters
type Base struct{}
//go:noinline
//go:nosplit
//go:uintptrescapes
//llgo:env
//go:wasmimport host marked
//go:linkname Base.Marked C.marked
func (Base) Marked(p uintptr) {}
type Outer struct { Base }
func use(o Outer) { o.Marked(0); f := o.Marked; f(0); Outer.Marked(o, 0) }
func ordinary(o Outer) { o.Marked(0) }
`)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	functions := NewFunctionSet(prog, nil)
	adapters := 0
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if fn.Syntax() != nil || fn.Synthetic == "" || fn.Name() == "init" {
			continue
		}
		got := functions.Function(fn).Directives
		if !got.UintptrEscapes {
			continue
		}
		adapters++
		got.UintptrEscapes = false
		if got != (directive.FunctionDirectives{}) {
			t.Fatalf("adapter %s inherited body or binding directives: %+v", fn, got)
		}
	}
	if adapters < 3 {
		t.Fatalf("only %d adapters tested, want promoted, bound and thunk wrappers", adapters)
	}
	if functions.Function(pkg.Func("ordinary")).Directives.UintptrEscapes {
		t.Fatal("ordinary caller inherited callee pragma")
	}
}

func TestFunctionDirectivesRequirePreparation(t *testing.T) {
	pkg, _ := buildCallerFrameSSAPackage(t, "example.com/unprepared", "package unprepared; func F() {}")
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	defer func() {
		if recover() == nil {
			t.Fatal("unprepared source silently treated as an unannotated function")
		}
	}()
	NewFunctionSet(prog, nil).Function(pkg.Func("F"))
}

func TestFunctionWrappersConcurrentIdentity(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/concurrent", `package concurrent
//go:noinline
func F() {}
`)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	functions := NewFunctionSet(prog, nil)
	results := make(chan *directive.Function, 32)
	var wg sync.WaitGroup
	for range cap(results) {
		wg.Add(1)
		go func() { defer wg.Done(); results <- functions.Function(pkg.Func("F")) }()
	}
	wg.Wait()
	close(results)
	expected := functions.Function(pkg.Func("F"))
	for got := range results {
		if got != expected || !got.Directives.NoInline {
			t.Fatal("concurrent callers received different or incomplete wrappers")
		}
	}
}

func TestFunctionWrappersUsePreparedImportedDeclarations(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/imported", `package imported
//go:noinline
//go:uintptrescapes
//go:wasmimport host entry
//go:linkname F C.entry
func F(uintptr)
`)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	imported := ssa.NewProgram(pkg.Prog.Fset, ssa.SanityCheckFunctions).CreatePackage(pkg.Pkg, nil, nil, true)
	fn := imported.Func("F")
	if fn.Syntax() != nil {
		t.Fatal("fixture unexpectedly has source syntax")
	}
	got := NewFunctionSet(prog, nil).Function(fn)
	if got.SSA != fn || !got.Directives.NoInline || !got.Directives.UintptrEscapes ||
		got.Directives.WasmModule != "host" || got.Directives.WasmName != "entry" || got.Directives.Linkname != "C.entry" {
		t.Fatalf("imported declaration lost prepared directives: %+v", got)
	}
}

func TestPreparedFunctionDirectivesReachLowering(t *testing.T) {
	pkg, files := buildCallerFrameSSAPackage(t, "example.com/prepared", `package prepared
func collect()
//go:noinline
//go:nosplit
//go:uintptrescapes
func marked(p uintptr) { collect() }
func plain() {}
//export exported
func exported() {}
`)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	prog.EnableGCRoots(true)
	prog.EnableCooperativeSafepoints(true)
	if err := ParsePkgSyntax(prog, pkg.Prog.Fset, pkg.Pkg, files); err != nil {
		t.Fatal(err)
	}
	for _, node := range files[0].Decls {
		if decl, ok := node.(*ast.FuncDecl); ok {
			decl.Doc = nil
		}
	}
	files[0].Comments = nil
	// The SSA bodies retain their syntax for construction, but consumers must
	// not discover function attributes or exports by walking the input files.
	files[0].Decls = nil
	functions := NewFunctionSet(prog, nil)
	tracking := NewCallerTracking(functions)
	tracking.Precompute([]*ssa.Package{pkg})
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, tracking, nil, nil, pkg, files, nil, false, Options{PreloadedSyntax: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if got := compiled.ExportFuncs()["example.com/prepared.exported"]; got != "exported" {
		t.Fatalf("prepared export was lost after removing function declarations: %q", got)
	}
	marked := compiled.Module().NamedFunction("example.com/prepared.marked").String()
	if !strings.Contains(marked, "noinline") || !strings.Contains(marked, "llvm_gc_root_chain") || strings.Contains(marked, "CooperativeSafepoint") {
		t.Fatalf("lowering did not consume the prepared noinline/uintptrescapes/nosplit value:\n%s", marked)
	}
	plain := compiled.Module().NamedFunction("example.com/prepared.plain").String()
	if strings.Contains(plain, "noinline") || !strings.Contains(plain, "CooperativeSafepoint") {
		t.Fatalf("unannotated function inherited another function's directives:\n%s", plain)
	}
}
