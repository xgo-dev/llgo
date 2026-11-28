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
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/callgraph/rta"
	llpkgs "github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func buildNativeTestSSAPackages(t *testing.T, sources map[string]string, spec target.Spec) (*ssa.Program, map[string]*ssa.Package) {
	t.Helper()
	fset := token.NewFileSet()
	tctx := types.NewContextWithConfig(types.ContextConfig{Spec: spec})
	pkgs := make(map[string]*types.Package)
	filesByPkg := make(map[string][]*ast.File)

	order := []string{"example.com/lib", "example.com/app"}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics|ssa.GlobalDebug|ssa.SanityCheckFunctions)
	prog.Target = spec
	ssaPkgs := make(map[string]*ssa.Package)
	stdImp := llpkgs.DefaultImporter()

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
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
			Instances:  make(map[*ast.Ident]types.Instance),
		}
		conf := types.Config{
			Context: tctx,
			Sizes:   tctx,
			Importer: nativeImporterFunc(func(impPath string) (*types.Package, error) {
				if p, ok := pkgs[impPath]; ok {
					return p, nil
				}
				return stdImp.Import(impPath)
			}),
		}
		p, err := conf.Check(path, fset, filesByPkg[path], info)
		if err != nil {
			t.Fatalf("typecheck %s: %v", path, err)
		}
		pkgs[path] = p
		ssaPkgs[path] = prog.CreatePackage(p, filesByPkg[path], info, true)
	}
	prog.Build()
	return prog, ssaPkgs
}

type nativeImporterFunc func(path string) (*types.Package, error)

func (f nativeImporterFunc) Import(path string) (*types.Package, error) { return f(path) }

func TestNativeSSARTACallgraphAndExportDataPipeline(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	llProg, llPkgs := buildNativeTestSSAPackages(t, map[string]string{
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
	}, spec)

	libLL := llPkgs["example.com/lib"]
	appLL := llPkgs["example.com/app"]
	if libLL == nil || appLL == nil {
		t.Fatalf("missing native SSA packages: %v", llPkgs)
	}

	// 1. Run pure-Go llgo/callgraph/rta directly on native llgo/ssa.Function roots.
	mainFn := appLL.Func("main")
	if mainFn == nil {
		t.Fatal("missing app.main")
	}
	rtaRes := rta.Analyze([]*ssa.Function{mainFn}, true)
	if rtaRes.CallGraph == nil {
		t.Fatal("expected CallGraph from rta.Analyze")
	}
	addFn := libLL.Func("Add")
	deadLibFn := libLL.Func("DeadHelper")
	deadAppFn := appLL.Func("unusedAppFunc")
	if _, ok := rtaRes.Reachable[mainFn]; !ok {
		t.Fatal("expected main reachable in RTA")
	}
	if _, ok := rtaRes.Reachable[addFn]; !ok {
		t.Errorf("expected lib.Add reachable via closure in RTA")
	}
	if _, ok := rtaRes.Reachable[deadLibFn]; ok {
		t.Errorf("expected lib.DeadHelper unreachable in RTA")
	}
	if _, ok := rtaRes.Reachable[deadAppFn]; ok {
		t.Errorf("expected app.unusedAppFunc unreachable in RTA")
	}
	var foundSpeak bool
	for fn := range rtaRes.Reachable {
		if fn.Name() == "Speak" {
			foundSpeak = true
			break
		}
	}
	if !foundSpeak {
		t.Errorf("expected (*Dog).Speak reachable via RTA interface dispatch")
	}
	path := callgraph.PathSearch(rtaRes.CallGraph.Root, func(n *callgraph.Node) bool {
		return n.Func == addFn
	})
	if len(path) == 0 {
		t.Errorf("expected callgraph.PathSearch path from Root to lib.Add")
	}
	if len(llProg.RuntimeTypes()) == 0 {
		t.Errorf("expected non-empty RuntimeTypes on llgo/ssa.Program")
	}

	// 2. Verify compiler-internal pkgmeta binary summary round-trip & deadcode.
	libMeta := pkgmeta.BuildPackageMeta(libLL, nil, nil, pkgmeta.DefaultBudget)
	rawMeta, err := pkgmeta.Marshal(libMeta)
	if err != nil {
		t.Fatalf("pkgmeta.Marshal: %v", err)
	}
	decodedLibMeta, err := pkgmeta.Unmarshal(rawMeta)
	if err != nil {
		t.Fatalf("pkgmeta.Unmarshal: %v", err)
	}
	var foundLeak, foundAdd bool
	for _, fe := range decodedLibMeta.Escape.Funcs {
		if fe.Name == "example.com/lib.LeakPtr" {
			foundLeak = true
			if len(fe.Params) != 1 || fe.Params[0]&pkgmeta.ParamToReturn == 0 {
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
		t.Fatalf("missing lib summaries: foundLeak=%v foundAdd=%v", foundLeak, foundAdd)
	}

	appMeta := pkgmeta.BuildPackageMeta(appLL, nil, map[string]*pkgmeta.EscapeSummary{"example.com/lib": &decodedLibMeta.Escape}, pkgmeta.DefaultBudget)
	dc := pkgmeta.AnalyzeDeadcode([]*pkgmeta.PackageMeta{decodedLibMeta, appMeta}, []string{"example.com/app.main"})
	if !slices.Contains(dc.ReachableFuncs, "example.com/lib.Add") || !slices.Contains(dc.DeadFuncs, "example.com/lib.DeadHelper") {
		t.Fatalf("unexpected pkgmeta deadcode: reachable=%v dead=%v", dc.ReachableFuncs, dc.DeadFuncs)
	}

	// 3. Verify 1:1 gcexportdata-aligned exportdata.Write / Read / NewImporter directly on native types.Package.
	var buf bytes.Buffer
	if err := exportdata.Write(&buf, llProg.Fset, libLL.Pkg); err != nil {
		t.Fatalf("exportdata.Write: %v", err)
	}
	imports := make(map[string]*types.Package)
	decodedLib, err := exportdata.Read(bytes.NewReader(buf.Bytes()), llProg.Fset, imports, "example.com/lib")
	if err != nil {
		t.Fatalf("exportdata.Read: %v", err)
	}
	if decodedLib.Path() != "example.com/lib" || decodedLib.Scope().Lookup("Dog") == nil || decodedLib.Scope().Lookup("Add") == nil {
		t.Fatalf("decoded lib missing expected symbols: %v", decodedLib)
	}
}
