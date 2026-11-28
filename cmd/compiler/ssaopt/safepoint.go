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

package ssaopt

import (
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/env"
	"github.com/xgo-dev/llgo/cmd/compiler/safepointplan"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// SafepointPass integrates pure-Go cooperative safepoint eligibility and
// backedge analysis into FunctionPlan so backend lowering and GCRootPass share
// a single unified plan.
type SafepointPass struct{}

func (SafepointPass) Name() string { return "safepoint" }

func (SafepointPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if !cfg.CooperativeSafepoints || fn == nil || len(fn.Blocks) == 0 {
		return nil
	}
	plan.SafepointPlanned = true
	if fn.CallKind() != ssa.CallGo || HasNoSplitDirective(fn) {
		return nil
	}
	path := safepointPackagePath(fn)
	if path == "" || excludeSafepointPackage(path) {
		return nil
	}
	plan.SafepointEntry = true
	plan.SafepointBackedges = safepointplan.Backedges(fn)
	return nil
}

// HasNoSplitDirective reports whether fn (or its generic origin) carries a
// first-class //go:nosplit directive on its *types.Func declaration.
func HasNoSplitDirective(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if obj, ok := fn.Object().(*types.Func); ok && obj.NoSplit() {
		return true
	}
	if orig := fn.Origin(); orig != nil && orig != fn {
		return HasNoSplitDirective(orig)
	}
	return false
}

func safepointPackagePath(fn *ssa.Function) string {
	for current := fn; current != nil; current = current.Parent() {
		if pkg := current.Package(); pkg != nil {
			return pkg.Pkg.Path()
		}
		if origin := current.Origin(); origin != nil {
			if pkg := origin.Package(); pkg != nil {
				return pkg.Pkg.Path()
			}
		}
	}
	return ""
}

func excludeSafepointPackage(path string) bool {
	if path == "runtime" || path == "sync/atomic" || strings.HasPrefix(path, "internal/runtime/") {
		return true
	}
	return path == env.LLGoRuntimePkg || strings.HasPrefix(path, env.LLGoRuntimePkg+"/")
}
