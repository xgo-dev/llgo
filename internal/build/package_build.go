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

type packageBuildSpec struct {
	pkg       *aPackage
	kind      int
	kindParam string
	runtime   bool
}

func newPackageBuildSpec(pkg *aPackage) packageBuildSpec {
	kind, kindParam := cl.PkgKindOf(pkg.Types)
	return packageBuildSpec{
		pkg:       pkg,
		kind:      kind,
		kindParam: kindParam,
		runtime:   isRuntimePkg(pkg.PkgPath),
	}
}

func (s packageBuildSpec) isDeclOnly() bool {
	return s.kind == cl.PkgDeclOnly
}

func (s packageBuildSpec) isLinkOnly() bool {
	return s.kind == cl.PkgLinkIR || s.kind == cl.PkgLinkExtern || s.kind == cl.PkgPyModule
}

func (s packageBuildSpec) hasSource() bool {
	return len(s.pkg.GoFiles) > 0
}

func (s packageBuildSpec) needsRuntimeSignals() bool {
	return !s.isLinkOnly() && !s.isDeclOnly()
}

type packageBuildResult struct {
	needRuntime bool
	needPyInit  bool
}

type packagePreflight struct {
	spec packageBuildSpec
	skip bool
}

func preflightPackageBuilds(ctx *context, specs []packageBuildSpec, verbose bool) (map[*aPackage]packagePreflight, error) {
	preflights := make(map[*aPackage]packagePreflight, len(specs))
	for _, spec := range specs {
		skip, err := preflightPackageBuild(ctx, spec, verbose)
		if err != nil {
			return nil, err
		}
		preflights[spec.pkg] = packagePreflight{spec: spec, skip: skip}
	}
	return preflights, nil
}

func packageBuildSpecsForRuntime(specs []packageBuildSpec, runtime bool) []packageBuildSpec {
	filtered := make([]packageBuildSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.runtime == runtime {
			filtered = append(filtered, spec)
		}
	}
	return filtered
}

// buildPreflightedPackageGroup executes package backends first and then
// publishes archives/cache entries serially in deterministic package order.
func buildPreflightedPackageGroup(ctx *context, specs []packageBuildSpec, preflights map[*aPackage]packagePreflight, verbose bool) ([]packageBuildResult, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	patched, coordinator, isolated, err := partitionPackageExecutions(ctx, specs)
	if err != nil {
		return nil, err
	}
	for _, index := range patched {
		if err := executePreflightedPackage(ctx, preflights[specs[index].pkg], verbose); err != nil {
			return nil, err
		}
	}
	for _, index := range coordinator {
		preflight := preflights[specs[index].pkg]
		if preflight.skip {
			continue
		}
		if err := executePackageBuild(ctx, preflight.spec, verbose); err != nil {
			return nil, err
		}
	}
	if err := executeIsolatedPackages(ctx, specs, isolated, preflights, verbose); err != nil {
		return nil, err
	}

	results := make([]packageBuildResult, len(specs))
	for i, spec := range specs {
		preflight := preflights[spec.pkg]
		if preflight.skip {
			results[i] = packageBuildResultFor(spec)
			continue
		}
		result, err := finalizePackageBuild(ctx, spec, verbose)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func partitionPackageExecutions(ctx *context, specs []packageBuildSpec) (patched, coordinator, isolated []int, err error) {
	for i, spec := range specs {
		if _, ok := ctx.patches[spec.pkg.PkgPath]; ok {
			patched = append(patched, i)
			continue
		}
		serial, serialErr := ctx.packageRequiresCoordinator(spec)
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

func executePreflightedPackage(ctx *context, preflight packagePreflight, verbose bool) error {
	if preflight.skip {
		return nil
	}
	serial, err := ctx.packageRequiresCoordinator(preflight.spec)
	if err != nil {
		return err
	}
	if serial {
		return executePackageBuild(ctx, preflight.spec, verbose)
	}
	return ctx.executeIsolatedPackage(preflight.spec, verbose)
}

func executeIsolatedPackages(ctx *context, specs []packageBuildSpec, indexes []int, preflights map[*aPackage]packagePreflight, verbose bool) error {
	if len(indexes) == 0 {
		return nil
	}
	return runBoundedPackageJobs(ctx.buildConf.parallelism(), indexes, func(index int) error {
		preflight := preflights[specs[index].pkg]
		if preflight.skip {
			return nil
		}
		return ctx.executeIsolatedPackage(preflight.spec, verbose)
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

func (ctx *context) packageRequiresCoordinator(spec packageBuildSpec) (bool, error) {
	if !ctx.canUseIsolatedBackend() {
		return true, nil
	}
	usesPlan9, err := ctx.packageUsesPlan9Asm(spec.pkg)
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

func (ctx *context) executeIsolatedPackage(spec packageBuildSpec, verbose bool) error {
	return ctx.executeIsolatedPackageWithCallerTracking(spec, ctx.callerTracking, verbose)
}

func (ctx *context) executeIsolatedPackageWithCallerTracking(spec packageBuildSpec, tracking *cl.CallerTracking, verbose bool) error {
	session, err := ctx.backend.newSession()
	if err != nil {
		return fmt.Errorf("create backend session for %s: %w", spec.pkg.PkgPath, err)
	}
	defer session.prog.Dispose()
	task := ctx.newBackendTaskWithCallerTracking(session, tracking)
	defer func() { spec.pkg.LPkg = nil }()

	if err := buildPkg(task, spec.pkg, verbose); err != nil {
		return err
	}
	if spec.pkg.LPkg == nil {
		if spec.pkg.Summary == nil {
			spec.pkg.Summary = summarizePackage(spec.pkg)
		}
		return nil
	}
	if !spec.pkg.CacheHit {
		if spec.needsRuntimeSignals() {
			spec.pkg.setNeedRuntimeOrPyInit(spec.pkg.LPkg.NeedRuntime, spec.pkg.LPkg.NeedPyInit)
		}
		spec.pkg.Summary = summarizePackage(spec.pkg)
	} else {
		spec.pkg.Summary.AbiTypes = session.prog.AbiTypes()
	}
	if spec.pkg.Summary == nil {
		return fmt.Errorf("package %s produced no linker summary", spec.pkg.PkgPath)
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
	return ctx.newBackendTaskWithCallerTracking(session, ctx.callerTracking)
}

func (ctx *context) newBackendTaskWithCallerTracking(session backendSession, tracking *cl.CallerTracking) *context {
	return &context{
		conf:            ctx.conf,
		progSSA:         ctx.progSSA,
		prog:            session.prog,
		dedup:           ctx.dedup,
		patches:         ctx.patches,
		callerTracking:  tracking,
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

func (ctx *context) executeCoordinatorPackageWithCallerTracking(spec packageBuildSpec, tracking *cl.CallerTracking, verbose bool) error {
	task := *ctx
	task.callerTracking = tracking
	return executePackageBuild(&task, spec, verbose)
}

func packageBuildResultFor(spec packageBuildSpec) packageBuildResult {
	return packageBuildResult{
		needRuntime: spec.pkg.NeedRt,
		needPyInit:  spec.pkg.NeedPyInit,
	}
}
