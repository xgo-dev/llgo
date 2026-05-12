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

	"github.com/goplus/llgo/internal/metadata"
	"github.com/goplus/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// unsafeEface(t *abi.Type, data unsafe.Pointer) Eface
func (b Builder) unsafeEface(t, data llvm.Value) llvm.Value {
	return aggregateValue(b.impl, b.Prog.rtEface(), t, data)
}

// unsafeIface(itab *runtime.Itab, data unsafe.Pointer) Eface
func (b Builder) unsafeIface(itab, data llvm.Value) llvm.Value {
	return aggregateValue(b.impl, b.Prog.rtIface(), itab, data)
}

// func NewItab(tintf *InterfaceType, typ *Type) *runtime.Itab
func (b Builder) newItab(tintf, typ Expr) Expr {
	return b.Call(b.Pkg.rtFunc("NewItab"), tintf, typ)
}

func (b Builder) unsafeInterface(rawIntf *types.Interface, t Expr, data llvm.Value) llvm.Value {
	if rawIntf.Empty() {
		return b.unsafeEface(t.impl, data)
	}
	tintf := b.abiType(rawIntf)
	itab := b.newItab(tintf, t)
	return b.unsafeIface(itab.impl, data)
}

func iMethodOf(rawIntf *types.Interface, name string) int {
	n := rawIntf.NumMethods()
	for i := 0; i < n; i++ {
		m := rawIntf.Method(i)
		if m.Name() == name {
			// TODO(xsw): check signature
			return i
		}
	}
	return -1
}

// emitIfaceMethodUse records an interface method call for metadata.
func (b Builder) emitIfaceMethodUse(ownerName, intfTypeName string, method *types.Func) {
	mb := b.Pkg.MetaBuilder()
	if mb == nil {
		return
	}
	prog := b.Prog
	mtypName, _ := prog.abi.TypeName(funcType(prog, method.Type()))
	mb.AddUseIfaceMethod(mb.String(ownerName), []metadata.IfaceMethodDemand{{
		Target: mb.String(intfTypeName),
		Sig: metadata.MethodSig{
			Name:  mb.String(mthName(method)),
			MType: mb.String(mtypName),
		},
	}})
}

// emitInterfaceInfo records the complete method set of an interface.
func (b Builder) emitInterfaceInfo(intfTypeName string, rawIntf *types.Interface) {
	mb := b.Pkg.MetaBuilder()
	if mb == nil {
		return
	}
	prog := b.Prog
	n := rawIntf.NumMethods()
	methods := make([]metadata.MethodSig, 0, n)
	for i := 0; i < n; i++ {
		im := rawIntf.Method(i)
		imtypName, _ := prog.abi.TypeName(funcType(prog, im.Type()))
		methods = append(methods, metadata.MethodSig{
			Name:  mb.String(mthName(im)),
			MType: mb.String(imtypName),
		})
	}
	mb.AddIfaceEntry(mb.String(intfTypeName), methods)
}

// emitUseIface records that a concrete type enters the interface domain.
func (b Builder) emitUseIface(ownerName, typeName string) {
	if mb := b.Pkg.MetaBuilder(); mb != nil {
		mb.AddUseIface(mb.String(ownerName), []metadata.Symbol{mb.String(typeName)})
	}
}

// Imethod returns closure of an interface method.
func (b Builder) Imethod(intf Expr, method *types.Func) Expr {
	prog := b.Prog
	rawIntf := intf.raw.Type.Underlying().(*types.Interface)
	sig := method.Type().(*types.Signature)
	if sig.Recv() == nil && sig.Params().Len() > 0 {
		pt := types.Unalias(sig.Params().At(0).Type())
		if types.Identical(pt, rawIntf) {
			n := sig.Params().Len()
			vars := make([]*types.Var, n-1)
			for i := 1; i < n; i++ {
				vars[i-1] = sig.Params().At(i)
			}
			sig = types.NewSignatureType(nil, nil, nil, types.NewTuple(vars...), sig.Results(), sig.Variadic())
		}
	}
	tclosure := prog.Type(sig, InGo)
	i := iMethodOf(rawIntf, method.Name())
	ownerName := b.Func.impl.Name()
	intfTypeName, _ := prog.abi.TypeName(intf.raw.Type)
	b.emitIfaceMethodUse(ownerName, intfTypeName, method)
	b.emitInterfaceInfo(intfTypeName, rawIntf)
	data := b.InlineCall(b.Pkg.rtFunc("IfacePtrData"), intf)
	impl := intf.impl
	itab := Expr{b.faceItab(impl), prog.VoidPtrPtr()}
	pfn := b.Advance(itab, prog.IntVal(uint64(i+3), prog.Int()))
	fn := b.Load(pfn)
	ret := b.aggregateValue(tclosure, fn.impl, data.impl)
	return ret
}

