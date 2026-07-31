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
	"go/ast"

	"github.com/goplus/llgo/cl"
	"golang.org/x/tools/go/ssa"
)

func preparePackagePatches(ctx *context, pkgs []*aPackage) {
	for _, pkg := range pkgs {
		patch, ok := ctx.patches[pkg.PkgPath]
		if !ok {
			continue
		}
		files := append([]*ast.File(nil), pkg.Syntax...)
		if pkg.AltPkg != nil {
			files = append(files, pkg.AltPkg.Syntax...)
		}
		ctx.patches[pkg.PkgPath] = cl.PreparePatch(patch, pkg.Types, files)
	}
}

type packagePipelineClass uint8

const (
	pipelineIsolated packagePipelineClass = iota
	pipelineCoordinator
)

type packagePipelineStage uint8

const (
	pipelineStageSSA packagePipelineStage = iota
	pipelineStageBackend
	pipelineStagePatchedBackend
)

type packagePipelineSSA struct {
	entry         ssaBuildEntry
	order         int
	done          bool
	callerSummary cl.CallerTrackingSummary
}

type packagePipelineTask struct {
	index       int
	spec        packageBuildSpec
	preflight   packagePreflight
	class       packagePipelineClass
	ownSSA      *packagePipelineSSA
	ssaDeps     []*packagePipelineSSA
	callerRoots []*ssa.Package
	callerNodes []*packagePipelineSSA
	enabled     bool
	running     bool
	done        bool
	executed    bool
	patched     bool
}

type packagePipelineEvent struct {
	ssa  *packagePipelineSSA
	task *packagePipelineTask
	err  error
}

