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

	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// The FieldAddr instruction yields the address of Field of *struct X.
//
// The field is identified by its index within the field list of the
// struct type of X.
//
// Dynamically, this instruction panics if X evaluates to a nil
// pointer.
//
// Type() returns a (possibly named) *types.Pointer.
//
// Example printed form:
//
//	t1 = &t0.name [#1]
func (b Builder) FieldAddr(x Expr, idx int) Expr {
	dbgInstrf("FieldAddr %v, %d\n", x.impl, idx)
	prog := b.Prog
	tstruc := prog.Elem(x.Type)
	telem := prog.Field(tstruc, idx)
	pt := prog.Pointer(telem)
	return Expr{llvm.CreateStructGEP(b.impl, tstruc.ll, x.impl, idx), pt}
}

// The Field instruction yields the value of Field of struct X.
func (b Builder) Field(x Expr, idx int) Expr {
	dbgInstrf("Field %v, %d\n", x.impl, idx)
	return b.getField(x, idx)
}

func (b Builder) getField(x Expr, idx int) Expr {
	tfld := b.Prog.Field(x.Type, idx)
	fld := llvm.CreateExtractValue(b.impl, x.impl, idx)
	return Expr{fld, tfld}
}

// -----------------------------------------------------------------------------

func (b Builder) Complex(r, i Expr) Expr {
	dbgInstrf("Complex %v, %v\n", r.impl, i.impl)
	prog := b.Prog
	var t Type
	switch kind := r.raw.Type.Underlying().(*types.Basic).Kind(); kind {
	case types.Float64:
		t = prog.Complex128()
	case types.Float32:
		t = prog.Complex64()
	}
	return b.aggregateValue(t, r.impl, i.impl)
}

// MakeString creates a new string from a C string pointer and length.
func (b Builder) MakeString(cstr Expr, n ...Expr) (ret Expr) {
	dbgInstrf("MakeString %v\n", cstr.impl)
	pkg := b.Pkg
	prog := b.Prog
	ret.Type = prog.String()
	if len(n) == 0 {
		ret.impl = b.Call(pkg.rtFunc("StringFromCStr"), cstr).impl
	} else {
		// TODO(xsw): remove Convert
		ret.impl = b.Call(pkg.rtFunc("StringFrom"), cstr, b.Convert(prog.Int(), n[0])).impl
	}
	return
}

// StringData returns the data pointer of a string.
func (b Builder) StringData(x Expr) Expr {
	dbgInstrf("StringData %v\n", x.impl)
	ptr := llvm.CreateExtractValue(b.impl, x.impl, 0)
	return Expr{ptr, b.Prog.CStr()}
}

// StringLen returns the length of a string.
func (b Builder) StringLen(x Expr) Expr {
	dbgInstrf("StringLen %v\n", x.impl)
	ptr := llvm.CreateExtractValue(b.impl, x.impl, 1)
	return Expr{ptr, b.Prog.Int()}
}

// -----------------------------------------------------------------------------

// SliceData returns the data pointer of a slice.
func (b Builder) SliceData(x Expr) Expr {
	dbgInstrf("SliceData %v\n", x.impl)
	ptr := llvm.CreateExtractValue(b.impl, x.impl, 0)
	ty := x.Type.RawType()
	tySlice := ty.Underlying().(*types.Slice)
	return Expr{ptr, b.Prog.Pointer(b.Prog.rawType(tySlice.Elem()))}
}

// SliceLen returns the length of a slice.
func (b Builder) SliceLen(x Expr) Expr {
	dbgInstrf("SliceLen %v\n", x.impl)
	ptr := llvm.CreateExtractValue(b.impl, x.impl, 1)
	return Expr{ptr, b.Prog.Int()}
}

// SliceCap returns the length of a slice cap.
func (b Builder) SliceCap(x Expr) Expr {
	dbgInstrf("SliceCap %v\n", x.impl)
	ptr := llvm.CreateExtractValue(b.impl, x.impl, 2)
	return Expr{ptr, b.Prog.Int()}
}

// -----------------------------------------------------------------------------

