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
	"container/heap"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
)

type buildDAGClass int

const (
	dagPackage buildDAGClass = iota
	dagLink
	dagTest

	dagPriorityTest        = 0
	dagPriorityLink        = 1
	dagPriorityPrepareLink = 2
	dagPriorityPackageBase = 1_000_000
)

type buildDAGNode struct {
	name         string
	dependencies []int
	priority     int
	class        buildDAGClass
	coordinator  bool
	run          func() error
	complete     func(error)
	skip         func() bool
	skipped      func()
	blocked      func()
}

type buildDAGNodeResult struct {
	err     error
	blocked bool
}

type buildDAGCompletion struct {
	index int
	err   error
}

type buildDAGState struct {
	remaining  int
	dependents []int
	blocked    bool
}

type buildDAGReadyItem struct {
	index    int
	priority int
	order    int
}

func (item buildDAGReadyItem) less(other buildDAGReadyItem) bool {
	return item.priority < other.priority ||
		item.priority == other.priority && item.order < other.order
}

type buildDAGReadyQueue []buildDAGReadyItem

func (queue buildDAGReadyQueue) Len() int { return len(queue) }
func (queue buildDAGReadyQueue) Less(i, j int) bool {
	return queue[i].less(queue[j])
}
func (queue buildDAGReadyQueue) Swap(i, j int) { queue[i], queue[j] = queue[j], queue[i] }
func (queue *buildDAGReadyQueue) Push(value any) {
	*queue = append(*queue, value.(buildDAGReadyItem))
}
func (queue *buildDAGReadyQueue) Pop() any {
	old := *queue
	last := len(old) - 1
	item := old[last]
	*queue = old[:last]
	return item
}

