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
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/cmd/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
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

	order := []string{"example.com/lib", "example.com/app"}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics|ssa.GlobalDebug)
	ssaPkgs := make(map[string]*ssa.Package)

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
		info := &gotypes.Info{
			Types:      make(map[ast.Expr]gotypes.TypeAndValue),
			Defs:       make(map[*ast.Ident]gotypes.Object),
			Uses:       make(map[*ast.Ident]gotypes.Object),
			Implicits:  make(map[ast.Node]gotypes.Object),
			Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
			Scopes:     make(map[ast.Node]*gotypes.Scope),
			Instances:  make(map[*ast.Ident]gotypes.Instance),
		}
		conf := gotypes.Config{
			Importer: importerFunc(func(impPath string) (*gotypes.Package, error) {
				if p, ok := goPkgs[impPath]; ok {
					return p, nil
				}
				return importer.Default().Import(impPath)
			}),
		}
		gp, err := conf.Check(path, fset, filesByPkg[path], info)
		if err != nil {
			t.Fatalf("typecheck %s: %v", path, err)
		}
		goPkgs[path] = gp
		ssaPkgs[path] = prog.CreatePackage(gp, filesByPkg[path], info, true)
	}
	prog.Build()
	return prog, ssaPkgs
}

type importerFunc func(path string) (*gotypes.Package, error)

func (f importerFunc) Import(path string) (*gotypes.Package, error) { return f(path) }

func TestLowerProgramRTACallgraphAndExportDataPipeline(t *testing.T) {
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

	// 1. Lower both packages into a unified llgo/ssa.Program (1:1 x/tools/go/ssa parity).
	llProg, llPkgs, err := cl.LowerProgram(prog, []*ssa.Package{ssaPkgs["example.com/lib"], ssaPkgs["example.com/app"]}, spec, tctx)
	if err != nil {
		t.Fatalf("LowerProgram: %v", err)
	}
	libLL := llPkgs["example.com/lib"]
	appLL := llPkgs["example.com/app"]
	if libLL == nil || appLL == nil {
		t.Fatalf("missing lowered packages: %v", llPkgs)
	}

	// 2. Run 1:1 x/tools/go/callgraph/rta directly on llgo/ssa.Function roots!
	mainFn := appLL.Func("main")
	if mainFn == nil {
		t.Fatal("missing app.main")
	}
	rtaRes := rta.Analyze([]*llssa.Function{mainFn}, true)
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
	// Verify *Dog.Speak() was discovered via MakeInterface + Invoke!
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
	// Verify callgraph.PathSearch finds a path from Root -> lib.Add.
	path := callgraph.PathSearch(rtaRes.CallGraph.Root, func(n *callgraph.Node) bool {
		return n.Func == addFn
	})
	if len(path) == 0 {
		t.Errorf("expected callgraph.PathSearch path from Root to lib.Add")
	}
	if len(llProg.RuntimeTypes()) == 0 {
		t.Errorf("expected non-empty RuntimeTypes on llgo/ssa.Program")
	}

	// 3. Verify compiler-internal pkgmeta binary summary round-trip & deadcode.
	libMeta := cl.BuildPackageMeta(libLL, nil, pkgmeta.DefaultBudget)
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

	appMeta := cl.BuildPackageMeta(appLL, map[string]*pkgmeta.EscapeSummary{"example.com/lib": &decodedLibMeta.Escape}, pkgmeta.DefaultBudget)
	dc := pkgmeta.AnalyzeDeadcode([]*pkgmeta.PackageMeta{decodedLibMeta, appMeta}, []string{"example.com/app.main"})
	if !slices.Contains(dc.ReachableFuncs, "example.com/lib.Add") || !slices.Contains(dc.DeadFuncs, "example.com/lib.DeadHelper") {
		t.Fatalf("unexpected pkgmeta deadcode: reachable=%v dead=%v", dc.ReachableFuncs, dc.DeadFuncs)
	}

	// 4. Verify 1:1 gcexportdata-aligned exportdata.Write / Read / NewImporter!
	var buf bytes.Buffer
	if err := exportdata.Write(&buf, prog.Fset, libLL.Pkg); err != nil {
		t.Fatalf("exportdata.Write: %v", err)
	}
	imports := make(map[string]*lltypes.Package)
	decodedPkg, err := exportdata.Read(bytes.NewReader(buf.Bytes()), prog.Fset, imports, "example.com/lib")
	if err != nil {
		t.Fatalf("exportdata.Read: %v", err)
	}
	if decodedPkg.Path() != "example.com/lib" || decodedPkg.Scope().Lookup("Dog") == nil {
		t.Fatalf("decoded package missing expected symbols: %v", decodedPkg)
	}
	imp := exportdata.NewImporter(prog.Fset, imports)
	if got, err := imp.Import("example.com/lib"); err != nil || got != decodedPkg {
		t.Fatalf("exportdata.NewImporter.Import = (%v, %v)", got, err)
	}
}
