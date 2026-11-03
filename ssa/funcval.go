/*
 * Copyright (c) 2026 The llgo Authors (llgo@xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package ssa

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llvm"
)

// A Go function value is a pointer to a descriptor. Its first word is the Go
// entry, followed by the entry's environment pointer. The synthetic closure
// type describes these two parts without making them two value-storage words.
// Native function pointers remain vkFuncPtr, and interface invocation pairs
// remain aggregates whose receiver is an ordinary call argument.
func (p Program) funcvalDescriptorType(t Type) Type {
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "code", p.Field(t, 0).raw.Type, false),
		types.NewField(token.NoPos, nil, "env", types.Typ[types.UnsafePointer], false),
	}
	return p.rawType(types.NewStruct(fields, nil))
}

// Static functions share a stable, address-significant descriptor. There is
// no unnamed_addr: unsafe copies and interface boxing observe its identity.
// External entries use ODR linkage so references in separate packages name the
// same descriptor; local entries keep both code and descriptor module-local.
func (p Package) constantFuncval(t Type, code, env llvm.Value) Expr {
	prog := p.Prog
	header := prog.funcvalDescriptorType(t)
	// A global name is independent of whether this module owns the entry's
	// definition. Constant expressions use their symbolic operand spelling.
	key := code.Name()
	if key == "" {
		key = code.String()
	}
	if !env.IsNull() {
		part := env.Name()
		if part == "" {
			part = env.String()
		}
		key += "\x00" + part
	}
	name := fmt.Sprintf("_llgo_funcval$%x", sha256.Sum256([]byte(key)))
	global := p.mod.NamedGlobal(name)
	if global.IsNil() {
		global = llvm.AddGlobal(p.mod, header.ll, name)
		global.SetInitializer(prog.constStructValue(header, []llvm.Value{
			code, env,
		}))
		global.SetGlobalConstant(true)
		switch {
		case funcvalModuleLocal(code) || funcvalModuleLocal(env):
			global.SetLinkage(llvm.PrivateLinkage)
		default:
			p.setODRLinkage(global, llvm.LinkOnceODRLinkage)
		}
	}
	return Expr{global, t}
}

// ConstFuncval constructs a constant, noncapturing Go function value owned by
// this package. Function-valued global initializers use this instead of a
// two-field ConstStruct; their descriptor's address is significant.
func (p Package) ConstFuncval(t Type, entry Expr) Expr {
	if t.kind != vkClosure || entry.kind != vkFuncDecl || entry.impl.IsAFunction().IsNil() {
		panic("ssa: constant function value requires a Go entry and function type")
	}
	if fn := p.FuncOf(entry.impl.Name()); fn != nil && fn.NeedsEnv() {
		panic("ssa: constant function value cannot omit a required environment")
	}
	return p.constantFuncval(t, entry.impl, p.Prog.Nil(p.Prog.VoidPtr()).impl)
}

// A constant expression may conceal a module-local capture (for example a
// GEP into private @ctx). Such descriptors cannot participate in ODR merging
// even if their entry is externally visible and their operand names agree.
func funcvalModuleLocal(value llvm.Value) bool {
	if !value.IsAGlobalValue().IsNil() {
		switch value.Linkage() {
		case llvm.PrivateLinkage, llvm.InternalLinkage:
			return true
		}
		if value.IsAGlobalAlias().IsNil() {
			return false
		}
	}
	for i := 0; i < value.OperandsCount(); i++ {
		if funcvalModuleLocal(value.Operand(i)) {
			return true
		}
	}
	return false
}

func (b Builder) makeFuncval(t Type, code, env llvm.Value) Expr {
	prog := b.Prog
	if code.IsNull() {
		return prog.Nil(t)
	}
	if code.IsConstant() && env.IsConstant() {
		return b.Pkg.constantFuncval(t, code, env)
	}
	header := prog.funcvalDescriptorType(t)
	allocate := func() Expr {
		ptr := b.AllocU(header)
		b.aggregateInit(ptr.impl, header, code, env)
		return Expr{ptr.impl, t}
	}
	if code.IsConstant() {
		return allocate()
	}
	// Native adapters may select a nil entry at runtime. Preserve nil without
	// allocating a descriptor for that branch. The temporary holds a Go
	// pointer, so normal value-root tracking retains the resulting descriptor.
	result := b.AllocaT(t)
	b.Store(result, prog.Nil(t))
	nonNil := Expr{llvm.CreateICmp(b.impl, llvm.IntNE, code, llvm.ConstNull(code.Type())), prog.Bool()}
	b.IfThen(nonNil, func() {
		b.Store(result, allocate())
	})
	return b.Load(result)
}

// A captured closure owns its header and captures in one allocation. The
// environment remains the layout expected by the compiled entry; the value
// and all its copies point to the same header and therefore share identity.
func (b Builder) makeCapturedFuncval(t Type, code llvm.Value, captures Type, fields []llvm.Value) Expr {
	prog := b.Prog
	if prog.SizeOf(captures) == 0 {
		env := b.Pkg.moduleZeroSizedAlloc(captures)
		return b.makeFuncval(t, code, env.impl)
	}
	header := prog.funcvalDescriptorType(t)
	allocation := b.AllocU(prog.Struct(header, captures))
	ptr := b.FieldAddr(allocation, 0)
	env := b.FieldAddr(allocation, 1)
	b.aggregateInit(env.impl, captures, fields...)
	b.aggregateInit(ptr.impl, header, code, env.impl)
	return Expr{ptr.impl, t}
}

func (b Builder) funcvalField(value Expr, index int) Expr {
	if index < 0 || index > 1 {
		panic("ssa: invalid function descriptor field")
	}
	prog := b.Prog
	field := prog.Field(value.Type, index)
	if value.impl.IsNull() {
		return prog.Zero(field)
	}
	header := prog.funcvalDescriptorType(value.Type)
	if global := value.impl.IsAGlobalVariable(); !global.IsNil() && global.IsGlobalConstant() {
		if initial := global.Initializer(); !initial.IsNil() && initial.Type() == header.ll {
			part := llvm.CreateExtractValue(b.impl, initial, index)
			return Expr{b.unwrapStructField(header, index, part), field}
		}
	}
	// Parts are also inspected by native callback conversions and nil-safe
	// printing. Use a zero header for nil queries; Call separately checks the
	// descriptor before dispatching, so a nil function call still panics.
	name := "_llgo_funcval_nil"
	zero := b.Pkg.mod.NamedGlobal(name)
	if zero.IsNil() {
		zero = llvm.AddGlobal(b.Pkg.mod, header.ll, name)
		zero.SetInitializer(llvm.ConstNull(header.ll))
		zero.SetGlobalConstant(true)
		zero.SetLinkage(llvm.PrivateLinkage)
		zero.SetUnnamedAddr(true)
	}
	isNil := llvm.CreateICmp(b.impl, llvm.IntEQ, value.impl, llvm.ConstNull(value.ll))
	safe := llvm.CreateSelect(b.impl, isNil, zero, value.impl)
	ptr := Expr{safe, prog.Pointer(header)}
	return b.Load(b.FieldAddr(ptr, index))
}
