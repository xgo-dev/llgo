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
	"fmt"
	"go/types"
	"unsafe"

	"github.com/goplus/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

var (
	tyAny = types.NewInterfaceType(nil, nil)

	NoArgsNoRet = types.NewSignatureType(nil, nil, nil, nil, nil, false)
)

// -----------------------------------------------------------------------------

type valueKind = int

const (
	vkInvalid valueKind = iota
	vkSigned
	vkUnsigned
	vkFloat
	vkComplex
	vkString
	vkBool
	vkPtr
	vkFuncDecl
	vkFuncPtr
	vkClosure
	vkBuiltin
	vkPyFuncRef
	vkPyVarRef
	vkTuple
	vkSlice
	vkArray
	vkMap
	vkEface
	vkIface
	vkStruct
	vkChan
)

// -----------------------------------------------------------------------------

func indexType(t types.Type) types.Type {
	typ := t
retry:
	switch t := typ.(type) {
	case *types.Named:
		typ = t.Underlying()
		goto retry
	case *types.Slice:
		return t.Elem()
	case *types.Pointer:
		switch t := t.Elem().Underlying().(type) {
		case *types.Array:
			return t.Elem()
		}
	case *types.Array:
		return t.Elem()
	}
	panic("index: type doesn't support index - " + t.String())
}

// -----------------------------------------------------------------------------

type goProgram aProgram

// Alignof returns the alignment of a variable of type T.
// Alignof must implement the alignment guarantees required by the spec.
// The result must be >= 1.
func (p *goProgram) Alignof(T types.Type) int64 {
	return p.sizes.Alignof(T)
}

// Offsetsof returns the offsets of the given struct fields, in bytes.
// Offsetsof must implement the offset guarantees required by the spec.
// A negative entry in the result indicates that the struct is too large.
func (p *goProgram) Offsetsof(fields []*types.Var) (ret []int64) {
	prog := Program(unsafe.Pointer(p))
	ptrSize := int64(prog.PointerSize())
	if abi.IsClosureFields(fields) {
		return []int64{0, ptrSize}
	}
	extra := int64(0)
	ret = p.sizes.Offsetsof(fields)
	for i, f := range fields {
		ret[i] += extra
		extra += p.extraSize(f.Type(), ptrSize)
	}
	return
}

// Sizeof returns the size of a variable of type T.
// Sizeof must implement the size guarantees required by the spec.
// A negative result indicates that T is too large.
func (p *goProgram) Sizeof(T types.Type) int64 {
	prog := Program(unsafe.Pointer(p))
	ptrSize := int64(prog.PointerSize())
	baseSize := prog.sizes.Sizeof(T) + p.extraSize(T, ptrSize)
	switch T.Underlying().(type) {
	case *types.Struct, *types.Array:
		return align(baseSize, prog.sizes.Alignof(T))
	}
	return baseSize
}

func (p *goProgram) extraSize(typ types.Type, ptrSize int64) (ret int64) {
retry:
	switch t := typ.(type) {
	case *types.Named:
		if v, ok := p.gocvt.typbg.Load(namedLinkname(t)); ok && v.(Background) == InC {
			return 0
		}
		typ = t.Underlying()
		goto retry
	case *types.Signature:
		return ptrSize
	case *types.Struct:
		if IsClosure(t) {
			return
		}
		n := t.NumFields()
		for i := 0; i < n; i++ {
			f := t.Field(i)
			ret += p.extraSize(f.Type(), ptrSize)
		}
		return
	case *types.Array:
		return p.extraSize(t.Elem(), ptrSize) * t.Len()
	}
	return 0
}

func align(x, a int64) int64 {
	return (x + a - 1) &^ (a - 1)
}

// -----------------------------------------------------------------------------

type rawType struct {
	Type types.Type
}

type aType struct {
	ll   llvm.Type
	raw  rawType
	kind valueKind // value kind of llvm.Type
}

type Type = *aType

