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
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// PyObjectPtrPtr returns the **py.Object type.
func (p Program) PyObjectPtrPtr() Type {
	if p.pyObjPPtr == nil {
		p.pyObjPPtr = p.Pointer(p.PyObjectPtr())
	}
	return p.pyObjPPtr
}

// PyObjectPtr returns the *py.Object type.
func (p Program) PyObjectPtr() Type {
	if p.pyObjPtr == nil {
		objPtr := types.NewPointer(p.pyNamed("Object"))
		p.pyObjPtr = p.rawType(objPtr)
	}
	return p.pyObjPtr
}

func (p Program) pyNamed(name string) *types.Named {
	// TODO(xsw): does python type need to convert?
	t := p.python().Scope().Lookup(name).Type().(*types.Named)
	return t
}

func (p Program) python() *types.Package {
	if p.py == nil {
		p.py = p.pyget()
	}
	return p.py
}

// SetPython sets the Python package.
// Its type can be *types.Package or func() *types.Package.
func (p Program) SetPython(py any) {
	switch v := py.(type) {
	case *types.Package:
		p.py = v
	case func() *types.Package:
		p.pyget = v
	}
}

// -----------------------------------------------------------------------------

func (p Package) pyFunc(fullName string, sig *types.Signature) Expr {
	p.NeedPyInit = true
	return p.NewFunc(fullName, sig, InC).Expr
}

func (p Program) paramObjPtr() *types.Var {
	if p.paramObjPtr_ == nil {
		objPtr := p.PyObjectPtr().raw.Type
		p.paramObjPtr_ = types.NewParam(token.NoPos, nil, "", objPtr)
	}
	return p.paramObjPtr_
}

// func(*char) *Object
func (p Program) tyImportPyModule() *types.Signature {
	if p.pyImpTy == nil {
		charPtr := types.NewPointer(types.Typ[types.Int8])
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "", charPtr))
		results := types.NewTuple(p.paramObjPtr())
		p.pyImpTy = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyImpTy
}

// func(*Object) *Object
func (p Program) tyCallNoArgs() *types.Signature {
	if p.callNoArgs == nil {
		params := types.NewTuple(p.paramObjPtr())
		p.callNoArgs = types.NewSignatureType(nil, nil, nil, params, params, false)
	}
	return p.callNoArgs
}

// func(*Object, *Object) *Object
func (p Program) tyCallOneArg() *types.Signature {
	if p.callOneArg == nil {
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, paramObjPtr)
		results := types.NewTuple(paramObjPtr)
		p.callOneArg = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.callOneArg
}

// func(*Object, ...) *Object
func (p Program) tyCallFunctionObjArgs() *types.Signature {
	if p.callFOArgs == nil {
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, VArg())
		results := types.NewTuple(paramObjPtr)
		p.callFOArgs = types.NewSignatureType(nil, nil, nil, params, results, true)
	}
	return p.callFOArgs
}

/*
// func(*Object, *Object, *Object) *Object
func (p Program) tyCall() *types.Signature {
	if p.callArgs == nil {
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, paramObjPtr, paramObjPtr)
		results := types.NewTuple(paramObjPtr)
		p.callArgs = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.callArgs
}
*/

// func(*Object, uintptr, *Object) cint
func (p Program) tyListSetItem() *types.Signature {
	if p.pyListSetI == nil {
		paramUintptr := types.NewParam(token.NoPos, nil, "", p.Uintptr().raw.Type)
		paramCInt := types.NewParam(token.NoPos, nil, "", p.CInt().raw.Type)
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, paramUintptr, paramObjPtr)
		results := types.NewTuple(paramCInt)
		p.pyListSetI = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyListSetI
}

