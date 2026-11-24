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
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// wrapperParam implements ssa.Value to represent a parameter of a compiler-emitted
// forwarding wrapper for an LLVM intrinsic or LLGo instruction binding without
// synthesizing private x/tools/go/ssa structures via unsafe.
type wrapperParam struct {
	index     int
	name      string
	typ       types.Type
	pos       token.Pos
	hasValist bool
}

func (w wrapperParam) Name() string                  { return w.name }
func (w wrapperParam) String() string                { return w.name }
func (w wrapperParam) Type() types.Type              { return w.typ }
func (w wrapperParam) Parent() *ssa.Function         { return nil }
func (w wrapperParam) Referrers() *[]ssa.Instruction { return nil }
func (w wrapperParam) Pos() token.Pos                { return w.pos }

// compileIntrinsicWrapper emits a linkonce (LinkOnceAny) addressable Go function
// wrapper for an LLVM intrinsic or LLGo instruction binding when used as a function value.
func (p *context) compileIntrinsicWrapper(v *ssa.Function) llssa.Function {
	if fn := p.funcs[v]; fn != nil {
		return fn
	}
	pkgTypes := p.goTyps
	if v.Pkg != nil {
		pkgTypes = v.Pkg.Pkg
	}
	name := funcName(pkgTypes, v, false)
	if fn := p.pkg.FuncOf(name); fn != nil && fn.HasBody() {
		p.funcs[v] = fn
		return fn
	}
	sig := func() *types.Signature {
		oldGoFn := p.goFn
		p.goFn = v
		defer func() {
			p.goFn = oldGoFn
		}()
		return p.patchType(v.Signature).(*types.Signature)
	}()
	fn := p.pkg.NewFuncEx(name, sig, llssa.GoDecl, false, true)
	noInlineDirective := hasNoInlineDirective(v)
	if disableInline || noInlineDirective {
		fn.Inline(llssa.NoInline)
	}
	if noInlineDirective {
		fn.DisableTailCalls()
	}
	p.funcs[v] = fn
	if p.options.DeclarationsOnly {
		return fn
	}
	p.emitFuncInfo(p.pkg, pkgTypes, v, fn)
	dbgEnabled := p.options.Debug
	p.inits = append(p.inits, func() {
		oldFn := p.fn
		oldGoFn := p.goFn
		oldInCFunc := p.inCFunc
		oldInWrapper := p.inIntrinsicWrapper
		p.fn = fn
		p.goFn = v
		p.inCFunc = false
		p.inIntrinsicWrapper = true
		defer func() {
			p.fn = oldFn
			p.goFn = oldGoFn
			p.inCFunc = oldInCFunc
			p.inIntrinsicWrapper = oldInWrapper
		}()

		b := fn.MakeBody(1)
		defer func() {
			b.EndBuild()
			b.Dispose()
		}()
		if dbgEnabled {
			pos := p.goProg.Fset.Position(v.Pos())
			bodyPos := p.getFuncBodyPos(v)
			b.DebugFunction(fn, debugFunctionScope(v), pos, bodyPos)
		}

		hasValist := llssa.HasNameValist(v.Signature)
		var args []ssa.Value
		addParam := func(param *types.Var) {
			pname := param.Name()
			if pname == "" {
				pname = fmt.Sprintf("arg%d", len(args))
			}
			args = append(args, wrapperParam{
				index:     len(args),
				name:      pname,
				typ:       param.Type(),
				pos:       v.Pos(),
				hasValist: hasValist,
			})
		}
		if recv := v.Signature.Recv(); recv != nil {
			addParam(recv)
		}
		params := v.Signature.Params()
		for i := 0; i < params.Len(); i++ {
			addParam(params.At(i))
		}

		call := ssa.CallCommon{
			Value: v,
			Args:  args,
		}
		ret := p.call(b, llssa.Call, &call)
		if b.HasTerminator() {
			return
		}
		switch n := sig.Results().Len(); n {
		case 0:
			b.Return()
		case 1:
			b.Return(ret)
		default:
			rets := make([]llssa.Expr, n)
			for i := 0; i < n; i++ {
				rets[i] = b.Extract(ret, i)
			}
			b.Return(rets...)
		}
	})
	return fn
}
