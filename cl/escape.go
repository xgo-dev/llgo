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
	"go/types"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/goescape"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// PrepareEscapeAnalysis runs after all source metadata and SSA bodies are ready,
// and before isolated backend Programs are created. Its result is immutable.
func PrepareEscapeAnalysis(prog llssa.Program, input *ssa.Program, patches Patches) *goescape.Result {
	ctx := &context{
		prog: prog, goProg: input, fset: input.Fset, patches: patches,
		options: Options{PreloadedSyntax: true},
		loaded:  make(map[*types.Package]*pkgInfo),
	}
	for _, pkg := range input.AllPackages() {
		ctx.loaded[pkg.Pkg] = &pkgInfo{}
		if ctx.goTyps == nil {
			ctx.goTyps = pkg.Pkg
		}
	}
	return ctx.analyzeEscape()
}

func (p *context) analyzeEscape() *goescape.Result {
	// Layout queries must not populate code generation's named-type cache
	// before source replacements are lowered. This applies both to the build
	// coordinator and to direct NewPackageEx callers.
	layout := *p
	layout.prog = p.prog.NewBackendProgram()
	defer layout.prog.Dispose()
	layout.loaded = make(map[*types.Package]*pkgInfo, len(p.loaded))
	for pkg, info := range p.loaded {
		layout.loaded[pkg] = info
	}
	layout.prog.SetPatch(layout.patchType)
	p = &layout
	var roots []*ssa.Function
	for fn := range ssautil.AllFunctions(p.goProg) {
		roots = append(roots, fn)
	}
	return goescape.Analyze(roots, goescape.Config{
		Inspect:  p.inspectEscapeBody,
		External: p.escapeCallContract,
		IndirectInterface: func(typ types.Type) bool {
			return p.prog.InterfaceIndirect(p.type_(typ, llssa.GoDecl))
		},
		ForceHeap:     p.forceEscapeHeap,
		StackLayout:   p.escapeStackLayout,
		MaxStackBytes: llabi.MaxEscapeStackBytes,
	})
}

func (p *context) escapeStackLayout(site ssa.Instruction, typ types.Type) (size, alignment uint64) {
	switch v := site.(type) {
	case *ssa.MakeSlice:
		elem := typ.Underlying().(*types.Slice).Elem()
		n := p.prog.SizeOf(p.type_(elem, llssa.GoDecl))
		if n == 0 || n > llabi.SmallStackBufferSize {
			return 0, 1
		}
		typ = types.NewArray(elem, int64(llabi.SmallStackBufferSize/n))
	case *ssa.Convert, *ssa.BinOp:
		elem := types.Typ[types.Byte]
		if _, conversion := site.(*ssa.Convert); conversion {
			if dst, ok := typ.Underlying().(*types.Slice); ok {
				if basic, ok := dst.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.Int32 {
					elem = types.Typ[types.Rune]
				}
			}
		}
		typ = types.NewArray(elem, llabi.SmallStackBufferSize)
	case *ssa.MakeMap:
		header, bucket := p.prog.MapStackStorage(p.type_(v.Type(), llssa.GoDecl))
		size, alignment = p.prog.SizeOf(header), p.prog.AlignOf(header)
		if bucket != nil {
			a := p.prog.AlignOf(bucket)
			size += -size & (a - 1)
			size += p.prog.SizeOf(bucket)
			if a > alignment {
				alignment = a
			}
		}
		return size, alignment
	}
	lowered := p.type_(typ, llssa.GoDecl)
	return p.prog.SizeOf(lowered), p.prog.AlignOf(lowered)
}

func (p *context) escapeCallContract(fn *ssa.Function) (goescape.Summary, bool) {
	_, name, kind := p.funcName(fn)
	if kind == llgoInstr {
		// Use the resolved instruction ID shared with call lowering. A visible
		// Go body with a linkname is deliberately not consulted here.
		return instructionEscapeContract(llgoInstrs[name], fn.Signature)
	}
	if len(fn.Blocks) == 0 && hasFuncDirective(fn, "go:noescape") {
		summary := signatureEscapeSummary(fn.Signature)
		for i := range summary.Parameters {
			// As in cmd/compile, noescape promises no retention, not purity.
			// Assembly may mutate pointees and invoke function arguments.
			summary.Parameters[i].Mutator = 1
			summary.Parameters[i].Callee = 1
		}
		return summary, true
	}
	return goescape.Summary{}, false
}