// The IndexAddr instruction yields the address of the element at
// index `idx` of collection `x`.  `idx` is an integer expression.
//
// The elements of maps and strings are not addressable; use Lookup (map),
// Index (string), or MapUpdate instead.
//
// Dynamically, this instruction panics if `x` evaluates to a nil *array
// pointer.
//
// Example printed form:
//
//	t2 = &t0[t1]
func (b Builder) IndexAddr(x, idx Expr) Expr {
	dbgInstrf("IndexAddr %v, %v\n", x.impl, idx.impl)
	prog := b.Prog
	telem := prog.Index(x.Type)
	pt := prog.Pointer(telem)
	switch t := x.raw.Type.Underlying().(type) {
	case *types.Slice:
		ptr := b.SliceData(x)
		max := b.SliceLen(x)
		idx = b.checkIndex(idx, max)
		indices := []llvm.Value{idx.impl}
		return Expr{llvm.CreateInBoundsGEP(b.impl, telem.ll, ptr.impl, indices), pt}
	case *types.Pointer:
		ar := t.Elem().Underlying().(*types.Array)
		max := prog.IntVal(uint64(ar.Len()), prog.Int())
		idx = b.checkIndex(idx, max)
	}
	indices := []llvm.Value{idx.impl}
	return Expr{llvm.CreateInBoundsGEP(b.impl, telem.ll, x.impl, indices), pt}
}

func isConstantInt(x Expr) (v int64, ok bool) {
	if rv := x.impl.IsAConstantInt(); !rv.IsNil() {
		v = rv.SExtValue()
		ok = true
	}
	return
}

func isConstantUint(x Expr) (v uint64, ok bool) {
	if rv := x.impl.IsAConstantInt(); !rv.IsNil() {
		v = rv.ZExtValue()
		ok = true
	}
	return
}

func checkRange(idx Expr, max Expr) (checkMin, checkMax bool) {
	if idx.kind == vkSigned {
		if v, ok := isConstantInt(idx); ok {
			if v < 0 {
				checkMin = true
			}
			if m, ok := isConstantInt(max); ok {
				if v >= m {
					checkMax = true
				}
			} else {
				checkMax = true
			}
		} else {
			checkMin = true
			checkMax = true
		}
	} else {
		if v, ok := isConstantUint(idx); ok {
			if m, ok := isConstantUint(max); ok {
				if v >= m {
					checkMax = true
				}
			} else {
				checkMax = true
			}
		} else {
			checkMax = true
		}
	}
	return
}

// check index >= 0 && index < max and size to uint
func (b Builder) checkIndex(idx Expr, max Expr) Expr {
	prog := b.Prog
	// check range
	checkMin, checkMax := checkRange(idx, max)
	// fit size
	var typ Type
	if idx.kind == vkSigned {
		typ = prog.Int()
	} else {
		typ = prog.Uint()
	}
	if prog.SizeOf(idx.Type) != prog.SizeOf(typ) {
		srcType := idx.Type
		idx.Type = typ
		idx.impl = castUintptr(b, idx.impl, srcType, typ)
	}
	// check range expr
	var check Expr
	if checkMin {
		zero := llvm.ConstInt(idx.ll, 0, false)
		check = Expr{llvm.CreateICmp(b.impl, llvm.IntSLT, idx.impl, zero), prog.Bool()}
	}
	if checkMax {
		// max is a non-negative len/cap value. Unsigned comparison is valid for
		// both signed and unsigned indexes, and signed negatives fail as large
		// unsigned values.
		r := Expr{llvm.CreateICmp(b.impl, llvm.IntUGE, idx.impl, max.impl), prog.Bool()}
		if check.IsNil() {
			check = r
		} else {
			check = Expr{b.impl.CreateOr(r.impl, check.impl, ""), prog.Bool()}
		}
	}
	if !check.IsNil() {
		b.InlineCall(b.Pkg.rtFunc("AssertIndexRange"), check)
	}
	return idx
}

// The Index instruction yields element Index of collection X, an array,
// string or type parameter containing an array, a string, a pointer to an,
// array or a slice.
//
// Example printed form:
//
//	t2 = t0[t1]
func (b Builder) Index(x, idx Expr, takeAddr func() (addr Expr, zero bool)) Expr {
	dbgInstrf("Index %v, %v\n", x.impl, idx.impl)
	prog := b.Prog
	var telem Type
	var ptr Expr
	var max Expr
	var zero bool
	switch t := x.raw.Type.Underlying().(type) {
	case *types.Basic:
		if t.Kind() != types.String {
			panic(fmt.Errorf("invalid operation: cannot index %v", t))
		}
		telem = prog.rawType(types.Typ[types.Byte])
		ptr = b.StringData(x)
		max = b.StringLen(x)
	case *types.Array:
		telem = prog.Index(x.Type)
		ptr, zero = takeAddr()
		max = prog.IntVal(uint64(t.Len()), prog.Int())
	}
	idx = b.checkIndex(idx, max)
	if zero {
		return prog.Zero(telem)
	}
	if ptr.IsNil() {
		ptr = b.Alloc(x.Type, false)
		b.impl.CreateStore(x.impl, ptr.impl)
	}
	pt := prog.Pointer(telem)
	indices := []llvm.Value{idx.impl}
	buf := Expr{llvm.CreateInBoundsGEP(b.impl, telem.ll, ptr.impl, indices), pt}
	return b.Load(buf)
}

