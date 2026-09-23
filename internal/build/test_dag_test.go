//go:build !llgo

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
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestCanUseNativeTestDAG(t *testing.T) {
	base := &context{mode: ModeTest, buildConf: &Config{BuildMode: BuildModeExe}}
	if !canUseNativeTestDAG(base, 2) {
		t.Fatal("two native test roots did not enable the DAG")
	}
	for _, test := range []struct {
		name  string
		ctx   *context
		count int
	}{
		{name: "one link", ctx: base, count: 1},
		{name: "build mode", ctx: &context{mode: ModeBuild, buildConf: &Config{BuildMode: BuildModeExe}}, count: 2},
		{name: "embedded", ctx: &context{mode: ModeTest, buildConf: &Config{BuildMode: BuildModeExe, Target: "rp2040"}}, count: 2},
		{name: "archive", ctx: &context{mode: ModeTest, buildConf: &Config{BuildMode: BuildModeCArchive}}, count: 2},
		{name: "nil context", count: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			if canUseNativeTestDAG(test.ctx, test.count) {
				t.Fatal("DAG unexpectedly enabled")
			}
		})
	}
}

func TestRunBuildDAGPipelinesReadyBranches(t *testing.T) {
	var mu sync.Mutex
	var events []string
	record := func(event string) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}
	firstPackageDone := make(chan struct{})
	releaseSecondPackage := make(chan struct{})
	nodes := []buildDAGNode{
		{name: "first package", priority: 3, class: dagPackage, run: func() error {
			record("package-1")
			close(firstPackageDone)
			return nil
		}},
		{name: "second package", priority: 3, class: dagPackage, run: func() error {
			<-firstPackageDone
			<-releaseSecondPackage
			record("package-2")
			return nil
		}},
		{name: "prepare", dependencies: []int{0}, priority: 2, class: dagLink, coordinator: true, run: func() error {
			record("prepare")
			return nil
		}},
		{name: "link", dependencies: []int{2}, priority: 1, class: dagLink, run: func() error {
			record("link")
			return nil
		}},
		{name: "test", dependencies: []int{3}, priority: 0, class: dagTest, run: func() error {
			record("test")
			close(releaseSecondPackage)
			return nil
		}},
	}
	results, err := runBuildDAG(nodes, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	for index, result := range results {
		if result.err != nil || result.blocked {
			t.Fatalf("node %d result = %+v", index, result)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	positions := make(map[string]int, len(events))
	for index, event := range events {
		positions[event] = index
	}
	if !(positions["package-1"] < positions["prepare"] &&
		positions["prepare"] < positions["link"] &&
		positions["link"] < positions["test"] &&
		positions["test"] < positions["package-2"]) {
		t.Fatalf("DAG did not pipeline the ready branch: %v", events)
	}
}

func TestRunBuildDAGBoundsWorkersAndPropagatesFailure(t *testing.T) {
	wantErr := errors.New("package failed")
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	var active atomic.Int32
	var maximum atomic.Int32
	worker := func(err error) func() error {
		return func() error {
			current := active.Add(1)
			for {
				observed := maximum.Load()
				if current <= observed || maximum.CompareAndSwap(observed, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			active.Add(-1)
			return err
		}
	}
	var dependentRan atomic.Bool
	var blockedCalled atomic.Bool
	nodes := []buildDAGNode{
		{name: "failure", class: dagPackage, run: worker(wantErr)},
		{name: "success one", class: dagPackage, run: worker(nil)},
		{name: "success two", class: dagPackage, run: worker(nil)},
		{name: "blocked", dependencies: []int{0}, class: dagLink, run: func() error {
			dependentRan.Store(true)
			return nil
		}, blocked: func() { blockedCalled.Store(true) }},
	}
	done := make(chan []buildDAGNodeResult, 1)
	go func() {
		results, err := runBuildDAG(nodes, 2, false)
		if err != nil {
			t.Errorf("runBuildDAG: %v", err)
		}
		done <- results
	}()
	<-started
	<-started
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrent workers = %d, want 2", got)
	}
	close(release)
	results := <-done
	if !errors.Is(results[0].err, wantErr) {
		t.Fatalf("failure result = %v, want %v", results[0].err, wantErr)
	}
	if !results[3].blocked || dependentRan.Load() || !blockedCalled.Load() {
		t.Fatalf("dependent result = %+v, ran = %v, blocked callback = %v",
			results[3], dependentRan.Load(), blockedCalled.Load())
	}
}

func TestRunBuildDAGFillsCapacityAtProducerTail(t *testing.T) {
	producerRelease := make(chan struct{})
	testRelease := make(chan struct{})
	defer func() {
		for _, release := range []chan struct{}{producerRelease, testRelease} {
			select {
			case <-release:
			default:
				close(release)
			}
		}
	}()
	started := make(chan buildDAGClass, 3)
	nodes := []buildDAGNode{
		{name: "producer", class: dagPackage, run: func() error {
			started <- dagPackage
			<-producerRelease
			return nil
		}},
		{name: "test one", class: dagTest, priority: dagPriorityTest, run: func() error {
			started <- dagTest
			<-testRelease
			return nil
		}},
		{name: "test two", class: dagTest, priority: dagPriorityTest, run: func() error {
			started <- dagTest
			<-testRelease
			return nil
		}},
	}
	done := make(chan error, 1)
	go func() {
		_, err := runBuildDAG(nodes, 3, false)
		done <- err
	}()

	counts := map[buildDAGClass]int{}
	for range 3 {
		select {
		case class := <-started:
			counts[class]++
		case <-time.After(time.Second):
			t.Fatalf("workers remained idle at producer tail; started classes = %v", counts)
		}
	}
	if counts[dagPackage] != 1 || counts[dagTest] != 2 {
		t.Fatalf("initial node classes = %v, want one producer and two tests", counts)
	}
	close(producerRelease)
	close(testRelease)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunBuildDAGPrefersReadyProducersAfterOneTest(t *testing.T) {
	producerRelease := make(chan struct{})
	testRelease := make(chan struct{})
	defer func() {
		for _, release := range []chan struct{}{producerRelease, testRelease} {
			select {
			case <-release:
			default:
				close(release)
			}
		}
	}()
	started := make(chan buildDAGClass, 4)
	worker := func(class buildDAGClass, release <-chan struct{}) func() error {
		return func() error {
			started <- class
			<-release
			return nil
		}
	}
	nodes := []buildDAGNode{
		{name: "producer one", priority: dagPriorityPackageBase, class: dagPackage, run: worker(dagPackage, producerRelease)},
		{name: "producer two", priority: dagPriorityPackageBase, class: dagPackage, run: worker(dagPackage, producerRelease)},
		{name: "test one", priority: dagPriorityTest, class: dagTest, run: worker(dagTest, testRelease)},
		{name: "test two", priority: dagPriorityTest, class: dagTest, run: worker(dagTest, testRelease)},
	}
	done := make(chan error, 1)
	go func() {
		_, err := runBuildDAG(nodes, 3, false)
		done <- err
	}()

	counts := map[buildDAGClass]int{}
	for range 3 {
		select {
		case class := <-started:
			counts[class]++
		case <-time.After(time.Second):
			t.Fatalf("workers remained idle; started classes = %v", counts)
		}
	}
	if counts[dagPackage] != 2 || counts[dagTest] != 1 {
		t.Fatalf("initial node classes = %v, want two producers and one test", counts)
	}
	close(producerRelease)
	close(testRelease)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunBuildDAGHonorsClassLimitAndDynamicSkip(t *testing.T) {
	var activeTests atomic.Int32
	var maximumTests atomic.Int32
	var failed atomic.Bool
	var skipped atomic.Int32
	nodes := make([]buildDAGNode, 4)
	for index := range nodes {
		index := index
		nodes[index] = buildDAGNode{
			name:     "test",
			priority: 0,
			class:    dagTest,
			skip:     failed.Load,
			skipped:  func() { skipped.Add(1) },
			run: func() error {
				current := activeTests.Add(1)
				if current > maximumTests.Load() {
					maximumTests.Store(current)
				}
				activeTests.Add(-1)
				if index == 0 {
					failed.Store(true)
				}
				return nil
			},
		}
	}
	_, err := runBuildDAG(nodes, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := maximumTests.Load(); got != 1 {
		t.Fatalf("maximum concurrent tests = %d, want 1", got)
	}
	if got := skipped.Load(); got != 3 {
		t.Fatalf("skipped tests = %d, want 3", got)
	}
}

func TestRunBuildDAGRejectsInvalidGraphs(t *testing.T) {
	t.Run("invalid dependency", func(t *testing.T) {
		_, err := runBuildDAG([]buildDAGNode{{name: "invalid", dependencies: []int{1}}}, 1, false)
		if err == nil || !strings.Contains(err.Error(), "invalid dependency 1") {
			t.Fatalf("invalid dependency error = %v", err)
		}
	})

	t.Run("cycle", func(t *testing.T) {
		_, err := runBuildDAG([]buildDAGNode{
			{name: "first", dependencies: []int{1}},
			{name: "second", dependencies: []int{0}},
		}, 1, false)
		if err == nil || !strings.Contains(err.Error(), "stalled with 0/2 nodes complete") {
			t.Fatalf("cycle error = %v", err)
		}
	})

	t.Run("progress before cycle", func(t *testing.T) {
		_, err := runBuildDAG([]buildDAGNode{
			{name: "independent", coordinator: true},
			{name: "first", dependencies: []int{2}},
			{name: "second", dependencies: []int{1}},
		}, 1, false)
		if err == nil || !strings.Contains(err.Error(), "stalled with 1/3 nodes complete") {
			t.Fatalf("partially completed cycle error = %v", err)
		}
	})
}

func TestRunBuildDAGRecoversNodePanics(t *testing.T) {
	wantErr := errors.New("error panic")
	results, err := runBuildDAG([]buildDAGNode{
		{name: "nil", coordinator: true},
		{name: "error", coordinator: true, run: func() error { panic(wantErr) }},
		{name: "value", coordinator: true, run: func() error { panic("value panic") }},
	}, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].err != nil {
		t.Fatalf("nil node error = %v", results[0].err)
	}
	if !errors.Is(results[1].err, wantErr) {
		t.Fatalf("error panic result = %v, want %v", results[1].err, wantErr)
	}
	if got := results[2].err; got == nil || !strings.Contains(got.Error(), "node 2 (value) panicked: value panic") {
		t.Fatalf("value panic result = %v", got)
	}
}

func TestSeparatedLinkPhaseValidation(t *testing.T) {
	if _, err := buildMainLink(nil, nil, nil, "", false); err == nil {
		t.Fatal("buildMainLink accepted a nil preparation")
	}
	if err := executeMainLink(nil, nil, false); err == nil {
		t.Fatal("executeMainLink accepted a nil plan")
	}
}

func TestExecuteInitialPackageLinkPropagatesStagingFailure(t *testing.T) {
	output := filepath.Join(t.TempDir(), "missing", "test.wasm")
	conf := &Config{BuildMode: BuildModeExe}
	ctx := &context{buildConf: conf}
	ctx.crossCompile.WasmPostLink.Asyncify = true
	link := &initialPackageLink{
		pkg:     &packages.Package{PkgPath: "example.com/test.test"},
		conf:    conf,
		outFmts: &OutFmtDetails{Out: output},
		plan:    &mainLinkPlan{outputPath: output},
	}
	if _, err := executeInitialPackageLink(ctx, link, false, false); err == nil {
		t.Fatal("executeInitialPackageLink succeeded with a missing staging directory")
	}
	if link.plan != nil {
		t.Fatal("failed link retained its consumed plan")
	}
}

func TestRunNativeTestDAGPropagatesPackageFailures(t *testing.T) {
	t.Run("preparation", func(t *testing.T) {
		fset, pkg := invalidEmbedPackage(t)
		conf := &Config{Mode: ModeTest}
		prog := llssa.NewProgram(nil)
		defer prog.Dispose()
		ctx := &context{
			prog:      prog,
			conf:      &packages.Config{Fset: fset},
			mode:      ModeGen,
			buildConf: conf,
		}
		_, err := runNativeTestDAG(ctx, []*aPackage{pkg}, nil, conf, false)
		if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
			t.Fatalf("preparation error = %v", err)
		}
	})

	t.Run("package node", func(t *testing.T) {
		fset, pkg := invalidEmbedPackage(t)
		coordinator := llssa.NewProgram(&llssa.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
		defer coordinator.Dispose()
		conf := &Config{Mode: ModeTest, BuildMode: BuildModeExe, Goos: runtime.GOOS, Goarch: runtime.GOARCH}
		ctx := &context{
			conf:      &packages.Config{Fset: fset},
			prog:      coordinator,
			mode:      ModeTest,
			buildConf: conf,
		}
		_, err := runNativeTestDAG(ctx, []*aPackage{pkg}, nil, conf, false)
		if err == nil || !strings.Contains(err.Error(), "only allowed in Go files that import") {
			t.Fatalf("package node error = %v", err)
		}
	})
}

func TestRunNativeTestDAGPropagatesPlanFailure(t *testing.T) {
	root := &packages.Package{
		ID:      "example.com/cycle.test",
		PkgPath: "example.com/cycle.test",
		Name:    "main",
		Imports: make(map[string]*packages.Package),
	}
	root.Imports["self"] = root
	conf := &Config{Mode: ModeTest, BuildMode: BuildModeExe}
	ctx := &context{
		mode:      ModeTest,
		buildConf: conf,
		commands:  commandEnv{dir: t.TempDir()},
		pkgs:      make(map[*packages.Package]Package),
		pkgByID:   make(map[string]Package),
	}
	readStderr := captureStderr(t)
	result, err := runNativeTestDAG(ctx, nil, []*packages.Package{root}, conf, false)
	stderr := readStderr()
	if err != nil {
		t.Fatal(err)
	}
	if result.links[0].err == nil || !strings.Contains(result.links[0].err.Error(), "contains a cycle") {
		t.Fatalf("link result error = %v", result.links[0].err)
	}
	if !result.tests.failed || !strings.Contains(stderr, "FAIL\texample.com/cycle [build failed]") {
		t.Fatalf("test result = %+v; stderr = %q", result.tests, stderr)
	}

	conf.CompileOnly = true
	result, err = runNativeTestDAG(ctx, nil, []*packages.Package{root}, conf, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.links[0].err == nil || result.tests.failed || result.tests.skipped != 0 {
		t.Fatalf("compile-only result = %+v", result)
	}
}

func TestRunNativeTestDAGPropagatesEntryFailure(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	root := &packages.Package{
		ID:      "example.com/entry.test",
		PkgPath: "example.com/entry.test",
		Name:    "main",
		Imports: make(map[string]*packages.Package),
	}
	conf := &Config{
		Mode:      ModeTest,
		BuildMode: BuildModeExe,
		Goos:      runtime.GOOS,
		Goarch:    runtime.GOARCH,
	}
	ctx := &context{
		prog:      prog,
		mode:      ModeTest,
		buildConf: conf,
		commands:  commandEnv{dir: t.TempDir()},
		pkgs:      make(map[*packages.Package]Package),
		pkgByID:   make(map[string]Package),
	}
	ctx.crossCompile.ExtraFiles = []string{"missing-entry-file.c"}

	readStderr := captureStderr(t)
	result, err := runNativeTestDAG(ctx, nil, []*packages.Package{root}, conf, false)
	stderr := readStderr()
	if err != nil {
		t.Fatal(err)
	}
	if result.links[0].err == nil || !strings.Contains(result.links[0].err.Error(), "extra file not found") {
		t.Fatalf("entry result error = %v", result.links[0].err)
	}
	if !result.tests.failed || !strings.Contains(stderr, "FAIL\texample.com/entry [build failed]") {
		t.Fatalf("test result = %+v; stderr = %q", result.tests, stderr)
	}
}
