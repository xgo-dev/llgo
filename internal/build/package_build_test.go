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
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/packages"
	"golang.org/x/tools/go/ssa"
)

func TestPackageBuildTaskAndResult(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		PkgPath: "example.com/p",
		GoFiles: []string{"p.go"},
		Types:   types.NewPackage("example.com/p", "p"),
	}, NeedRt: true, NeedPyInit: true}
	task := newPackageBuildTask(pkg)
	if task.isDeclOnly() || task.isLinkOnly() || !task.hasSource() || task.isRuntime() || !task.needsRuntimeSignals() {
		t.Fatalf("unexpected normal package task: %+v", task)
	}
	result := packageBuildResultFor(task)
	if !result.needRuntime || !result.needPyInit {
		t.Fatalf("unexpected package result: %+v", result)
	}
}

func TestPackageBuildTaskSpecialKinds(t *testing.T) {
	decl := newPackageBuildTask(&aPackage{Package: &packages.Package{
		PkgPath: "unsafe",
		Types:   types.Unsafe,
	}})
	if !decl.isDeclOnly() || decl.needsRuntimeSignals() {
		t.Fatalf("unexpected declaration-only task: %+v", decl)
	}
	runtime := newPackageBuildTask(&aPackage{Package: &packages.Package{
		PkgPath: env.LLGoRuntimePkg,
		Types:   types.NewPackage(env.LLGoRuntimePkg, "runtime"),
	}})
	if !runtime.isRuntime() {
		t.Fatalf("runtime package was not marked runtime: %+v", runtime)
	}
}

func TestConfigParallelism(t *testing.T) {
	if got := (&Config{BuildParallelism: 3}).parallelism(); got != 3 {
		t.Fatalf("parallelism = %d, want 3", got)
	}
	if got := (&Config{}).parallelism(); got < 1 {
		t.Fatalf("default parallelism = %d, want positive value", got)
	}
}

func TestPreparePackageModuleReturnsEmbedError(t *testing.T) {
	fset := token.NewFileSet()
	filename := filepath.Join(t.TempDir(), "p.go")
	file, err := parser.ParseFile(fset, filename, `package p

//go:embed missing.txt
var content string
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/p",
		PkgPath: "example.com/p",
		Syntax:  []*ast.File{file},
	}}
	ctx := &context{
		conf:      &packages.Config{Fset: fset},
		buildConf: &Config{},
	}
	externs, err := preparePackageModule(ctx, pkg, true)
	if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
		t.Fatalf("preparePackageModule embed error = %v", err)
	}
	if externs != nil {
		t.Fatalf("preparePackageModule externs = %v, want nil", externs)
	}
}

func TestBuildOnePackageReturnsFrontendError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	fset := token.NewFileSet()
	filename := filepath.Join(t.TempDir(), "p.go")
	file, err := parser.ParseFile(fset, filename, `package p

//go:embed missing.txt
var content string
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{
		Package: &packages.Package{
			ID:      "example.com/frontend-error",
			PkgPath: "example.com/frontend-error",
			GoFiles: []string{filename},
			Syntax:  []*ast.File{file},
			Types:   types.NewPackage("example.com/frontend-error", "p"),
		},
		Manifest:    "already fingerprinted",
		Fingerprint: "frontend-error",
	}
	ctx := &context{
		conf:      &packages.Config{Fset: fset},
		buildConf: &Config{},
		built:     make(map[string]none),
	}

	_, err = buildOnePackage(ctx, newPackageBuildTask(pkg), false)
	if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
		t.Fatalf("buildOnePackage frontend error = %v", err)
	}
}