// RawType returns the raw type.
func (t Type) RawType() types.Type {
	return t.raw.Type
}

// TypeSizes returns the sizes of the types.
func (p Program) TypeSizes(sizes types.Sizes) types.Sizes {
	p.sizes = sizes
	return (*goProgram)(unsafe.Pointer(p))
}

// TODO(xsw):
// how to generate platform independent code?
func (p Program) SizeOf(typ Type, n ...int64) uint64 {
	size := p.td.TypeAllocSize(typ.ll)
	if len(n) != 0 {
		size *= uint64(n[0])
	}
	return size
}

// OffsetOf returns the offset of a field in a struct.
func (p Program) OffsetOf(typ Type, i int) uint64 {
	return p.td.ElementOffset(typ.ll, i)
}

// SizeOf returns the size of a type.
func SizeOf(prog Program, t Type, n ...int64) Expr {
	size := prog.SizeOf(t, n...)
	return prog.IntVal(size, prog.Uintptr())
}

/*
func OffsetOf(prog Program, t Type, i int) Expr {
	offset := prog.OffsetOf(t, i)
	return prog.IntVal(offset, prog.Uintptr())
}
*/

func (p Program) PointerSize() int {
	return p.ptrSize
}

func (p Program) Slice(typ Type) Type {
	return p.rawType(types.NewSlice(typ.raw.Type))
}

func (p Program) Pointer(typ Type) Type {
	return p.rawType(types.NewPointer(typ.raw.Type))
}

func (p Program) Elem(typ Type) Type {
	elem := typ.raw.Type.Underlying().(interface {
		Elem() types.Type
	}).Elem()
	return p.rawType(elem)
}

func (p Program) Index(typ Type) Type {
	return p.rawType(indexType(typ.raw.Type))
}

func (p Program) Field(typ Type, i int) Type {
	var fld *types.Var
	switch t := typ.raw.Type.Underlying().(type) {
	case *types.Tuple:
		fld = t.At(i)
	case *types.Basic:
		switch t.Kind() {
		case types.Complex128:
			return p.Float64()
		case types.Complex64:
			return p.Float32()
		}
		panic("Field: basic type doesn't have fields")
	default:
		st := t.(*types.Struct)
		if i < 0 || i >= st.NumFields() {
			panic(fmt.Sprintf("Field: struct index out of range: idx=%d fields=%d type=%s",
				i, st.NumFields(), typeStringWithPkg(typ.raw.Type)))
		}
		fld = st.Field(i)
	}
	return p.rawType(fld.Type())
}

func typeStringWithPkg(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Path()
	})
}

func (p Program) rawType(raw types.Type) Type {
	if v := p.typs.At(raw); v != nil {
		return v.(Type)
	}
	ret := p.toType(raw)
	p.typs.Set(raw, ret)
	return ret
}

func (p Program) tyVoidPtr() llvm.Type {
	if p.voidPtrTy.IsNil() {
		// No void* in llvm, all the pointers are opaque pointer.
		p.voidPtrTy = llvm.PointerType(p.tyInt8(), 0)
	}
	return p.voidPtrTy
}

func (p Program) tyVoid() llvm.Type {
	if p.voidType.IsNil() {
		p.voidType = p.ctx.VoidType()
	}
	return p.voidType
}

func (p Program) tyInt1() llvm.Type {
	if p.int1Type.IsNil() {
		p.int1Type = p.ctx.Int1Type()
	}
	return p.int1Type
}

func (p Program) tyInt() llvm.Type {
	if p.intType.IsNil() {
		p.intType = llvmIntType(p.ctx, p.td.PointerSize())
	}
	return p.intType
}

func llvmIntType(ctx llvm.Context, size int) llvm.Type {
	if size <= 4 {
		return ctx.Int32Type()
	}
	return ctx.Int64Type()
}

func (p Program) tyInt8() llvm.Type {
	if p.int8Type.IsNil() {
		p.int8Type = p.ctx.Int8Type()
	}
	return p.int8Type
}

