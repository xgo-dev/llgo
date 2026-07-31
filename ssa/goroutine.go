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
	"go/token"
	"go/types"
	"strconv"

	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// func(c.Pointer) c.Pointer
func (p Program) tyRoutine() *types.Signature {
	if p.routineTy == nil {
		paramPtr := types.NewParam(token.NoPos, nil, "", p.VoidPtr().raw.Type)
		params := types.NewTuple(paramPtr)
		p.routineTy = types.NewSignatureType(nil, nil, nil, params, params, false)
	}
	return p.routineTy
}

// The Go instruction creates a new goroutine and calls the specified
// function within it.
//
// Example printed form:
//
//	go println(t0, t1)
//	go t3()
//	go invoke t5.Println(...t6)
func (b Builder) Go(fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
	dbgInstrCall("Go", fn, args)

	prog := b.Prog
	pkg := b.Pkg

	var offset int
	if fn != Nil && fn.kind != vkBuiltin {
		offset = 1
	}
	typs := make([]Type, len(args)+offset)
	flds := make([]llvm.Value, len(args)+offset)
	if offset == 1 {
		typs[0] = fn.Type
		flds[0] = fn.impl
	}
	for i, arg := range args {
		typs[i+offset] = arg.Type
		flds[i+offset] = arg.impl
	}
	t := prog.Struct(typs...)
	voidPtr := prog.VoidPtr()
	// Goroutine startup data may carry Go pointers, including closure ctx.
	// Keep the startup record in scanned, uncollectable GC memory until the
	// new routine has finished its entry call.
	dataPtr := b.Call(pkg.rtFunc("AllocRoot"), prog.IntVal(prog.SizeOf(t), prog.Uintptr())).impl
	aggregateInit(b.impl, dataPtr, t.ll, flds...)
	data := Expr{dataPtr, voidPtr}
	stackSize := prog.IntVal(prog.pthreadStackSize, prog.Uintptr())
	b.Call(pkg.rtFunc("NewProc"), pkg.routine(t, fn, buildCall, len(args)), data, stackSize)
}

func (p Package) routineName() string {
	p.iRoutine++
	return p.Path() + "._llgo_routine$" + strconv.Itoa(p.iRoutine)
}

func (p Package) routine(t Type, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, n int) Expr {
	prog := p.Prog
	routine := p.NewFunc(p.routineName(), prog.tyRoutine(), InC)
	b := routine.MakeBody(1)
	var localCtx, previousLocalCtx Expr
	hasLocalContext := prog.NeedsLocalContext()
	if hasLocalContext {
		localCtx, previousLocalCtx = b.EnterLocalContext()
	}
	param := routine.Param(0)
	data := Expr{llvm.CreateLoad(b.impl, t.ll, param.impl), t}
	args := make([]Expr, n)
	var offset int
	if fn != Nil && fn.kind != vkBuiltin {
		savedType := fn.Type
		fn = b.getField(data, 0)
		// Interface invocation pairs are structurally funcvals, but their data
		// word remains an ordinary receiver argument after crossing the
		// goroutine startup record.
		if savedType.kind == vkIfaceMethod {
			fn.Type = savedType
		}
		offset = 1
	}
	for i := 0; i < n; i++ {
		args[i] = b.getField(data, i+offset)
	}
	b.Call(p.rtFunc("FreeRoot"), param)
	buildCall(b, fn, args...)
	lastInst := b.impl.GetInsertBlock().LastInstruction()
	if lastInst.IsNil() || lastInst.IsAUnreachableInst().IsNil() {
		if hasLocalContext {
			b.LeaveLocalContext(localCtx, previousLocalCtx)
		}
		b.Return(prog.Nil(prog.VoidPtr()))
	}
	return routine.Expr
}
