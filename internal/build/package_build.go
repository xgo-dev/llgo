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

import "github.com/goplus/llgo/cl"

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
