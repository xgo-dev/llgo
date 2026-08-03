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

func packageBuildResultFor(task *packageBuildTask) packageBuildResult {
	return packageBuildResult{
		needRuntime: task.pkg.NeedRt,
		needPyInit:  task.pkg.NeedPyInit,
	}
}
