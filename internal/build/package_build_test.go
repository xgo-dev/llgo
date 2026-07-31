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
	"fmt"
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
)

func TestPackageBuildSpecAndResult(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		PkgPath: "example.com/p",
		GoFiles: []string{"p.go"},
		Types:   types.NewPackage("example.com/p", "p"),
	}, NeedRt: true, NeedPyInit: true}
	spec := newPackageBuildSpec(pkg)
	if spec.isDeclOnly() || spec.isLinkOnly() || !spec.hasSource() || spec.runtime || !spec.needsRuntimeSignals() {
		t.Fatalf("unexpected normal package spec: %+v", spec)
	}
	result := packageBuildResultFor(spec)
	if !result.needRuntime || !result.needPyInit {
		t.Fatalf("unexpected package result: %+v", result)
	}
}

func TestPackageBuildSpecSpecialKinds(t *testing.T) {
	decl := newPackageBuildSpec(&aPackage{Package: &packages.Package{
		PkgPath: "unsafe",
		Types:   types.Unsafe,
	}})
	if !decl.isDeclOnly() || decl.needsRuntimeSignals() {
		t.Fatalf("unexpected declaration-only spec: %+v", decl)
	}
	runtime := newPackageBuildSpec(&aPackage{Package: &packages.Package{
		PkgPath: env.LLGoRuntimePkg,
		Types:   types.NewPackage(env.LLGoRuntimePkg, "runtime"),
	}})
	if !runtime.runtime {
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

func TestBuildOnePackageSkipsAlreadyBuiltPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/already-built",
		PkgPath: "example.com/already-built",
		GoFiles: []string{"already.go"},
		Types:   types.NewPackage("example.com/already-built", "already"),
	}, NeedRt: true}
	ctx := &context{built: map[string]none{pkg.ID: {}}}

	result, err := buildOnePackage(ctx, newPackageBuildSpec(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.needRuntime {
		t.Fatalf("build result = %+v, want runtime requirement preserved", result)
	}
}

func TestPreflightPackageBuildSkipsDeclarationOnlyPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:         "unsafe",
		PkgPath:    "unsafe",
		Types:      types.Unsafe,
		ExportFile: "stale.a",
	}}
	ctx := &context{
		conf:      &packages.Config{},
		buildConf: &Config{Goos: "linux", Goarch: "amd64", ForceRebuild: true},
		built:     make(map[string]none),
	}

	skip, err := preflightPackageBuild(ctx, newPackageBuildSpec(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !skip {
		t.Fatal("declaration-only package was not skipped")
	}
	if pkg.ExportFile != "" {
		t.Fatalf("ExportFile = %q, want empty", pkg.ExportFile)
	}
	if _, ok := ctx.built[pkg.ID]; !ok {
		t.Fatal("declaration-only package was not recorded as built")
	}
}

func TestFinalizePackageBuildReturnsCachedResult(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/cached",
		PkgPath: "example.com/cached",
		Types:   types.NewPackage("example.com/cached", "cached"),
	}, CacheHit: true, NeedPyInit: true}

	result, err := finalizePackageBuild(&context{}, newPackageBuildSpec(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.needPyInit {
		t.Fatalf("build result = %+v, want Python initialization requirement preserved", result)
	}
}

func TestPreflightFingerprintsSkippedPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "unsafe",
		PkgPath: "unsafe",
		Types:   types.Unsafe,
	}}
	ctx := &context{
		conf:      &packages.Config{},
		buildConf: &Config{Goos: "linux", Goarch: "amd64", ForceRebuild: true},
		built:     make(map[string]none),
	}
	skip, err := preflightPackageBuild(ctx, newPackageBuildSpec(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !skip || pkg.Fingerprint == "" || pkg.Manifest == "" || pkg.Summary == nil {
		t.Fatalf("skipped package was not fully prepared: skip=%v fingerprint=%q manifest=%q summary=%#v", skip, pkg.Fingerprint, pkg.Manifest, pkg.Summary)
	}
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
	patched, coordinator, isolated, err := partitionPackageExecutions(ctx, []packageBuildSpec{
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

func TestPublishPackagePipelineTasksUsesBoundedWorkers(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	tasks := make([]*packagePipelineTask, 5)
	for i := range tasks {
		pkgPath := fmt.Sprintf("example.com/p%d", i)
		tasks[i] = &packagePipelineTask{
			executed: i != 4,
			spec: packageBuildSpec{pkg: &aPackage{Package: &packages.Package{
				ID:      pkgPath,
				PkgPath: pkgPath,
				Name:    fmt.Sprintf("p%d", i),
			}}},
		}
	}

	started := make(chan struct{}, len(tasks))
	release := make(chan struct{})
	done := make(chan error, 1)
	var active atomic.Int32
	var maximum atomic.Int32
	go func() {
		ctx := &context{buildConf: &Config{BuildParallelism: 8}}
		done <- publishPackagePipelineTasksWith(ctx, tasks, false, func(_ *context, _ packageBuildSpec, _ bool) (packageBuildResult, error, error) {
			current := active.Add(1)
			for {
				old := maximum.Load()
				if current <= old || maximum.CompareAndSwap(old, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			active.Add(-1)
			return packageBuildResult{}, nil, nil
		})
	}()

	for range maxConcurrentPackagePublishes {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatal("publication workers did not start concurrently")
		}
	}
	if got := maximum.Load(); got != maxConcurrentPackagePublishes {
		close(release)
		t.Fatalf("maximum concurrent publications = %d, want %d", got, maxConcurrentPackagePublishes)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := len(started); got != 2 {
		t.Fatalf("remaining executed publications = %d, want 2", got)
	}
}

func TestPublishPackagePipelineTasksReturnsFirstOrderedError(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	first := errors.New("first")
	second := errors.New("second")
	tasks := make([]*packagePipelineTask, 3)
	for i := range tasks {
		tasks[i] = &packagePipelineTask{
			executed: true,
			spec: packageBuildSpec{pkg: &aPackage{Package: &packages.Package{
				PkgPath: fmt.Sprintf("example.com/p%d", i),
			}}},
		}
	}
	err := publishPackagePipelineTasksWith(
		&context{buildConf: &Config{BuildParallelism: 3}},
		tasks,
		false,
		func(_ *context, spec packageBuildSpec, _ bool) (packageBuildResult, error, error) {
			switch spec.pkg.PkgPath {
			case "example.com/p0":
				time.Sleep(10 * time.Millisecond)
				return packageBuildResult{}, nil, first
			case "example.com/p1":
				return packageBuildResult{}, nil, second
			default:
				return packageBuildResult{}, nil, nil
			}
		},
	)
	if !errors.Is(err, first) {
		t.Fatalf("error = %v, want first submitted error", err)
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

func TestPreflightPackageBuildsRecordsSkippedPackages(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/already-built",
		PkgPath: "example.com/already-built",
	}}
	ctx := &context{built: map[string]none{pkg.ID: {}}}
	preflights, err := preflightPackageBuilds(ctx, []packageBuildSpec{{pkg: pkg}}, false)
	if err != nil {
		t.Fatal(err)
	}
	preflight, ok := preflights[pkg]
	if !ok || !preflight.skip || preflight.spec.pkg != pkg {
		t.Fatalf("preflights = %#v, want skipped package", preflights)
	}
}

func TestPackageSchedulingHandlesNonBackendPackages(t *testing.T) {
	spec := packageBuildSpec{pkg: &aPackage{}}
	ctx := &context{mode: ModeGen, buildConf: &Config{BuildMode: BuildModeExe}}
	serial, err := ctx.packageRequiresCoordinator(spec)
	if err != nil || !serial {
		t.Fatalf("generation package coordinator = %v, %v; want true, nil", serial, err)
	}

	usesPlan9, err := (&context{}).packageUsesPlan9Asm(spec.pkg)
	if err != nil || usesPlan9 {
		t.Fatalf("nil package Plan9 asm = %v, %v; want false, nil", usesPlan9, err)
	}
	patched, coordinator, isolated, err := partitionPackageExecutions(ctx, nil)
	if err != nil || patched != nil || coordinator != nil || isolated != nil {
		t.Fatalf("empty partition = %v, %v, %v, %v", patched, coordinator, isolated, err)
	}
}