// -----------------------------------------------------------------------------

// MakeInterface constructs an instance of an interface type from a
// value of a concrete type.
//
// Use Program.MethodSets.MethodSet(X.Type()) to find the method-set
// of X, and Program.MethodValue(m) to find the implementation of a method.
//
// To construct the zero value of an interface type T, use:
//
//	NewConst(constant.MakeNil(), T, pos)
//
// Example printed form:
//
//	t1 = make interface{} <- int (42:int)
//	t2 = make Stringer <- t0
func (b Builder) MakeInterface(tinter Type, x Expr) (ret Expr) {
	rawIntf := tinter.raw.Type.Underlying().(*types.Interface)
	dbgInstrf("MakeInterface %v, %v\n", rawIntf, x.impl)
	if x.kind == vkFuncDecl {
		typ := b.Prog.Type(x.raw.Type, InGo)
		x = checkExpr(x, typ.raw.Type, b)
	}
	prog := b.Prog
	typ := x.Type
	tabi := b.abiType(typ.raw.Type)
	if _, ok := typ.raw.Type.Underlying().(*types.Interface); !ok {
		ownerName := b.Func.impl.Name()
		typeName, _ := prog.abi.TypeName(typ.raw.Type)
		b.emitUseIface(ownerName, typeName)
	}
	if !directIfaceType(typ.raw.Type) {
		vptr := b.AllocU(typ)
		b.Store(vptr, x)
		return Expr{b.unsafeInterface(rawIntf, tabi, vptr.impl), tinter}
	}
	kind, _, lvl := abi.DataKindOf(typ.raw.Type, 0, prog.is32Bits)
	switch kind {
	case abi.Indirect:
		vptr := b.AllocU(typ)
		b.Store(vptr, x)
		return Expr{b.unsafeInterface(rawIntf, tabi, vptr.impl), tinter}
	}
	ximpl := x.impl
	if lvl > 0 {
		ximpl = extractVal(b.impl, ximpl, lvl)
	}
	var u llvm.Value
	switch kind {
	case abi.Pointer:
		return Expr{b.unsafeInterface(rawIntf, tabi, ximpl), tinter}
	case abi.Integer:
		tu := prog.Uintptr()
		u = llvm.CreateIntCast(b.impl, ximpl, tu.ll)
	case abi.BitCast:
		tu := prog.Uintptr()
		if b.Prog.td.TypeAllocSize(typ.ll) < b.Prog.td.TypeAllocSize(tu.ll) {
			u = llvm.CreateBitCast(b.impl, ximpl, prog.Uint32().ll)
		} else {
			u = llvm.CreateBitCast(b.impl, ximpl, tu.ll)
		}
	default:
		panic("todo")
	}
	data := llvm.CreateIntToPtr(b.impl, u, prog.tyVoidPtr())
	return Expr{b.unsafeInterface(rawIntf, tabi, data), tinter}
}

func (b Builder) MakeInterfaceFromPtr(tinter Type, ptr Expr) (ret Expr) {
	rawIntf := tinter.raw.Type.Underlying().(*types.Interface)
	prog := b.Prog
	b.AssertNilDeref(ptr)

	typ := prog.Elem(ptr.Type)
	tabi := b.abiType(typ.raw.Type)
	if kind, _, _ := abi.DataKindOf(typ.raw.Type, 0, prog.is32Bits); kind != abi.Indirect {
		return b.MakeInterface(tinter, b.Load(ptr))
	}

	vptr := b.AllocU(typ)
	dst := b.Convert(prog.VoidPtr(), vptr)
	src := b.Convert(prog.VoidPtr(), ptr)
	b.Call(b.Pkg.rtFunc("Typedmemmove"), tabi, dst, src)
	return Expr{b.unsafeInterface(rawIntf, tabi, vptr.impl), tinter}
}