func runBuildDAG(nodes []buildDAGNode, parallelism int, sequentialTests bool) ([]buildDAGNodeResult, error) {
	results := make([]buildDAGNodeResult, len(nodes))
	states := make([]buildDAGState, len(nodes))
	for index, node := range nodes {
		states[index].remaining = len(node.dependencies)
		for _, dependency := range node.dependencies {
			if dependency < 0 || dependency >= len(nodes) || dependency == index {
				return nil, fmt.Errorf("DAG node %d (%s) has invalid dependency %d", index, node.name, dependency)
			}
			states[dependency].dependents = append(states[dependency].dependents, index)
		}
	}

	readyNonTests := make(buildDAGReadyQueue, 0, len(nodes))
	readyTests := make(buildDAGReadyQueue, 0, len(nodes))
	readyBlocked := make(buildDAGReadyQueue, 0)
	readyOrder := 0
	pushReady := func(index int) {
		item := buildDAGReadyItem{index: index, priority: nodes[index].priority, order: readyOrder}
		readyOrder++
		if states[index].blocked {
			heap.Push(&readyBlocked, item)
		} else if nodes[index].class == dagTest {
			heap.Push(&readyTests, item)
		} else {
			heap.Push(&readyNonTests, item)
		}
	}
	for index := range nodes {
		if states[index].remaining == 0 {
			pushReady(index)
		}
	}
	workers := max(1, parallelism)
	testLimit := workers
	if sequentialTests {
		testLimit = 1
	}
	completions := make(chan buildDAGCompletion, workers)
	active := 0
	activeTests := 0
	finished := 0
	unfinishedNonTests := 0
	for _, node := range nodes {
		if node.class != dagTest {
			unfinishedNonTests++
		}
	}

	var finish func(int, error, bool, bool)
	finish = func(index int, err error, blocked, skipped bool) {
		state := &states[index]
		results[index] = buildDAGNodeResult{err: err, blocked: blocked}
		finished++
		if nodes[index].class != dagTest {
			unfinishedNonTests--
		}
		if nodes[index].complete != nil && !blocked && !skipped {
			nodes[index].complete(err)
		}
		if skipped && nodes[index].skipped != nil {
			nodes[index].skipped()
		}
		if blocked && nodes[index].blocked != nil {
			nodes[index].blocked()
		}
		failed := err != nil || blocked
		for _, dependent := range state.dependents {
			next := &states[dependent]
			next.remaining--
			next.blocked = next.blocked || failed
			if next.remaining == 0 {
				pushReady(dependent)
			}
		}
	}

	popReady := func() (int, bool) {
		if readyBlocked.Len() > 0 {
			return heap.Pop(&readyBlocked).(buildDAGReadyItem).index, true
		}
		testReady := readyTests.Len() > 0 && activeTests < testLimit
		// Test execution overlaps production, but package/link work is the
		// long pole in full CI. After starting one test, prefer every ready
		// producer; if only active producers remain, tests may fill otherwise
		// idle slots instead of forming a serial build tail.
		if testReady && unfinishedNonTests > 0 &&
			(workers == 1 || activeTests >= 1 && readyNonTests.Len() > 0) {
			testReady = false
		}
		if !testReady && readyNonTests.Len() == 0 {
			return 0, false
		}
		if testReady && (readyNonTests.Len() == 0 || readyTests[0].less(readyNonTests[0])) {
			return heap.Pop(&readyTests).(buildDAGReadyItem).index, true
		}
		return heap.Pop(&readyNonTests).(buildDAGReadyItem).index, true
	}

	runWorker := func(index int) {
		go func() {
			completions <- buildDAGCompletion{index: index, err: runBuildDAGNode(index, nodes[index])}
		}()
	}
	handleCompletion := func(completion buildDAGCompletion) {
		active--
		if nodes[completion.index].class == dagTest {
			activeTests--
		}
		finish(completion.index, completion.err, false, false)
	}
	drainCompletions := func() bool {
		drained := false
		for active != 0 {
			select {
			case completion := <-completions:
				handleCompletion(completion)
				drained = true
			default:
				return drained
			}
		}
		return drained
	}

	for finished < len(nodes) {
		progress := drainCompletions()
		for active < workers {
			if drainCompletions() {
				progress = true
			}
			index, ok := popReady()
			if !ok {
				break
			}
			state := &states[index]
			node := nodes[index]
			if state.blocked {
				finish(index, nil, true, false)
				progress = true
				continue
			}
			if node.skip != nil && node.skip() {
				finish(index, nil, false, true)
				progress = true
				continue
			}
			if node.coordinator {
				finish(index, runBuildDAGNode(index, node), false, false)
				progress = true
				continue
			}
			active++
			if node.class == dagTest {
				activeTests++
			}
			runWorker(index)
			progress = true
		}
		if active == 0 {
			if finished == len(nodes) {
				break
			}
			if !progress {
				return nil, fmt.Errorf("build DAG stalled with %d/%d nodes complete", finished, len(nodes))
			}
			continue
		}
		handleCompletion(<-completions)
	}
	return results, nil
}

func runBuildDAGNode(index int, node buildDAGNode) (err error) {
	defer func() {
		if value := recover(); value != nil {
			if recovered, ok := value.(error); ok {
				err = recovered
			} else {
				err = fmt.Errorf("DAG node %d (%s) panicked: %v", index, node.name, value)
			}
		}
	}()
	if node.run == nil {
		return nil
	}
	return node.run()
}

type testLinkResult struct {
	program *testProgram
	err     error
}

type nativeTestDAGResult struct {
	links []testLinkResult
	tests testRunResult
}

// The native test DAG is deliberately split at LLVM ownership boundaries:
//
//	package backend+.a -> package link snapshot+Program disposal
//	-> coordinator root link plan -> isolated entry .o
//	-> link/finalize -> test run
//
// All non-coordinator nodes share the single -p budget. A link-plan node is
// coordinator-only because patched packages may still belong to the shared
// coordinator Program, which LLVM does not permit workers to read concurrently.
// Isolated package workers publish LLVM-free snapshots and dispose their own
// Programs first. Root plans consume those snapshots, so entry generation can
// use a fresh Program and downstream link/run work retains no package Context.
func canUseNativeTestDAG(ctx *context, count int) bool {
	return count > 1 && ctx != nil && ctx.buildConf != nil &&
		ctx.mode == ModeTest && ctx.buildConf.Target == "" &&
		ctx.buildConf.BuildMode == BuildModeExe && !ctx.buildConf.deadcodeDropEnabled()
}

