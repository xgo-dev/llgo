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
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// unsafeEface(t *abi.Type, data unsafe.Pointer) Eface
func (b Builder) unsafeEface(t, data llvm.Value) llvm.Value {
	return b.aggregateValue(b.Prog.rtType("Eface"), t, data).impl
}

// unsafeIface(itab *runtime.Itab, data unsafe.Pointer) Eface
func (b Builder) unsafeIface(itab, data llvm.Value) llvm.Value {
	return b.aggregateValue(b.Prog.rtType("Iface"), itab, data).impl
}

// func NewItab(tintf *InterfaceType, typ *Type) *runtime.Itab
func (b Builder) newItab(tintf, typ Expr) Expr {
	return b.Call(b.Pkg.rtFunc("NewItab"), tintf, typ)
}

func (b Builder) staticItab(rawIntf *types.Interface, concrete types.Type, tintf, typ Expr) (Expr, bool) {
	prog := b.Prog
	if !prog.enableGoGlobalDCE || !prog.enableLTOPluginMarker ||
		rawIntf.NumMethods() == 0 || concrete == nil {
		return Expr{}, false
	}
	if !types.AssignableTo(concrete, rawIntf) {
		return Expr{}, false
	}
	mset := types.NewMethodSet(concrete)
	methods := make([]*types.Selection, rawIntf.NumMethods())
	for i := range methods {
		im := rawIntf.Method(i)
		method := mset.Lookup(im.Pkg(), im.Name())
		if method == nil || prog.isNoInterfaceMethod(method.Obj().(*types.Func)) {
			return Expr{}, false
		}
		methods[i] = method
	}

	intfName, _ := prog.abi.TypeName(rawIntf)
	typeName, _ := prog.abi.TypeName(concrete)
	sum := sha256.Sum256([]byte(intfName + "\x00" + typeName))
	name := "_llgo_itab$" + base64.RawURLEncoding.EncodeToString(sum[:])
	if global := b.Pkg.VarOf(name); global != nil {
		return Expr{global.impl, prog.Pointer(prog.rtType("Itab"))}, true
	}

	ptr := prog.VoidPtr()
	funArray := prog.rawType(types.NewArray(ptr.RawType(), int64(len(methods))))
	staticType := prog.rawType(types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "inter", ptr.RawType()),
		types.NewVar(token.NoPos, nil, "typ", ptr.RawType()),
		types.NewVar(token.NoPos, nil, "hash", types.Typ[types.Uint32]),
		types.NewVar(token.NoPos, nil, "fun", funArray.RawType()),
	}, nil))
	global := b.Pkg.NewVarEx(name, prog.Pointer(staticType))
	funcs := make([]llvm.Value, len(methods))
	for i, method := range methods {
		funcs[i], _ = b.abiMethodFuncs(concrete, method)
	}
	hashBytes := sha256.Sum256([]byte(typeName))
	hash := binary.LittleEndian.Uint32(hashBytes[:4])
	global.impl.SetInitializer(prog.constStructValue(staticType, []llvm.Value{
		tintf.impl,
		typ.impl,
		prog.IntVal(uint64(hash), prog.Uint32()).impl,
		llvm.ConstArray(prog.storageType(ptr), func() []llvm.Value {
			stored := make([]llvm.Value, len(funcs))
			for i, fn := range funcs {
				stored[i] = prog.toStorageConstant(ptr, fn)
			}
			return stored
		}()),
	}))
	global.impl.SetGlobalConstant(true)
	b.Pkg.setODRLinkage(global.impl, llvm.WeakODRLinkage)

	// Describe each function slot with private LLGo metadata. The template is a
	// compile-time certificate, not a runtime vtable, so it must not participate
	// in LLVM's type-test candidate sets before the plugin consumes it.
	slotKind := prog.ctx.MDKindID("llgo.static.itab.slot")
	funOffset := uint64(prog.td.ElementOffset(staticType.ll, 3))
	stride := uint64(prog.td.TypeAllocSize(prog.storageType(ptr)))
	interfaceTypeID := prog.interfaceCapabilityKey(rawIntf)
	for i := range methods {
		offset := funOffset + uint64(i)*stride
		typeID := interfaceMethodCapabilityKeyFromID(interfaceTypeID, i)
		node := prog.ctx.MDNode([]llvm.Metadata{
			llvm.ConstInt(prog.Int64().ll, offset, false).ConstantAsMetadata(),
			prog.ctx.MDString(typeID),
		})
		global.impl.AddMetadata(slotKind, node)
	}
	// Keep the otherwise-dormant template through package optimization without
	// perturbing function IR. The LTO plugin removes this compiler.used entry
	// after using the template as a compile-time devirtualization certificate.
	b.Pkg.markLLVMUsed(global.impl)
	return Expr{global.impl, prog.Pointer(prog.rtType("Itab"))}, true
}

