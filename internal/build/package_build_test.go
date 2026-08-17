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
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestPackageBuildTask(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		PkgPath: "example.com/p",
		GoFiles: []string{"p.go"},
		Types:   types.NewPackage("example.com/p", "p"),
	}, NeedRt: true, NeedPyInit: true}
	task := newPackageBuildTask(pkg)
	if task.isDeclOnly() || task.isLinkOnly() || !task.hasSource() || task.isRuntime() || !task.needsRuntimeSignals() {
		t.Fatalf("unexpected normal package task: %+v", task)
	}
	needRuntime, needPyInit := packageRuntimeNeeds([]*packageBuildTask{task})
	if !needRuntime || !needPyInit {
		t.Fatalf("package runtime needs = %v, %v; want true, true", needRuntime, needPyInit)
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
		ctx := &context{buildConf: &Config{}}
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

func TestPrePackageBuildSkipsDeclarationOnlyPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:         "unsafe",
		PkgPath:    "unsafe",
		Types:      types.Unsafe,
		ExportFile: "stale.a",
	}}
	ctx := &context{}

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
}

func TestPrePackageBuildSkipsExternalLinkOnlyPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:         "example.com/linkonly",
		PkgPath:    "example.com/linkonly",
		Name:       "linkonly",
		Types:      types.NewPackage("example.com/linkonly", "linkonly"),
		ExportFile: "stale.a",
	}}
	ctx := &context{buildConf: &Config{}}
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

func TestFinalizePackageBuildPreservesCachedRuntimeNeeds(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/cached",
		PkgPath: "example.com/cached",
		Types:   types.NewPackage("example.com/cached", "cached"),
	}, CacheHit: true, NeedPyInit: true}

	if err := finalizePackageBuild(&context{}, newPackageBuildTask(pkg), false); err != nil {
		t.Fatal(err)
	}
	if !pkg.NeedPyInit {
		t.Fatal("cached package lost Python initialization requirement")
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

func TestPreparePackageBuildsKeepsPatchesIsolatedAndSerial(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	patched := &aPackage{Package: &packages.Package{
		ID:      "example.com/patched",
		PkgPath: "example.com/patched",
	}}
	normal := &aPackage{Package: &packages.Package{
		ID:      "example.com/normal",
		PkgPath: "example.com/normal",
	}}
	ctx := &context{
		mode:        ModeBuild,
		buildConf:   &Config{BuildMode: BuildModeExe},
		patches:     cl.Patches{patched.PkgPath: {}},
		sfilesCache: make(map[string][]string),
	}
	patchedTask := &packageBuildTask{pkg: patched}
	normalTask := &packageBuildTask{pkg: normal}
	if err := preparePackageBuilds(ctx, []*packageBuildTask{patchedTask, normalTask}, false); err != nil {
		t.Fatal(err)
	}
	if !patchedTask.isolated || patchedTask.parallel {
		t.Fatalf("patched execution = isolated %v, parallel %v; want true, false", patchedTask.isolated, patchedTask.parallel)
	}
	if !normalTask.isolated || !normalTask.parallel {
		t.Fatalf("normal execution = isolated %v, parallel %v; want true, true", normalTask.isolated, normalTask.parallel)
	}
}

func TestBuildPackageGroupReturnsPreparationError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/missing-source",
		PkgPath: "example.com/missing-source",
		GoFiles: []string{filepath.Join(t.TempDir(), "missing.go")},
		Types:   types.NewPackage("example.com/missing-source", "missing"),
	}}
	ctx := &context{buildConf: &Config{}, llvmVersion: "test"}
	err := buildPackageGroup(ctx, []*packageBuildTask{newPackageBuildTask(pkg)}, false)
	if err == nil || !strings.Contains(err.Error(), "digest go files") {
		t.Fatalf("build package group preparation error = %v", err)
	}
}

func TestPreparePackageBuildsReturnsCoordinatorReadError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	pkg := &aPackage{
		Package: &packages.Package{
			ID:      "example.com/unprepared-asm",
			PkgPath: "example.com/unprepared-asm",
			Types:   types.NewPackage("example.com/unprepared-asm", "unprepared"),
		},
		Manifest:    "prepared manifest",
		Fingerprint: "prepared fingerprint",
	}
	ctx := &context{
		mode:         ModeBuild,
		buildConf:    &Config{BuildMode: BuildModeExe},
		sfilesFrozen: true,
	}
	err := preparePackageBuilds(ctx, []*packageBuildTask{{pkg: pkg}}, false)
	if err == nil || !strings.Contains(err.Error(), "were not prepared") {
		t.Fatalf("prepare package coordinator read error = %v", err)
	}
}

