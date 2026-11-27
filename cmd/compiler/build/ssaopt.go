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
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func (ctx *context) newOptPlan() *ssaopt.ProgramPlan {
	if ctx == nil || ctx.progSSA == nil {
		return nil
	}
	var patchPkgs map[string]*ssa.Package
	var patchTypes map[string]*types.Package
	if len(ctx.patches) > 0 {
		patchPkgs = make(map[string]*ssa.Package, len(ctx.patches))
		patchTypes = make(map[string]*types.Package, len(ctx.patches))
		for path, patch := range ctx.patches {
			if patch.Alt != nil {
				patchPkgs[path] = patch.Alt
			}
			if patch.Types != nil {
				patchTypes[path] = patch.Types
			}
		}
	}
	var typeCtx *types.Context
	var gcRoots bool
	var coopSafepoints bool
	var runtimePkg *types.Package
	if ctx.prog != nil {
		typeCtx = ctx.prog.TypeContext()
		gcRoots = ctx.prog.GCRootsEnabled()
		coopSafepoints = ctx.prog.CooperativeSafepointsEnabled()
		runtimePkg = ctx.prog.RuntimePkg()
	}
	// Use a lazy ProgramPlan so cached dependency packages never incur planning
	// work, while every lowered function gets a verified FunctionPlan on demand.
	return ssaopt.NewProgramPlan(ssaopt.Config{
		TypeCtx:               typeCtx,
		OptLevel:              ctx.buildConf.OptLevel,
		DisableBoundsChecks:   ctx.buildConf.DisableBoundsChecks,
		GCRoots:               gcRoots,
		CooperativeSafepoints: coopSafepoints,
		Escape:                ctx.frontendOptions.EscapeAnalysis,
		Patches:               patchPkgs,
		PatchTypes:            patchTypes,
		RuntimePkg:            runtimePkg,
	}, ssaopt.DefaultPipeline())
}
