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
	"github.com/goplus/llgo/internal/wasmresume"
	"github.com/xgo-dev/llvm"
)

// EnableWasmResumeABI controls emission of the function and call inventory
// consumed by the experimental WebAssembly resumable ABI lowering.
func (p Program) EnableWasmResumeABI(enable bool) {
	p.enableWasmResumeABI = enable
}

// WasmResumeABIEnabled reports whether resumable ABI lowering is enabled for a
// WebAssembly target.
func (p Program) WasmResumeABIEnabled() bool {
	return p.enableWasmResumeABI && p.target != nil && p.target.GOARCH == "wasm"
}

func (p Program) markWasmResumeFunction(fn llvm.Value) {
	if !p.WasmResumeABIEnabled() ||
		wasmresume.IsRuntimeABIImplementation(fn.Name()) ||
		wasmresume.IsNonSuspendingBoundary(fn.Name()) {
		return
	}
	fn.AddFunctionAttr(p.ctx.CreateStringAttribute(wasmresume.FunctionAttribute, "1"))
}

func (p Package) wasmResumeStart(fn llvm.Value) llvm.Value {
	if !p.Prog.WasmResumeABIEnabled() ||
		wasmresume.IsRuntimeABIImplementation(fn.Name()) ||
		wasmresume.IsNonSuspendingBoundary(fn.Name()) {
		return fn
	}
	name := wasmresume.StartSymbol(fn.Name())
	if start := p.mod.NamedFunction(name); !start.IsNil() {
		return start
	}
	fnType := fn.GlobalValueType()
	params := append([]llvm.Type{p.Prog.tyVoidPtr()}, fnType.ParamTypes()...)
	startType := llvm.FunctionType(p.Prog.tyVoidPtr(), params, false)
	return llvm.AddFunction(p.mod, name, startType)
}

func (b Builder) markWasmResumeCall(call llvm.Value, background Background) {
	if background != InGo || !b.wasmResumeFunctionEnabled() {
		return
	}
	callee := call.CalledValue()
	if !callee.IsAFunction().IsNil() &&
		wasmresume.IsNonSuspendingBoundary(callee.Name()) {
		return
	}
	kind := b.Prog.ctx.MDKindID(wasmresume.CallMetadata)
	version := llvm.ConstInt(b.Prog.Int32().ll, wasmresume.MarkerVersion, false).ConstantAsMetadata()
	call.SetMetadata(kind, b.Prog.ctx.MDNode([]llvm.Metadata{version}))
}

func (b Builder) wasmResumeFunctionEnabled() bool {
	return b != nil && b.Prog.WasmResumeABIEnabled() &&
		b.Func != nil && b.Func.background == InGo &&
		!wasmresume.IsRuntimeABIImplementation(b.Func.Name()) &&
		!wasmresume.IsNonSuspendingBoundary(b.Func.Name())
}

func (b Builder) registerWasmResumeUnwind(frame Expr, handler BasicBlock) {
	if !b.wasmResumeFunctionEnabled() {
		return
	}
	ctx := b.Prog.ctx
	typ := llvm.FunctionType(ctx.VoidType(), []llvm.Type{
		b.Prog.tyVoidPtr(),
		b.Prog.tyVoidPtr(),
	}, false)
	fn := b.Pkg.mod.NamedFunction(wasmresume.RegisterUnwindSymbol)
	if fn.IsNil() {
		fn = llvm.AddFunction(b.Pkg.mod, wasmresume.RegisterUnwindSymbol, typ)
	}
	llvm.CreateCall(b.impl, typ, fn, []llvm.Value{frame.impl, handler.Addr().impl})
}

func (b Builder) clearWasmResumeUnwind() {
	if !b.wasmResumeFunctionEnabled() {
		return
	}
	ctx := b.Prog.ctx
	typ := llvm.FunctionType(ctx.VoidType(), nil, false)
	fn := b.Pkg.mod.NamedFunction(wasmresume.ClearUnwindSymbol)
	if fn.IsNil() {
		fn = llvm.AddFunction(b.Pkg.mod, wasmresume.ClearUnwindSymbol, typ)
	}
	llvm.CreateCall(b.impl, typ, fn, nil)
}

func (b Builder) directCallBackground(fn Expr) Background {
	if fn.impl.IsNil() || fn.impl.IsAFunction().IsNil() {
		return inUnknown
	}
	if decl := b.Pkg.FuncOf(fn.impl.Name()); decl != nil {
		return decl.background
	}
	return inUnknown
}
