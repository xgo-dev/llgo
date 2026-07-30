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

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/packages"
)

type packageBuildTask struct {
	pkg       *aPackage
	kind      int
	kindParam string
	skip      bool
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

type packageBuildResult struct {
	needRuntime bool
	needPyInit  bool
}

func prePackageBuilds(ctx *context, tasks []*packageBuildTask, verbose bool) error {
	for _, task := range tasks {
		if err := prePackageBuild(ctx, task, verbose); err != nil {
			return err
		}
	}
	return nil
}

func packageBuildTasksForRuntime(tasks []*packageBuildTask, runtime bool) []*packageBuildTask {
	filtered := make([]*packageBuildTask, 0, len(tasks))
	for _, task := range tasks {
		if task.isRuntime() == runtime {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// buildPrePackageGroup executes package backends first and then
// publishes archives/cache entries serially in deterministic package order.
func buildPrePackageGroup(ctx *context, tasks []*packageBuildTask, verbose bool) ([]packageBuildResult, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	patched, coordinator, isolated, err := partitionPackageExecutions(ctx, tasks)
	if err != nil {
		return nil, err
	}
	for _, index := range patched {
		if err := executePrePackage(ctx, tasks[index], verbose); err != nil {
			return nil, err
		}
	}
	for _, index := range coordinator {
		task := tasks[index]
		if task.skip {
			continue
		}
		if err := executePackageBuild(ctx, task, verbose); err != nil {
			return nil, err
		}
	}
	if err := executeIsolatedPackages(ctx, tasks, isolated, verbose); err != nil {
		return nil, err
	}

	results := make([]packageBuildResult, len(tasks))
	for i, task := range tasks {
		if task.skip {
			results[i] = packageBuildResultFor(task)
			continue
		}
		result, err := finalizePackageBuild(ctx, task, verbose)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func partitionPackageExecutions(ctx *context, tasks []*packageBuildTask) (patched, coordinator, isolated []int, err error) {
	for i, task := range tasks {
		if _, ok := ctx.patches[task.pkg.PkgPath]; ok {
			patched = append(patched, i)
			continue
		}
		serial, serialErr := ctx.packageRequiresCoordinator(task)
		if serialErr != nil {
			return nil, nil, nil, serialErr
		}
		if serial {
			coordinator = append(coordinator, i)
		} else {
			isolated = append(isolated, i)
		}
	}
	return
}

func executePrePackage(ctx *context, task *packageBuildTask, verbose bool) error {
	if task.skip {
		return nil
	}
	serial, err := ctx.packageRequiresCoordinator(task)
	if err != nil {
		return err
	}
	if serial {
		return executePackageBuild(ctx, task, verbose)
	}
	return ctx.executeIsolatedPackage(task, verbose)
}

func executeIsolatedPackages(ctx *context, tasks []*packageBuildTask, indexes []int, verbose bool) error {
	if len(indexes) == 0 {
		return nil
	}
	return runBoundedPackageJobs(ctx.buildConf.parallelism(), indexes, func(index int) error {
		task := tasks[index]
		if task.skip {
			return nil
		}
		return ctx.executeIsolatedPackage(task, verbose)
	})
}

func runBoundedPackageJobs(parallelism int, indexes []int, run func(index int) error) error {
	if len(indexes) == 0 {
		return nil
	}
	workers := min(max(1, parallelism), len(indexes))
	jobs := make(chan int, len(indexes))
	errs := make(map[int]error, len(indexes))
	var errsMu sync.Mutex
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				if err := runPackageJob(index, run); err != nil {
					errsMu.Lock()
					errs[index] = err
					errsMu.Unlock()
				}
			}
		}()
	}
	for _, index := range indexes {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	for _, index := range indexes {
		if err := errs[index]; err != nil {
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
	session, err := ctx.backend.newSession()
	if err != nil {
		return fmt.Errorf("create backend session for %s: %w", task.pkg.PkgPath, err)
	}
	defer session.prog.Dispose()
	backendCtx := ctx.newBackendTask(session)
	defer func() { task.pkg.LPkg = nil }()

	if err := buildPkg(backendCtx, task.pkg, verbose); err != nil {
		return err
	}
	if task.pkg.LPkg == nil {
		if task.pkg.Summary == nil {
			task.pkg.Summary = summarizePackage(task.pkg)
		}
		return nil
	}
	if !task.pkg.CacheHit {
		if task.needsRuntimeSignals() {
			task.pkg.setNeedRuntimeOrPyInit(task.pkg.LPkg.NeedRuntime, task.pkg.LPkg.NeedPyInit)
		}
		task.pkg.Summary = summarizePackage(task.pkg)
	} else {
		task.pkg.Summary.AbiTypes = session.prog.AbiTypes()
	}
	if task.pkg.Summary == nil {
		return fmt.Errorf("package %s produced no linker summary", task.pkg.PkgPath)
	}
	return nil
}

func preparePackageSFiles(ctx *context, pkg *aPackage) error {
	if _, err := pkgSFiles(ctx, pkg.Package); err != nil {
		return err
	}
	if pkg.AltPkg != nil {
		if _, err := pkgSFiles(ctx, pkg.AltPkg.Package); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *context) newBackendTask(session backendSession) *context {
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
		backend:         ctx.backend,
		sfilesCache:     ctx.sfilesCache,
		sfilesFrozen:    true,
		plan9asmReady:   true,
		plan9asmMode:    ctx.plan9asmMode,
		plan9asmPkgs:    ctx.plan9asmPkgs,
		plan9asmSigs:    make(map[string]map[string]struct{}),
	}
}

func packageBuildResultFor(task *packageBuildTask) packageBuildResult {
	return packageBuildResult{
		needRuntime: task.pkg.NeedRt,
		needPyInit:  task.pkg.NeedPyInit,
	}
}
