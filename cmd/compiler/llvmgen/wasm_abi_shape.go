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

package llvmgen

import (
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"strconv"
	"strings"

	"github.com/xgo-dev/llvm"
)

// wasmCallShape records the physical prototypes and the Go frame layout.
// Keep C integer extensions last to preserve cross-package adapter names.
func (p Program) wasmCallShape(goSig, nativeSig *types.Signature, goType, nativeType llvm.Type) (string, bool) {
	var shape strings.Builder
	p.appendWasmLLVMTypeShape(&shape, goType)
	if nativeType != goType {
		shape.WriteByte('/')
		p.appendWasmLLVMTypeShape(&shape, nativeType)
	}
	shape.WriteByte('/')
	p.appendWasmTupleRootShape(&shape, goSig.Params())
	// Reflection supplies a distinct destination slot for each result.
	shape.WriteByte('/')
	p.appendWasmTupleRootShape(&shape, goSig.Results())
	hasAttrs := p.appendNativeIntegerAttrShape(&shape, nativeSig, nativeType)
	return shape.String(), hasAttrs
}

func (p Program) appendWasmLLVMTypeShape(shape *strings.Builder, typ llvm.Type) {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind:
		shape.WriteByte('v')
	case llvm.FloatTypeKind:
		shape.WriteByte('f')
	case llvm.DoubleTypeKind:
		shape.WriteByte('d')
	case llvm.X86_FP80TypeKind:
		shape.WriteString("x80")
	case llvm.FP128TypeKind:
		shape.WriteString("f128")
	case llvm.PPC_FP128TypeKind:
		shape.WriteString("p128")
	case llvm.IntegerTypeKind:
		shape.WriteByte('i')
		shape.WriteString(strconv.Itoa(typ.IntTypeWidth()))
	case llvm.FunctionTypeKind:
		shape.WriteByte('(')
		for _, param := range typ.ParamTypes() {
			p.appendWasmLLVMTypeShape(shape, param)
			shape.WriteByte(',')
		}
		if typ.IsFunctionVarArg() {
			shape.WriteByte('*')
		}
		shape.WriteByte(')')
		p.appendWasmLLVMTypeShape(shape, typ.ReturnType())
	case llvm.StructTypeKind:
		// A lookalike C record can use a header's padding bytes for a live field.
		if p.usesWideGoStorage() && p.isWasmABIPaddedHeader(typ) {
			shape.WriteByte('h')
		}
		if typ.IsStructPacked() {
			shape.WriteByte('<')
		} else {
			shape.WriteByte('{')
		}
		for _, elem := range typ.StructElementTypes() {
			p.appendWasmLLVMTypeShape(shape, elem)
			shape.WriteByte(',')
		}
		if typ.IsStructPacked() {
			shape.WriteByte('>')
		} else {
			shape.WriteByte('}')
		}
	case llvm.ArrayTypeKind:
		shape.WriteByte('[')
		shape.WriteString(strconv.Itoa(typ.ArrayLength()))
		shape.WriteByte(':')
		p.appendWasmLLVMTypeShape(shape, typ.ElementType())
		shape.WriteByte(']')
	case llvm.PointerTypeKind:
		shape.WriteByte('p')
		shape.WriteString(strconv.Itoa(typ.PointerAddressSpace()))
	case llvm.VectorTypeKind:
		shape.WriteByte('V')
		shape.WriteString(strconv.Itoa(typ.VectorSize()))
		shape.WriteByte(':')
		p.appendWasmLLVMTypeShape(shape, typ.ElementType())
	default:
		panic(fmt.Sprintf("ssa: unsupported LLVM type in WebAssembly ABI: %v", typ.TypeKind()))
	}
}

func (p Program) appendWasmTupleRootShape(shape *strings.Builder, tuple *types.Tuple) {
	for i := 0; i < tuple.Len(); i++ {
		p.appendWasmRootShape(shape, p.rawType(tuple.At(i).Type()))
		shape.WriteByte(';')
	}
}

func (p Program) appendWasmRootShape(shape *strings.Builder, typ Type) {
	switch typ.kind {
	case vkPtr, vkMap, vkChan:
		shape.WriteByte('r')
	case vkString:
		shape.WriteByte('s')
		shape.WriteString("0r")
	case vkSlice:
		shape.WriteByte('l')
		shape.WriteString("0r")
	case vkEface, vkIface:
		shape.WriteByte('i')
		shape.WriteString("1r")
	case vkClosure:
		// Closures and interfaces are both two-word Wasm values with their
		// second word rooted, but their first words use different function-table
		// and linear-memory storage conversions.
		shape.WriteByte('c')
		shape.WriteString("1r")
	case vkStruct, vkTuple:
		shape.WriteByte('{')
		var count int
		switch raw := typ.raw.Type.Underlying().(type) {
		case *types.Struct:
			count = raw.NumFields()
		case *types.Tuple:
			count = raw.Len()
		}
		for i := 0; i < count; i++ {
			p.appendWasmRootShape(shape, p.Field(typ, i))
			shape.WriteByte(',')
		}
		shape.WriteByte('}')
	case vkArray:
		raw := typ.raw.Type.Underlying().(*types.Array)
		shape.WriteByte('[')
		shape.WriteString(strconv.FormatInt(raw.Len(), 10))
		shape.WriteByte(':')
		p.appendWasmRootShape(shape, p.Index(typ))
		shape.WriteByte(']')
	default:
		shape.WriteByte('-')
	}
}

func (p Program) appendNativeIntegerAttrShape(shape *strings.Builder, sig *types.Signature, prototype llvm.Type) (hasAttrs bool) {
	p.nativeIntegerAttrs(sig, prototype, func(index int, attr llvm.Attribute) {
		hasAttrs = true
		ext := 's'
		if uint(attr.GetEnumKind()) == nativeZeroExtKind {
			ext = 'z'
		}
		fmt.Fprintf(shape, "/%d%c", index, ext)
	})
	return hasAttrs
}