func (b Builder) valFromData(typ Type, data llvm.Value) Expr {
	prog := b.Prog
	if !directIfaceType(typ.raw.Type) {
		impl := b.impl
		tll := typ.ll
		tptr := llvm.PointerType(tll, 0)
		ptr := llvm.CreatePointerCast(impl, data, tptr)
		return Expr{llvm.CreateLoad(impl, tll, ptr), typ}
	}
	kind, real, lvl := abi.DataKindOf(typ.raw.Type, 0, prog.is32Bits)
	switch kind {
	case abi.Indirect:
		impl := b.impl
		tll := typ.ll
		tptr := llvm.PointerType(tll, 0)
		ptr := llvm.CreatePointerCast(impl, data, tptr)
		return Expr{llvm.CreateLoad(impl, tll, ptr), typ}
	}
	t := typ
	if lvl > 0 {
		t = prog.rawType(real)
	}
	switch kind {
	case abi.Pointer:
		return b.buildVal(typ, data, lvl)
	case abi.Integer:
		x := castUintptr(b, data, prog.VoidPtr(), prog.Uintptr())
		return b.buildVal(typ, castInt(b, x, prog.Uintptr(), t), lvl)
	case abi.BitCast:
		x := castUintptr(b, data, prog.VoidPtr(), prog.Uintptr())
		if int(prog.SizeOf(t)) != prog.PointerSize() {
			x = castInt(b, x, prog.Uintptr(), prog.Int32())
		}
		return b.buildVal(typ, llvm.CreateBitCast(b.impl, x, t.ll), lvl)
	}
	panic("todo")
}

func extractVal(b llvm.Builder, val llvm.Value, lvl int) llvm.Value {
	for lvl > 0 {
		// TODO(xsw): check array support
		val = llvm.CreateExtractValue(b, val, 0)
		lvl--
	}
	return val
}

func (b Builder) buildVal(typ Type, val llvm.Value, lvl int) Expr {
	if lvl == 0 {
		return Expr{val, typ}
	}
	switch t := typ.raw.Type.Underlying().(type) {
	case *types.Struct:
		telem := b.Prog.rawType(t.Field(0).Type())
		elem := b.buildVal(telem, val, lvl-1)
		return Expr{aggregateValue(b.impl, typ.ll, elem.impl), typ}
	case *types.Array:
		telem := b.Prog.rawType(t.Elem())
		elem := b.buildVal(telem, val, lvl-1)
		return Expr{aggregateValue(b.impl, typ.ll, elem.impl), typ}
	}
	panic("todo")
}

