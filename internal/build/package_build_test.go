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
	"testing"

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