func (p Program) tyInt16() llvm.Type {
	if p.int16Type.IsNil() {
		p.int16Type = p.ctx.Int16Type()
	}
	return p.int16Type
}

func (p Program) tyInt32() llvm.Type {
	if p.int32Type.IsNil() {
		p.int32Type = p.ctx.Int32Type()
	}
	return p.int32Type
}

func (p Program) tyInt64() llvm.Type {
	if p.int64Type.IsNil() {
		p.int64Type = p.ctx.Int64Type()
	}
	return p.int64Type
}

/*
func (p Program) toTuple(typ *types.Tuple) Type {
	return &aType{p.toLLVMTuple(typ), rawType{typ}, vkTuple}
}
*/

func (p Program) toType(raw types.Type) Type {
	if typ, ok := cvtGoSSAOpaqueType(raw); ok {
		return p.rawType(typ)
	}
	typ := rawType{raw}
	switch t := raw.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Int:
			return &aType{p.tyInt(), typ, vkSigned}
		case types.Uint, types.Uintptr:
			return &aType{p.tyInt(), typ, vkUnsigned}
		case types.Bool:
			return &aType{p.tyInt1(), typ, vkBool}
		case types.Uint8:
			return &aType{p.tyInt8(), typ, vkUnsigned}
		case types.Int8:
			return &aType{p.tyInt8(), typ, vkSigned}
		case types.Int16:
			return &aType{p.tyInt16(), typ, vkSigned}
		case types.Uint16:
			return &aType{p.tyInt16(), typ, vkUnsigned}
		case types.Int32:
			return &aType{p.tyInt32(), typ, vkSigned}
		case types.Uint32:
			return &aType{p.tyInt32(), typ, vkUnsigned}
		case types.Int64:
			return &aType{p.tyInt64(), typ, vkSigned}
		case types.Uint64:
			return &aType{p.tyInt64(), typ, vkUnsigned}
		case types.Float32:
			return &aType{p.ctx.FloatType(), typ, vkFloat}
		case types.Float64:
			return &aType{p.ctx.DoubleType(), typ, vkFloat}
		case types.Complex64:
			return &aType{p.tyComplex64(), typ, vkComplex}
		case types.Complex128:
			return &aType{p.tyComplex128(), typ, vkComplex}
		case types.String:
			return &aType{p.rtString(), typ, vkString}
		case types.UnsafePointer:
			return &aType{p.tyVoidPtr(), typ, vkPtr}
		}
	case *types.Pointer:
		elem := p.rawType(t.Elem())
		return &aType{llvm.PointerType(elem.ll, 0), typ, vkPtr}
	case *types.Interface:
		if t.Empty() {
			return &aType{p.rtEface(), typ, vkEface}
		}
		return &aType{p.rtIface(), typ, vkIface}
	case *types.Slice:
		return &aType{p.rtSlice(), typ, vkSlice}
	case *types.Map:
		return &aType{llvm.PointerType(p.rtMap(), 0), typ, vkMap}
	case *types.Struct:
		ll, kind := p.toLLVMStruct(t)
		return &aType{ll, typ, kind}
	case *types.Named:
		return p.toNamed(t)
	case *types.Signature: // represents a C function pointer in raw type
		return &aType{p.toLLVMFuncPtr(t), typ, vkFuncPtr}
	case *types.Tuple:
		return &aType{p.toLLVMTuple(t), typ, vkTuple}
	case *types.Array:
		elem := p.rawType(t.Elem())
		return &aType{llvm.ArrayType(elem.ll, int(t.Len())), typ, vkArray}
	case *types.Chan:
		return &aType{llvm.PointerType(p.rtChan(), 0), typ, vkChan}
	case *types.Alias:
		return p.toType(types.Unalias(t))
	case *types.TypeParam:
		return p.toType(t.Constraint())
	}
	panic(fmt.Sprintf("toLLVMType: todo - %T\n", raw))
}

