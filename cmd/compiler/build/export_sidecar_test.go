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
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestExportSidecarMetaAndStdlibTypesRoundTrip(t *testing.T) {
	fset := token.NewFileSet()
	src := `package stdsim

type Box[T any] struct {
	Value T
}

func (b *Box[T]) Get() T { return b.Value }

func Keep(x int) int { return x + 1 }

func Unused() int { return 42 }
`
	f, err := parser.ParseFile(fset, "stdsim.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
	conf := types.Config{Importer: packages.DefaultImporter()}
	goPkg, err := conf.Check("stdsim", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(goPkg, []*ast.File{f}, info, true)
	prog.Build()

	pkg := &aPackage{
		Package: &packages.Package{
			ID:      "stdsim",
			PkgPath: "stdsim",
			Name:    "stdsim",
			Module:  nil,
			Types:   goPkg,
		},
		SSA: ssaPkg,
	}
	ctx := &context{
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
		progSSA:   prog,
		pkgByID:   map[string]Package{"stdsim": pkg},
	}

	dir := t.TempDir()
	metaPath := filepath.Join(dir, "stdsim"+cacheExportExt)
	if err := ctx.writeExportMetaSidecar(metaPath, pkg); err != nil {
		t.Fatalf("writeExportMetaSidecar: %v", err)
	}
	loadedMeta, err := ctx.readExportMetaSidecar(metaPath)
	if err != nil {
		t.Fatalf("readExportMetaSidecar: %v", err)
	}
	if loadedMeta.PkgPath != "stdsim" {
		t.Fatalf("loaded meta path = %q, want stdsim", loadedMeta.PkgPath)
	}
	dc := pkgmeta.AnalyzeDeadcode([]*pkgmeta.PackageMeta{loadedMeta}, []string{"stdsim.Keep"})
	if !slices.Contains(dc.ReachableFuncs, "stdsim.Keep") || !slices.Contains(dc.DeadFuncs, "stdsim.Unused") {
		t.Fatalf("unexpected deadcode result from cached sidecar: reachable=%v dead=%v", dc.ReachableFuncs, dc.DeadFuncs)
	}

	typesPath := filepath.Join(dir, "stdsim"+cacheTypesExt)
	if err := ctx.writePackageTypesSidecar(typesPath, pkg); err != nil {
		t.Fatalf("writePackageTypesSidecar: %v", err)
	}
	decodedPkg, err := ctx.readPackageTypesSidecar(typesPath, "stdsim", make(map[string]*types.Package))
	if err != nil {
		t.Fatalf("readPackageTypesSidecar: %v", err)
	}
	if decodedPkg.Path() != "stdsim" || decodedPkg.Scope().Lookup("Box") == nil || decodedPkg.Scope().Lookup("Keep") == nil {
		t.Fatalf("decoded stdlib-like package missing expected symbols: %v", decodedPkg)
	}
}

func TestM1IncrementalCacheSkipsUnchangedDeps(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("GOWORK", "off")
	t.Setenv(llgoBuildCache, "1")
	t.Setenv(llgoEscapeCache, "1")
	cache := t.TempDir()
	old := cacheRootFunc
	cacheRootFunc = func() string { return cache }
	t.Cleanup(func() { cacheRootFunc = old })

	modDir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/m1inc\n\ngo 1.24\n",
		"dep/dep.go": `package dep

import (
	"strconv"
	"strings"
)

func Format(tag string, n int) string {
	var b strings.Builder
	b.WriteString(strings.ToUpper(tag))
	b.WriteByte('-')
	b.WriteString(strconv.Itoa(n))
	return b.String()
}
`,
		"main.go": `package main

import "example.com/m1inc/dep"

func main() {
	println("v1:" + dep.Format("dep", 42))
}
`,
	} {
		full := filepath.Join(modDir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	buildAndRun := func(wantOut string) []buildTraceEvent {
		t.Helper()
		cfg := NewDefaultConf(ModeBuild)
		cfg.OptLevel = optlevel.O0
		cfg.BuildParallelism = 4
		cfg.OutFile = filepath.Join(t.TempDir(), "app")
		if runtime.GOOS == "windows" {
			cfg.OutFile += ".exe"
		}
		cfg.BuildTrace = filepath.Join(t.TempDir(), "trace.json")
		if _, err := Build(Invocation{Args: []string{"."}, Dir: modDir, Config: cfg}); err != nil {
			t.Fatalf("build (%s): %v", wantOut, err)
		}
		out, err := exec.Command(cfg.OutFile).CombinedOutput()
		if err != nil || !strings.Contains(string(out), wantOut) {
			t.Fatalf("run (%s): err=%v out=%q", wantOut, err, string(out))
		}
		data, err := os.ReadFile(cfg.BuildTrace)
		if err != nil {
			t.Fatalf("read trace: %v", err)
		}
		var events []buildTraceEvent
		if err := json.Unmarshal(data, &events); err != nil {
			t.Fatalf("parse trace: %v", err)
		}
		return events
	}

	// Build 1: cold cache populates .a, .linksnap, .llgotypes, and frontend certs.
	_ = buildAndRun("v1:DEP-42")

	// Verify .llgotypes sidecars were actually emitted into the build cache.
	llgotypesCount := 0
	_ = filepath.WalkDir(cache, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, cacheTypesExt) {
			llgotypesCount++
		}
		return nil
	})
	if llgotypesCount == 0 {
		t.Fatal("expected cold build to write .llgotypes sidecar files into cache")
	}

	// Edit only main.go (leaving dep and stdlib completely untouched).
	mainV2 := `package main

import "example.com/m1inc/dep"

func main() {
	println("v2:" + dep.Format("dep", 42))
}
`
	if err := os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainV2), 0o600); err != nil {
		t.Fatal(err)
	}

	// Build 2: hot incremental rebuild must skip parse, typecheck, and SSA for unchanged deps.
	events := buildAndRun("v2:DEP-42")

	var skippedParse, typeExports, typesExportData, skippedSSA int
	for _, e := range events {
		if e.Name == "certify frontend exports" {
			if n, ok := e.Args["skipped_parse"].(float64); ok {
				skippedParse = int(n)
			}
			if n, ok := e.Args["type_exports"].(float64); ok {
				typeExports = int(n)
			}
			if n, ok := e.Args["types_exportdata"].(float64); ok {
				typesExportData = int(n)
			}
			if n, ok := e.Args["skipped_ssa"].(float64); ok {
				skippedSSA = int(n)
			}
		}
		if e.Args["stage"] == "ssa" {
			if id, ok := e.Args["package_id"].(string); ok && id == "example.com/m1inc/dep" {
				t.Fatalf("incremental build unexpectedly reconstructed SSA for unchanged dep %s", id)
			}
		}
	}
	if skippedParse == 0 || typeExports == 0 || typesExportData == 0 || skippedSSA == 0 {
		t.Fatalf("expected M1 cache hits after editing main.go, got skipped_parse=%d type_exports=%d types_exportdata=%d skipped_ssa=%d (llgotypes_files=%d)",
			skippedParse, typeExports, typesExportData, skippedSSA, llgotypesCount)
	}
}
