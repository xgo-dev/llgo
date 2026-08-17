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

package cl

import (
	"reflect"
	"sync"
	"testing"

	gossa "golang.org/x/tools/go/ssa"
)

func TestCallerTrackingPrecomputeSupportsConcurrentReads(t *testing.T) {
	var nilTracking *CallerTracking
	nilTracking.Precompute(nil)
	dep, root := buildCallerFrameSSAProgram(t,
		"example.com/dep", `package dep
import "runtime"
func Where() { runtime.Caller(0) }
func Recovering() any { return recover() }
func Inspect() { recover(); runtime.Caller(0) }
func Owner() { defer Inspect(); Leaf() }
func Leaf() {}
func Plain() {}
`,
		"example.com/root", `package root
import "example.com/dep"
func Logs() { dep.Where() }
func CrossOwner() { defer dep.Inspect(); CrossLeaf() }
func CrossLeaf() {}
`)
	lazyCrossPackageSites := recoverPanicSiteFuncSet(NewCallerTracking(), root)
	if !lazyCrossPackageSites[root.Func("CrossOwner")] || !lazyCrossPackageSites[root.Func("CrossLeaf")] {
		t.Fatal("lazy analysis lost the subtree below a cross-package recovering defer")
	}
	tracking := NewCallerTracking()
	tracking.Precompute([]*gossa.Package{dep, root})
	if !runtimeCallerBaseSet(tracking, dep)[dep.Func("Where")] {
		t.Fatal("precomputed base set lost runtime caller function")
	}
	if !runtimeCallerFuncSet(tracking, root)[root.Func("Logs")] {
		t.Fatal("precomputed extended set lost cross-package caller")
	}
	panicSites := recoverPanicSiteFuncSet(tracking, dep)
	if !panicSites[dep.Func("Owner")] || !panicSites[dep.Func("Leaf")] {
		t.Fatal("precomputed recover panic-site set lost synchronous callees")
	}
	if panicSites[dep.Func("Where")] {
		t.Fatal("ordinary caller-tracked function entered recover panic-site set")
	}
	rootPanicSites := recoverPanicSiteFuncSet(tracking, root)
	if !rootPanicSites[root.Func("CrossOwner")] || !rootPanicSites[root.Func("CrossLeaf")] {
		t.Fatal("precomputed analysis lost the subtree below a cross-package recovering defer")
	}
	recovering := dep.Func("Recovering")
	plain := dep.Func("Plain")
	if needs, ok := tracking.recover.scopes[recovering]; !ok || !needs {
		t.Fatal("precompute did not cache the dependency's recover scope")
	}
	if needs, ok := tracking.recover.scopes[plain]; !ok || needs {
		t.Fatal("precompute did not cache the dependency's plain function")
	}

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !runtimeCallerBaseSet(tracking, dep)[dep.Func("Where")] ||
				!runtimeCallerFuncSet(tracking, root)[root.Func("Logs")] ||
				!recoverPanicSiteFuncSet(tracking, dep)[dep.Func("Leaf")] ||
				!recoverPanicSiteFuncSet(tracking, root)[root.Func("CrossLeaf")] ||
				!tracking.recover.needsRecoverScope(recovering) ||
				tracking.recover.needsRecoverScope(plain) {
				t.Error("concurrent read lost precomputed caller tracking data")
			}
		}()
	}
	wg.Wait()
}

func TestCallerTrackingPrecomputeMatchesLazyAnalysis(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t,
		"example.com/dep", `package dep
import "runtime"
func Where() { runtime.Caller(0) }
func Quiet() {}
`,
		"example.com/root", `package root
import "example.com/dep"
func Logs() { dep.Where() }
func Plain() { dep.Quiet() }
`)
	pkgs := []*gossa.Package{root, dep}
	lazy := NewCallerTracking()
	for _, pkg := range pkgs {
		runtimeCallerBaseSet(lazy, pkg)
	}
	for _, pkg := range pkgs {
		runtimeCallerFuncSet(lazy, pkg)
	}
	precomputed := NewCallerTracking()
	precomputed.Precompute(pkgs)
	for _, pkg := range pkgs {
		if got, want := precomputed.base[pkg], lazy.base[pkg]; !reflect.DeepEqual(got, want) {
			t.Fatalf("precomputed base set for %s differs from lazy set", pkg.Pkg.Path())
		}
		if got, want := precomputed.extended[pkg], lazy.extended[pkg]; !reflect.DeepEqual(got, want) {
			t.Fatalf("precomputed extended set for %s differs from lazy set", pkg.Pkg.Path())
		}
	}
}

func TestCallerTrackingPrecomputeRejectsLatePackages(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t,
		"example.com/dep", `package dep
func Where() {}
`,
		"example.com/root", `package root
import "example.com/dep"
func Logs() { dep.Where() }
`)
	tests := []struct {
		name   string
		lookup func(*CallerTracking, *gossa.Package)
	}{
		{name: "base", lookup: func(c *CallerTracking, pkg *gossa.Package) {
			runtimeCallerBaseSet(c, pkg)
		}},
		{name: "extended", lookup: func(c *CallerTracking, pkg *gossa.Package) {
			runtimeCallerFuncSet(c, pkg)
		}},
		{name: "recover-panic-sites", lookup: func(c *CallerTracking, pkg *gossa.Package) {
			recoverPanicSiteFuncSet(c, pkg)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracking := NewCallerTracking()
			tracking.Precompute([]*gossa.Package{dep})
			defer func() {
				if recover() == nil {
					t.Fatal("late caller-tracking lookup did not panic")
				}
			}()
			test.lookup(tracking, root)
		})
	}
}