func (b Builder) unsafeInterface(rawIntf *types.Interface, concrete types.Type, t Expr, data llvm.Value) llvm.Value {
	if rawIntf.Empty() {
		return b.unsafeEface(t.impl, data)
	}
	tintf := b.abiType(rawIntf)
	// Emit a constant template for LTO analysis. Keep the runtime NewItab call
	// even after devirtualization so dynamically-created interfaces continue to
	// share the runtime's canonical itab pointer. Every template disappears
	// before GlobalDCE.
	b.staticItab(rawIntf, concrete, tintf, t)
	itab := b.newItab(tintf, t)
	return b.unsafeIface(itab.impl, data)
}

func iMethodOf(rawIntf *types.Interface, method *types.Func) int {
	id := types.Id(method.Pkg(), method.Name())
	n := rawIntf.NumMethods()
	for i := 0; i < n; i++ {
		m := rawIntf.Method(i)
		if types.Id(m.Pkg(), m.Name()) == id {
			return i
		}
	}
	return -1
}

// Imethod returns closure of an interface method.
func (b Builder) Imethod(intf Expr, method *types.Func) Expr {
	return b.imethod(intf, method, false)
}

// ImethodWithRecoverToken returns an interface method invocation whose code
// pointer may also be used as recover-frame bookkeeping data.
func (b Builder) ImethodWithRecoverToken(intf Expr, method *types.Func) Expr {
	return b.imethod(intf, method, true)
}