func (p Program) toLLVMNamedStruct(name string, raw *types.Named, st *types.Struct, kind valueKind) Type {
	t := p.ctx.StructCreateNamed(name)
	typ := &aType{t, rawType{raw}, kind}
	p.named[name] = typ
	p.typs.Set(raw, typ)
	fields := p.toLLVMFields(st)
	t.StructSetBody(fields, false)
	return typ
}

func (p Program) toLLVMStruct(raw *types.Struct) (ret llvm.Type, kind valueKind) {
	fields := p.toLLVMFields(raw)
	ret = p.ctx.StructType(fields, false)
	if IsClosure(raw) {
		kind = vkClosure
	} else {
		kind = vkStruct
	}
	return
}

func IsClosure(raw *types.Struct) bool {
	return abi.IsClosure(raw)
}

func (p Program) toLLVMFields(raw *types.Struct) (fields []llvm.Type) {
	n := raw.NumFields()
	if n > 0 {
		fields = make([]llvm.Type, n)
		for i := 0; i < n; i++ {
			fields[i] = p.rawType(p.patch(raw.Field(i).Type())).ll
		}
	}
	return
}

func (p Program) toLLVMTuple(t *types.Tuple) llvm.Type {
	return p.ctx.StructType(p.toLLVMTypes(t, t.Len()), false)
}

func (p Program) toLLVMTypes(t *types.Tuple, n int) (ret []llvm.Type) {
	if n > 0 {
		ret = make([]llvm.Type, n)
		for i := 0; i < n; i++ {
			ret[i] = p.rawType(p.patch(t.At(i).Type())).ll
		}
	}
	return
}

func (p Program) toLLVMFunc(sig *types.Signature) llvm.Type {
	tParams := sig.Params()
	n := tParams.Len()
	hasVArg := HasNameValist(sig)
	if hasVArg {
		n--
	}
	params := p.toLLVMTypes(tParams, n)
	out := sig.Results()
	var ret llvm.Type
	switch nret := out.Len(); nret {
	case 0:
		ret = p.tyVoid()
	case 1:
		ret = p.rawType(out.At(0).Type()).ll
	default:
		ret = p.toLLVMTuple(out)
	}
	return llvm.FunctionType(ret, params, hasVArg)
}

func (p Program) toLLVMFuncPtr(_ *types.Signature) llvm.Type {
	// LLVM opaque pointers don't retain the pointee function type. Avoid
	// recursively expanding named function types such as F func(F) F.
	return p.tyVoidPtr()
}

func (p Program) retType(raw *types.Signature) Type {
	out := raw.Results()
	switch n := out.Len(); n {
	case 0:
		return p.Void()
	case 1:
		return p.rawType(out.At(0).Type())
	default:
		return &aType{p.toLLVMTuple(out), rawType{out}, vkTuple}
	}
}

func (p Program) llvmNameOf(named *types.Named) (name string) {
	name = NameOf(named)
	if obj := named.Obj(); obj != nil {
		parent := obj.Parent()
		pkg := obj.Pkg()
		if parent != nil && pkg != nil && !isPkgScope(parent, pkg.Scope()) {
			index := p.fnnamed[name]
			p.fnnamed[name] = index + 1
			name += fmt.Sprintf("#%v", index)
		}
	}
	return name
}

func isPkgScope(parent, pkgScope *types.Scope) bool {
	if parent == pkgScope {
		return true
	}
	if parent == nil || pkgScope == nil {
		return false
	}
	// Some importer paths can materialize equivalent package scopes as distinct
	// pointers. Keep package-level names stable when scopes describe the same
	// package scope textually.
	return parent.String() == pkgScope.String()
}

