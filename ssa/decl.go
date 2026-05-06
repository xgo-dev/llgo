/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package ssa

import (
	"go/types"
	"strconv"
	"strings"

	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

const (
	NameValist = "__llgo_va_list"
)

func VArg() *types.Var {
	return types.NewParam(0, nil, NameValist, types.NewSlice(tyAny))
}

func HasNameValist(sig *types.Signature) bool {
	if sig.Variadic() {
		if params := sig.Params(); params.At(params.Len()-1).Name() == NameValist {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------

type aNamedConst struct {
}

// A NamedConst is a Member of a Package representing a package-level
// named constant.
//
// Pos() returns the position of the declaring ast.ValueSpec.Names[*]
// identifier.
//
// NB: a NamedConst is not a Value; it contains a constant Value, which
// it augments with the name and position of its 'const' declaration.
type NamedConst = *aNamedConst

/*
// NewConst creates a new named constant.
func (p Package) NewConst(name string, val constant.Value) NamedConst {
	return &aNamedConst{}
}
*/

// -----------------------------------------------------------------------------

type aGlobal struct {
	Expr
	//array bool
}

// A Global is a named Value holding the address of a package-level
// variable.
type Global = *aGlobal

const (
	moduleZeroName              = "__llgo.moduleZeroSizedAlloc$"
	runtimeZeroSizedAllocSymbol = "zeroSizedAlloc"
)

func (p Package) moduleZeroSizedAlloc(elem Type) Expr {
	zerobase := p.mod.NamedGlobal(moduleZeroName)
	if zerobase.IsNil() {
		byteTy := p.Prog.Byte()
		zerobase = llvm.AddGlobal(p.mod, byteTy.ll, moduleZeroName)
		zerobase.SetInitializer(llvm.ConstNull(byteTy.ll))
		zerobase.SetLinkage(llvm.LinkOnceODRLinkage)
		zerobase.SetUnnamedAddr(true)
	}
	return Expr{zerobase, p.Prog.Pointer(elem)}
}

func (p Package) ownsGlobal(name string) bool {
	if p.path == "" {
		return true
	}
	return strings.HasPrefix(name, p.path+".")
}

// NewVar creates a new global variable.
func (p Package) NewVar(name string, typ types.Type, bg Background) Global {
	if v, ok := p.vars[name]; ok {
		return v
	}
	t := p.Prog.Type(typ, bg)
	return p.doNewVar(name, t)
}

// NewVarEx creates a new global variable.
func (p Package) NewVarEx(name string, t Type) Global {
	if v, ok := p.vars[name]; ok {
		return v
	}
	return p.doNewVar(name, t)
}

func (p Package) doNewVar(name string, t Type) Global {
	typ := p.Prog.Elem(t).ll
	if p.Prog.td.TypeAllocSize(typ) == 0 {
		var rt *types.Package
		if p.Prog.rt != nil || p.Prog.rtget != nil {
			rt = p.Prog.runtime()
		}
		if rt != nil {
			// Do not redirect the runtime's own zero-sized allocation sentinel.
			zeroName := FullName(rt, runtimeZeroSizedAllocSymbol)
			if name != zeroName && p.ownsGlobal(name) {
				zero := p.moduleZeroSizedAlloc(p.Prog.Elem(t))
				alias := llvm.AddAlias(p.mod, typ, 0, zero.impl, name)
				alias.SetLinkage(llvm.ExternalLinkage)
				// The returned Global intentionally points at the shared
				// sentinel; the alias above preserves this package variable's
				// symbol for external references.
				ret := &aGlobal{zero}
				p.vars[name] = ret
				return ret
			}
		}
	}
	gbl := llvm.AddGlobal(p.mod, typ, name)
	alignment := p.Prog.td.ABITypeAlignment(typ)
	gbl.SetAlignment(alignment)
	ret := &aGlobal{Expr{gbl, t}}
	p.vars[name] = ret
	return ret
}

// VarOf returns a global variable by name.
func (p Package) VarOf(name string) Global {
	return p.vars[name]
}

// Init initializes the global variable with the given value.
func (g Global) Init(v Expr) {
	g.impl.SetInitializer(v.impl)
}

func (g Global) InitNil() {
	g.impl.SetInitializer(llvm.ConstNull(g.impl.GlobalValueType()))
}

func (g Global) ReplaceAllUsesWith(v Expr) {
	g.impl.ReplaceAllUsesWith(v.impl)
}

// -----------------------------------------------------------------------------

// Function represents the parameters, results, and code of a function
// or method.
//
// If Blocks is nil, this indicates an external function for which no
// Go source code is available.  In this case, FreeVars, Locals, and
// Params are nil too.  Clients performing whole-program analysis must
// handle external functions specially.
//
// Blocks contains the function's control-flow graph (CFG).
// Blocks[0] is the function entry point; block order is not otherwise
// semantically significant, though it may affect the readability of
// the disassembly.
// To iterate over the blocks in dominance order, use DomPreorder().
//
// Recover is an optional second entry point to which control resumes
// after a recovered panic.  The Recover block may contain only a return
// statement, preceded by a load of the function's named return
// parameters, if any.
//
// A nested function (Parent()!=nil) that refers to one or more
// lexically enclosing local variables ("free variables") has FreeVars.
// Such functions cannot be called directly but require a
// value created by MakeClosure which, via its Bindings, supplies
// values for these parameters.
//
// If the function is a method (Signature.Recv() != nil) then the first
// element of Params is the receiver parameter.
//
// A Go package may declare many functions called "init".
// For each one, Object().Name() returns "init" but Name() returns
// "init#1", etc, in declaration order.
//
// Pos() returns the declaring ast.FuncLit.Type.Func or the position
// of the ast.FuncDecl.Name, if the function was explicit in the
// source.  Synthetic wrappers, for which Synthetic != "", may share
// the same position as the function they wrap.
// Syntax.Pos() always returns the position of the declaring "func" token.
//
// Type() returns the function's Signature.
//
// A generic function is a function or method that has uninstantiated type
// parameters (TypeParams() != nil). Consider a hypothetical generic
// method, (*Map[K,V]).Get. It may be instantiated with all ground
// (non-parameterized) types as (*Map[string,int]).Get or with
// parameterized types as (*Map[string,U]).Get, where U is a type parameter.
// In both instantiations, Origin() refers to the instantiated generic
// method, (*Map[K,V]).Get, TypeParams() refers to the parameters [K,V] of
// the generic method. TypeArgs() refers to [string,U] or [string,int],
// respectively, and is nil in the generic method.
type aFunction struct {
	Expr
	Pkg  Package
	Prog Program

	blks []BasicBlock

	defer_           *aDefer
	pendingLoopCases []loopDeferCase
	nextDeferID      uintptr
	recov            BasicBlock

	params   []Type
	freeVars Expr
	base     int // base = 1 if hasFreeVars; base = 0 otherwise
	hasVArg  bool

	diFunc DIFunction
}

// Function represents a function or method.
type Function = *aFunction

// NewFunc creates a new function.
func (p Package) NewFunc(name string, sig *types.Signature, bg Background) Function {
	return p.NewFuncEx(name, sig, bg, false, false)
}

// NewFuncEx creates a new function.
func (p Package) NewFuncEx(name string, sig *types.Signature, bg Background, hasFreeVars bool, instantiated bool) Function {
	if v, ok := p.fns[name]; ok {
		return v
	}
	t := p.Prog.FuncDecl(sig, bg)
	dbgInstrln("NewFunc", name, t.raw.Type, "hasFreeVars:", hasFreeVars)
	fn := llvm.AddFunction(p.mod, name, t.ll)
	if bg == InGo {
		fn.AddFunctionAttr(p.nullPointerIsValidAttr)
	}
	if instantiated {
		fn.SetLinkage(llvm.LinkOnceAnyLinkage)
	}
	if p.isPreservedName(name) {
		p.markLLVMUsed(fn)
	}
	ret := newFunction(fn, t, p, p.Prog, hasFreeVars)
	p.fns[name] = ret
	return ret
}

// FuncOf returns a function by name.
func (p Package) FuncOf(name string) Function {
	return p.fns[name]
}

func newFunction(fn llvm.Value, t Type, pkg Package, prog Program, hasFreeVars bool) Function {
	params, hasVArg := newParams(t, prog)
	base := 0
	if hasFreeVars {
		base = 1
	}
	return &aFunction{
		Expr:    Expr{fn, t},
		Pkg:     pkg,
		Prog:    prog,
		params:  params,
		base:    base,
		hasVArg: hasVArg,
	}
}

func newParams(fn Type, prog Program) (params []Type, hasVArg bool) {
	sig := fn.raw.Type.(*types.Signature)
	in := sig.Params()
	if n := in.Len(); n > 0 {
		if hasVArg = HasNameValist(sig); hasVArg {
			n--
		}
		params = make([]Type, n)
		for i := 0; i < n; i++ {
			params[i] = prog.rawType(in.At(i).Type())
		}
	}
	return
}

// Name returns the function's name.
func (p Function) Name() string {
	return p.impl.Name()
}

// Params returns the function's ith parameter.
func (p Function) Param(i int) Expr {
	i += p.base // skip if hasFreeVars
	return Expr{p.impl.Param(i), p.params[i]}
}

func (p Function) closureCtx(b Builder) Expr {
	if p.freeVars.IsNil() {
		if p.base == 0 {
			panic("ssa: function has no free variables")
		}
		ptr := Expr{p.impl.Param(0), p.params[0]}
		if b.blk.Index() != 0 {
			blk := b.impl.GetInsertBlock()
			b.SetBlockEx(p.blks[0], AtStart, false)
			p.freeVars = b.Load(ptr)
			b.impl.SetInsertPointAtEnd(blk)
		} else {
			p.freeVars = b.Load(ptr)
		}
	}
	return p.freeVars
}

// FreeVar returns the function's ith free variable.
func (p Function) FreeVar(b Builder, i int) Expr {
	ctx := p.closureCtx(b)
	return b.getField(ctx, i)
}

// NewBuilder creates a new Builder for the function.
func (p Function) NewBuilder() Builder {
	prog := p.Prog
	b := prog.ctx.NewBuilder()
	// TODO(xsw): Finalize may cause panic, so comment it.
	// b.Finalize()
	return &aBuilder{b, nil, p, p.Pkg, prog,
		make(map[*types.Scope]DIScope)}
}

// HasBody reports whether the function has a body.
func (p Function) HasBody() bool {
	return len(p.blks) > 0
}

// MakeBody creates nblk basic blocks for the function, and creates
// a new Builder associated to #0 block.
func (p Function) MakeBody(nblk int) Builder {
	p.MakeBlocks(nblk)
	b := p.NewBuilder()
	b.blk = p.blks[0]
	b.impl.SetInsertPointAtEnd(b.blk.last)
	return b
}

// MakeBlocks creates nblk basic blocks for the function.
func (p Function) MakeBlocks(nblk int) []BasicBlock {
	n := len(p.blks)
	if n == 0 {
		p.blks = make([]BasicBlock, 0, nblk)
	}
	for i := 0; i < nblk; i++ {
		p.addBlock(n + i)
	}
	return p.blks[n:]
}

func (p Function) addBlock(idx int) BasicBlock {
	label := "_llgo_" + strconv.Itoa(idx)
	blk := p.Prog.ctx.AddBasicBlock(p.impl, label)
	ret := &aBasicBlock{blk, blk, p, idx}
	p.blks = append(p.blks, ret)
	return ret
}

// MakeBlock creates a new basic block for the function.
func (p Function) MakeBlock() BasicBlock {
	return p.addBlock(len(p.blks))
}

// Block returns the ith basic block of the function.
func (p Function) Block(idx int) BasicBlock {
	return p.blks[idx]
}

// SetRecover sets the recover block for the function.
func (p Function) SetRecover(blk BasicBlock) {
	p.recov = blk
}

// -----------------------------------------------------------------------------

type inlineAttr int

const (
	NoInline inlineAttr = iota
	AlwaysInline
	InlineHint
)

func (p Function) Inline(inline inlineAttr) {
	inlineAttrName := map[inlineAttr]string{
		NoInline:     "noinline",
		AlwaysInline: "alwaysinline",
		InlineHint:   "inlinehint",
	}[inline]
	inlineAttr := p.Pkg.mod.Context().CreateEnumAttribute(llvm.AttributeKindID(inlineAttrName), 0)
	p.impl.AddFunctionAttr(inlineAttr)
}

// -----------------------------------------------------------------------------
