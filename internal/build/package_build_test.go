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
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/packages"
)

func TestPackageBuildSpecAndResult(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		PkgPath: "example.com/p",
		GoFiles: []string{"p.go"},
		Types:   types.NewPackage("example.com/p", "p"),
	}, NeedRt: true, NeedPyInit: true, CacheHit: true, ArchiveFile: "p.a"}
	spec := newPackageBuildSpec(pkg)
	if spec.isDeclOnly() || spec.isLinkOnly() || !spec.hasSource() || spec.runtime || !spec.needsRuntimeSignals() {
		t.Fatalf("unexpected normal package spec: %+v", spec)
	}
	result := packageBuildResultFor(spec)
	if !result.cacheHit || result.archiveFile != "p.a" || !result.needRuntime || !result.needPyInit {
		t.Fatalf("unexpected package result: %+v", result)
	}
}

func TestPackageBuildPlanReadyLevels(t *testing.T) {
	leaf := planPackage("leaf")
	left := planPackage("left", leaf.Package)
	right := planPackage("right", leaf.Package)
	root := planPackage("root", left.Package, right.Package)
	plan, err := newPackageBuildPlan([]*aPackage{root, right, left, leaf})
	if err != nil {
		t.Fatal(err)
	}
	var levels [][]string
	for _, level := range plan.levels {
		ids := make([]string, len(level))
		for i, spec := range level {
			ids[i] = spec.pkg.ID
		}
		levels = append(levels, ids)
	}
	if want := [][]string{{"leaf"}, {"left", "right"}, {"root"}}; !reflect.DeepEqual(levels, want) {
		t.Fatalf("ready levels = %v, want %v", levels, want)
	}
}

func TestPackageBuildPlanExcludesAlternateDependencies(t *testing.T) {
	baseDep := planPackage("base")
	altDep := planPackage("alt")
	pkg := planPackage("pkg", baseDep.Package)
	pkg.AltPkg = &packages.Cached{Package: &packages.Package{ID: "patch/pkg", Imports: map[string]*packages.Package{"alt": altDep.Package}}}
	plan, err := newPackageBuildPlan([]*aPackage{pkg, baseDep, altDep})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := plan.deps["pkg"], []string{"base"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("plan dependencies = %v, want %v", got, want)
	}
}

func TestPackageBuildPlanRejectsCycles(t *testing.T) {
	a := planPackage("a")
	b := planPackage("b")
	a.Imports = map[string]*packages.Package{"b": b.Package}
	b.Imports = map[string]*packages.Package{"a": a.Package}
	if _, err := newPackageBuildPlan([]*aPackage{a, b}); err == nil {
		t.Fatal("newPackageBuildPlan succeeded, want cycle error")
	} else if !strings.Contains(err.Error(), "a, b") {
		t.Fatalf("cycle error = %q, want package IDs", err)
	}
}

func planPackage(id string, imports ...*packages.Package) *aPackage {
	depMap := make(map[string]*packages.Package, len(imports))
	for _, dep := range imports {
		depMap[dep.ID] = dep
	}
	return &aPackage{Package: &packages.Package{
		ID:      id,
		PkgPath: "example.com/" + id,
		Imports: depMap,
		Types:   types.NewPackage("example.com/"+id, id),
	}}
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

func TestPreflightFingerprintsSkippedPackage(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{
		ID:      "unsafe",
		PkgPath: "unsafe",
		Types:   types.Unsafe,
	}}
	ctx := &context{
		conf:             &packages.Config{},
		buildConf:        &Config{Goos: "linux", Goarch: "amd64", ForceRebuild: true},
		built:            make(map[string]none),
		llvmVersionReady: true,
	}
	skip, err := preflightPackageBuild(ctx, newPackageBuildSpec(pkg), false)
	if err != nil {
		t.Fatal(err)
	}
	if !skip || pkg.Fingerprint == "" || pkg.Manifest == "" || pkg.Summary == nil {
		t.Fatalf("skipped package was not fully prepared: skip=%v fingerprint=%q manifest=%q summary=%#v", skip, pkg.Fingerprint, pkg.Manifest, pkg.Summary)
	}
}