// buildPackagePipeline overlaps completed Go SSA packages with isolated LLVM
// backends. SSA and backend work share one concurrency budget, so -p remains a
// bound on CPU-heavy package work rather than creating two independent pools.
func buildPackagePipeline(ctx *context, entries []ssaBuildEntry, pkgs []*aPackage, verbose bool) ([]*aPackage, error) {
	specs := make([]packageBuildSpec, len(pkgs))
	for i, pkg := range pkgs {
		specs[i] = newPackageBuildSpec(pkg)
	}
	preflights, err := preflightPackageBuilds(ctx, specs, verbose)
	if err != nil {
		return nil, err
	}
	ctx.sfilesFrozen = true

	ssaNodes, ssaByPackage := newPackagePipelineSSANodes(entries)
	tasks, serialOrder, err := newPackagePipelineTasks(ctx, specs, preflights, ssaByPackage)
	if err != nil {
		return nil, err
	}
	ssaNodes = orderPackagePipelineSSA(ssaNodes, tasks)

	hostRuntime := ctx.buildConf.Target == ""
	normalRemaining := 0
	backendFailed := false
	var needRuntime, needPyInit bool
	for _, task := range tasks {
		task.enabled = !task.spec.runtime || hostRuntime
		if task.preflight.skip {
			task.done = true
			if !task.spec.runtime {
				result := packageBuildResultFor(task.spec)
				needRuntime = needRuntime || result.needRuntime
				needPyInit = needPyInit || result.needPyInit
			}
			continue
		}
		if !task.spec.runtime {
			normalRemaining++
		}
	}

	runtimeResolved := hostRuntime
	resolveRuntime := func() {
		if runtimeResolved || normalRemaining != 0 {
			return
		}
		runtimeResolved = true
		buildRuntime := !backendFailed && (needRuntime || needPyInit)
		for _, task := range tasks {
			if !task.spec.runtime || task.done {
				continue
			}
			task.enabled = buildRuntime
			if !buildRuntime {
				task.done = true
			}
		}
	}
	resolveRuntime()

	parallelism := ctx.buildConf.parallelism()
	events := make(chan packagePipelineEvent, parallelism)
	active := 0
	ssaActive := 0
	nextSSA := 0
	ssaDone := 0
	serialPos := 0
	serialActive := false
	exclusiveActive := false
	var ssaErr error
	backendErrs := make(map[int]error)

	advanceSerial := func() {
		for serialPos < len(serialOrder) && serialOrder[serialPos].done {
			serialPos++
		}
	}
	advanceSerial()

	ready := func(task *packagePipelineTask) bool {
		if task == nil || !task.enabled || task.running || task.done {
			return false
		}
		// Legacy patch compilation rewrites shared go/ssa package type pointers.
		// Its immutable merged type view is prepared up front for ordinary
		// backends, but the patched package itself still runs exclusively after
		// SSA construction has stopped mutating the program.
		if task.patched && ssaDone != len(ssaNodes) {
			return false
		}
		if task.ownSSA != nil && !task.ownSSA.done {
			return false
		}
		for _, dep := range task.ssaDeps {
			if !dep.done {
				return false
			}
		}
		return true
	}
	nextBackend := func() *packagePipelineTask {
		advanceSerial()
		if !serialActive && serialPos < len(serialOrder) {
			task := serialOrder[serialPos]
			if task.patched && ssaDone == len(ssaNodes) {
				if active == 0 && ready(task) {
					return task
				}
				return nil
			}
			if ready(task) {
				return task
			}
		}
		for _, task := range tasks {
			if task.class == pipelineIsolated && ready(task) {
				return task
			}
		}
		return nil
	}
	allBackendsDone := func() bool {
		for _, task := range tasks {
			if !task.done {
				return false
			}
		}
		return true
	}
	launchSSA := func() bool {
		if nextSSA >= len(ssaNodes) {
			return false
		}
		node := ssaNodes[nextSSA]
		nextSSA++
		active++
		ssaActive++
		go func() {
			pkgPath := packagePipelineSSAPath(node.entry.pkg)
			observePackagePipeline(ctx, pipelineStageSSA, pkgPath, true)
			runErr := buildPackagePipelineSSA(node.entry)
			observePackagePipeline(ctx, pipelineStageSSA, pkgPath, false)
			events <- packagePipelineEvent{ssa: node, err: runErr}
		}()
		return true
	}

	for ssaDone != len(ssaNodes) || !allBackendsDone() || active != 0 {
		for ssaErr == nil && active < parallelism {
			if exclusiveActive {
				break
			}
			// Keep at least one SSA producer active while packages remain. At
			// -p=1 this preserves the legacy all-SSA-then-backend order; with
			// more workers, the remaining slots consume ready backend work.
			if ssaActive == 0 && launchSSA() {
				continue
			}
			if task := nextBackend(); task != nil {
				task.running = true
				if task.class == pipelineCoordinator {
					serialActive = true
				}
				if task.patched {
					exclusiveActive = true
				}
				active++
				go func() {
					stage := pipelineStageBackend
					if task.patched {
						stage = pipelineStagePatchedBackend
					}
					observePackagePipeline(ctx, stage, task.spec.pkg.PkgPath, true)
					runErr := runPackageJob(task.index, func(int) error {
						summaries := make([]cl.CallerTrackingSummary, 0, len(task.callerNodes))
						for _, node := range task.callerNodes {
							summaries = append(summaries, node.callerSummary)
						}
						tracking := cl.NewPackageCallerTrackingForPackages(task.callerRoots, summaries...)
						if task.class == pipelineCoordinator {
							return ctx.executeCoordinatorPackageWithCallerTracking(task.spec, tracking, verbose)
						}
						return ctx.executeIsolatedPackageWithCallerTracking(task.spec, tracking, verbose)
					})
					observePackagePipeline(ctx, stage, task.spec.pkg.PkgPath, false)
					events <- packagePipelineEvent{task: task, err: runErr}
				}()
				continue
			}
			if launchSSA() {
				continue
			}
			break
		}

		if active == 0 {
			if ssaErr != nil {
				return nil, ssaErr
			}
			resolveRuntime()
			if allBackendsDone() && ssaDone == len(ssaNodes) {
				break
			}
			return nil, fmt.Errorf("package pipeline stalled with %d/%d SSA packages complete", ssaDone, len(ssaNodes))
		}

		event := <-events
		active--
		switch {
		case event.ssa != nil:
			ssaActive--
			if event.err == nil {
				event.ssa.callerSummary, event.err = finishPackagePipelineSSA(event.ssa.entry)
			}
			if event.err != nil {
				if ssaErr == nil {
					ssaErr = event.err
				}
				continue
			}
			event.ssa.done = true
			ssaDone++
		case event.task != nil:
			task := event.task
			task.running = false
			task.done = true
			task.executed = event.err == nil
			if task.class == pipelineCoordinator {
				serialActive = false
			}
			if task.patched {
				exclusiveActive = false
			}
			if event.err != nil {
				backendErrs[task.index] = event.err
				backendFailed = true
			}
			if !task.spec.runtime {
				normalRemaining--
			}
			if event.err == nil && !task.spec.runtime {
				result := packageBuildResultFor(task.spec)
				needRuntime = needRuntime || result.needRuntime
				needPyInit = needPyInit || result.needPyInit
			}
			resolveRuntime()
			advanceSerial()
		}
	}

	if ssaErr != nil {
		return nil, ssaErr
	}
	for i := range tasks {
		if err := backendErrs[i]; err != nil {
			return nil, err
		}
	}
	for _, task := range tasks {
		if !task.executed {
			continue
		}
		if _, err := finalizePackageBuild(ctx, task.spec, verbose); err != nil {
			return nil, err
		}
	}
	return pkgs, nil
}