// The TypeAssert instruction tests whether interface value X has type
// AssertedType.
//
// If !CommaOk, on success it returns v, the result of the conversion
// (defined below); on failure it panics.
//
// If CommaOk: on success it returns a pair (v, true) where v is the
// result of the conversion; on failure it returns (z, false) where z
// is AssertedType's zero value.  The components of the pair must be
// accessed using the Extract instruction.
//
// If Underlying: tests whether interface value X has the underlying
// type AssertedType.
//
// If AssertedType is a concrete type, TypeAssert checks whether the
// dynamic type in interface X is equal to it, and if so, the result
// of the conversion is a copy of the value in the interface.
//
// If AssertedType is an interface, TypeAssert checks whether the
// dynamic type of the interface is assignable to it, and if so, the
// result of the conversion is a copy of the interface value X.
// If AssertedType is a superinterface of X.Type(), the operation will
// fail iff the operand is nil.  (Contrast with ChangeInterface, which
// performs no nil-check.)
//
// Type() reflects the actual type of the result, possibly a
// 2-types.Tuple; AssertedType is the asserted type.
//
// Depending on the TypeAssert's purpose, Pos may return:
//   - the ast.CallExpr.Lparen of an explicit T(e) conversion;
//   - the ast.TypeAssertExpr.Lparen of an explicit e.(T) operation;
//   - the ast.CaseClause.Case of a case of a type-switch statement;
//   - the Ident(m).NamePos of an interface method value i.m
//     (for which TypeAssert may be used to effect the nil check).
//
// Example printed form:
//
//	t1 = typeassert t0.(int)
//	t3 = typeassert,ok t2.(T)
func (b Builder) TypeAssert(x Expr, assertedTyp Type, commaOk bool) Expr {
	dbgInstrf("TypeAssert %v, %v, %v\n", x.impl, assertedTyp.raw.Type, commaOk)
	tx := b.faceAbiType(x)
	tabi := b.abiType(assertedTyp.raw.Type)
	var eq Expr
	var val func() Expr
	if x.RawType() == assertedTyp.RawType() {
		eq = b.BinOp(token.NEQ, tx, b.Prog.Zero(b.Prog.AbiTypePtr()))
		val = func() Expr { return x }
	} else {
		if rawIntf, ok := assertedTyp.raw.Type.Underlying().(*types.Interface); ok {
			eq = b.InlineCall(b.Pkg.rtFunc("Implements"), tabi, tx)
			val = func() Expr { return Expr{b.unsafeInterface(rawIntf, tx, b.faceData(x.impl)), assertedTyp} }
		} else if assertedTyp.kind == vkClosure {
			eq = b.InlineCall(b.Pkg.rtFunc("MatchesClosure"), tabi, tx)
			val = func() Expr { return b.valFromData(assertedTyp, b.faceData(x.impl)) }
		} else {
			eq = b.BinOp(token.EQL, tx, tabi)
			val = func() Expr { return b.valFromData(assertedTyp, b.faceData(x.impl)) }
		}
	}

	if commaOk {
		prog := b.Prog
		t := prog.Struct(assertedTyp, prog.Bool())
		blks := b.Func.MakeBlocks(3)
		b.If(eq, blks[0], blks[1])

		b.SetBlockEx(blks[2], AtEnd, false)
		phi := b.Phi(t)
		phi.AddIncoming(b, blks[:2], func(i int, blk BasicBlock) Expr {
			b.SetBlockEx(blk, AtEnd, false)
			if i == 0 {
				valTrue := aggregateValue(b.impl, t.ll, val().impl, prog.BoolVal(true).impl)
				b.Jump(blks[2])
				return Expr{valTrue, t}
			}
			zero := prog.Zero(assertedTyp)
			valFalse := aggregateValue(b.impl, t.ll, zero.impl, prog.BoolVal(false).impl)
			b.Jump(blks[2])
			return Expr{valFalse, t}
		})
		b.SetBlockEx(blks[2], AtEnd, false)
		b.blk.last = blks[2].last
		return phi.Expr
	}
	blks := b.Func.MakeBlocks(2)
	b.If(eq, blks[0], blks[1])
	b.SetBlockEx(blks[1], AtEnd, false)
	b.Panic(b.MakeInterface(b.Prog.Any(), b.Str("type assertion "+x.RawType().String()+" -> "+assertedTyp.RawType().String()+" failed")))
	b.SetBlockEx(blks[0], AtEnd, false)
	b.blk.last = blks[0].last
	return val()
}

// ChangeInterface constructs a value of one interface type from a
// value of another interface type known to be assignable to it.
// This operation cannot fail.
//
// Pos() returns the ast.CallExpr.Lparen if the instruction arose from
// an explicit T(e) conversion; the ast.TypeAssertExpr.Lparen if the
// instruction arose from an explicit e.(T) operation; or token.NoPos
// otherwise.
//
// Example printed form:
//
//	t1 = change interface interface{} <- I (t0)
func (b Builder) ChangeInterface(typ Type, x Expr) (ret Expr) {
	rawIntf := typ.raw.Type.Underlying().(*types.Interface)
	tabi := b.faceAbiType(x)
	data := b.faceData(x.impl)
	return Expr{b.unsafeInterface(rawIntf, tabi, data), typ}
}

// -----------------------------------------------------------------------------

// InterfaceData returns the data pointer of an interface.
func (b Builder) InterfaceData(x Expr) Expr {
	dbgInstrf("InterfaceData %v\n", x.impl)
	return Expr{b.faceData(x.impl), b.Prog.VoidPtr()}
}

func (b Builder) faceData(x llvm.Value) llvm.Value {
	return llvm.CreateExtractValue(b.impl, x, 1)
}

func (b Builder) faceItab(x llvm.Value) llvm.Value {
	return llvm.CreateExtractValue(b.impl, x, 0)
}

func (b Builder) faceAbiType(x Expr) Expr {
	if x.kind == vkIface {
		return b.InlineCall(b.Pkg.rtFunc("IfaceType"), x)
	}
	typ := llvm.CreateExtractValue(b.impl, x.impl, 0)
	return Expr{typ, b.Prog.AbiTypePtr()}
}

// -----------------------------------------------------------------------------
