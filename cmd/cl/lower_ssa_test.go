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

package cl_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/analysis/callgraph"
	"github.com/xgo-dev/llgo/analysis/deadcode"
	"github.com/xgo-dev/llgo/analysis/escape"
	"github.com/xgo-dev/llgo/analysis/inline"
	"github.com/xgo-dev/llgo/cmd/cl"
	"github.com/xgo-dev/llgo/exportdata"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
	"golang.org/x/tools/go/ssa"
)

func buildTestSSAPackages(t *testing.T, sources map[string]string) (*ssa.Program, map[string]*ssa.Package) {
	t.Helper()
	fset := token.NewFileSet()
	goPkgs := make(map[string]*gotypes.Package)
	filesByPkg := make(map[string][]*ast.File)

	// Parse in deterministic order; compile dependency "example.com/lib" before "example.com/app".
	order := []string{"example.com/lib", "example.com/app"}
	for _, path := range order {
		src, ok := sources[path]
		if !ok {
			continue
		}
		f, err := parser.ParseFile(fset, path+".go", src, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		filesByPkg[path] = []*ast.File{f}
		conf := gotypes.Config{
			Importer: importerFunc(func(impPath string) (*gotypes.Package, error) {
				if p, ok := goPkgs[impPath]; ok {
					return p, nil
				}
				return importer.Default().Import(impPath)
			}),
		}
		info := &gotypes.Info{
			Types:      make(map[ast.Expr]gotypes.TypeAndValue),
			Defs:       make(map[*ast.Ident]gotypes.Object),
			Uses:       make(map[*ast.Ident]gotypes.Object),
			Implicits:  make(map[ast.Node]gotypes.Object),
			Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
			Scopes:     make(map[ast.Node]*gotypes.Scope),
			Instances:  make(map[*ast.Ident]gotypes.Instance),
		}
		gp, err := conf.Check(path, fset, []*ast.File{f}, info)
		if err != nil {
			t.Fatalf("typecheck %s: %v", path, err)
		}
		goPkgs[path] = gp
	}

	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics|ssa.GlobalDebug)
	ssaPkgs := make(map[string]*ssa.Package)
	for _, path := range order {
		gp := goPkgs[path]
		if gp == nil {
			continue
		}
		info := &gotypes.Info{
			Types:      make(map[ast.Expr]gotypes.TypeAndValue),
			Defs:       make(map[*ast.Ident]gotypes.Object),
			Uses:       make(map[*ast.Ident]gotypes.Object),
			Implicits:  make(map[ast.Node]gotypes.Object),
			Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
			Scopes:     make(map[ast.Node]*gotypes.Scope),
			Instances:  make(map[*ast.Ident]gotypes.Instance),
		}
		// Re-check into fresh info so prog.CreatePackage has complete TypesInfo.
		conf := gotypes.Config{
			Importer: importerFunc(func(impPath string) (*gotypes.Package, error) {
				if p, ok := goPkgs[impPath]; ok {
					return p, nil
				}
				return importer.Default().Import(impPath)
			}),
		}
		gp2, err := conf.Check(path, fset, filesByPkg[path], info)
		if err != nil {
			t.Fatalf("re-typecheck %s: %v", path, err)
		}
		goPkgs[path] = gp2
		sp := prog.CreatePackage(gp2, filesByPkg[path], info, true)
		ssaPkgs[path] = sp
	}
	prog.Build()
	return prog, ssaPkgs
}

type importerFunc func(path string) (*gotypes.Package, error)

func (f importerFunc) Import(path string) (*gotypes.Package, error) { return f(path) }