// -----------------------------------------------------------------------------

// The Slice instruction yields a slice of an existing string, slice
// or *array X between optional integer bounds Low and High.
//
// Dynamically, this instruction panics if X evaluates to a nil *array
// pointer.
//
// Type() returns string if the type of X was string, otherwise a
// *types.Slice with the same element type as X.
//
// Example printed form:
//
//	t1 = slice t0[1:]
func (b Builder) Slice(x, low, high, max Expr) (ret Expr) {
	dbgInstrf("Slice %v, %v, %v\n", x.impl, low.impl, high.impl)
	prog := b.Prog
	var nCap Expr
	var nEltSize Expr
	var base Expr
	var lowIsNil = low.IsNil()
	if lowIsNil {
		low = prog.IntVal(0, prog.Int())
	} else {
		low = b.fitIntSize(low)
	}
	if !high.IsNil() {
		high = b.fitIntSize(high)
	}
	if !max.IsNil() {
		max = b.fitIntSize(max)
	}
	switch t := x.raw.Type.Underlying().(type) {
	case *types.Basic:
		if t.Kind() != types.String {
			panic(fmt.Errorf("invalid operation: cannot slice %v", t))
		}
		if high.IsNil() {
			high = b.StringLen(x)
		}
		ret.Type = x.Type
		ret.impl = b.InlineCall(b.Pkg.rtFunc("StringSlice"), x, low, high).impl
		return
	case *types.Slice:
		nEltSize = SizeOf(prog, prog.Index(x.Type))
		nCap = b.SliceCap(x)
		if high.IsNil() {
			high = b.SliceLen(x)
		}
		ret.Type = x.Type
		base = b.SliceData(x)
	case *types.Pointer:
		telem := t.Elem()
		switch te := telem.Underlying().(type) {
		case *types.Array:
			elem := prog.rawType(te.Elem())
			ret.Type = prog.Slice(elem)
			nEltSize = SizeOf(prog, elem)
			nCap = prog.IntVal(uint64(te.Len()), prog.Int())
			if high.IsNil() {
				if lowIsNil && max.IsNil() {
					ret.impl = b.unsafeSlice(x, nCap.impl, nCap.impl).impl
					return
				}
				high = nCap
			}
			base = x
		}
	}
	if max.IsNil() {
		max = nCap
	}
	ret.impl = b.InlineCall(b.Pkg.rtFunc("NewSlice3"), base, nEltSize, nCap, low, high, max).impl
	return
}

// SliceLit creates a new slice with the specified elements.
func (b Builder) SliceLit(t Type, elts ...Expr) Expr {
	prog := b.Prog
	telem := prog.Index(t)
	ptr := b.AllocU(telem, int64(len(elts)))
	for i, elt := range elts {
		b.Store(b.Advance(ptr, prog.Val(i)), elt)
	}
	size := llvm.ConstInt(prog.tyInt(), uint64(len(elts)), false)
	return b.unsafeSlice(ptr, size, size)
}

// The MakeSlice instruction yields a slice of length Len backed by a
// newly allocated array of length Cap.
//
// Both Len and Cap must be non-nil Values of integer type.
//
// (Alloc(types.Array) followed by Slice will not suffice because
// Alloc can only create arrays of constant length.)
//
// Type() returns a (possibly named) *types.Slice.
//
// Example printed form:
//
//	t1 = make []string 1:int t0
//	t1 = make StringSlice 1:int t0
func (b Builder) MakeSlice(t Type, len, cap Expr) (ret Expr) {
	dbgInstrf("MakeSlice %v, %v, %v\n", t.RawType(), len.impl, cap.impl)
	prog := b.Prog
	len = b.fitIntSize(len)
	cap = b.fitIntSize(cap)
	telem := prog.Index(t)
	ret = b.InlineCall(b.Pkg.rtFunc("MakeSlice"), len, cap, prog.IntVal(prog.SizeOf(telem), prog.Int()))
	ret.Type = t
	return
}