func (p Program) toNamed(raw *types.Named) Type {
	name := p.llvmNameOf(raw)
	if typ, ok := p.named[name]; ok {
		if namedTypeEquivalent(typ.raw.Type, raw) || p.namedStructLayoutEquivalent(typ, raw) {
			return typ
		}
		// Some toolchains produce distinct instantiated named types that share
		// the same object name (e.g. generic local helper structs in stdlib).
		// Disambiguate instead of reusing an incompatible LLVM named struct.
		base := name
		for i := 0; ; i++ {
			name = fmt.Sprintf("%s#%d", base, i)
			if typ2, ok2 := p.named[name]; ok2 {
				if namedTypeEquivalent(typ2.raw.Type, raw) || p.namedStructLayoutEquivalent(typ2, raw) {
					return typ2
				}
				continue
			}
			break
		}
	}
	switch t := raw.Underlying().(type) {
	case *types.Struct:
		kind := vkStruct
		if IsClosure(t) {
			kind = vkClosure
		}
		return p.toLLVMNamedStruct(name, raw, t, kind)
	default:
		typ := p.rawType(t)
		return &aType{typ.ll, rawType{raw}, typ.kind}
	}
}

func namedTypeEquivalent(a, b types.Type) bool {
	na, okA := types.Unalias(a).(*types.Named)
	nb, okB := types.Unalias(b).(*types.Named)
	if !okA || !okB {
		return false
	}
	if types.Identical(na, nb) {
		return true
	}
	if NameOf(na) != NameOf(nb) {
		return false
	}
	_, sigA := na.Underlying().(*types.Signature)
	_, sigB := nb.Underlying().(*types.Signature)
	if sigA || sigB {
		return false
	}
	// go/types may materialize the same package/type in distinct instances
	// (e.g. mixed importer paths across toolchains). Reuse LLVM named structs
	// when full-name and underlying shape are equivalent.
	return typeStringWithPkg(na.Underlying()) == typeStringWithPkg(nb.Underlying())
}

func (p Program) namedStructLayoutEquivalent(existing Type, raw *types.Named) bool {
	if existing == nil || raw == nil {
		return false
	}
	en, ok := types.Unalias(existing.raw.Type).(*types.Named)
	if !ok || NameOf(en) != NameOf(raw) {
		return false
	}
	rs, ok := raw.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	existingFields := existing.ll.StructElementTypes()
	rawFields := p.toLLVMFields(rs)
	if len(existingFields) != len(rawFields) {
		return false
	}
	for i := range existingFields {
		if existingFields[i].String() != rawFields[i].String() {
			return false
		}
	}
	return true
}

// NameOf returns the full name of a named type.
func NameOf(typ *types.Named) string {
	return abi.FullName(typ.Obj().Pkg(), abi.NamedName(typ))
}

// FullName returns the full name of a package member.
func FullName(pkg *types.Package, name string) string {
	return abi.FullName(pkg, name)
}

// PathOf returns the package path of the specified package.
func PathOf(pkg *types.Package) string {
	return abi.PathOf(pkg)
}

// FuncName:
// - func: pkg.name
// - method: pkg.T.name, pkg.(*T).name
func FuncName(pkg *types.Package, name string, recv *types.Var, org bool) string {
	if recv != nil {
		named, ptr := recvNamed(recv.Type())
		var tName string
		if named != nil {
			if org {
				tName = named.Obj().Name()
			} else {
				tName = abi.NamedName(named)
			}
			if ptr {
				tName = "(*" + tName + ")"
			}
		} else {
			tName = types.TypeString(recv.Type(), PathOf)
		}
		return PathOf(pkg) + "." + tName + "." + name
	}
	ret := FullName(pkg, name)
	return ret
}

func recvNamed(t types.Type) (typ *types.Named, ptr bool) {
	if tp, ok := t.(*types.Pointer); ok {
		t = tp.Elem()
		ptr = true
	}
	if _, ok := t.(*types.Alias); ok {
		t = types.Unalias(t)
	}
	typ, _ = t.(*types.Named)
	return
}

func TypeArgs(typeArgs []types.Type) string {
	return abi.TypeArgs(typeArgs)
}

// -----------------------------------------------------------------------------
