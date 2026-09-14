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

package ssa

import (
	"go/token"
	"go/types"
)

const runtimeLocalContext = "LocalContext"

// EnterLocalContext creates the stack root used by TLS/GLS locality blocks and
// installs it for the current outermost Go entry. previous is nonzero only for
// a nested entry that inherited an existing context.
func (b Builder) EnterLocalContext() (ctx, previous Expr) {
	fn := b.Pkg.rtFunc("EnterLocalContext")
	params := fn.raw.Type.(*types.Signature).Params()
	ctxPtr := b.Prog.rawType(params.At(0).Type())
	ctxType := b.Prog.rawType(ctxPtr.RawType().(*types.Pointer).Elem())
	ctx = b.Alloc(ctxType, false)
	previous = b.Call(fn, ctx)
	return
}

// LeaveLocalContext restores an inherited context or drops the stack roots
// installed by EnterLocalContext.
func (b Builder) LeaveLocalContext(ctx, previous Expr) {
	b.Call(b.Pkg.rtFunc("LeaveLocalContext"), ctx, previous)
}

// BuildLocalPackageAccessor builds an always-hot package block lookup around a
// generated owner-local cache slot. The runtime owns allocation and rooting;
// the compiler needs no LocalContext or block-header layout knowledge.
func (p Function) BuildLocalPackageAccessor(cache, size, align Expr) {
	prog := p.Prog
	b := p.MakeBody(3)
	hit := p.Block(1)
	slow := p.Block(2)
	cached := b.Load(cache)
	b.If(b.BinOp(token.NEQ, cached, prog.IntVal(0, prog.Uintptr())), hit, slow)

	b.SetBlock(hit)
	result := p.raw.Type.(*types.Signature).Results().At(0).Type()
	b.Return(b.Convert(prog.rawType(result), cached))

	b.SetBlock(slow)
	raw := b.Call(p.Pkg.rtFunc("LocalPackage"), cache, size, align)
	b.Return(b.Convert(prog.rawType(result), raw))
	b.EndBuild()
}