func prepareNativeTestPackageTasks(ctx *context, pkgs []*aPackage, verbose bool) ([]*packageBuildTask, error) {
	// Resolve the lazy Plan 9 policy before isolated package workers start.
	_ = ctx.plan9asmEnabled("")
	var normalTasks, runtimeTasks []*packageBuildTask
	for _, pkg := range pkgs {
		task := newPackageBuildTask(pkg)
		if task.isRuntime() {
			runtimeTasks = append(runtimeTasks, task)
		} else {
			normalTasks = append(normalTasks, task)
		}
	}
	var parallel []*packageBuildTask
	for _, tasks := range [][]*packageBuildTask{normalTasks, runtimeTasks} {
		indexes, err := preparePackageGroup(ctx, tasks, verbose)
		if err != nil {
			return nil, err
		}
		for _, index := range indexes {
			parallel = append(parallel, tasks[index])
		}
	}
	return parallel, nil
}

func runNativeTestDAG(ctx *context, allPkgs []*aPackage, roots []*packages.Package, conf *Config, verbose bool) (nativeTestDAGResult, error) {
	result := nativeTestDAGResult{links: make([]testLinkResult, len(roots))}
	defer func() {
		for _, pkg := range allPkgs {
			if pkg != nil {
				pkg.linkSnapshot = nil
			}
		}
	}()
	packageTasks, err := prepareNativeTestPackageTasks(ctx, allPkgs, verbose)
	if err != nil {
		return result, err
	}

	nodes := make([]buildDAGNode, 0, len(packageTasks)+len(roots)*4)
	packageNodes := make(map[*aPackage]int, len(packageTasks))
	for _, task := range packageTasks {
		task := task
		index := len(nodes)
		packageNodes[task.pkg] = index
		nodes = append(nodes, buildDAGNode{
			name:     "package " + task.pkg.PkgPath,
			priority: dagPriorityPackageBase,
			class:    dagPackage,
			run: func() error {
				if err := tracePackageBuild(ctx, task, verbose, true, true); err != nil {
					return err
				}
				span := ctx.buildTrace.startWorker("link-snapshot", task.pkg.PkgPath)
				ctx.snapshotBackendPackage(task.pkg)
				ctx.disposeBackendPackage(task.pkg)
				span.done()
				return nil
			},
		})
	}

	prepared := make([]*initialPackageLink, len(roots))
	defer func() {
		// Scheduler cancellation and fail-fast may leave a planned test without
		// a run node. Successful runs have already removed the same directories.
		for _, link := range prepared {
			if link != nil && link.outFmts.tempDir != "" {
				removeOutFmts(link.outFmts)
			}
		}
	}()
	runResults := make([]testProgramResult, len(roots))
	stdout := &lockedTestWriter{w: os.Stdout}
	stderr := &lockedTestWriter{w: os.Stderr}
	report := testResultReporter(len(roots), stdout)
	packageFanout := make([]int, len(packageTasks))
	packageConsumers := make(map[*aPackage]int, len(packageTasks))
	for rootIndex, root := range roots {
		rootIndex, root := rootIndex, root
		dependencies := make([]int, 0)
		consumerPackages := make([]*aPackage, 0)
		seen := make(map[int]bool)
		for _, pkg := range linkedPackageClosure(ctx, root, allPkgs) {
			if index, ok := packageNodes[pkg]; ok && !seen[index] {
				seen[index] = true
				packageFanout[index]++
				packageConsumers[pkg]++
				dependencies = append(dependencies, index)
				consumerPackages = append(consumerPackages, pkg)
			}
		}
		planIndex := len(nodes)
		nodes = append(nodes, buildDAGNode{
			name:         "plan link " + root.PkgPath,
			dependencies: dependencies,
			priority:     dagPriorityPrepareLink,
			class:        dagLink,
			coordinator:  true,
			run: func() error {
				link, err := planInitialPackageLink(ctx, root, allPkgs, conf, false)
				prepared[rootIndex] = link
				return err
			},
			complete: func(err error) {
				for _, pkg := range consumerPackages {
					packageConsumers[pkg]--
					if packageConsumers[pkg] == 0 {
						pkg.linkSnapshot = nil
						ctx.disposeBackendPackage(pkg)
					}
				}
				if err != nil {
					result.links[rootIndex].err = err
				}
			},
		})
		entryIndex := len(nodes)
		nodes = append(nodes, buildDAGNode{
			name:         "entry object " + root.PkgPath,
			dependencies: []int{planIndex},
			priority:     dagPriorityLink,
			class:        dagLink,
			run: func() error {
				return buildInitialPackageEntry(ctx, prepared[rootIndex], verbose, true)
			},
			complete: func(err error) {
				if err != nil {
					result.links[rootIndex].err = err
					if link := prepared[rootIndex]; link != nil {
						removeOutFmts(link.outFmts)
						prepared[rootIndex] = nil
					}
				}
			},
		})
		linkIndex := len(nodes)
		nodes = append(nodes, buildDAGNode{
			name:         "link " + root.PkgPath,
			dependencies: []int{entryIndex},
			priority:     dagPriorityLink,
			class:        dagLink,
			run: func() error {
				program, err := executeInitialPackageLink(ctx, prepared[rootIndex], verbose, true)
				result.links[rootIndex] = testLinkResult{program: program, err: err}
				return err
			},
		})
		if conf.CompileOnly {
			continue
		}
		nodes = append(nodes, buildDAGNode{
			name:         "test " + root.PkgPath,
			dependencies: []int{linkIndex},
			priority:     dagPriorityTest,
			class:        dagTest,
			skip: func() bool {
				return conf.TestFailFast && result.tests.failed
			},
			skipped: func() {
				result.tests.skipped++
				report(rootIndex, nil)
			},
			blocked: func() {
				result.tests.failed = true
				fmt.Fprintf(os.Stderr, "FAIL\t%s [build failed]\n", strings.TrimSuffix(root.PkgPath, ".test"))
				report(rootIndex, nil)
			},
			run: func() error {
				linked := result.links[rootIndex].program
				if linked == nil {
					err := fmt.Errorf("link %s completed without a test program", root.PkgPath)
					runResults[rootIndex] = testProgramResult{
						program: testProgram{pkgName: strings.TrimSuffix(root.PkgPath, ".test")},
						output:  []byte(err.Error() + "\n"),
						err:     err,
					}
					return err
				}
				program := *linked
				span := ctx.buildTrace.startWorker("test", program.pkgName)
				defer span.done()
				runResults[rootIndex] = runTestProgram(program, conf, len(roots), stdout, stderr,
					func(output io.Writer) error { return runNativeTest(ctx.commands, program, conf, output, output) })
				return runResults[rootIndex].err
			},
			complete: func(err error) {
				report(rootIndex, &runResults[rootIndex])
				if err != nil {
					result.tests.failed = true
				}
			},
		})
	}
	// Package backends are independent after shared frontend preparation. Favor
	// the packages on the most root closures so shared critical-path work makes
	// the largest number of prepare-link nodes ready first.
	for index, fanout := range packageFanout {
		nodes[index].priority = dagPriorityPackageBase - fanout
	}

	nodeResults, err := runBuildDAG(nodes, conf.parallelism(), conf.TestRunSequential)
	if err != nil {
		return result, err
	}
	var buildErrs []error
	for _, task := range packageTasks {
		index := packageNodes[task.pkg]
		if nodeResults[index].err != nil {
			buildErrs = append(buildErrs, fmt.Errorf("%s: %w", task.pkg.PkgPath, nodeResults[index].err))
		}
	}
	return result, errors.Join(buildErrs...)
}