// fit size to int
func (b Builder) fitIntSize(n Expr) Expr {
	prog := b.Prog
	typ := prog.Int()
	if prog.SizeOf(n.Type) != prog.SizeOf(typ) {
		srcType := n.Type
		n.Type = typ
		n.impl = castInt(b, n.impl, srcType, typ)
	}
	return n
}

// -----------------------------------------------------------------------------

// The MakeMap instruction creates a new hash-table-based map object
// and yields a value of kind map.
//
// t is a (possibly named) *types.Map.
//
// Example printed form:
//
//	t1 = make map[string]int t0
//	t1 = make StringIntMap t0
func (b Builder) MakeMap(t Type, nReserve Expr) (ret Expr) {
	dbgInstrf("MakeMap %v, %v\n", t.RawType(), nReserve.impl)
	if nReserve.IsNil() {
		nReserve = b.Prog.Val(0)
	}
	typ := b.abiType(t.raw.Type)
	ret = b.InlineCall(b.Pkg.rtFunc("MakeMap"), typ, nReserve)
	ret.Type = t
	return
}

// The Lookup instruction yields element Index of collection map X.
// Index is the appropriate key type.
//
// If CommaOk, the result is a 2-tuple of the value above and a
// boolean indicating the result of a map membership test for the key.
// The components of the tuple are accessed using Extract.
//
// Example printed form:
//
//	t2 = t0[t1]
//	t5 = t3[t4],ok
func (b Builder) Lookup(x, key Expr, commaOk bool) (ret Expr) {
	dbgInstrf("Lookup %v, %v, %v\n", x.impl, key.impl, commaOk)
	prog := b.Prog
	typ := b.abiType(x.raw.Type)
	vtyp := prog.Elem(x.Type)
	ptr := b.mapKeyPtr(key)
	if commaOk {
		vals := b.Call(b.Pkg.rtFunc("MapAccess2"), typ, x, ptr)
		val := b.Load(Expr{b.impl.CreateExtractValue(vals.impl, 0, ""), prog.Pointer(vtyp)})
		ok := b.impl.CreateExtractValue(vals.impl, 1, "")
		t := prog.Struct(vtyp, prog.Bool())
		return b.aggregateValue(t, val.impl, ok)
	} else {
		val := b.Call(b.Pkg.rtFunc("MapAccess1"), typ, x, ptr)
		val.Type = prog.Pointer(vtyp)
		ret = b.Load(val)
	}
	return
}

// The MapUpdate instruction updates the association of Map[Key] to
// Value.
//
// Pos() returns the ast.KeyValueExpr.Colon or ast.IndexExpr.Lbrack,
// if explicit in the source.
//
// Example printed form:
//
//	t0[t1] = t2
func (b Builder) MapUpdate(m, k, v Expr) {
	// Convert function declarations to proper closure form when stored in maps.
	// This ensures function values are correctly wrapped as closures, similar to
	// interface assignment (see MakeInterface).
	if v.kind == vkFuncDecl {
		typ := b.Prog.Type(v.raw.Type, InGo)
		v = checkExpr(v, typ.raw.Type, b)
	}
	dbgInstrf("MapUpdate %v[%v] = %v\n", m.impl, k.impl, v.impl)
	typ := b.abiType(m.raw.Type)
	ptr := b.mapKeyPtr(k)
	ret := b.Call(b.Pkg.rtFunc("MapAssign"), typ, m, ptr)
	ret.Type = b.Prog.Pointer(v.Type)
	b.Store(ret, v)
}

// key => unsafe.Pointer
func (b Builder) mapKeyPtr(x Expr) Expr {
	typ := x.Type
	vtyp := b.Prog.VoidPtr()
	vptr := b.AllocU(typ)
	b.Store(vptr, x)
	return Expr{vptr.impl, vtyp}
}

// -----------------------------------------------------------------------------

// The Range instruction yields an iterator over the domain and range
// of X, which must be a string or map.
//
// Elements are accessed via Next.
//
// Type() returns an opaque and degenerate "rangeIter" type.
//
// Pos() returns the ast.RangeStmt.For.
//
// Example printed form:
//
//	t0 = range "hello":string
func (b Builder) Range(x Expr) Expr {
	switch x.kind {
	case vkString:
		return b.InlineCall(b.Pkg.rtFunc("NewStringIter"), x)
	case vkMap:
		typ := b.abiType(x.raw.Type)
		return b.InlineCall(b.Pkg.rtFunc("NewMapIter"), typ, x)
	}
	panic("unsupport range for " + x.raw.Type.String())
}