func (b Builder) imethod(intf Expr, method *types.Func, recoverToken bool) Expr {
	prog := b.Prog
	intfType := types.Unalias(intf.raw.Type)
	patchedIntfType := prog.patch(intfType)
	rawIntf := patchedIntfType.Underlying().(*types.Interface)
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
	i := iMethodOf(rawIntf, method)
	b.recordUseIfaceMethod(rawIntf, i)
	data := b.InlineCall(b.Pkg.rtFunc("IfacePtrData"), intf)
	impl := intf.impl
	itab := Expr{b.faceItab(impl), prog.VoidPtrPtr()}
	pfn := b.Advance(itab, prog.IntVal(uint64(i+3), prog.Int()))
	var fn Expr
	if prog.enableGoGlobalDCE && !recoverToken {
		fnType := prog.Elem(pfn.Type)
		fn = Expr{
			prog.interfaceMethodCheckedLoad(b.impl, pfn.impl, rawIntf, i),
			fnType,
		}
	} else {
		fn = b.Load(pfn)
		if prog.enableGoGlobalDCE {
			// A type.checked.load result is a virtual-call capability, not a
			// general-purpose pointer. Recover bookkeeping needs the same code
			// address as ordinary data, so retain the capability check while
			// carrying the raw itab load into the transient invocation pair.
			prog.interfaceMethodCheckedLoad(b.impl, pfn.impl, rawIntf, i)
		}
	}
	// This is a transient interface invocation pair, not a first-class
	// funcval. The method receiver remains an ordinary ABI parameter.
	tmethod := &aType{tclosure.ll, tclosure.raw, vkIfaceMethod}
	ret := b.aggregateValue(tmethod, fn.impl, data.impl)
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
	b.recordUseIface(typ)
	tabi := b.abiType(typ.raw.Type)
	if !directIfaceType(typ.raw.Type) {
		vptr := b.AllocU(typ)
		b.Store(vptr, x)
		return Expr{b.unsafeInterface(rawIntf, typ.raw.Type, tabi, vptr.impl), tinter}
	}
	kind, _, lvl := abi.DataKindOf(typ.raw.Type, 0, prog.is32Bits)
	switch kind {
	case abi.Indirect:
		vptr := b.AllocU(typ)
		b.Store(vptr, x)
		return Expr{b.unsafeInterface(rawIntf, typ.raw.Type, tabi, vptr.impl), tinter}
	}
	value := x
	if lvl > 0 {
		for range lvl {
			switch value.raw.Type.Underlying().(type) {
			case *types.Struct:
				value = b.getField(value, 0)
			case *types.Array:
				telem := prog.Index(value.Type)
				impl := llvm.CreateExtractValue(b.impl, value.impl, 0)
				value = Expr{b.fromStorageValue(telem, impl), telem}
			default:
				panic("direct interface wrapper is not a singleton aggregate")
			}
		}
	}
	ximpl := value.impl
	var u llvm.Value
	switch kind {
	case abi.Pointer:
		return Expr{b.unsafeInterface(rawIntf, typ.raw.Type, tabi, ximpl), tinter}
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
	return Expr{b.unsafeInterface(rawIntf, typ.raw.Type, tabi, data), tinter}
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

	b.recordUseIface(typ)
	vptr := b.AllocU(typ)
	dst := b.Convert(prog.VoidPtr(), vptr)
	src := b.Convert(prog.VoidPtr(), ptr)
	b.Call(b.Pkg.rtFunc("Typedmemmove"), tabi, dst, src)
	return Expr{b.unsafeInterface(rawIntf, typ.raw.Type, tabi, vptr.impl), tinter}
}

func (b Builder) recordUseIface(typ Type) {
	if mb := b.Pkg.metaBuilder; mb != nil {
		if _, ok := types.Unalias(typ.raw.Type).Underlying().(*types.Interface); !ok {
			typeName, _ := b.Prog.abi.TypeName(typ.raw.Type)
			mb.AddIfaceUse(mb.Sym(b.Func.Name()), mb.Sym(typeName))
		}
	}
}

func (b Builder) recordUseIfaceMethod(rawIntf *types.Interface, methodIndex int) {
	if mb := b.Pkg.metaBuilder; mb != nil {
		intfSymName, _ := b.Prog.abi.TypeName(rawIntf)
		intfSym := mb.Sym(intfSymName)
		b.recordInterfaceInfo(rawIntf, intfSymName)
		mb.AddIfaceMethodUse(mb.Sym(b.Func.Name()), intfSym, uint32(methodIndex))
	}
}

func (b Builder) recordInterfaceInfo(t *types.Interface, typeName string) {
	mb := b.Pkg.metaBuilder
	if mb == nil {
		return
	}
	prog := b.Prog
	intfSym := mb.Sym(typeName)
	for i := 0; i < t.NumMethods(); i++ {
		f := t.Method(i)
		ftypName, _ := prog.abi.TypeName(funcType(prog, f.Type()))
		mb.AddIfaceMethod(intfSym, abiMethodName(f), mb.Sym(ftypName))
	}
}

func (b Builder) valFromData(typ Type, data llvm.Value) Expr {
	prog := b.Prog
	if !directIfaceType(typ.raw.Type) {
		stored := llvm.CreateLoad(b.impl, prog.storageType(typ), data)
		return Expr{b.fromStorageValue(typ, stored), typ}
	}
	kind, real, lvl := abi.DataKindOf(typ.raw.Type, 0, prog.is32Bits)
	switch kind {
	case abi.Indirect:
		stored := llvm.CreateLoad(b.impl, prog.storageType(typ), data)
		return Expr{b.fromStorageValue(typ, stored), typ}
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
		if int(prog.SizeOf(t)) != prog.GoWordSize() {
			x = castInt(b, x, prog.Uintptr(), prog.Int32())
		}
		return b.buildVal(typ, llvm.CreateBitCast(b.impl, x, t.ll), lvl)
	}
	panic("todo")
}

func (b Builder) buildVal(typ Type, val llvm.Value, lvl int) Expr {
	if lvl == 0 {
		return Expr{val, typ}
	}
	switch t := typ.raw.Type.Underlying().(type) {
	case *types.Struct:
		telem := b.Prog.rawType(t.Field(0).Type())
		elem := b.buildVal(telem, val, lvl-1)
		return b.aggregateValue(typ, elem.impl)
	case *types.Array:
		telem := b.Prog.rawType(t.Elem())
		elem := b.buildVal(telem, val, lvl-1)
		return b.aggregateValue(typ, elem.impl)
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
			val = func() Expr { return Expr{b.unsafeInterface(rawIntf, nil, tx, b.faceData(x.impl)), assertedTyp} }
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
				valTrue := b.aggregateValue(t, val().impl, prog.BoolVal(true).impl)
				b.Jump(blks[2])
				return valTrue
			}
			zero := prog.Zero(assertedTyp)
			valFalse := b.aggregateValue(t, zero.impl, prog.BoolVal(false).impl)
			b.Jump(blks[2])
			return valFalse
		})
		b.SetBlockEx(blks[2], AtEnd, false)
		b.blk.last = blks[2].last
		return phi.Expr
	}
	blks := b.Func.MakeBlocks(2)
	b.If(eq, blks[0], blks[1])
	b.SetBlockEx(blks[1], AtEnd, false)
	var source Expr
	if rawIntf, ok := x.RawType().Underlying().(*types.Interface); ok && rawIntf.NumMethods() > 0 {
		source = b.abiType(x.RawType())
	} else {
		source = b.Prog.Nil(b.Prog.AbiTypePtr())
	}
	b.Call(b.Pkg.rtFunc("PanicTypeAssert"), source, tx, tabi)
	b.Unreachable()
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
	return Expr{b.unsafeInterface(rawIntf, nil, tabi, data), typ}
}

// -----------------------------------------------------------------------------

// InterfaceData returns the data pointer of an interface.
func (b Builder) InterfaceData(x Expr) Expr {
	dbgInstrf("InterfaceData %v\n", x.impl)
	return Expr{b.faceData(x.impl), b.Prog.VoidPtr()}
}

func (b Builder) faceData(x llvm.Value) llvm.Value {
	stored := llvm.CreateExtractValue(b.impl, x, 1)
	return b.fromStorageValue(b.Prog.VoidPtr(), stored)
}

func (b Builder) faceItab(x llvm.Value) llvm.Value {
	stored := llvm.CreateExtractValue(b.impl, x, 0)
	return b.fromStorageValue(b.Prog.VoidPtrPtr(), stored)
}

func (b Builder) faceAbiType(x Expr) Expr {
	if x.kind == vkIface {
		return b.InlineCall(b.Pkg.rtFunc("IfaceType"), x)
	}
	typ := b.Prog.AbiTypePtr()
	stored := llvm.CreateExtractValue(b.impl, x.impl, 0)
	return Expr{b.fromStorageValue(typ, stored), typ}
}

// -----------------------------------------------------------------------------
