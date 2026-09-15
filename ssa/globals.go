/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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
	"fmt"
	"go/types"

	"github.com/xgo-dev/llvm"
)

func (pkg Package) AddGlobalString(name string, value string) {
	prog := pkg.Prog
	styp := prog.String()
	data := pkg.createGlobalStr(value)
	length := prog.IntVal(uint64(len(value)), prog.Uintptr())
	cv := prog.constStructValue(styp, []llvm.Value{data, length.impl})
	pkg.NewVarEx(name, prog.Pointer(styp)).Init(Expr{cv, styp})
}

// ConstString creates an SSA expression representing a Go string literal. The
// returned value is backed by an anonymous global constant and can be used to
// initialize package-level variables or other constant contexts that expect a
// Go string value.
func (pkg Package) ConstString(value string) Expr {
	prog := pkg.Prog
	styp := prog.String()
	data := pkg.createGlobalStr(value)
	length := prog.IntVal(uint64(len(value)), prog.Uintptr())
	cv := prog.constStructValue(styp, []llvm.Value{data, length.impl})
	return Expr{cv, styp}
}

// ConstBytes creates an SSA expression for a []byte backed by writable static data.
// Each call gets its own backing store, matching []byte mutability semantics.
func (pkg Package) ConstBytes(value []byte) Expr {
	prog := pkg.Prog
	styp := prog.Slice(prog.Byte())
	if len(value) == 0 {
		return prog.Zero(styp)
	}
	data := pkg.createGlobalBytes(value)
	n := prog.IntVal(uint64(len(value)), prog.Int())
	cv := prog.constStructValue(styp, []llvm.Value{data, n.impl, n.impl})
	return Expr{cv, styp}
}

// ConstArray creates an LLVM constant array expression.
func (prog Program) ConstArray(t Type, values []Expr) Expr {
	elem := prog.Index(t)
	fields := make([]llvm.Value, len(values))
	for i, value := range values {
		fields[i] = prog.toStorageConstant(elem, value.impl)
	}
	return Expr{llvm.ConstArray(prog.storageType(elem), fields), t}
}

// ConstByteArray creates a compact LLVM constant for a Go byte array. Unlike
// ConstArray, it does not allocate one LLVM value wrapper per array element.
func (prog Program) ConstByteArray(t Type, value []byte) Expr {
	return Expr{prog.ctx.ConstString(string(value), false), t}
}

// ConstSlice creates a slice constant backed by a writable package global.
// The backing store must remain writable because a Go slice literal may be
// mutated after package initialization.
func (pkg Package) ConstSlice(name string, t Type, values []Expr) Expr {
	prog := pkg.Prog
	elem := prog.Index(t)
	array := prog.rawType(types.NewArray(elem.RawType(), int64(len(values))))
	data := pkg.NewVarEx(name, prog.Pointer(array))
	data.Init(prog.ConstArray(array, values))

	n := prog.IntVal(uint64(len(values)), prog.Int())
	cv := prog.constStructValue(t, []llvm.Value{data.impl, n.impl, n.impl})
	return Expr{cv, t}
}

// ConstStruct creates an LLVM constant struct expression.
func (prog Program) ConstStruct(t Type, values []Expr) Expr {
	fields := make([]llvm.Value, len(values))
	for i, value := range values {
		fields[i] = value.impl
	}
	return Expr{prog.constStructValue(t, fields), t}
}

func (prog Program) constStructValue(t Type, values []llvm.Value) llvm.Value {
	return prog.constStructValueAs(t, t.ll, values)
}

// constStructValueAs constructs a constant using the exact LLVM type owned by
// its destination. LLVM may uniquify an otherwise equivalent identified
// struct while linking modules, so global initializers must not use a second
// Type instance merely because it has the same Go shape.
func (prog Program) constStructValueAs(t Type, structType llvm.Type, values []llvm.Value) llvm.Value {
	elements := structType.StructElementTypes()
	fields := make([]llvm.Value, len(elements))
	for i, value := range values {
		fields[i] = prog.wrapStructConstantAs(t, structType, i, value)
	}
	for i := len(values); i < len(fields); i++ {
		fields[i] = llvm.ConstNull(elements[i])
	}
	if structType.StructName() != "" {
		return llvm.ConstNamedStruct(structType, fields)
	}
	return prog.ctx.ConstStruct(fields, structType.IsStructPacked())
}

func (prog Program) wrapStructConstant(t Type, index int, value llvm.Value) llvm.Value {
	return prog.wrapStructConstantAs(t, t.ll, index, value)
}

func (prog Program) wrapStructConstantAs(t Type, structType llvm.Type, index int, value llvm.Value) llvm.Value {
	value = prog.toStorageConstant(prog.Field(t, index), value)
	layout, ok := prog.structLayout(t)
	if !ok || index >= len(layout.wrapped) || !layout.wrapped[index] {
		return value
	}
	elem := structType.StructElementTypes()[index]
	parts := elem.StructElementTypes()
	values := []llvm.Value{value}
	if len(parts) == 2 {
		values = append(values, llvm.ConstNull(parts[1]))
	}
	return prog.ctx.ConstStruct(values, true)
}

// Undefined global string var by names
func (pkg Package) Undefined(names ...string) error {
	prog := pkg.Prog
	styp := prog.rtString()
	for _, name := range names {
		global := pkg.VarOf(name)
		if global == nil {
			continue
		}
		typ := prog.Elem(global.Type)
		if typ.ll != styp {
			return fmt.Errorf("%s: not a var of type string (type:%v)", name, typ.RawType())
		}
		newGlobal := llvm.AddGlobal(pkg.mod, styp, "")
		global.impl.ReplaceAllUsesWith(newGlobal)
		global.impl.EraseFromParentAsGlobal()
		newGlobal.SetName(name)
	}
	return nil
}