func invalidEmbedPackage(t *testing.T) (*token.FileSet, *aPackage) {
	t.Helper()
	fset := token.NewFileSet()
	filename := filepath.Join(t.TempDir(), "p.go")
	file, err := parser.ParseFile(fset, filename, `package p

//go:embed missing.txt
var content string
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return fset, &aPackage{
		Package: &packages.Package{
			ID:      "example.com/invalid-embed",
			PkgPath: "example.com/invalid-embed",
			GoFiles: []string{filename},
			Syntax:  []*ast.File{file},
			Types:   types.NewPackage("example.com/invalid-embed", "p"),
		},
		Manifest:    "prepared manifest",
		Fingerprint: "prepared fingerprint",
	}
}

func TestBuildPackageGroupReturnsCoordinatorBuildError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	fset, pkg := invalidEmbedPackage(t)
	ctx := &context{
		conf:      &packages.Config{Fset: fset},
		mode:      ModeGen,
		buildConf: &Config{},
	}
	err := buildPackageGroup(ctx, []*packageBuildTask{newPackageBuildTask(pkg)}, false)
	if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
		t.Fatalf("coordinator build error = %v", err)
	}
}

func TestBuildPackageGroupReturnsParallelBuildError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	fset, pkg := invalidEmbedPackage(t)
	coordinator := llssa.NewProgram(&llssa.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	defer coordinator.Dispose()
	ctx := &context{
		conf: &packages.Config{Fset: fset},
		prog: coordinator,
		mode: ModeBuild,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      runtime.GOOS,
			Goarch:    runtime.GOARCH,
		},
	}
	err := buildPackageGroup(ctx, []*packageBuildTask{newPackageBuildTask(pkg)}, false)
	if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
		t.Fatalf("parallel build error = %v", err)
	}
}

func TestBuildAllPkgsReturnsPackageGroupErrors(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	newContext := func(fset *token.FileSet) *context {
		return &context{
			conf:      &packages.Config{Fset: fset},
			mode:      ModeGen,
			buildConf: &Config{},
		}
	}

	t.Run("ordinary package", func(t *testing.T) {
		fset, pkg := invalidEmbedPackage(t)
		_, err := buildAllPkgs(newContext(fset), []*aPackage{pkg}, false)
		if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
			t.Fatalf("ordinary package group error = %v", err)
		}
	})

	t.Run("runtime package", func(t *testing.T) {
		fset, pkg := invalidEmbedPackage(t)
		pkg.ID = env.LLGoRuntimePkg
		pkg.PkgPath = env.LLGoRuntimePkg
		pkg.Types = types.NewPackage(env.LLGoRuntimePkg, "runtime")
		_, err := buildAllPkgs(newContext(fset), []*aPackage{pkg}, false)
		if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
			t.Fatalf("runtime package group error = %v", err)
		}
	})
}

func TestExecuteIsolatedPackageReleasesProgramWithoutModule(t *testing.T) {
	coordinator := llssa.NewProgram(&llssa.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	defer coordinator.Dispose()
	ctx := &context{
		prog: coordinator,
		buildConf: &Config{
			Goos:   runtime.GOOS,
			Goarch: runtime.GOARCH,
		},
	}

	t.Run("frontend error", func(t *testing.T) {
		fset, pkg := invalidEmbedPackage(t)
		ctx.conf = &packages.Config{Fset: fset}
		task := newPackageBuildTask(pkg)
		err := ctx.executeIsolatedPackage(task, false)
		if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
			t.Fatalf("isolated frontend error = %v", err)
		}
		if pkg.LPkg != nil {
			t.Fatal("isolated frontend error retained LPkg")
		}
	})

	t.Run("skipped package", func(t *testing.T) {
		pkg := &aPackage{Package: &packages.Package{ID: "unsafe", PkgPath: "unsafe", Types: types.Unsafe}}
		if err := ctx.executeIsolatedPackage(newPackageBuildTask(pkg), false); err != nil {
			t.Fatal(err)
		}
		if pkg.LPkg != nil {
			t.Fatal("skipped isolated package retained LPkg")
		}
	})
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
	err = runPackageJob(7, func(int) error { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "package job 7 panicked: boom") {
		t.Fatalf("non-error panic = %v", err)
	}
}

func TestBuildPackageGroupEmpty(t *testing.T) {
	ctx := &context{buildConf: &Config{}}
	if err := buildPackageGroup(ctx, nil, false); err != nil {
		t.Fatalf("empty package group = %v", err)
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
	if task.plan9asmPkgs["example.com/p"] != coordinator.plan9asmPkgs["example.com/p"] {
		t.Fatal("backend task lost prepared Plan9 package policy")
	}
}

func TestPackageSchedulingHandlesNonBackendPackages(t *testing.T) {
	task := &packageBuildTask{pkg: &aPackage{}}
	ctx := &context{mode: ModeGen, buildConf: &Config{BuildMode: BuildModeExe}}
	serial, err := ctx.packageRequiresCoordinator(task)
	if err != nil || !serial {
		t.Fatalf("generation package coordinator = %v, %v; want true, nil", serial, err)
	}

	usesPlan9, err := (&context{}).packageUsesPlan9Asm(task.pkg)
	if err != nil || usesPlan9 {
		t.Fatalf("nil package Plan9 asm = %v, %v; want false, nil", usesPlan9, err)
	}
}