func observePackagePipeline(ctx *context, stage packagePipelineStage, pkgPath string, start bool) {
	if observer := ctx.buildConf.packagePipelineObserver; observer != nil {
		observer(stage, pkgPath, start)
	}
}

func packagePipelineSSAPath(pkg *ssa.Package) string {
	if pkg == nil || pkg.Pkg == nil {
		return ""
	}
	return pkg.Pkg.Path()
}

func newPackagePipelineSSANodes(entries []ssaBuildEntry) ([]*packagePipelineSSA, map[*ssa.Package]*packagePipelineSSA) {
	nodes := make([]*packagePipelineSSA, 0, len(entries))
	byPackage := make(map[*ssa.Package]*packagePipelineSSA, len(entries))
	for _, entry := range entries {
		if entry.pkg == nil {
			continue
		}
		if node := byPackage[entry.pkg]; node != nil {
			node.entry.fixOrder = node.entry.fixOrder || entry.fixOrder
			continue
		}
		node := &packagePipelineSSA{entry: entry, order: len(nodes)}
		nodes = append(nodes, node)
		byPackage[entry.pkg] = node
	}
	return nodes, byPackage
}

func newPackagePipelineTasks(
	ctx *context,
	specs []packageBuildSpec,
	preflights map[*aPackage]packagePreflight,
	ssaByPackage map[*ssa.Package]*packagePipelineSSA,
) ([]*packagePipelineTask, []*packagePipelineTask, error) {
	tasks := make([]*packagePipelineTask, len(specs))
	var patchedNormal, coordinatorNormal, patchedRuntime, coordinatorRuntime []*packagePipelineTask
	for i, spec := range specs {
		task := &packagePipelineTask{
			index:     i,
			spec:      spec,
			preflight: preflights[spec.pkg],
			ownSSA:    ssaByPackage[spec.pkg.SSA],
		}
		if spec.pkg.SSA != nil {
			task.callerRoots = append(task.callerRoots, spec.pkg.SSA)
		}
		seenCallerNodes := make(map[*packagePipelineSSA]bool)
		addCallerNode := func(pkg *ssa.Package) {
			node := ssaByPackage[pkg]
			if node == nil || seenCallerNodes[node] {
				return
			}
			seenCallerNodes[node] = true
			task.callerNodes = append(task.callerNodes, node)
		}
		addCallerNode(spec.pkg.SSA)
		seen := make(map[*packagePipelineSSA]bool)
		addDep := func(pkg *ssa.Package) {
			node := ssaByPackage[pkg]
			if node == nil || node == task.ownSSA || seen[node] {
				return
			}
			seen[node] = true
			task.ssaDeps = append(task.ssaDeps, node)
		}
		visitedDeps := make(map[string]bool)
		var addPackageDeps func(*aPackage)
		addPackageDeps = func(pkg *aPackage) {
			if pkg == nil || visitedDeps[pkg.ID] {
				return
			}
			visitedDeps[pkg.ID] = true
			addDep(pkg.SSA)
			for _, dep := range effectiveDependencies(pkg) {
				addPackageDeps(ctx.pkgByID[dep.ID])
			}
		}
		for _, dep := range effectiveDependencies(spec.pkg) {
			depPkg := ctx.pkgByID[dep.ID]
			if depPkg != nil {
				addCallerNode(depPkg.SSA)
			}
			addPackageDeps(depPkg)
		}
		if patch, ok := ctx.patches[spec.pkg.PkgPath]; ok {
			addDep(patch.Alt)
			addCallerNode(patch.Alt)
			if patch.Alt != nil && patch.Alt != spec.pkg.SSA {
				task.callerRoots = append(task.callerRoots, patch.Alt)
			}
		}

		patched := false
		if _, ok := ctx.patches[spec.pkg.PkgPath]; ok {
			patched = true
			task.patched = true
			task.class = pipelineCoordinator
		} else {
			serial, err := ctx.packageRequiresCoordinator(spec)
			if err != nil {
				return nil, nil, err
			}
			if serial {
				task.class = pipelineCoordinator
			}
		}
		if task.class == pipelineCoordinator && !task.preflight.skip {
			switch {
			case !spec.runtime && patched:
				patchedNormal = append(patchedNormal, task)
			case !spec.runtime:
				coordinatorNormal = append(coordinatorNormal, task)
			case patched:
				patchedRuntime = append(patchedRuntime, task)
			default:
				coordinatorRuntime = append(coordinatorRuntime, task)
			}
		}
		tasks[i] = task
	}
	serial := append(patchedNormal, coordinatorNormal...)
	serial = append(serial, patchedRuntime...)
	serial = append(serial, coordinatorRuntime...)
	return tasks, serial, nil
}

