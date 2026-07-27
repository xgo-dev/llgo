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
	"sort"
	"strings"

	"github.com/goplus/llgo/cl"
)

// packageBuildSpec is the immutable scheduler input for one package. It keeps
// package classification out of the execution loop, so a later DAG scheduler
// can choose work without reinterpreting frontend metadata.
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

// packageBuildResult carries the observable output of a serial package build.
// Subsequent scheduler PRs can pass this value between worker and finalization
// stages without exposing the mutable aPackage implementation details.
type packageBuildResult struct {
	spec        packageBuildSpec
	cacheHit    bool
	archiveFile string
	needRuntime bool
	needPyInit  bool
}

func packageBuildResultFor(spec packageBuildSpec) packageBuildResult {
	pkg := spec.pkg
	return packageBuildResult{
		spec:        spec,
		cacheHit:    pkg.CacheHit,
		archiveFile: pkg.ArchiveFile,
		needRuntime: pkg.NeedRt,
		needPyInit:  pkg.NeedPyInit,
	}
}

// packageBuildPlan is the immutable dependency graph consumed by a future
// scheduler. specs retains the existing deterministic execution order until
// the LLVM backend is made worker-local; levels records the safe ready sets.
type packageBuildPlan struct {
	specs  []packageBuildSpec
	byID   map[string]packageBuildSpec
	deps   map[string][]string
	levels [][]packageBuildSpec
}

func newPackageBuildPlan(pkgs []*aPackage) (*packageBuildPlan, error) {
	plan := &packageBuildPlan{
		specs: make([]packageBuildSpec, 0, len(pkgs)),
		byID:  make(map[string]packageBuildSpec, len(pkgs)),
		deps:  make(map[string][]string, len(pkgs)),
	}
	for _, pkg := range pkgs {
		spec := newPackageBuildSpec(pkg)
		id := spec.pkg.ID
		if _, exists := plan.byID[id]; exists {
			return nil, fmt.Errorf("duplicate package build spec for %s", id)
		}
		plan.specs = append(plan.specs, spec)
		plan.byID[id] = spec
	}
	for _, spec := range plan.specs {
		id := spec.pkg.ID
		for _, dep := range packageBuildDependencies(spec.pkg) {
			if _, inPlan := plan.byID[dep.ID]; inPlan {
				plan.deps[id] = append(plan.deps[id], dep.ID)
			}
		}
		sort.Strings(plan.deps[id])
	}
	levels, err := plan.readyLevels()
	if err != nil {
		return nil, err
	}
	plan.levels = levels
	return plan, nil
}

func (p *packageBuildPlan) readyLevels() ([][]packageBuildSpec, error) {
	remaining := make(map[string]int, len(p.specs))
	dependents := make(map[string][]string, len(p.specs))
	for _, spec := range p.specs {
		id := spec.pkg.ID
		remaining[id] = len(p.deps[id])
		for _, dep := range p.deps[id] {
			dependents[dep] = append(dependents[dep], id)
		}
	}
	for _, ids := range dependents {
		sort.Strings(ids)
	}

	ready := make([]string, 0, len(p.specs))
	for id, count := range remaining {
		if count == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	levels := make([][]packageBuildSpec, 0, len(p.specs))
	for len(ready) > 0 {
		ids := ready
		ready = nil
		level := make([]packageBuildSpec, 0, len(ids))
		for _, id := range ids {
			level = append(level, p.byID[id])
			for _, dependent := range dependents[id] {
				remaining[dependent]--
				if remaining[dependent] == 0 {
					ready = append(ready, dependent)
				}
			}
		}
		sort.Strings(ready)
		levels = append(levels, level)
	}
	if len(levels) == 0 && len(p.specs) == 0 {
		return levels, nil
	}
	count := 0
	for _, level := range levels {
		count += len(level)
	}
	if count != len(p.specs) {
		cycle := make([]string, 0, len(p.specs)-count)
		for id, unresolved := range remaining {
			if unresolved > 0 {
				cycle = append(cycle, id)
			}
		}
		sort.Strings(cycle)
		return nil, fmt.Errorf("package build dependency cycle involving %s", strings.Join(cycle, ", "))
	}
	return levels, nil
}