func (p *context) inspectEscapeBody(fn *ssa.Function) bool {
	if fn.TypeParams().Len() != 0 && len(fn.TypeArgs()) == 0 {
		return false
	}
	pkg := fn.Pkg
	for parent := fn.Parent(); pkg == nil && parent != nil; parent = parent.Parent() {
		pkg = parent.Pkg
	}
	if pkg != nil {
		if !p.options.PreloadedSyntax && pkg != p.goPkg {
			// A direct compiler API user may not have parsed declarations in
			// imported bodies. Until it does, their source stubs prove nothing.
			return false
		}
		if patch, ok := p.patches[pkg.Pkg.Path()]; ok && pkg != patch.Alt {
			return false
		}
	}
	owner, name, kind := p.funcName(fn)
	if kind != goFunc || owner == nil {
		return false
	}
	// A Go-to-Go linkname can also replace a visible body. Analyze a body only
	// when it is the symbol the backend will actually call.
	return name == funcName(owner, fn, false)
}

func (p *context) forceEscapeHeap(site ssa.Instruction, typ types.Type) string {
	switch site.(type) {
	case *ssa.MakeSlice:
		elem := typ.Underlying().(*types.Slice).Elem()
		size := p.prog.SizeOf(p.type_(elem, llssa.GoDecl))
		if size == 0 || size > llabi.SmallStackBufferSize {
			return "no small slice stack buffer"
		}
		typ = types.NewArray(elem, int64(llabi.SmallStackBufferSize/size))
	case *ssa.Convert, *ssa.BinOp:
		// These implicit allocations contain scalar characters, not the
		// pointer-bearing string/slice header used to describe the result.
		typ = types.NewArray(types.Typ[types.Byte], llabi.SmallStackBufferSize)
	case *ssa.MakeMap:
		// The runtime map header contains managed pointers even when its
		// keys and values do not. Explicit-root targets keep its heap path.
		if p.prog.GCRootsEnabled() {
			return "explicit GC roots do not scan map stack fields"
		}
	}
	lowered := p.type_(typ, llssa.GoDecl)
	size := p.prog.SizeOf(lowered)
	limit := llabi.MaxImplicitStackVarSize
	if alloc, ok := site.(*ssa.Alloc); ok {
		limit = llabi.MaxStackVarSize
		if alloc.Heap {
			// Newly promoted objects must fit the implicit allocation limit.
			// In particular, a 128 KiB new(array) consumes an entire Wasm stack.
			limit = llabi.MaxImplicitStackVarSize
		}
		if size == 0 && alloc.Heap {
			return "zero-size heap identity"
		}
		if alloc.Heap && alloc.Parent().Recover != nil {
			return "recover-visible storage"
		}
	} else if size == 0 {
		return "zero-size heap identity"
	}
	if size > limit {
		return "stack size limit"
	}
	if p.prog.GCRootsEnabled() && mayContainGCRoot(typ) {
		if alloc, ok := site.(*ssa.Alloc); ok && !alloc.Heap {
			// SSA-local addresses never leave the function. Their loads already
			// publish the individual pointer values through the GC root plan.
			// New promotions can pass their address to a borrowing callee, whose
			// writes are not covered by that plan, so keep those on the heap.
			return ""
		}
		return "explicit GC roots do not scan stack fields"
	}
	return ""
}

func (p *context) stackStorage(site ssa.Instruction) bool {
	if decision, ok := p.options.EscapeAnalysis.LookupDecision(site); ok {
		return !decision.Heap
	}
	// A compiled C-ABI body can be excluded from Go call summaries. Preserve
	// the SSA builder's existing local-storage proof in such bodies; missing
	// facts never justify promoting an object that SSA placed on the heap.
	alloc, ok := site.(*ssa.Alloc)
	return ok && !alloc.Heap
}
