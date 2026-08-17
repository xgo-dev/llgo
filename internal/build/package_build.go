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
	"fmt"
	"sync"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/packages"
)

type packageBuildTask struct {
	pkg       *aPackage
	kind      int
	kindParam string
	skip      bool
	isolated  bool
	parallel  bool
}

func newPackageBuildTask(pkg *aPackage) *packageBuildTask {
	kind, kindParam := cl.PkgKindOf(pkg.Types)
	return &packageBuildTask{
		pkg:       pkg,
		kind:      kind,
		kindParam: kindParam,
	}
}

func (t *packageBuildTask) isRuntime() bool {
	return isRuntimePkg(t.pkg.PkgPath)
}

func (t *packageBuildTask) isDeclOnly() bool {
	return t.kind == cl.PkgDeclOnly
}

func (t *packageBuildTask) isLinkOnly() bool {
	return t.kind == cl.PkgLinkIR || t.kind == cl.PkgLinkExtern || t.kind == cl.PkgPyModule
}

func (t *packageBuildTask) hasSource() bool {
	return len(t.pkg.GoFiles) > 0
}

func (t *packageBuildTask) needsRuntimeSignals() bool {
	return !t.isLinkOnly() && !t.isDeclOnly()
}

// preparePackageBuilds completes every coordinator-only read before workers
// start. In particular, packageRequiresCoordinator populates both primary and
// alternate SFiles entries, so isolated backends share that cache read-only.
func preparePackageBuilds(ctx *context, tasks []*packageBuildTask, verbose bool) error {
	for _, task := range tasks {
		if err := prePackageBuild(ctx, task, verbose); err != nil {
			return err
		}
		if task.skip {
			continue
		}
		coordinator, err := ctx.packageRequiresCoordinator(task)
		if err != nil {
			return err
		}
		task.isolated = !coordinator
		_, patched := ctx.patches[task.pkg.PkgPath]
		// typepatch.Merge still updates the shared patch.Types scope while a
		// package is lowered. Keep patched packages isolated from the coordinator
		// Program, but do not execute two such merges concurrently.
		task.parallel = task.isolated && !patched
	}
	return nil
}

// buildPackageGroup prepares the group serially, then runs eligible package
// backends with bounded parallelism. Archive creation and cache publication
// stay in the same package job as the backend that produced the object data.
func buildPackageGroup(ctx *context, tasks []*packageBuildTask, verbose bool) error {
	if len(tasks) == 0 {
		return nil
	}
	prepareSpan := ctx.buildTrace.startCoordinator("prepare packages", map[string]any{
		"count":   len(tasks),
		"runtime": tasks[0].isRuntime(),
	})
	if err := preparePackageBuilds(ctx, tasks, verbose); err != nil {
		prepareSpan.done()
		return err
	}
	prepareSpan.done()

	isolated := make([]int, 0, len(tasks))
	// Complete coordinator and patched work before starting ordinary workers:
	// patched lowering finalizes shared type scopes that other packages read.
	for i, task := range tasks {
		if task.skip {
			continue
		}
		if task.parallel {
			isolated = append(isolated, i)
			continue
		}
		if err := tracePackageBuild(ctx, task, verbose, task.isolated, false); err != nil {
			return err
		}
	}
	if err := runBoundedPackageJobs(ctx.buildConf.parallelism(), isolated, func(index int) error {
		return tracePackageBuild(ctx, tasks[index], verbose, true, true)
	}); err != nil {
		return err
	}
	return nil
}

func tracePackageBuild(ctx *context, task *packageBuildTask, verbose, isolated, worker bool) (err error) {
	class := "coordinator"
	var traceSpan *buildTraceSpan
	if worker {
		class = "isolated"
		traceSpan = ctx.buildTrace.startWorker("backend+publish", task.pkg.PkgPath)
	} else {
		if isolated {
			class = "patched"
		}
		traceSpan = ctx.buildTrace.startPackageCoordinator("backend+publish", task.pkg.PkgPath)
	}
	traceSpan.setArg("package_id", task.pkg.ID)
	traceSpan.setArg("class", class)
	traceSpan.setArg("archive_publication", !task.pkg.CacheHit)
	ctx.buildTrace.flowFromSSA(task.pkg.ID, traceSpan)
	defer traceSpan.done()
	return buildPackage(ctx, task, verbose, isolated)
}

func buildPackage(ctx *context, task *packageBuildTask, verbose, isolated bool) error {
	var err error
	if isolated {
		err = ctx.executeIsolatedPackage(task, verbose)
	} else {
		err = executePackageBuild(ctx, task, verbose)
	}
	if err != nil {
		return err
	}
	return finalizePackageBuild(ctx, task, verbose)
}

