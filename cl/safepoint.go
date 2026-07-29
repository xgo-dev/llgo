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
	"strings"

	"github.com/goplus/llgo/internal/safepointplan"
	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) prepareCooperativeSafepoints(fn *ssa.Function, isCgo bool) {
	p.safepointEntry = false
	p.safepoints = nil
	if !p.prog.CooperativeSafepointsEnabled() || fn == nil || len(fn.Blocks) == 0 ||
		isCgo || hasFuncDirective(fn, "go:nosplit") {
		return
	}
	// Package-less SSA wrappers only forward into a declared function and may
	// represent runtime helpers, so the declared function owns the poll.
	if path := safepointPackagePath(fn); path == "" || excludeSafepointPackage(path) {
		return
	}
	p.safepointEntry = true
	p.safepoints = safepointplan.Backedges(fn)
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
	if path == "runtime" || strings.HasPrefix(path, "internal/runtime/") {
		return true
	}
	runtimeModule := strings.TrimSuffix(llssa.PkgRuntime, "/internal/runtime")
	return path == runtimeModule || strings.HasPrefix(path, runtimeModule+"/")
}

func (p *context) isCooperativeSafepoint(instr ssa.Instruction) bool {
	_, ok := p.safepoints[instr]
	return ok
}

func (p *context) emitCooperativeSafepoint(b llssa.Builder) {
	b.Call(p.pkg.RuntimeFunc("CooperativeSafepoint"))
}