func orderPackagePipelineSSA(nodes []*packagePipelineSSA, tasks []*packagePipelineTask) []*packagePipelineSSA {
	deps := make(map[*packagePipelineSSA][]*packagePipelineSSA)
	for _, task := range tasks {
		if task.ownSSA == nil {
			continue
		}
		deps[task.ownSSA] = append(deps[task.ownSSA], task.ssaDeps...)
	}
	ordered := make([]*packagePipelineSSA, 0, len(nodes))
	state := make(map[*packagePipelineSSA]uint8, len(nodes))
	var visit func(*packagePipelineSSA)
	visit = func(node *packagePipelineSSA) {
		if node == nil || state[node] == 2 {
			return
		}
		if state[node] == 1 {
			return
		}
		state[node] = 1
		for _, dep := range deps[node] {
			visit(dep)
		}
		state[node] = 2
		ordered = append(ordered, node)
	}
	for _, node := range nodes {
		visit(node)
	}
	return ordered
}

func buildPackagePipelineSSA(entry ssaBuildEntry) (err error) {
	pkgPath := packagePipelineSSAPath(entry.pkg)
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("build SSA for %s: %v", pkgPath, value)
		}
	}()
	if entry.pkg == nil {
		return fmt.Errorf("build SSA for %s: nil package", pkgPath)
	}
	entry.pkg.Build()
	return nil
}

func finishPackagePipelineSSA(entry ssaBuildEntry) (summary cl.CallerTrackingSummary, err error) {
	pkgPath := packagePipelineSSAPath(entry.pkg)
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("fix SSA order for %s: %v", pkgPath, value)
		}
	}()
	if entry.fixOrder {
		fixSSAOrder(entry.pkg, entry.syntax)
	}
	return cl.SummarizeCallerTracking(entry.pkg), nil
}