func TestPrePackageBuildErrorsAndCacheHit(t *testing.T) {
	t.Run("fingerprint error", func(t *testing.T) {
		pkg := &aPackage{Package: &packages.Package{
			ID:      "example.com/missing-source",
			PkgPath: "example.com/missing-source",
			GoFiles: []string{filepath.Join(t.TempDir(), "missing.go")},
			Types:   types.NewPackage("example.com/missing-source", "missing"),
		}}
		ctx := &context{
			buildConf:   &Config{},
			built:       make(map[string]none),
			llvmVersion: "test",
		}
		task := newPackageBuildTask(pkg)
		err := prePackageBuild(ctx, task, false)
		if err == nil || !strings.Contains(err.Error(), "digest go files") {
			t.Fatalf("pre fingerprint error = %v", err)
		}
		if task.skip {
			t.Fatal("pre skipped package after fingerprint error")
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		t.Setenv(llgoBuildCache, "off")
		pkg := &aPackage{
			Package: &packages.Package{
				ID:      "example.com/cache-hit",
				PkgPath: "example.com/cache-hit",
				GoFiles: []string{"cached.go"},
				Types:   types.NewPackage("example.com/cache-hit", "cached"),
			},
			Manifest:    "already fingerprinted",
			Fingerprint: "cache-hit",
			CacheHit:    true,
		}
		ctx := &context{buildConf: &Config{}, built: make(map[string]none)}
		task := newPackageBuildTask(pkg)
		err := prePackageBuild(ctx, task, true)
		if err != nil {
			t.Fatal(err)
		}
		if task.skip || !pkg.CacheHit {
			t.Fatalf("pre cache hit = skip %v, cache hit %v", task.skip, pkg.CacheHit)
		}
	})
}

func TestBuildOnePackageSkipsAlreadyBuiltPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/already-built",
		PkgPath: "example.com/already-built",
		GoFiles: []string{"already.go"},
		Types:   types.NewPackage("example.com/already-built", "already"),
	}, NeedRt: true}
	ctx := &context{built: map[string]none{pkg.ID: {}}}

	result, err := buildOnePackage(ctx, newPackageBuildTask(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.needRuntime {
		t.Fatalf("build result = %+v, want runtime requirement preserved", result)
	}
}

func TestPrePackageBuildSkipsDeclarationOnlyPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:         "unsafe",
		PkgPath:    "unsafe",
		Types:      types.Unsafe,
		ExportFile: "stale.a",
	}}
	ctx := &context{built: make(map[string]none)}

	task := newPackageBuildTask(pkg)
	err := prePackageBuild(ctx, task, false)
	if err != nil {
		t.Fatal(err)
	}
	if !task.skip {
		t.Fatal("declaration-only package was not skipped")
	}
	if pkg.ExportFile != "" {
		t.Fatalf("ExportFile = %q, want empty", pkg.ExportFile)
	}
	if _, ok := ctx.built[pkg.ID]; !ok {
		t.Fatal("declaration-only package was not recorded as built")
	}
}

func TestPrePackageBuildSkipsExternalLinkOnlyPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:         "example.com/linkonly",
		PkgPath:    "example.com/linkonly",
		Name:       "linkonly",
		Types:      types.NewPackage("example.com/linkonly", "linkonly"),
		ExportFile: "stale.a",
	}}
	ctx := &context{buildConf: &Config{}, built: make(map[string]none)}
	task := &packageBuildTask{pkg: pkg, kind: cl.PkgLinkExtern, kindParam: "-lexample"}
	err := prePackageBuild(ctx, task, false)
	if err != nil {
		t.Fatal(err)
	}
	if !task.skip || pkg.ExportFile != "" {
		t.Fatalf("external link-only pre = skip %v, export %q", task.skip, pkg.ExportFile)
	}
	if len(pkg.LinkArgs) != 1 || pkg.LinkArgs[0] != "-lexample" {
		t.Fatalf("external link args = %q, want [-lexample]", pkg.LinkArgs)
	}
}

func TestFinalizePackageBuildReturnsCachedResult(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/cached",
		PkgPath: "example.com/cached",
		Types:   types.NewPackage("example.com/cached", "cached"),
	}, CacheHit: true, NeedPyInit: true}

	result, err := finalizePackageBuild(&context{}, newPackageBuildTask(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.needPyInit {
		t.Fatalf("build result = %+v, want Python initialization requirement preserved", result)
	}
}

func TestBuildSSAPkgsEmptyAndNilEntries(t *testing.T) {
	ctx := &context{buildConf: &Config{}}
	buildSSAPkgs(ctx, nil)
	buildSSAPkgs(ctx, []ssaBuildEntry{{}, {fixOrder: true}})

	prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
	pkg := prog.CreatePackage(types.NewPackage("example.com/ssa", "ssa"), nil, nil, true)
	buildSSAPkgs(ctx, []ssaBuildEntry{{pkg: pkg}, {pkg: pkg}})
}