// The Next instruction reads and advances the (map or string)
// iterator Iter and returns a 3-tuple value (ok, k, v).  If the
// iterator is not exhausted, ok is true and k and v are the next
// elements of the domain and range, respectively.  Otherwise ok is
// false and k and v are undefined.
//
// Components of the tuple are accessed using Extract.
//
// The IsString field distinguishes iterators over strings from those
// over maps, as the Type() alone is insufficient: consider
// map[int]rune.
//
// Type() returns a *types.Tuple for the triple (ok, k, v).
// The types of k and/or v may be types.Invalid.
//
// Example printed form:
//
//	t1 = next t0
func (b Builder) Next(typ Type, iter Expr, isString bool) Expr {
	if isString {
		return b.InlineCall(b.Pkg.rtFunc("StringIterNext"), iter)
	}
	prog := b.Prog
	ktyp := prog.Type(typ.raw.Type.Underlying().(*types.Map).Key(), InGo)
	vtyp := prog.Type(typ.raw.Type.Underlying().(*types.Map).Elem(), InGo)
	rets := b.InlineCall(b.Pkg.rtFunc("MapIterNext"), iter)
	ok := b.impl.CreateExtractValue(rets.impl, 0, "")
	t := prog.Struct(prog.Bool(), ktyp, vtyp)
	blks := b.Func.MakeBlocks(3)
	b.If(Expr{ok, prog.Bool()}, blks[0], blks[1])
	b.SetBlockEx(blks[2], AtEnd, false)
	phi := b.Phi(t)
	phi.AddIncoming(b, blks[:2], func(i int, blk BasicBlock) Expr {
		b.SetBlockEx(blk, AtEnd, false)
		if i == 0 {
			k := b.impl.CreateExtractValue(rets.impl, 1, "")
			v := b.impl.CreateExtractValue(rets.impl, 2, "")
			valTrue := aggregateValue(b.impl, t.ll, prog.BoolVal(true).impl,
				llvm.CreateLoad(b.impl, ktyp.ll, k),
				llvm.CreateLoad(b.impl, vtyp.ll, v))
			b.Jump(blks[2])
			return Expr{valTrue, t}
		}
		valFalse := aggregateValue(b.impl, t.ll, prog.BoolVal(false).impl,
			llvm.ConstNull(ktyp.ll),
			llvm.ConstNull(vtyp.ll))
		b.Jump(blks[2])
		return Expr{valFalse, t}
	})
	b.SetBlockEx(blks[2], AtEnd, false)
	b.blk.last = blks[2].last
	return phi.Expr
}

// The MakeChan instruction creates a new channel object and yields a
// value of kind chan.
//
// Type() returns a (possibly named) *types.Chan.
//
// Pos() returns the ast.CallExpr.Lparen for the make(chan) that
// created it.
//
// Example printed form:
//
//	t0 = make chan int 0
//	t0 = make IntChan 0
//
//	type MakeChan struct {
//		register
//		Size Value // int; size of buffer; zero => synchronous.
//	}
func (b Builder) MakeChan(t Type, size Expr) (ret Expr) {
	dbgInstrf("MakeChan %v, %v\n", t.RawType(), size.impl)
	prog := b.Prog
	eltSize := prog.IntVal(prog.SizeOf(prog.Elem(t)), prog.Int())
	ret.Type = t
	ret.impl = b.InlineCall(b.Pkg.rtFunc("NewChan"), eltSize, size).impl
	return
}

// The Send instruction sends X on channel Chan.
//
// Pos() returns the ast.SendStmt.Arrow, if explicit in the source.
//
// Example printed form:
//
//	send t0 <- t1
func (b Builder) Send(ch Expr, x Expr) (ret Expr) {
	dbgInstrf("Send %v, %v\n", ch.impl, x.impl)
	prog := b.Prog
	eltSize := prog.IntVal(prog.SizeOf(prog.Elem(ch.Type)), prog.Int())
	ret = b.InlineCall(b.Pkg.rtFunc("ChanSend"), ch, b.toPtr(x), eltSize)
	return
}

func (b Builder) toPtr(x Expr) Expr {
	typ := x.Type
	vtyp := b.Prog.VoidPtr()
	vptr := b.Alloc(typ, false)
	b.Store(vptr, x)
	return Expr{vptr.impl, vtyp}
}