// func(uintptr) *Object
func (p Program) tyNewList() *types.Signature {
	if p.pyNewList == nil {
		paramUintptr := types.NewParam(token.NoPos, nil, "", p.Uintptr().raw.Type)
		params := types.NewTuple(paramUintptr)
		results := types.NewTuple(p.paramObjPtr())
		p.pyNewList = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyNewList
}

// func(*Object, uintptr, *Object) cint
func (p Program) tyTupleSetItem() *types.Signature {
	if p.pyTupleSetI == nil {
		paramUintptr := types.NewParam(token.NoPos, nil, "", p.Uintptr().raw.Type)
		paramCInt := types.NewParam(token.NoPos, nil, "", p.CInt().raw.Type)
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, paramUintptr, paramObjPtr)
		results := types.NewTuple(paramCInt)
		p.pyTupleSetI = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyTupleSetI
}

// func(uintptr) *Object
func (p Program) tyNewTuple() *types.Signature {
	if p.pyNewTuple == nil {
		paramUintptr := types.NewParam(token.NoPos, nil, "", p.Uintptr().raw.Type)
		params := types.NewTuple(paramUintptr)
		results := types.NewTuple(p.paramObjPtr())
		p.pyNewTuple = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyNewTuple
}

// func(int32) *Object
func (p Program) tyBoolFromLong() *types.Signature {
	if p.pyBoolFromInt32 == nil {
		paramObjPtr := p.paramObjPtr()
		paramFloat := types.NewParam(token.NoPos, nil, "", p.Int32().raw.Type)
		params := types.NewTuple(paramFloat)
		results := types.NewTuple(paramObjPtr)
		p.pyBoolFromInt32 = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyBoolFromInt32
}

// func(float64) *Object
func (p Program) tyFloatFromDouble() *types.Signature {
	if p.floatFromDbl == nil {
		paramObjPtr := p.paramObjPtr()
		paramFloat := types.NewParam(token.NoPos, nil, "", p.Float64().raw.Type)
		params := types.NewTuple(paramFloat)
		results := types.NewTuple(paramObjPtr)
		p.floatFromDbl = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.floatFromDbl
}

// func(int64) *Object
func (p Program) tyLongFromInt64() *types.Signature {
	if p.pyLongFromInt64 == nil {
		paramObjPtr := p.paramObjPtr()
		paramInt := types.NewParam(token.NoPos, nil, "", p.Int64().raw.Type)
		params := types.NewTuple(paramInt)
		results := types.NewTuple(paramObjPtr)
		p.pyLongFromInt64 = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyLongFromInt64
}

// func(uint64) *Object
func (p Program) tyLongFromUint64() *types.Signature {
	if p.pyLongFromUint64 == nil {
		paramObjPtr := p.paramObjPtr()
		paramInt := types.NewParam(token.NoPos, nil, "", p.Uint64().raw.Type)
		params := types.NewTuple(paramInt)
		results := types.NewTuple(paramObjPtr)
		p.pyLongFromUint64 = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyLongFromUint64
}

// func(*Object, ...)
func (p Program) tyLoadPyModSyms() *types.Signature {
	if p.loadPyModS == nil {
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, VArg())
		p.loadPyModS = types.NewSignatureType(nil, nil, nil, params, nil, true)
	}
	return p.loadPyModS
}

// func(*char) *Object
func (p Program) tyPyUnicodeFromString() *types.Signature {
	if p.pyUniStr == nil {
		charPtr := types.NewPointer(types.Typ[types.Int8])
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "", charPtr))
		results := types.NewTuple(p.paramObjPtr())
		p.pyUniStr = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyUniStr
}

// func(*char, int) *Object
func (p Program) tyPyUnicodeFromStringAndSize() *types.Signature {
	if p.pyUniFromStrAndSize == nil {
		charPtr := types.NewPointer(types.Typ[types.Int8])
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "", charPtr), types.NewParam(token.NoPos, nil, "", types.Typ[types.Int]))
		results := types.NewTuple(p.paramObjPtr())
		p.pyUniFromStrAndSize = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyUniFromStrAndSize
}

// func(float64, float64) *Object
func (p Program) tyPyComplexFromDoubles() *types.Signature {
	if p.pyComplexFromDbs == nil {
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Float64]), types.NewParam(token.NoPos, nil, "", types.Typ[types.Float64]))
		results := types.NewTuple(p.paramObjPtr())
		p.pyComplexFromDbs = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyComplexFromDbs
}

// func(*char, int) *Object
func (p Program) tyPyByteArrayFromStringAndSize() *types.Signature {
	if p.pyBytesFromStrAndSize == nil {
		charPtr := types.NewPointer(types.Typ[types.Int8])
		params := types.NewTuple(types.NewParam(token.NoPos, nil, "", charPtr), types.NewParam(token.NoPos, nil, "", types.Typ[types.Int]))
		results := types.NewTuple(p.paramObjPtr())
		p.pyBytesFromStrAndSize = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.pyBytesFromStrAndSize
}

// func(*Objecg, *char) *Object
func (p Program) tyGetAttrString() *types.Signature {
	if p.getAttrStr == nil {
		charPtr := types.NewPointer(types.Typ[types.Int8])
		paramObjPtr := p.paramObjPtr()
		params := types.NewTuple(paramObjPtr, types.NewParam(token.NoPos, nil, "", charPtr))
		results := types.NewTuple(paramObjPtr)
		p.getAttrStr = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.getAttrStr
}

// PyNewModVar creates a new global variable for a Python module.
func (p Package) PyNewModVar(name string, doInit bool) Global {
	if v, ok := p.pymods[name]; ok {
		return v
	}
	prog := p.Prog
	objPtr := prog.PyObjectPtrPtr().raw.Type
	g := p.NewVar(name, objPtr, InC)
	if doInit {
		g.InitNil()
		g.impl.SetLinkage(llvm.LinkOnceAnyLinkage)
	}
	p.pymods[name] = g
	return g
}

// PyImportMod imports a Python module.
func (b Builder) PyImportMod(path string) Expr {
	fnImp := b.Pkg.pyFunc("PyImport_ImportModule", b.Prog.tyImportPyModule())
	return b.Call(fnImp, b.CStr(path))
}

// PyLoadModSyms loads python objects from specified module.
func (b Builder) PyLoadModSyms(modName string, objs ...PyObjRef) Expr {
	pkg := b.Pkg
	fnLoad := pkg.pyFunc("llgoLoadPyModSyms", b.Prog.tyLoadPyModSyms())
	modPtr := pkg.PyNewModVar(modName, false).Expr
	mod := b.Load(modPtr)
	args := make([]Expr, 1, len(objs)*2+2)
	args[0] = mod
	nbase := len(modName) + 1
	for _, o := range objs {
		fullName := o.impl.Name()
		name := fullName[nbase:]
		args = append(args, b.CStr(name))
		args = append(args, o.Expr)
	}
	prog := b.Prog
	args = append(args, prog.Nil(prog.CStr()))
	return b.Call(fnLoad, args...)
}

func (b Builder) pyCall(fn Expr, args []Expr) (ret Expr) {
	prog := b.Prog
	pkg := b.Pkg
	fn = b.Load(fn)
	sig := fn.raw.Type.(*types.Signature)
	params := sig.Params()
	n := params.Len()
	switch n {
	case 0:
		call := pkg.pyFunc("PyObject_CallNoArgs", prog.tyCallNoArgs())
		ret = b.Call(call, fn)
	case 1:
		if !sig.Variadic() {
			call := pkg.pyFunc("PyObject_CallOneArg", prog.tyCallOneArg())
			return b.Call(call, fn, args[0])
		}
		fallthrough
	default:
		call := pkg.pyFunc("PyObject_CallFunctionObjArgs", prog.tyCallFunctionObjArgs())
		n = len(args)
		callargs := make([]Expr, n+2)
		callargs[0] = fn
		copy(callargs[1:], args)
		callargs[n+1] = prog.Nil(prog.PyObjectPtr())
		ret = b.Call(call, callargs...)
	}
	return
}

// PyNewList(n uintptr) *Object
func (b Builder) PyNewList(n Expr) (ret Expr) {
	prog := b.Prog
	fn := b.Pkg.pyFunc("PyList_New", prog.tyNewList())
	return b.Call(fn, n)
}

// PyListSetItem(list *Object, index uintptr, item *Object) c.Int
func (b Builder) PyListSetItem(list, index, item Expr) (ret Expr) {
	prog := b.Prog
	fn := b.Pkg.pyFunc("PyList_SetItem", prog.tyListSetItem())
	return b.Call(fn, list, index, item)
}

// PyList(args ...Expr) *Object
func (b Builder) PyList(args ...Expr) (ret Expr) {
	prog := b.Prog
	n := len(args)
	uintPtr := prog.Uintptr()
	list := b.PyNewList(prog.IntVal(uint64(n), uintPtr))
	for i, arg := range args {
		b.PyListSetItem(list, prog.IntVal(uint64(i), uintPtr), b.PyVal(arg))
	}
	return list
}

// PyNewTuple(n int) *Object
func (b Builder) PyNewTuple(n Expr) (ret Expr) {
	prog := b.Prog
	fn := b.Pkg.pyFunc("PyTuple_New", prog.tyNewTuple())
	return b.Call(fn, n)
}

// PyTupleSetItem(list *Object, index uintptr, item *Object) c.Int
func (b Builder) PyTupleSetItem(list, index, item Expr) (ret Expr) {
	prog := b.Prog
	fn := b.Pkg.pyFunc("PyTuple_SetItem", prog.tyTupleSetItem())
	return b.Call(fn, list, index, item)
}

// PyTuple(args ...Expr) *Object
func (b Builder) PyTuple(args ...Expr) (ret Expr) {
	prog := b.Prog
	n := len(args)
	uintPtr := prog.Uintptr()
	list := b.PyNewTuple(prog.IntVal(uint64(n), uintPtr))
	for i, arg := range args {
		b.PyTupleSetItem(list, prog.IntVal(uint64(i), uintPtr), b.PyVal(arg))
	}
	return list
}

// PyVal(v any) *Object
func (b Builder) PyVal(v Expr) (ret Expr) {
	switch t := v.raw.Type.Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			return b.PyBool(v)
		case types.Float32:
			typ := b.Prog.Float64()
			return b.PyFloat(Expr{castFloat(b, v.impl, typ), typ})
		case types.Float64:
			return b.PyFloat(v)
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			typ := b.Prog.Int64().ll
			if b.Prog.td.TypeAllocSize(v.ll) < b.Prog.td.TypeAllocSize(typ) {
				v.impl = llvm.CreateSExt(b.impl, v.impl, typ)
				v.ll = typ
			}
			return b.PyInt64(v)
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
			typ := b.Prog.Uint64().ll
			if b.Prog.td.TypeAllocSize(v.ll) < b.Prog.td.TypeAllocSize(typ) {
				v.impl = llvm.CreateZExt(b.impl, v.impl, typ)
				v.ll = typ
			}
			return b.PyUint64(v)
		case types.String:
			return b.PyStrExpr(v)
		case types.Complex64:
			return b.PyComplex64(v)
		case types.Complex128:
			return b.PyComplex128(v)
		case types.UnsafePointer:
			typ := b.Prog.Uint64()
			return b.PyUint64(Expr{llvm.CreatePtrToInt(b.impl, v.impl, typ.ll), typ})
		}
	case *types.Slice:
		if elem, ok := t.Elem().Underlying().(*types.Basic); ok && elem.Kind() == types.Byte {
			return b.PyByteArray(v)
		}
	case *types.Array:
		if elem, ok := t.Elem().Underlying().(*types.Basic); ok && elem.Kind() == types.Byte {
			return b.PyBytes(v)
		}
	case *types.Pointer:
		if v.Type == b.Prog.PyObjectPtr() {
			return v
		}
		if v.kind == vkPyFuncRef {
			return b.Load(v)
		}
		typ := b.Prog.Uint64()
		return b.PyUint64(Expr{llvm.CreatePtrToInt(b.impl, v.impl, typ.ll), typ})
	}
	panic("PyVal: todo " + v.raw.Type.String())
}

// PyBool(bVal bool) *Object
func (b Builder) PyBool(bVal Expr) (ret Expr) {
	fn := b.Pkg.pyFunc("PyBool_FromLong", b.Prog.tyBoolFromLong())
	typ := b.Prog.Int32()
	return b.Call(fn, Expr{castInt(b, bVal.impl, bVal.Type, typ), typ})
}

// PyFloat(fltVal float64) *Object
func (b Builder) PyFloat(fltVal Expr) (ret Expr) {
	fn := b.Pkg.pyFunc("PyFloat_FromDouble", b.Prog.tyFloatFromDouble())
	return b.Call(fn, fltVal)
}

// PyInt64(val int64) *Object
func (b Builder) PyInt64(intVal Expr) (ret Expr) {
	fn := b.Pkg.pyFunc("PyLong_FromLongLong", b.Prog.tyLongFromInt64())
	return b.Call(fn, intVal)
}

// PyUint64(val uint64) *Object
func (b Builder) PyUint64(uintVal Expr) (ret Expr) {
	fn := b.Pkg.pyFunc("PyLong_FromUnsignedLongLong", b.Prog.tyLongFromUint64())
	return b.Call(fn, uintVal)
}

// PyStr returns a py-style string constant expression.
func (b Builder) PyStr(v string) Expr {
	fn := b.Pkg.pyFunc("PyUnicode_FromString", b.Prog.tyPyUnicodeFromString())
	return b.Call(fn, b.CStr(v))
}

// PyStrExpr(str string) *Object
func (b Builder) PyStrExpr(v Expr) Expr {
	fn := b.Pkg.pyFunc("PyUnicode_FromStringAndSize", b.Prog.tyPyUnicodeFromStringAndSize())
	return b.Call(fn, b.StringData(v), b.StringLen(v))
}

// PyComplex128(val complex128) *Object
func (b Builder) PyComplex128(v Expr) Expr {
	fn := b.Pkg.pyFunc("PyComplex_FromDoubles", b.Prog.tyPyComplexFromDoubles())
	return b.Call(fn, b.getField(v, 0), b.getField(v, 1))
}

// PyComplex64(val complex64) *Object
func (b Builder) PyComplex64(v Expr) Expr {
	fn := b.Pkg.pyFunc("PyComplex_FromDoubles", b.Prog.tyPyComplexFromDoubles())
	typ := b.Prog.Float64()
	return b.Call(fn, Expr{castFloat(b, b.getField(v, 0).impl, typ), typ}, Expr{castFloat(b, b.getField(v, 1).impl, typ), typ})
}

// PyByteArray(val []byte) *Object
func (b Builder) PyByteArray(v Expr) Expr {
	fn := b.Pkg.pyFunc("PyByteArray_FromStringAndSize", b.Prog.tyPyByteArrayFromStringAndSize())
	return b.Call(fn, b.SliceData(v), b.SliceLen(v))
}

// PyBytes(val [...]byte) *Object
func (b Builder) PyBytes(v Expr) Expr {
	fn := b.Pkg.pyFunc("PyBytes_FromStringAndSize", b.Prog.tyPyByteArrayFromStringAndSize())
	n := v.raw.Type.Underlying().(*types.Array).Len()
	typ := b.Prog.Pointer(b.Prog.Byte())
	if n == 0 {
		return b.Call(fn, Expr{llvm.ConstNull(typ.ll), typ}, b.Prog.Zero(b.Prog.Int()))
	}
	ptr := b.Alloc(v.Type, false)
	b.impl.CreateStore(v.impl, ptr.impl)
	p := llvm.CreateInBoundsGEP(b.impl, typ.ll, ptr.impl, []llvm.Value{b.Prog.Zero(b.Prog.Int()).impl})
	return b.Call(fn, Expr{p, typ}, b.Const(constant.MakeInt64(n), b.Prog.Int()))
}

// -----------------------------------------------------------------------------

type aPyGlobal struct {
	Expr
}

type PyGlobal = *aPyGlobal

// PyNewVar creates a Python variable.
func (b Builder) PyNewVar(modName, name string) PyGlobal {
	modPtr := b.Pkg.PyNewModVar(modName, false).Expr
	mod := b.Load(modPtr)
	return &aPyGlobal{pyVarExpr(mod, name)}
}

func (b Builder) pyLoad(ptr Expr) Expr {
	t := ptr.raw.Type.(*pyVarTy)
	fn := b.Pkg.pyFunc("PyObject_GetAttrString", b.Prog.tyGetAttrString())
	return b.Call(fn, t.mod, b.CStr(t.name))
}

// -----------------------------------------------------------------------------

type aPyObjRef struct {
	Expr
	Obj Global
}

// PyObjRef represents a python object reference.
type PyObjRef = *aPyObjRef

// PyNewFunc creates a new python function.
func (p Package) PyNewFunc(name string, sig *types.Signature, doInit bool) PyObjRef {
	if v, ok := p.pyobjs[name]; ok {
		return v
	}
	prog := p.Prog
	obj := p.NewVar(name, prog.PyObjectPtrPtr().RawType(), InC)
	if doInit {
		p.NeedPyInit = true
		obj.InitNil()
		obj.impl.SetLinkage(llvm.LinkOnceAnyLinkage)
	}
	ty := &aType{obj.ll, rawType{types.NewPointer(sig)}, vkPyFuncRef}
	expr := Expr{obj.impl, ty}
	ret := &aPyObjRef{expr, obj}
	p.pyobjs[name] = ret
	return ret
}

// PyObjOf returns a python object by name.
func (p Package) PyObjOf(name string) PyObjRef {
	return p.pyobjs[name]
}

func (p Package) pyHasModSyms() bool {
	return len(p.pyobjs) > 0
}

// pyLoadModSyms loads module symbols used in this package.
func (p Package) pyLoadModSyms(b Builder) {
	objs := p.pyobjs
	names := make([]string, 0, len(objs))
	for name := range objs {
		names = append(names, name)
	}
	sort.Strings(names)

	mods := make(map[string][]PyObjRef)
	modNames := make([]string, 0, 8)
	lastMod := ""
	for _, name := range names {
		modName := modOf(name)
		mods[modName] = append(mods[modName], objs[name])
		if modName != lastMod {
			modNames = append(modNames, modName)
			lastMod = modName
		}
	}

	for _, modName := range modNames {
		objs := mods[modName]
		b.PyLoadModSyms(modName, objs...)
	}
}

func modOf(name string) string {
	if pos := strings.LastIndexByte(name, '.'); pos > 0 {
		return name[:pos]
	}
	panic("unreachable")
}

// -----------------------------------------------------------------------------