func TestLowerPackageAndPureGoAnalysisPipeline(t *testing.T) {
	prog, ssaPkgs := buildTestSSAPackages(t, map[string]string{
		"example.com/lib": `package lib

type Speaker interface {
	Speak() int
}

type Dog struct{ N int }

func (d *Dog) Speak() int { return d.N }

func Add(a, b int) int { return a + b }

func LeakPtr(x *int) *int { return x }

func KeepLocal() int {
	x := 41
	return x + 1
}

func DeadHelper() int { return 999 }

func GenericID[T any](v T) T { return v }
`,
		"example.com/app": `package main

import "example.com/lib"

var globalSink *int

func useClosure(base int) func(int) int {
	return func(delta int) int {
		return lib.Add(base, delta)
	}
}

func useDefer(n int) (res int) {
	defer func() { res += 1 }()
	if n > 0 {
		return n * 2
	}
	return 0
}

func unusedAppFunc() int { return 123 }

func main() {
	var s lib.Speaker = &lib.Dog{N: lib.GenericID(7)}
	_ = s.Speak()
	fn := useClosure(10)
	_ = fn(5)
	_ = useDefer(3)
	_ = lib.KeepLocal()
	v := 42
	globalSink = lib.LeakPtr(&v)
}
`,
	})

	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	tctx := lltypes.NewContextWithConfig(lltypes.ContextConfig{Spec: spec})

	// 1. Lower lib -> llgo/ssa -> Verify -> BuildPackageMeta -> MarshalMeta/UnmarshalMeta round-trip
	libLL, err := cl.LowerPackage(prog, ssaPkgs["example.com/lib"], spec, tctx)
	if err != nil {
		t.Fatalf("LowerPackage(lib): %v", err)
	}
	if err := llssa.VerifyPackage(libLL); err != nil {
		t.Fatalf("VerifyPackage(lib): %v", err)
	}
	libMeta := cl.BuildPackageMeta(libLL, nil, inline.DefaultBudget)
	rawMeta, err := exportdata.MarshalMeta(libMeta)
	if err != nil {
		t.Fatalf("MarshalMeta(lib): %v", err)
	}
	decodedLibMeta, err := exportdata.UnmarshalMeta(rawMeta)
	if err != nil {
		t.Fatalf("UnmarshalMeta(lib): %v", err)
	}

	// Verify escape summary for lib.LeakPtr (param 0 escapes to return) vs lib.KeepLocal (no heap alloc).
	var foundLeak, foundAdd bool
	for _, fe := range decodedLibMeta.Escape.Funcs {
		if fe.Name == "example.com/lib.LeakPtr" {
			foundLeak = true
			if len(fe.Params) != 1 || fe.Params[0]&exportdata.ParamToReturn == 0 {
				t.Errorf("lib.LeakPtr param flags = %v, want ParamToReturn", fe.Params)
			}
		}
	}
	for _, fi := range decodedLibMeta.Inline.Funcs {
		if fi.Name == "example.com/lib.Add" {
			foundAdd = true
			if !fi.Inlineable {
				t.Errorf("lib.Add expected inlineable, got reason=%q cost=%d", fi.Reason, fi.Cost)
			}
		}
	}
	if !foundLeak || !foundAdd {
		t.Fatalf("missing expected lib summaries: foundLeak=%v foundAdd=%v", foundLeak, foundAdd)
	}

	// 2. Lower app using imported lib escape summary from binary exportdata sidecar!
	appLL, err := cl.LowerPackage(prog, ssaPkgs["example.com/app"], spec, tctx)
	if err != nil {
		t.Fatalf("LowerPackage(app): %v", err)
	}
	appEsc := escape.AnalyzePackage(appLL, map[string]*exportdata.EscapeSummary{
		"example.com/lib": &decodedLibMeta.Escape,
	})
	if appEsc == nil || len(appEsc.Funcs) == 0 {
		t.Fatal("expected non-empty app escape summary")
	}

	// 3. Verify hybrid live (appLL) + cached binary sidecar (decodedLibMeta) callgraph & deadcode!
	dc := deadcode.Analyze(
		[]*llssa.Package{appLL},
		[]*exportdata.PackageMeta{decodedLibMeta},
		nil,
		[]string{"example.com/app.main"},
	)
	if !slices.Contains(dc.ReachableFuncs, "example.com/app.main") {
		t.Fatalf("expected app.main reachable, got %v", dc.ReachableFuncs)
	}
	if !slices.Contains(dc.ReachableFuncs, "example.com/lib.Add") {
		t.Errorf("expected lib.Add reachable through closure, got reachable=%v", dc.ReachableFuncs)
	}
	if !slices.Contains(dc.ReachableMethods, "Speak") {
		t.Errorf("expected interface method Speak live, got %v", dc.ReachableMethods)
	}
	if !slices.Contains(dc.DeadFuncs, "example.com/lib.DeadHelper") {
		t.Errorf("expected lib.DeadHelper dead, got dead=%v", dc.DeadFuncs)
	}
	if !slices.Contains(dc.DeadFuncs, "example.com/app.unusedAppFunc") {
		t.Errorf("expected app.unusedAppFunc dead, got dead=%v", dc.DeadFuncs)
	}

	// 4. Verify callgraph captures closure & interface edges directly.
	cg := callgraph.Analyze([]*llssa.Package{appLL}, []*exportdata.PackageMeta{decodedLibMeta}, []string{"example.com/app.main"})
	if !cg.InterfaceCalls["Speak"] {
		t.Errorf("expected callgraph InterfaceCalls[Speak]")
	}

	// 5. Verify binary llgo/types.Package exportdata round-trip via LowerGoPackageTypes.
	libTypesPkg := cl.LowerGoPackageTypes(ssaPkgs["example.com/lib"].Pkg, tctx)
	rawPkg, err := exportdata.MarshalPackage(libTypesPkg)
	if err != nil {
		t.Fatalf("MarshalPackage(lib): %v", err)
	}
	decodedPkg, err := exportdata.UnmarshalPackage(rawPkg, make(map[string]*lltypes.Package))
	if err != nil {
		t.Fatalf("UnmarshalPackage(lib): %v", err)
	}
	if decodedPkg.Path() != "example.com/lib" || decodedPkg.Scope().Lookup("Dog") == nil || decodedPkg.Scope().Lookup("Add") == nil {
		t.Fatalf("decoded lib package missing symbols: %v", decodedPkg)
	}
}