func (b Builder) Recv(ch Expr, commaOk bool) (ret Expr) {
	dbgInstrf("Recv %v, %v\n", ch.impl, commaOk)
	prog := b.Prog
	eltSize := prog.IntVal(prog.SizeOf(prog.Elem(ch.Type)), prog.Int())
	etyp := prog.Elem(ch.Type)
	ptr := b.Alloc(etyp, false)
	ok := b.InlineCall(b.Pkg.rtFunc("ChanRecv"), ch, ptr, eltSize)
	if commaOk {
		val := b.Load(ptr)
		t := prog.Struct(etyp, prog.Bool())
		return b.aggregateValue(t, val.impl, ok.impl)
	} else {
		return b.Load(ptr)
	}
}

type SelectState struct {
	Chan  Expr // channel to use (for send or receive)
	Value Expr // value to send (for send)
	Send  bool // direction of case (SendOnly or RecvOnly)
}

// The Select instruction tests whether (or blocks until) one
// of the specified sent or received states is entered.
//
// Let n be the number of States for which Dir==RECV and T_i (0<=i<n)
// be the element type of each such state's Chan.
// Select returns an n+2-tuple
//
//	(index int, recvOk bool, r_0 T_0, ... r_n-1 T_n-1)
//
// The tuple's components, described below, must be accessed via the
// Extract instruction.
//
// If Blocking, select waits until exactly one state holds, i.e. a
// channel becomes ready for the designated operation of sending or
// receiving; select chooses one among the ready states
// pseudorandomly, performs the send or receive operation, and sets
// 'index' to the index of the chosen channel.
//
// If !Blocking, select doesn't block if no states hold; instead it
// returns immediately with index equal to -1.
//
// If the chosen channel was used for a receive, the r_i component is
// set to the received value, where i is the index of that state among
// all n receive states; otherwise r_i has the zero value of type T_i.
// Note that the receive index i is not the same as the state
// index index.
//
// The second component of the triple, recvOk, is a boolean whose value
// is true iff the selected operation was a receive and the receive
// successfully yielded a value.
//
// Pos() returns the ast.SelectStmt.Select.
//
// Example printed form:
//
//	t3 = select nonblocking [<-t0, t1<-t2]
//	t4 = select blocking []
func (b Builder) Select(states []*SelectState, blocking bool) (ret Expr) {
	ops := make([]Expr, len(states))
	for i, s := range states {
		ops[i] = b.chanOp(s)
	}
	var fn Expr
	if blocking {
		fn = b.Pkg.rtFunc("Select")
	} else {
		fn = b.Pkg.rtFunc("TrySelect")
	}
	prog := b.Prog
	tSlice := lastParamType(prog, fn)
	slice := b.SliceLit(tSlice, ops...)
	ret = b.Call(fn, slice)
	chosen := b.impl.CreateExtractValue(ret.impl, 0, "")
	recvOK := b.impl.CreateExtractValue(ret.impl, 1, "")
	if !blocking {
		// runtime.TrySelect returns (isel, recvOK, tryOK). recvOK is only meaningful
		// for receives; selection success is reported by tryOK.
		tryOK := b.impl.CreateExtractValue(ret.impl, 2, "")
		chosen = llvm.CreateSelect(b.impl, tryOK, chosen, prog.Val(-1).impl)
	}
	results := []llvm.Value{chosen, recvOK}
	typs := []Type{prog.Int(), prog.Bool()}
	for i, s := range states {
		if !s.Send {
			etyp := b.Prog.Elem(s.Chan.Type)
			typs = append(typs, etyp)
			r := b.Load(Expr{b.impl.CreateExtractValue(ops[i].impl, 1, ""), prog.Pointer(etyp)})
			results = append(results, r.impl)
		}
	}
	return b.aggregateValue(b.Prog.Struct(typs...), results...)
}

func lastParamType(prog Program, fn Expr) Type {
	params := fn.raw.Type.(*types.Signature).Params()
	return prog.rawType(params.At(params.Len() - 1).Type())
}

func (b Builder) chanOp(s *SelectState) Expr {
	prog := b.Prog
	var val Expr
	var size Expr
	if s.Send {
		val = b.toPtr(s.Value)
		size = prog.IntVal(prog.SizeOf(s.Value.Type), prog.Int32())
	} else {
		etyp := prog.Elem(s.Chan.Type)
		val = b.Alloc(etyp, false)
		size = prog.IntVal(prog.SizeOf(etyp), prog.Int32())
	}
	send := prog.BoolVal(s.Send)
	typ := b.Prog.rtType("ChanOp")
	return b.aggregateValue(typ, s.Chan.impl, val.impl, size.impl, send.impl)
}

// -----------------------------------------------------------------------------
