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
	"github.com/xgo-dev/llgo/types"

	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
)

// PrepareEscapeAnalysis runs pure-Go escape analysis after all source metadata
// and SSA bodies are ready, before isolated backend Programs are created.
func PrepareEscapeAnalysis(prog llvmgen.Program, input *ssa.Program, patches Patches) *goescape.Result {
	return PrepareEscapeAnalysisCached(prog, input, patches, nil)
}

// PrepareEscapeAnalysisCached optionally reuses complete validated facts.
func PrepareEscapeAnalysisCached(prog llvmgen.Program, input *ssa.Program, patches Patches, cache *goescape.Cache) *goescape.Result {
	return PrepareEscapeAnalysisImported(prog, input, patches, cache, nil, nil, nil)
}

// PrepareEscapeAnalysisImported joins cached archive contracts with source
// bodies that are being rebuilt or imported. It delegates directly to pure-Go
// ssaopt.PrepareEscapeAnalysis without creating an LLVM backend Program.
func PrepareEscapeAnalysisImported(prog llvmgen.Program, input *ssa.Program, patches Patches, cache *goescape.Cache, imports map[*ssa.Function]goescape.Summary, keys map[*ssa.Function]string, extraRoots []*ssa.Function) *goescape.Result {
	ctx := &context{
		prog: prog, goProg: input, fset: input.Fset, patches: patches,
		options: Options{PreloadedSyntax: true, EscapeImports: imports, EscapeImportKeys: keys, EscapeRoots: extraRoots},
		loaded:  make(map[*types.Package]*pkgInfo),
	}
	for _, pkg := range input.AllPackages() {
		ctx.loaded[pkg.Pkg] = &pkgInfo{}
		if ctx.goTyps == nil {
			ctx.goTyps = pkg.Pkg
		}
	}
	return ctx.analyzeEscapeCached(cache)
}

func (p *context) analyzeEscape() *goescape.Result {
	return p.analyzeEscapeCached(nil)
}

func (p *context) analyzeEscapeCached(cache *goescape.Cache) *goescape.Result {
	patchPkgs := make(map[string]*ssa.Package, len(p.patches))
	patchTypes := make(map[string]*types.Package, len(p.patches))
	for path, patch := range p.patches {
		if patch.Alt != nil {
			patchPkgs[path] = patch.Alt
		}
		if patch.Types != nil {
			patchTypes[path] = patch.Types
		}
	}
	return ssaopt.PrepareEscapeAnalysis(p.goProg, ssaopt.EscapeConfig{
		TypeCtx:         p.prog.TypeContext(),
		GCRoots:         p.prog.GCRootsEnabled(),
		PreloadedSyntax: p.options.PreloadedSyntax,
		TargetPkg:       p.goPkg,
		RuntimePkg:      p.prog.RuntimePkg(),
		Patches:         patchPkgs,
		PatchTypes:      patchTypes,
		Cache:           cache,
		Imports:         p.options.EscapeImports,
		ImportKeys:      p.options.EscapeImportKeys,
		ExtraRoots:      p.options.EscapeRoots,
		ResolveIntrinsic: func(fn *ssa.Function) (bool, ssa.Intrinsic) {
			owner, name, kind := p.funcName(fn)
			if kind == llgoInstr {
				return false, ssa.LookupIntrinsic(name)
			}
			if kind != goFunc || owner == nil {
				return false, ssa.IntrinsicNone
			}
			return name == funcName(owner, fn, false), ssa.IntrinsicNone
		},
	})
}

func (p *context) stackStorage(site ssa.Instruction) bool {
	if value, ok := site.(ssa.Value); ok && freshNilAppendAllocation(value) {
		return false
	}
	if decision, ok := p.options.EscapeAnalysis.LookupDecision(site); ok {
		return !decision.Heap
	}
	// A compiled C-ABI body can be excluded from Go call summaries. Preserve
	// the SSA builder's existing local-storage proof in such bodies; missing
	// facts never justify promoting an object that SSA placed on the heap.
	alloc, ok := site.(*ssa.Alloc)
	return ok && !alloc.Heap
}