func runBoundedPackageJobs(parallelism int, indexes []int, run func(index int) error) error {
	if len(indexes) == 0 {
		return nil
	}
	workers := min(max(1, parallelism), len(indexes))
	jobs := make(chan int, len(indexes))
	errs := make([]error, len(indexes))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pos := range jobs {
				errs[pos] = runPackageJob(indexes[pos], run)
			}
		}()
	}
	for pos := range indexes {
		jobs <- pos
	}
	close(jobs)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func runPackageJob(index int, run func(index int) error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			if recovered, ok := value.(error); ok {
				err = recovered
			} else {
				err = fmt.Errorf("package job %d panicked: %v", index, value)
			}
		}
	}()
	return run(index)
}

func (ctx *context) canUseIsolatedBackend() bool {
	return ctx.mode != ModeGen &&
		ctx.buildConf.BuildMode == BuildModeExe &&
		ctx.buildConf.ModuleHook == nil
}

func (ctx *context) packageRequiresCoordinator(task *packageBuildTask) (bool, error) {
	if !ctx.canUseIsolatedBackend() {
		return true, nil
	}
	usesPlan9, err := ctx.packageUsesPlan9Asm(task.pkg)
	if err != nil {
		return false, err
	}
	return usesPlan9, nil
}

func (ctx *context) packageUsesPlan9Asm(pkg *aPackage) (bool, error) {
	check := func(p *packages.Package) (bool, error) {
		if p == nil {
			return false, nil
		}
		sfiles, err := pkgSFiles(ctx, p)
		if err != nil {
			return false, err
		}
		return len(sfiles) != 0 && ctx.plan9asmEnabled(p.PkgPath), nil
	}
	if yes, err := check(pkg.Package); err != nil || yes {
		return yes, err
	}
	if pkg.AltPkg != nil {
		return check(pkg.AltPkg.Package)
	}
	return false, nil
}

func (ctx *context) executeIsolatedPackage(task *packageBuildTask, verbose bool) error {
	session := ctx.newBackendSession()
	owned := true
	defer func() {
		if owned {
			task.pkg.LPkg = nil
			session.prog.Dispose()
		}
	}()
	backendCtx := ctx.newBackendTask(session)

	if err := buildPkg(backendCtx, task.pkg, verbose); err != nil {
		return err
	}
	if task.pkg.LPkg == nil {
		return nil
	}
	owned = false
	if task.needsRuntimeSignals() {
		task.pkg.setNeedRuntimeOrPyInit(task.pkg.LPkg.NeedRuntime, task.pkg.LPkg.NeedPyInit)
	}
	// Linking still consumes live package state: method tables, globals,
	// funcinfo/PCLN, C exports, and DCE source modules. Cache hits still rebuild
	// that frontend module in the isolated Program, skip backend emission, and
	// keep the module alive until every link has completed.
	//
	// LPkg retains the Program that owns its LLVM context. Ownership therefore
	// moves to the coordinator on every success, not only when dead-code
	// dropping is enabled.
	return nil
}

func (ctx *context) newBackendTask(session backendSession) *context {
	// preparePackageBuilds populated every task's SFiles entry before workers start.
	// Backend tasks share that map read-only; a frozen miss returns an error.
	return &context{
		conf:            ctx.conf,
		progSSA:         ctx.progSSA,
		prog:            session.prog,
		dedup:           ctx.dedup,
		patches:         ctx.patches,
		callerTracking:  ctx.callerTracking,
		initial:         ctx.initial,
		pkgs:            ctx.pkgs,
		pkgByID:         ctx.pkgByID,
		mode:            ctx.mode,
		output:          ctx.output,
		passOpt:         ctx.passOpt,
		buildConf:       ctx.buildConf,
		crossCompile:    ctx.crossCompile,
		commands:        ctx.commands,
		frontendOptions: ctx.frontendOptions,
		cTransformer:    session.transformer,
		buildTrace:      ctx.buildTrace,
		sfilesCache:     ctx.sfilesCache,
		sfilesFrozen:    true,
		plan9asmReady:   true,
		plan9asmMode:    ctx.plan9asmMode,
		plan9asmPkgs:    ctx.plan9asmPkgs,
	}
}

func packageRuntimeNeeds(tasks []*packageBuildTask) (needRuntime, needPyInit bool) {
	for _, task := range tasks {
		needRuntime = needRuntime || task.pkg.NeedRt
		needPyInit = needPyInit || task.pkg.NeedPyInit
	}
	return
}
