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
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
func TestCanUseIsolatedBackend(t *testing.T) {
	ctx := &context{
		mode:      ModeBuild,
		buildConf: &Config{BuildMode: BuildModeExe},
	}
	if !ctx.canUseIsolatedBackend() {
		t.Fatal("normal executable build should use isolated backends")
	}
	ctx.mode = ModeGen
	if ctx.canUseIsolatedBackend() {
		t.Fatal("generation mode should remain on the coordinator")
	}
	ctx.mode = ModeTest
	if !ctx.canUseIsolatedBackend() {
		t.Fatal("test mode should use isolated executable backends")
	}
	ctx.buildConf.BuildMode = BuildModeCShared
	if ctx.canUseIsolatedBackend() {
		t.Fatal("c-shared mode should remain on the coordinator")
	}
	ctx.buildConf.BuildMode = BuildModeExe
	ctx.buildConf.ModuleHook = func(*aPackage) {}
	if ctx.canUseIsolatedBackend() {
		t.Fatal("module hooks should remain on the coordinator")
	}
}

func TestPartitionPackageExecutions(t *testing.T) {
	patchedPkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/patched",
		PkgPath: "example.com/patched",
	}}
	normalPkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/normal",
		PkgPath: "example.com/normal",
	}}
	ctx := &context{
		mode:        ModeBuild,
		buildConf:   &Config{BuildMode: BuildModeExe},
		patches:     cl.Patches{"example.com/patched": {}},
		sfilesCache: make(map[string][]string),
	}
	patched, coordinator, isolated, err := partitionPackageExecutions(ctx, []*packageBuildTask{
		{pkg: patchedPkg},
		{pkg: normalPkg},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patched) != 1 || patched[0] != 0 {
		t.Fatalf("patched = %v, want [0]", patched)
	}
	if len(coordinator) != 0 {
		t.Fatalf("coordinator = %v, want empty", coordinator)
	}
	if len(isolated) != 1 || isolated[0] != 1 {
		t.Fatalf("isolated = %v, want [1]", isolated)
	}
}

func TestPkgSFilesRejectsUnpreparedBackendRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "asm.s"), []byte("TEXT ·f(SB),$0-0\n\tRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{
		sfilesCache:  make(map[string][]string),
		sfilesFrozen: true,
	}
	_, err := pkgSFiles(ctx, &packages.Package{
		ID:      "example.com/asm",
		PkgPath: "example.com/asm",
		Dir:     dir,
	})
	if err == nil {
		t.Fatal("expected frozen SFiles cache to reject an unprepared package")
	}
}

func TestRunBoundedPackageJobs(t *testing.T) {
	started := make(chan int, 4)
	release := make(chan struct{})
	done := make(chan error, 1)
	var active atomic.Int32
	var maximum atomic.Int32
	go func() {
		done <- runBoundedPackageJobs(2, []int{0, 1, 2, 3}, func(index int) error {
			current := active.Add(1)
			for {
				old := maximum.Load()
				if current <= old || maximum.CompareAndSwap(old, current) {
					break
				}
			}
			started <- index
			<-release
			active.Add(-1)
			return nil
		})
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatal("two package workers did not start concurrently")
		}
	}
	if got := maximum.Load(); got != 2 {
		close(release)
		t.Fatalf("maximum concurrent jobs = %d, want 2", got)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrent jobs = %d after completion, want 2", got)
	}
}

func TestRunBoundedPackageJobsReturnsFirstOrderedError(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	err := runBoundedPackageJobs(3, []int{3, 1, 2}, func(index int) error {
		switch index {
		case 3:
			return first
		case 1:
			return second
		default:
			return nil
		}
	})
	if !errors.Is(err, first) {
		t.Fatalf("error = %v, want first submitted error", err)
	}
}

func TestRunBoundedPackageJobsConvertsPanicToError(t *testing.T) {
	boom := errors.New("boom")
	err := runBoundedPackageJobs(2, []int{0, 1}, func(index int) error {
		if index == 0 {
			panic(boom)
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want recovered panic %v", err, boom)
	}
}
func TestPackageBuildStageEmptyAndSkippedInputs(t *testing.T) {
	runtimePkg := &aPackage{Package: &packages.Package{PkgPath: env.LLGoRuntimePkg}}
	normalPkg := &aPackage{Package: &packages.Package{PkgPath: "example.com/normal"}}
	specs := []packageBuildSpec{
		{pkg: runtimePkg, runtime: true},
		{pkg: normalPkg},
	}
	if got := packageBuildSpecsForRuntime(specs, true); len(got) != 1 || got[0].pkg != runtimePkg {
		t.Fatalf("runtime specs = %+v, want runtime package", got)
	}
	if got := packageBuildSpecsForRuntime(specs, false); len(got) != 1 || got[0].pkg != normalPkg {
		t.Fatalf("non-runtime specs = %+v, want normal package", got)
	}

	ctx := &context{buildConf: &Config{}}
	preflights, err := preflightPackageBuilds(ctx, nil, false)
	if err != nil || len(preflights) != 0 {
		t.Fatalf("empty preflights = %#v, %v", preflights, err)
	}
	results, err := buildPreflightedPackageGroup(ctx, nil, preflights, false)
	if err != nil || results != nil {
		t.Fatalf("empty package group = %#v, %v", results, err)
	}
	if err := executePreflightedPackage(ctx, packagePreflight{skip: true}, false); err != nil {
		t.Fatal(err)
	}
	if err := executeIsolatedPackages(ctx, nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if err := runBoundedPackageJobs(0, nil, func(int) error {
		t.Fatal("empty package jobs invoked callback")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := runPackageJob(7, func(int) error { panic("boom") }); err == nil || !strings.Contains(err.Error(), "package job 7 panicked: boom") {
		t.Fatalf("non-error panic = %v", err)
	}
}

func TestNewBackendTaskUsesPackageLocalState(t *testing.T) {
	coordinator := &context{
		conf:            &packages.Config{},
		mode:            ModeBuild,
		buildConf:       &Config{BuildMode: BuildModeExe},
		commands:        commandEnv{dir: t.TempDir()},
		frontendOptions: cl.Options{Debug: true},
		sfilesCache:     map[string][]string{"example.com/p": {"asm.s"}},
		plan9asmReady:   true,
		plan9asmMode:    plan9asmEnvSelected,
		plan9asmPkgs:    map[string]bool{"example.com/p": true},
	}

	task := coordinator.newBackendTask(backendSession{})
	if task == coordinator {
		t.Fatal("backend task aliases coordinator")
	}
	if task.buildConf != coordinator.buildConf || task.conf != coordinator.conf {
		t.Fatal("backend task did not retain immutable build inputs")
	}
	if !task.sfilesFrozen || !task.plan9asmReady || task.plan9asmMode != plan9asmEnvSelected {
		t.Fatalf("backend task state = %+v", task)
	}
	if !task.frontendOptions.Debug || task.commands.dir != coordinator.commands.dir {
		t.Fatal("backend task lost invocation settings")
	}
	task.plan9asmSigs["example.com/p"] = map[string]struct{}{"f": {}}
	if coordinator.plan9asmSigs != nil {
		t.Fatal("backend task signature cache aliases coordinator")
	}
}
