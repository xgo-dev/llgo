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
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"log"

	"github.com/xgo-dev/llvm"
)

type AtomicOrdering = llvm.AtomicOrdering

const (
	OrderingNotAtomic      = llvm.AtomicOrderingNotAtomic
	OrderingUnordered      = llvm.AtomicOrderingUnordered
	OrderingMonotonic      = llvm.AtomicOrderingMonotonic
	OrderingAcquire        = llvm.AtomicOrderingAcquire
	OrderingRelease        = llvm.AtomicOrderingRelease
	OrderingAcquireRelease = llvm.AtomicOrderingAcquireRelease
	OrderingSeqConsistent  = llvm.AtomicOrderingSequentiallyConsistent
)

// -----------------------------------------------------------------------------

type Expr struct {
	impl llvm.Value
	Type
}

var Nil Expr // Zero value is a nil Expr

// IsNil checks if the expression is nil or not.
func (v Expr) IsNil() bool {
	return v.Type == nil
}

// SetOrdering sets the ordering of the atomic operation.
func (v Expr) SetOrdering(ordering AtomicOrdering) Expr {
	v.impl.SetOrdering(ordering)
	return v
}

func (v Expr) SetName(alias string) Expr {
	v.impl.SetName(alias)
	return v
}

func (v Expr) Name() string {
	return v.impl.Name()
}

// -----------------------------------------------------------------------------

type builtinTy struct {
	name string
}

func (p builtinTy) Underlying() types.Type {
	panic("don't call")
}

func (p builtinTy) String() string {
	return "builtinTy"
}

// Builtin returns a builtin function expression.
func Builtin(name string) Expr {
	tbi := &aType{raw: rawType{&builtinTy{name}}, kind: vkBuiltin}
	return Expr{Type: tbi}
}

// -----------------------------------------------------------------------------

type pyVarTy struct {
	mod  Expr
	name string
}

func (p pyVarTy) Underlying() types.Type {
	panic("don't call")
}

func (p pyVarTy) String() string {
	return "pyVar"
}

func pyVarExpr(mod Expr, name string) Expr {
	tvar := &aType{raw: rawType{&pyVarTy{mod, name}}, kind: vkPyVarRef}
	return Expr{Type: tvar}
}

// -----------------------------------------------------------------------------

// Zero returns a zero constant expression.
func (p Program) Zero(t Type) Expr {
	var ret llvm.Value
	_, isNamed := t.raw.Type.(*types.Named)
	switch u := t.raw.Type.Underlying().(type) {
	case *types.Basic:
		kind := u.Kind()
		switch {
		case kind >= types.Bool && kind <= types.Uintptr:
			ret = llvm.ConstInt(p.rawType(u).ll, 0, false)
		case kind == types.String:
			ret = p.Zero(p.rtType("String")).impl
		case kind == types.UnsafePointer:
			ret = llvm.ConstPointerNull(p.tyVoidPtr())
		case kind == types.Float64:
			ret = llvm.ConstFloat(p.Float64().ll, 0)
		case kind == types.Float32:
			ret = llvm.ConstFloat(p.Float32().ll, 0)
		case kind == types.Complex64:
			v := llvm.ConstFloat(p.Float32().ll, 0)
			ret = p.ctx.ConstStruct([]llvm.Value{v, v}, false)
		case kind == types.Complex128:
			v := llvm.ConstFloat(p.Float64().ll, 0)
			ret = p.ctx.ConstStruct([]llvm.Value{v, v}, false)
		default:
			panic("todo")
		}
	case *types.Pointer, *types.Signature, *types.Chan:
		return Expr{llvm.ConstNull(t.ll), t}
	case *types.Struct:
		n := u.NumFields()
		flds := make([]llvm.Value, n)
		for i := 0; i < n; i++ {
			flds[i] = p.Zero(p.rawType(u.Field(i).Type())).impl
		}
		if isNamed {
			ret = llvm.ConstNamedStruct(t.ll, flds)
		} else {
			ret = p.ctx.ConstStruct(flds, false)
		}
	case *types.Slice:
		ret = p.Zero(p.rtType("Slice")).impl
	case *types.Array:
		ret = llvm.ConstNull(t.ll)
	case *types.Interface:
		var name string
		if u.Empty() {
			name = "Eface"
		} else {
			name = "Iface"
		}
		ret = p.Zero(p.rtType(name)).impl
	case *types.Map:
		ret = p.Zero(p.Pointer(p.rtType("Map"))).impl
	case *types.Tuple:
		n := u.Len()
		flds := make([]llvm.Value, n)
		for i := 0; i < n; i++ {
			flds[i] = p.Zero(p.rawType(u.At(i).Type())).impl
		}
		ret = p.ctx.ConstStruct(flds, false)
	default:
		log.Panicln("todo:", u)
	}
	return Expr{ret, t}
}

// Nil returns a null constant expression. t should be a pointer type.
func (p Program) Nil(t Type) Expr {
	return Expr{llvm.ConstNull(t.ll), t}
}

// BoolVal returns a boolean constant expression.
func (p Program) BoolVal(v bool) Expr {
	t := p.Bool()
	var bv uint64
	if v {
		bv = 1
	}
	ret := llvm.ConstInt(t.ll, bv, v)
	return Expr{ret, t}
}

// IntVal returns an integer constant expression.
func (p Program) IntVal(v uint64, t Type) Expr {
	ret := llvm.ConstInt(t.ll, v, false)
	return Expr{ret, t}
}

// FloatVal returns a float constant expression.
func (p Program) FloatVal(v float64, t Type) Expr {
	ret := llvm.ConstFloat(t.ll, v)
	return Expr{ret, t}
}

// ComplexVal returns a complex constant expression.
func (p Program) ComplexVal(v complex128, t Type) Expr {
	flt := p.Field(t, 0)
	re := p.FloatVal(real(v), flt)
	im := p.FloatVal(imag(v), flt)
	return Expr{p.ctx.ConstStruct([]llvm.Value{re.impl, im.impl}, false), t}
}

// Val returns a constant expression.
func (p Program) Val(v interface{}) Expr {
	switch v := v.(type) {
	case int:
		return p.IntVal(uint64(v), p.Int())
	case uintptr:
		return p.IntVal(uint64(v), p.Uintptr())
	case bool:
		return p.BoolVal(v)
	case float64:
		t := p.Float64()
		ret := llvm.ConstFloat(t.ll, v)
		return Expr{ret, t}
	}
	panic("todo")
}

// Const returns a constant expression.
func (b Builder) Const(v constant.Value, typ Type) Expr {
	prog := b.Prog
	if v == nil {
		return prog.Nil(typ)
	}
	raw := typ.raw.Type
	switch t := raw.Underlying().(type) {
	case *types.Basic:
		kind := t.Kind()
		switch {
		case kind == types.Bool:
			return Expr{prog.BoolVal(constant.BoolVal(v)).impl, typ}
		case kind >= types.Int && kind <= types.Int64:
			if v, exact := constant.Int64Val(constant.ToInt(v)); exact {
				return prog.IntVal(uint64(v), typ)
			}
		case kind >= types.Uint && kind <= types.Uintptr:
			if v, exact := constant.Uint64Val(constant.ToInt(v)); exact {
				return prog.IntVal(v, typ)
			}
		case kind == types.Float64 || kind == types.Float32:
			v, _ := constant.Float64Val(constant.ToFloat(v))
			return prog.FloatVal(v, typ)
		case kind == types.String:
			return Expr{b.Str(constant.StringVal(v)).impl, typ}
		case kind == types.Complex128 || kind == types.Complex64:
			v = constant.ToComplex(v)
			re, _ := constant.Float64Val(constant.Real(v))
			im, _ := constant.Float64Val(constant.Imag(v))
			return prog.ComplexVal(complex(re, im), typ)
		case kind == types.UnsafePointer:
			if v, exact := constant.Uint64Val(v); exact {
				return Expr{llvm.ConstIntToPtr(llvm.ConstInt(prog.Uintptr().ll, v, false), typ.ll), typ}
			}
		}
	}
	panic(fmt.Sprintf("unsupported Const: %v, %v", v, raw))
}

// CStr returns a c-style string constant expression.
func (b Builder) CStr(v string) Expr {
	return Expr{llvm.CreateGlobalStringPtr(b.impl, v), b.Prog.CStr()}
}

// CString returns a c-style string
func (b Builder) CString(v Expr) Expr {
	fn := b.Pkg.rtFunc("CString")
	return b.Call(fn, v)
}

// CBytes returns a c-style bytes
func (b Builder) CBytes(v Expr) Expr {
	fn := b.Pkg.rtFunc("CBytes")
	return b.Call(fn, v)
}

// InlineAsm generates inline assembly instruction
func (b Builder) InlineAsm(instruction string) {
	dbgInstrf("InlineAsm %s\n", instruction)

	typ := llvm.FunctionType(b.Prog.tyVoid(), nil, false)
	asm := llvm.InlineAsm(typ, instruction, "", true, false, llvm.InlineAsmDialectATT, false)
	b.impl.CreateCall(typ, asm, nil, "")
}

func (b Builder) InlineAsmFull(instruction, constraints string, retType Type, exprs []Expr) Expr {
	typs := make([]llvm.Type, len(exprs))
	vals := make([]llvm.Value, len(exprs))
	for i, expr := range exprs {
		typs[i], vals[i] = expr.Type.ll, expr.impl
	}

	ftype := llvm.FunctionType(retType.ll, typs, false)
	asm := llvm.InlineAsm(ftype, instruction, constraints, true, false, llvm.InlineAsmDialectATT, false)
	return Expr{b.impl.CreateCall(ftype, asm, vals, ""), retType}
}

// GoString returns a Go string
func (b Builder) GoString(v Expr) Expr {
	fn := b.Pkg.rtFunc("GoString")
	return b.Call(fn, v)
}

// GoStringN returns a Go string
func (b Builder) GoStringN(v Expr, n Expr) Expr {
	fn := b.Pkg.rtFunc("GoStringN")
	return b.Call(fn, v, n)
}

// GoBytes returns a Go bytes
func (b Builder) GoBytes(v Expr, n Expr) (ret Expr) {
	fn := b.Pkg.rtFunc("GoBytes")
	return b.Call(fn, v, n)
}

// CMalloc returns a c-style pointer
func (b Builder) CMalloc(n Expr) Expr {
	return b.malloc(n)
}

// Str returns a Go string constant expression.
func (b Builder) Str(v string) Expr {
	prog := b.Prog
	data := b.Pkg.createGlobalStr(v)
	size := llvm.ConstInt(prog.tyInt(), uint64(len(v)), false)
	return Expr{aggregateValue(b.impl, prog.rtString(), data, size), prog.String()}
}

// unsafeString(data *byte, size int) string
func (b Builder) unsafeString(data, size llvm.Value) Expr {
	prog := b.Prog
	return Expr{aggregateValue(b.impl, prog.rtString(), data, size), prog.String()}
}

// unsafeSlice(data *T, size, cap int) []T
func (b Builder) unsafeSlice(data Expr, size, cap llvm.Value) Expr {
	prog := b.Prog
	tslice := prog.Slice(prog.Elem(data.Type))
	return Expr{aggregateValue(b.impl, prog.rtSlice(), data.impl, size, cap), tslice}
}

// -----------------------------------------------------------------------------

const (
	mathOpBase = token.ADD
	mathOpLast = token.REM
)

var mathOpToLLVM = []llvm.Opcode{
	int(token.ADD-mathOpBase)<<2 | vkSigned:   llvm.Add,
	int(token.ADD-mathOpBase)<<2 | vkUnsigned: llvm.Add,
	int(token.ADD-mathOpBase)<<2 | vkFloat:    llvm.FAdd,

	int(token.SUB-mathOpBase)<<2 | vkSigned:   llvm.Sub,
	int(token.SUB-mathOpBase)<<2 | vkUnsigned: llvm.Sub,
	int(token.SUB-mathOpBase)<<2 | vkFloat:    llvm.FSub,

	int(token.MUL-mathOpBase)<<2 | vkSigned:   llvm.Mul,
	int(token.MUL-mathOpBase)<<2 | vkUnsigned: llvm.Mul,
	int(token.MUL-mathOpBase)<<2 | vkFloat:    llvm.FMul,

	int(token.QUO-mathOpBase)<<2 | vkSigned:   llvm.SDiv,
	int(token.QUO-mathOpBase)<<2 | vkUnsigned: llvm.UDiv,
	int(token.QUO-mathOpBase)<<2 | vkFloat:    llvm.FDiv,

	int(token.REM-mathOpBase)<<2 | vkSigned:   llvm.SRem,
	int(token.REM-mathOpBase)<<2 | vkUnsigned: llvm.URem,
	int(token.REM-mathOpBase)<<2 | vkFloat:    llvm.FRem,
}

func mathOpIdx(op token.Token, x valueKind) int {
	return int(op-mathOpBase)<<2 | x
}

// ADD SUB MUL QUO REM          + - * / %
func isMathOp(op token.Token) bool {
	return op >= mathOpBase && op <= mathOpLast
}

const (
	logicOpBase = token.AND
	logicOpLast = token.AND_NOT
)

var logicOpToLLVM = []llvm.Opcode{
	token.AND - logicOpBase: llvm.And,
	token.OR - logicOpBase:  llvm.Or,
	token.XOR - logicOpBase: llvm.Xor,
	token.SHL - logicOpBase: llvm.Shl,
	token.SHR - logicOpBase: llvm.AShr, // Arithmetic Shift Right
}

// AND OR XOR SHL SHR AND_NOT   & | ^ << >> &^
func isLogicOp(op token.Token) bool {
	return op >= logicOpBase && op <= logicOpLast
}

const (
	predOpBase = token.EQL
	predOpLast = token.GEQ
)

var intPredOpToLLVM = []llvm.IntPredicate{
	token.EQL - predOpBase: llvm.IntEQ,
	token.NEQ - predOpBase: llvm.IntNE,
	token.LSS - predOpBase: llvm.IntSLT,
	token.LEQ - predOpBase: llvm.IntSLE,
	token.GTR - predOpBase: llvm.IntSGT,
	token.GEQ - predOpBase: llvm.IntSGE,
}

var uintPredOpToLLVM = []llvm.IntPredicate{
	token.EQL - predOpBase: llvm.IntEQ,
	token.NEQ - predOpBase: llvm.IntNE,
	token.LSS - predOpBase: llvm.IntULT,
	token.LEQ - predOpBase: llvm.IntULE,
	token.GTR - predOpBase: llvm.IntUGT,
	token.GEQ - predOpBase: llvm.IntUGE,
}

var floatPredOpToLLVM = []llvm.FloatPredicate{
	token.EQL - predOpBase: llvm.FloatOEQ,
	token.NEQ - predOpBase: llvm.FloatUNE,
	token.LSS - predOpBase: llvm.FloatOLT,
	token.LEQ - predOpBase: llvm.FloatOLE,
	token.GTR - predOpBase: llvm.FloatOGT,
	token.GEQ - predOpBase: llvm.FloatOGE,
}

var boolPredOpToLLVM = []llvm.IntPredicate{
	token.EQL - predOpBase: llvm.IntEQ,
	token.NEQ - predOpBase: llvm.IntNE,
}

// EQL NEQ LSS LEQ GTR GEQ      == != < <= > >=
func isPredOp(op token.Token) bool {
	return op >= predOpBase && op <= predOpLast
}

// The BinOp instruction yields the result of binary operation (x op y).
// op can be:
// ADD SUB MUL QUO REM          + - * / %
// AND OR XOR SHL SHR AND_NOT   & | ^ << >> &^
// EQL NEQ LSS LEQ GTR GEQ      == != < <= > >=
func (b Builder) BinOp(op token.Token, x, y Expr) Expr {
	dbgInstrf("BinOp %d, %v, %v\n", op, x.impl, y.impl)
	switch {
	case isMathOp(op): // op: + - * / %
		kind := x.kind
		switch kind {
		case vkString:
			if op == token.ADD {
				return Expr{b.InlineCall(b.Pkg.rtFunc("StringCat"), x, y).impl, x.Type}
			}
		case vkComplex:
			xr, xi := b.impl.CreateExtractValue(x.impl, 0, ""), b.impl.CreateExtractValue(x.impl, 1, "")
			yr, yi := b.impl.CreateExtractValue(y.impl, 0, ""), b.impl.CreateExtractValue(y.impl, 1, "")
			switch op {
			case token.ADD:
				r := llvm.CreateBinOp(b.impl, llvm.FAdd, xr, yr)
				i := llvm.CreateBinOp(b.impl, llvm.FAdd, xi, yi)
				return b.aggregateValue(x.Type, r, i)
			case token.SUB:
				r := llvm.CreateBinOp(b.impl, llvm.FSub, xr, yr)
				i := llvm.CreateBinOp(b.impl, llvm.FSub, xi, yi)
				return b.aggregateValue(x.Type, r, i)
			case token.MUL:
				r := llvm.CreateBinOp(b.impl, llvm.FSub,
					llvm.CreateBinOp(b.impl, llvm.FMul, xr, yr),
					llvm.CreateBinOp(b.impl, llvm.FMul, xi, yi),
				)
				i := llvm.CreateBinOp(b.impl, llvm.FAdd,
					llvm.CreateBinOp(b.impl, llvm.FMul, xr, yi),
					llvm.CreateBinOp(b.impl, llvm.FMul, xi, yr),
				)
				return b.aggregateValue(x.Type, r, i)
			case token.QUO:
				x128 := b.Convert(b.Prog.Complex128(), x)
				y128 := b.Convert(b.Prog.Complex128(), y)
				ret := b.InlineCall(b.Pkg.rtFunc("Complex128Div"), x128, y128)
				return b.Convert(x.Type, ret)
			}
		default:
			idx := mathOpIdx(op, kind)
			if llop := mathOpToLLVM[idx]; llop != 0 {
				// Go requires integer division/modulo by zero to panic.
				// LLVM's integer div/rem are undefined on zero divisor, so
				// we insert an explicit runtime check here and lower through a
				// safe non-zero divisor so targets like x86 don't trap first.
				// safeY carries the zero-safe divisor into the signed overflow
				// lowering below when both guards are needed.
				var safeY llvm.Value
				if (op == token.QUO || op == token.REM) && (kind == vkSigned || kind == vkUnsigned) {
					needsCheck := true
					if rv := y.impl.IsAConstantInt(); !rv.IsNil() {
						needsCheck = rv.ZExtValue() == 0
					}
					if needsCheck {
						zero := llvm.ConstInt(y.ll, 0, false)
						isZero := llvm.CreateICmp(b.impl, llvm.IntEQ, y.impl, zero)
						check := Expr{isZero, b.Prog.Bool()}
						b.InlineCall(b.Pkg.rtFunc("AssertDivideByZero"), check)
						safeY = llvm.CreateSelect(b.impl, isZero, llvm.ConstInt(y.ll, 1, false), y.impl)
					}
				}
				// On x86/x86_64, signed div/rem of minInt by -1 traps even
				// though Go defines it to produce quotient=minInt and
				// remainder=0. Lower through safe operands and select the
				// Go-defined result for that overflow case.
				needsOverflowCheck := false
				if (op == token.QUO || op == token.REM) && kind == vkSigned {
					needsOverflowCheck = true
					if rv := y.impl.IsAConstantInt(); !rv.IsNil() {
						needsOverflowCheck = rv.SExtValue() == -1
					}
				}
				var minIntVal uint64
				if needsOverflowCheck {
					bits := b.Prog.SizeOf(x.Type) * 8
					if bits == 0 || bits > 64 {
						panic("unexpected integer bit width")
					}
					minIntVal = uint64(1) << (bits - 1)
					if rv := x.impl.IsAConstantInt(); !rv.IsNil() {
						needsOverflowCheck = rv.ZExtValue() == minIntVal
					}
				}
				if needsOverflowCheck {
					minInt := llvm.ConstInt(x.ll, minIntVal, false)
					negOne := llvm.ConstAllOnes(y.ll)
					isMinInt := llvm.CreateICmp(b.impl, llvm.IntEQ, x.impl, minInt)
					isNegOne := llvm.CreateICmp(b.impl, llvm.IntEQ, y.impl, negOne)
					overflow := llvm.CreateAnd(b.impl, isMinInt, isNegOne)

					safeX := llvm.CreateSelect(b.impl, overflow, llvm.ConstInt(x.ll, 0, false), x.impl)
					if safeY.IsNil() {
						safeY = y.impl
					}
					safeY = llvm.CreateSelect(b.impl, overflow, llvm.ConstInt(y.ll, 1, false), safeY)
					v := llvm.CreateBinOp(b.impl, llop, safeX, safeY)
					if op == token.QUO {
						v = llvm.CreateSelect(b.impl, overflow, x.impl, v)
					} else {
						v = llvm.CreateSelect(b.impl, overflow, llvm.ConstInt(x.ll, 0, false), v)
					}
					return Expr{v, x.Type}
				}
				if !safeY.IsNil() {
					return Expr{llvm.CreateBinOp(b.impl, llop, x.impl, safeY), x.Type}
				}
				return Expr{llvm.CreateBinOp(b.impl, llop, x.impl, y.impl), x.Type}
			}
		}
	case isLogicOp(op): // op: & | ^ << >> &^
		switch op {
		case token.AND_NOT:
			return Expr{llvm.CreateAnd(b.impl, x.impl, llvm.CreateNot(b.impl, y.impl)), x.Type}
		case token.SHL, token.SHR:
			if needsNegativeCheck(y) {
				zero := llvm.ConstInt(y.ll, 0, false)
				check := Expr{llvm.CreateICmp(b.impl, llvm.IntSLT, y.impl, zero), b.Prog.Bool()}
				b.InlineCall(b.Pkg.rtFunc("AssertNegativeShift"), check)
			}
			xsize, ysize := b.Prog.SizeOf(x.Type), b.Prog.SizeOf(y.Type)
			if xsize != ysize {
				y = b.Convert(x.Type, y)
			}
			overflows := llvm.CreateICmp(b.impl, llvm.IntUGE, y.impl, llvm.ConstInt(y.ll, xsize*8, false))
			xzero := llvm.ConstInt(x.ll, 0, false)
			if op == token.SHL {
				rhs := llvm.CreateShl(b.impl, x.impl, y.impl)
				return Expr{llvm.CreateSelect(b.impl, overflows, xzero, rhs), x.Type}
			} else {
				if x.kind == vkSigned {
					rhs := llvm.CreateSelect(b.impl, overflows, llvm.ConstInt(y.ll, 8*xsize-1, false), y.impl)
					return Expr{llvm.CreateAShr(b.impl, x.impl, rhs), x.Type}
				} else {
					rsh := llvm.CreateLShr(b.impl, x.impl, y.impl)
					return Expr{llvm.CreateSelect(b.impl, overflows, xzero, rsh), x.Type}
				}
			}
		default:
			llop := logicOpToLLVM[op-logicOpBase]
			return Expr{llvm.CreateBinOp(b.impl, llop, x.impl, y.impl), x.Type}
		}
	case isPredOp(op): // op: == != < <= > >=
		prog := b.Prog
		tret := prog.Bool()
		kind := x.kind
		switch kind {
		case vkSigned:
			pred := intPredOpToLLVM[op-predOpBase]
			return Expr{llvm.CreateICmp(b.impl, pred, x.impl, y.impl), tret}
		case vkUnsigned, vkPtr:
			pred := uintPredOpToLLVM[op-predOpBase]
			return Expr{llvm.CreateICmp(b.impl, pred, x.impl, y.impl), tret}
		case vkFloat:
			pred := floatPredOpToLLVM[op-predOpBase]
			return Expr{llvm.CreateFCmp(b.impl, pred, x.impl, y.impl), tret}
		case vkBool:
			pred := boolPredOpToLLVM[op-predOpBase]
			return Expr{llvm.CreateICmp(b.impl, pred, x.impl, y.impl), tret}
		case vkComplex:
			switch op {
			case token.EQL:
				xr, xi := b.impl.CreateExtractValue(x.impl, 0, ""), b.impl.CreateExtractValue(x.impl, 1, "")
				yr, yi := b.impl.CreateExtractValue(y.impl, 0, ""), b.impl.CreateExtractValue(y.impl, 1, "")
				return Expr{llvm.CreateAnd(b.impl,
					llvm.CreateFCmp(b.impl, llvm.FloatOEQ, xr, yr),
					llvm.CreateFCmp(b.impl, llvm.FloatOEQ, xi, yi),
				), tret}
			case token.NEQ:
				xr, xi := b.impl.CreateExtractValue(x.impl, 0, ""), b.impl.CreateExtractValue(x.impl, 1, "")
				yr, yi := b.impl.CreateExtractValue(y.impl, 0, ""), b.impl.CreateExtractValue(y.impl, 1, "")
				return Expr{b.impl.CreateOr(
					llvm.CreateFCmp(b.impl, llvm.FloatUNE, xr, yr),
					llvm.CreateFCmp(b.impl, llvm.FloatUNE, xi, yi),
					"",
				), tret}
			}
		case vkString:
			switch op {
			case token.EQL:
				return b.InlineCall(b.Pkg.rtFunc("StringEqual"), x, y)
			case token.NEQ:
				ret := b.InlineCall(b.Pkg.rtFunc("StringEqual"), x, y)
				ret.impl = llvm.CreateNot(b.impl, ret.impl)
				return ret
			case token.LSS:
				return b.InlineCall(b.Pkg.rtFunc("StringLess"), x, y)
			case token.LEQ:
				ret := b.InlineCall(b.Pkg.rtFunc("StringLess"), y, x)
				ret.impl = llvm.CreateNot(b.impl, ret.impl)
				return ret
			case token.GTR:
				return b.InlineCall(b.Pkg.rtFunc("StringLess"), y, x)
			case token.GEQ:
				ret := b.InlineCall(b.Pkg.rtFunc("StringLess"), x, y)
				ret.impl = llvm.CreateNot(b.impl, ret.impl)
				return ret
			}
		case vkClosure:
			x = b.Field(x, 0)
			if y.kind == vkClosure {
				y = b.Field(y, 0)
			}
			fallthrough
		case vkFuncPtr, vkFuncDecl, vkChan, vkMap:
			if y.kind == vkClosure {
				y = b.Field(y, 0)
			}
			switch op {
			case token.EQL, token.NEQ:
				pred := uintPredOpToLLVM[op-predOpBase]
				return Expr{llvm.CreateICmp(b.impl, pred, x.impl, y.impl), tret}
			}
		case vkArray:
			typ := x.raw.Type.Underlying().(*types.Array)
			elem := b.Prog.Elem(x.Type)
			ret := prog.BoolVal(true)
			for i, n := 0, int(typ.Len()); i < n; i++ {
				fx := b.impl.CreateExtractValue(x.impl, i, "")
				fy := b.impl.CreateExtractValue(y.impl, i, "")
				r := b.BinOp(token.EQL, Expr{fx, elem}, Expr{fy, elem})
				ret = Expr{b.impl.CreateAnd(ret.impl, r.impl, ""), tret}
			}
			switch op {
			case token.EQL:
				return ret
			case token.NEQ:
				return Expr{b.impl.CreateNot(ret.impl, ""), tret}
			}
		case vkStruct:
			typ := x.raw.Type.Underlying().(*types.Struct)
			ret := prog.BoolVal(true)
			for i, n := 0, typ.NumFields(); i < n; i++ {
				if typ.Field(i).Name() == "_" {
					continue
				}
				ft := prog.Type(typ.Field(i).Type(), InGo)
				fx := b.impl.CreateExtractValue(x.impl, i, "")
				fy := b.impl.CreateExtractValue(y.impl, i, "")
				r := b.BinOp(token.EQL, Expr{fx, ft}, Expr{fy, ft})
				ret = Expr{b.impl.CreateAnd(ret.impl, r.impl, ""), tret}
			}
			switch op {
			case token.EQL:
				return ret
			case token.NEQ:
				return Expr{b.impl.CreateNot(ret.impl, ""), tret}
			}
		case vkSlice:
			dx := b.impl.CreateExtractValue(x.impl, 0, "")
			dy := b.impl.CreateExtractValue(y.impl, 0, "")
			switch op {
			case token.EQL:
				return Expr{b.impl.CreateICmp(llvm.IntEQ, dx, dy, ""), tret}
			case token.NEQ:
				return Expr{b.impl.CreateICmp(llvm.IntNE, dx, dy, ""), tret}
			}
		case vkIface, vkEface:
			toEface := func(x Expr, emtpy bool) Expr {
				if emtpy {
					return x
				}
				return Expr{b.unsafeEface(b.faceAbiType(x).impl, b.faceData(x.impl)), prog.rtType("Eface")}
			}
			switch op {
			case token.EQL:
				return b.InlineCall(b.Pkg.rtFunc("EfaceEqual"), toEface(x, x.kind == vkEface), toEface(y, y.kind == vkEface))
			case token.NEQ:
				ret := b.InlineCall(b.Pkg.rtFunc("EfaceEqual"), toEface(x, x.kind == vkEface), toEface(y, y.kind == vkEface))
				ret.impl = llvm.CreateNot(b.impl, ret.impl)
				return ret
			}
		}
	}
	panic("todo")
}

// The UnOp instruction yields the result of (op x).
// ARROW is channel receive.
// MUL is pointer indirection (load).
// XOR is bitwise complement.
// SUB is negation.
// NOT is logical negation.
func (b Builder) UnOp(op token.Token, x Expr) (ret Expr) {
	dbgInstrf("UnOp %v, %v\n", op, x.impl)
	switch op {
	case token.MUL:
		return b.Load(x)
	case token.SUB:
		switch t := x.raw.Type.Underlying().(type) {
		case *types.Basic:
			ret.Type = x.Type
			if t.Info()&types.IsInteger != 0 {
				ret.impl = llvm.CreateNeg(b.impl, x.impl)
			} else if t.Info()&types.IsFloat != 0 {
				ret.impl = llvm.CreateFNeg(b.impl, x.impl)
			} else if t.Info()&types.IsComplex != 0 {
				r := b.impl.CreateExtractValue(x.impl, 0, "")
				i := b.impl.CreateExtractValue(x.impl, 1, "")
				return b.aggregateValue(x.Type, llvm.CreateFNeg(b.impl, r), llvm.CreateFNeg(b.impl, i))
			} else {
				panic("todo")
			}
		default:
			panic("unreachable")
		}
	case token.NOT:
		ret.Type = x.Type
		ret.impl = llvm.CreateNot(b.impl, x.impl)
	case token.XOR:
		ret.Type = x.Type
		ret.impl = llvm.CreateXor(b.impl, x.impl, llvm.ConstInt(x.Type.ll, ^uint64(0), false))
	case token.ARROW:
		panic("todo")
	}
	return
}

// -----------------------------------------------------------------------------

// The ChangeType instruction applies to X a value-preserving type
// change to Type().
//
// Type changes are permitted:
//   - between a named type and its underlying type.
//   - between two named types of the same underlying type.
//   - between (possibly named) pointers to identical base types.
//   - from a bidirectional channel to a read- or write-channel,
//     optionally adding/removing a name.
//   - between a type (t) and an instance of the type (tσ), i.e.
//     Type() == σ(X.Type()) (or X.Type()== σ(Type())) where
//     σ is the type substitution of Parent().TypeParams by
//     Parent().TypeArgs.
//
// This operation cannot fail dynamically.
//
// Type changes may to be to or from a type parameter (or both). All
// types in the type set of X.Type() have a value-preserving type
// change to all types in the type set of Type().
//
// Example printed form:
//
//	t1 = changetype *int <- IntPtr (t0)
func (b Builder) ChangeType(t Type, x Expr) (ret Expr) {
	dbgInstrf("ChangeType %v, %v\n", t.RawType(), x.impl)
	if t.kind == vkClosure {
		switch x.kind {
		case vkFuncDecl:
			ret.impl = checkExpr(x, t.raw.Type, b).impl
		case vkClosure:
			// TODO(xsw): change type should be a noop instruction
			convType := func() Expr {
				r := Expr{llvm.CreateAlloca(b.impl, t.ll), b.Prog.Pointer(t)}
				b.Store(r, x)
				return b.Load(r)
			}
			switch t.RawType().(type) {
			case *types.Named:
				if _, ok := x.RawType().(*types.Struct); ok {
					return convType()
				}
			case *types.Struct:
				if _, ok := x.RawType().(*types.Named); ok {
					return convType()
				}
			}
			fallthrough
		default:
			ret.impl = x.impl
		}
	} else if t.kind == vkStruct {
		xt := types.Unalias(x.RawType()).Underlying().(*types.Struct)
		agg := llvm.Undef(t.ll)
		for i := 0; i < xt.NumFields(); i++ {
			field := b.impl.CreateExtractValue(x.impl, i, "")
			agg = b.impl.CreateInsertValue(agg, field, i, "")
		}
		ret.impl = agg
	} else {
		if x.impl.Type().String() == t.ll.String() {
			ret.impl = x.impl
		} else {
			ptr := llvm.CreateAlloca(b.impl, t.ll)
			b.impl.CreateStore(x.impl, ptr)
			ret.impl = llvm.CreateLoad(b.impl, t.ll, ptr)
		}
	}
	ret.Type = t
	return
}

// The Convert instruction yields the conversion of value X to type
// Type().  One or both of those types is basic (but possibly named).
//
// A conversion may change the value and representation of its operand.
// Conversions are permitted:
//   - between real numeric types.
//   - between complex numeric types.
//   - between string and []byte or []rune.
//   - between pointers and unsafe.Pointer.
//   - between unsafe.Pointer and uintptr.
//   - from (Unicode) integer to (UTF-8) string.
//
// A conversion may imply a type name change also.
//
// Conversions may to be to or from a type parameter. All types in
// the type set of X.Type() can be converted to all types in the type
// set of Type().
//
// This operation cannot fail dynamically.
//
// Conversions of untyped string/number/bool constants to a specific
// representation are eliminated during SSA construction.
//
// Example printed form:
//
//	t1 = convert []byte <- string (t0)
func (b Builder) Convert(t Type, x Expr) (ret Expr) {
	dbgInstrf("Convert %v <- %v\n", t.RawType(), x.RawType())
	dst := t.raw.Type
	ret.Type = b.Prog.rawType(dst)
	switch typ := dst.Underlying().(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Uintptr:
			ret.impl = castUintptr(b, x.impl, x.Type, t)
			return
		case types.UnsafePointer:
			ret.impl = castPtr(b.impl, x.impl, t.ll)
			return
		case types.String:
			switch xtyp := x.RawType().Underlying().(type) {
			case *types.Slice:
				if etyp, ok := xtyp.Elem().Underlying().(*types.Basic); ok {
					switch etyp.Kind() {
					case types.Byte:
						ret.impl = b.InlineCall(b.Func.Pkg.rtFunc("StringFromBytes"), x).impl
						return
					case types.Rune:
						ret.impl = b.InlineCall(b.Func.Pkg.rtFunc("StringFromRunes"), x).impl
						return
					}
				}
			case *types.Basic:
				if xtyp.Info()&types.IsInteger == 0 {
					panic("unreachable: string conversion from non-integer basic type")
				}
				srcType := x.Type
				var arg Expr
				var fn string
				if xtyp.Info()&types.IsUnsigned != 0 {
					arg.Type = b.Prog.Uint64()
					fn = "StringFromUint64"
				} else {
					arg.Type = b.Prog.Int64()
					fn = "StringFromInt64"
				}
				arg.impl = x.impl
				if arg.impl.Type() != arg.Type.ll {
					arg.impl = castInt(b, x.impl, srcType, arg.Type)
				}
				ret.impl = b.InlineCall(b.Func.Pkg.rtFunc(fn), arg).impl
				return
			}
		case types.Complex128:
			switch xtyp := x.RawType().Underlying().(type) {
			case *types.Basic:
				if xtyp.Kind() == types.Complex64 {
					r := b.impl.CreateExtractValue(x.impl, 0, "")
					i := b.impl.CreateExtractValue(x.impl, 1, "")
					r = castFloat(b, r, b.Prog.Float64())
					i = castFloat(b, i, b.Prog.Float64())
					ret.impl = b.aggregateValue(t, r, i).impl
					return
				}
			}
		}
		switch xtyp := x.RawType().Underlying().(type) {
		case *types.Basic:
			if typ.Info()&types.IsInteger != 0 {
				// int <- int/float
				if xtyp.Info()&types.IsInteger != 0 {
					ret.impl = castInt(b, x.impl, x.Type, t)
					return
				} else if xtyp.Info()&types.IsFloat != 0 {
					ret.impl = castFloatToInt(b, x.impl, t)
					return
				}
			} else if typ.Info()&types.IsFloat != 0 {
				// float <- int/float
				if xtyp.Info()&types.IsInteger != 0 {
					if xtyp.Info()&types.IsUnsigned != 0 {
						ret.impl = llvm.CreateUIToFP(b.impl, x.impl, t.ll)
					} else {
						ret.impl = llvm.CreateSIToFP(b.impl, x.impl, t.ll)
					}
					return
				} else if xtyp.Info()&types.IsFloat != 0 {
					ret.impl = castFloat(b, x.impl, t)
					return
				}
			}
		}
		if x.kind == vkComplex && t.kind == vkComplex {
			ft := b.Prog.Float64()
			if t.raw.Type.Underlying().(*types.Basic).Kind() == types.Complex64 {
				ft = b.Prog.Float32()
			}
			r := b.impl.CreateExtractValue(x.impl, 0, "")
			i := b.impl.CreateExtractValue(x.impl, 1, "")
			ret.impl = b.Complex(Expr{castFloat(b, r, ft), ft}, Expr{castFloat(b, i, ft), ft}).impl
			return
		}
	case *types.Pointer:
		ret.impl = castPtr(b.impl, x.impl, t.ll)
		return
	case *types.Slice:
		if x.kind == vkString {
			if etyp, ok := typ.Elem().Underlying().(*types.Basic); ok {
				switch etyp.Kind() {
				case types.Byte:
					ret.impl = b.InlineCall(b.Func.Pkg.rtFunc("StringToBytes"), x).impl
					return
				case types.Rune:
					ret.impl = b.InlineCall(b.Func.Pkg.rtFunc("StringToRunes"), x).impl
					return
				}
			}
		}
	}
	if types.Identical(dst.Underlying(), x.RawType().Underlying()) {
		return b.ChangeType(t, x)
	}
	panic("todo")
}

func castUintptr(b Builder, x llvm.Value, xtyp Type, typ Type) llvm.Value {
	if x.Type().TypeKind() == llvm.PointerTypeKind {
		return llvm.CreatePtrToInt(b.impl, x, typ.ll)
	}
	if xtyp.kind == vkFloat {
		return castFloatToInt(b, x, typ)
	}
	return castInt(b, x, xtyp, typ)
}

func castInt(b Builder, x llvm.Value, xtyp Type, typ Type) llvm.Value {
	xsize := b.Prog.td.TypeAllocSize(x.Type())
	size := b.Prog.td.TypeAllocSize(typ.ll)
	if xsize > size {
		return llvm.CreateTrunc(b.impl, x, typ.ll)
	} else if xtyp.kind == vkUnsigned {
		// Source is unsigned, use zero extension
		return llvm.CreateZExt(b.impl, x, typ.ll)
	} else {
		// Source is signed, use sign extension
		return llvm.CreateSExt(b.impl, x, typ.ll)
	}
}

func castFloatToInt(b Builder, x llvm.Value, typ Type) llvm.Value {
	dstSize := b.Prog.td.TypeAllocSize(typ.ll)
	if dstSize < 8 {
		i64 := b.Prog.Int64()
		if typ.kind == vkUnsigned {
			// note(zzy): for unsigned targets, split negative vs non-negative.
			// Negative values need signed expansion (FPToSI) before truncation;
			// non-negative values can use unsigned expansion (FPToUI). This avoids
			// direct float->narrow-unsigned conversions and preserves wrap/trunc behavior.
			zero := llvm.ConstNull(x.Type())
			isNeg := b.impl.CreateFCmp(llvm.FloatOLT, x, zero, "")
			neg := llvm.CreateFPToSI(b.impl, x, i64.ll)
			pos := llvm.CreateFPToUI(b.impl, x, i64.ll)
			tmp := b.impl.CreateSelect(isNeg, neg, pos, "")
			return llvm.CreateTrunc(b.impl, tmp, typ.ll)
		}
		tmp := llvm.CreateFPToSI(b.impl, x, i64.ll)
		return llvm.CreateTrunc(b.impl, tmp, typ.ll)
	}
	// note(zzy): dst is already 64-bit wide, so no extra widen+trunc roundtrip is needed here;
	// see LLVM fptoui/fptosi semantics: https://llvm.org/docs/LangRef.html#fptoui-to-instruction
	// and https://llvm.org/docs/LangRef.html#fptosi-to-instruction
	if typ.kind == vkUnsigned {
		return llvm.CreateFPToUI(b.impl, x, typ.ll)
	}
	return llvm.CreateFPToSI(b.impl, x, typ.ll)
}

func castFloat(b Builder, x llvm.Value, typ Type) llvm.Value {
	xsize := b.Prog.td.TypeAllocSize(x.Type())
	size := b.Prog.td.TypeAllocSize(typ.ll)
	if xsize > size {
		return llvm.CreateFPTrunc(b.impl, x, typ.ll)
	} else {
		return llvm.CreateFPExt(b.impl, x, typ.ll)
	}
}

func castPtr(b llvm.Builder, x llvm.Value, t llvm.Type) llvm.Value {
	if x.Type().TypeKind() == llvm.PointerTypeKind {
		return llvm.CreatePointerCast(b, x, t)
	}
	return llvm.CreateIntToPtr(b, x, t)
}

// PtrCast converts a pointer/integer expression to a pointer type.
func (b Builder) PtrCast(t Type, x Expr) Expr {
	return Expr{castPtr(b.impl, x.impl, t.ll), t}
}

// -----------------------------------------------------------------------------

// The MakeClosure instruction yields a closure value whose code is
// Fn and whose free variables' values are supplied by Bindings.
//
// Type() returns a (possibly named) *types.Signature.
//
// Example printed form:
//
//	t0 = make closure anon@1.2 [x y z]
//	t1 = make closure bound$(main.I).add [i]
func (b Builder) MakeClosure(fn Expr, bindings []Expr) Expr {
	dbgInstrf("MakeClosure %v, %v\n", fn, bindings)
	prog := b.Prog
	tfn := fn.Type
	sig := tfn.raw.Type.(*types.Signature)
	data := prog.Nil(prog.VoidPtr()).impl
	if ctxParam := closureCtxParam(sig); ctxParam != nil {
		tctx := ctxParam.Type().Underlying().(*types.Pointer).Elem().(*types.Struct)
		ptr := b.aggregateAllocU(prog.rawType(tctx), llvmFields(bindings, tctx, b)...)
		data = ptr
	}
	return b.aggregateValue(prog.Closure(removeCtx(sig)), fn.impl, data)
}

// -----------------------------------------------------------------------------

// TODO(xsw): make inline call
func (b Builder) InlineCall(fn Expr, args ...Expr) (ret Expr) {
	return b.Call(fn, args...)
}

// The Call instruction represents a function call.
//
// The Call instruction yields the function result if there is exactly
// one.  Otherwise it returns a tuple, the components of which are
// accessed via Extract.
//
// Example printed form:
//
//	t2 = println(t0, t1)
//	t4 = t3()
func (b Builder) Call(fn Expr, args ...Expr) (ret Expr) {
	dbgInstrCall("Call", fn, args)
	var kind = fn.kind
	if kind == vkPyFuncRef {
		return b.pyCall(fn, args)
	}
	var ll llvm.Type
	var data Expr
	var sig *types.Signature
	var raw = fn.raw.Type
	switch kind {
	case vkClosure:
		data = b.Field(fn, 1)
		fn = b.Field(fn, 0)
		sig = fn.raw.Type.(*types.Signature)
		ctx := types.NewParam(token.NoPos, nil, closureCtx, types.Typ[types.UnsafePointer])
		sigCtx := FuncAddCtx(ctx, sig)
		ret.Type = b.Prog.retType(sig)
		ll = b.Prog.FuncDecl(sigCtx, InC).ll
		ret.impl = llvm.CreateCall(b.impl, ll, fn.impl, llvmParamsEx(data, args, sigCtx.Params(), b))
		return ret
	case vkFuncPtr:
		sig = raw.Underlying().(*types.Signature)
		ll = b.Prog.FuncDecl(sig, InC).ll
	case vkFuncDecl:
		sig = raw.(*types.Signature)
		ll = fn.ll
	case vkBuiltin:
		bi := raw.(*builtinTy)
		return b.BuiltinCall(bi.name, args...)
	default:
		log.Panicf("unreachable: %d(%T), %v\n", kind, raw, fn.RawType())
	}
	if b.Pkg.Path() != "reflect" {
		b.checkReflect(fn, args)
	}
	ret.Type = b.Prog.retType(sig)
	ret.impl = llvm.CreateCall(b.impl, ll, fn.impl, llvmParamsEx(data, args, sig.Params(), b))
	return
}

const (
	ReflectArrayOf = 1 << iota
	ReflectChanOf
	ReflectFuncOf
	ReflectMapOf
	ReflectPointerTo
	ReflectSliceOf
	ReflectStructOf
	ReflectMethodByIndex
	ReflectMethodByName
	ReflectMethodDynamic
	ReflectMethodMask = ReflectMethodByIndex | ReflectMethodByName | ReflectMethodDynamic
)

func (b Builder) checkReflect(fn Expr, args []Expr) {
	pkg := b.Pkg
	switch fn.Name() {
	case "reflect.ArrayOf":
		pkg.NeedAbiInit |= ReflectArrayOf
	case "reflect.ChanOf":
		pkg.NeedAbiInit |= ReflectChanOf
	case "reflect.FuncOf":
		pkg.NeedAbiInit |= ReflectFuncOf
	case "reflect.MapOf":
		pkg.NeedAbiInit |= ReflectMapOf
	case "reflect.PointerTo", "reflect.PtrTo":
		pkg.NeedAbiInit |= ReflectPointerTo
	case "reflect.SliceOf":
		pkg.NeedAbiInit |= ReflectSliceOf
	case "reflect.StructOf":
		pkg.NeedAbiInit |= ReflectStructOf
	case "reflect.Value.Method":
		if len(args) == 2 {
			if v, ok := extractConstInt(args[1].impl); ok {
				if pkg.MethodByIndex == nil {
					pkg.MethodByIndex = make(map[int]none)
				}
				pkg.MethodByIndex[v] = none{}
				pkg.NeedAbiInit |= ReflectMethodByIndex
				return
			}
			pkg.NeedAbiInit |= ReflectMethodDynamic
		}
	case "reflect.Value.MethodByName":
		if len(args) == 2 {
			if v, ok := extractConstString(args[1].impl); ok {
				if pkg.MethodByName == nil {
					pkg.MethodByName = make(map[string]none)
				}
				pkg.MethodByName[v] = none{}
				pkg.NeedAbiInit |= ReflectMethodByName
				return
			}
			pkg.NeedAbiInit |= ReflectMethodDynamic
		}
	}
}

func extractConstInt(v llvm.Value) (r int, ok bool) {
	if rv := v.IsAConstantInt(); !rv.IsNil() {
		return int(rv.SExtValue()), true
	}
	return
}

func extractConstString(v llvm.Value) (str string, ok bool) {
	if st := v.IsAConstantStruct(); !st.IsNil() {
		if init := st.Operand(0).Initializer(); !init.IsNil() {
			return init.ConstGetAsString(), true
		}
	}
	return
}

func logCall(da string, fn Expr, args []Expr) {
	if fn == Nil || fn.kind == vkBuiltin {
		return
	}
	var b bytes.Buffer
	name := fn.impl.Name()
	if name == "" {
		name = "closure"
	}
	fmt.Fprint(&b, da, " ", fn.kind, " ", fn.raw.Type, " ", name)
	sep := ": "
	for _, arg := range args {
		fmt.Fprint(&b, sep, arg.impl)
		sep = ", "
	}
	log.Println(b.String())
}

func dbgInstrCall(da string, fn Expr, args []Expr) {
	if debugInstr {
		logCall(da, fn, args)
	}
}

type DoAction int

const (
	Call DoAction = iota
	Go
	DeferAlways // defer statement executes always
	DeferInCond // defer statement executes in a conditional block
	DeferInLoop // defer statement executes in a loop block
)

// Do call a function with an action.
func (b Builder) Do(da DoAction, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) (ret Expr) {
	switch da {
	case Call:
		return buildCall(b, fn, args...)
	case Go:
		b.Go(fn, buildCall, args...)
	default:
		b.Defer(da, fn, buildCall, args...)
	}
	return
}

// compareSelect performs a series of comparisons and selections based on the
// given comparison op. It's used to implement operations like min and max.
//
// The function iterates through the provided expressions, comparing each with
// the current result using the specified comparison op. It selects the
// appropriate value based on the comparison.
func (b Builder) compareSelect(op token.Token, x Expr, y ...Expr) Expr {
	ret := x
	for _, v := range y {
		cond := b.BinOp(op, ret, v)
		sel := llvm.CreateSelect(b.impl, cond.impl, ret.impl, v.impl)
		ret = Expr{sel, ret.Type}
	}
	return ret
}

// SelectValue chooses between two values based on the condition.
func (b Builder) SelectValue(cond Expr, a Expr, bExpr Expr) Expr {
	sel := llvm.CreateSelect(b.impl, cond.impl, a.impl, bExpr.impl)
	return Expr{sel, a.Type}
}

// The SliceToArrayPointer instruction yields the conversion of slice X to
// array pointer.
//
// Pos() returns the ast.CallExpr.Lparen, if the instruction arose
// from an explicit conversion in the source.
//
// Conversion may to be to or from a type parameter. All types in
// the type set of X.Type() must be a slice types that can be converted to
// all types in the type set of Type() which must all be pointer to array
// types.
//
// This operation can fail dynamically if the length of the slice is less
// than the length of the array.
//
// Example printed form:
//
//	t1 = slice to array pointer *[4]byte <- []byte (t0)
func (b Builder) SliceToArrayPointer(x Expr, typ Type) (ret Expr) {
	ret.Type = typ
	max := b.Prog.IntVal(uint64(typ.RawType().Underlying().(*types.Pointer).Elem().Underlying().(*types.Array).Len()), b.Prog.Int())
	failed := Expr{llvm.CreateICmp(b.impl, llvm.IntSLT, b.SliceLen(x).impl, max.impl), b.Prog.Bool()}
	b.IfThen(failed, func() {
		b.InlineCall(b.Pkg.rtFunc("PanicSliceConvert"), b.SliceLen(x), max)
	})
	ret.impl = b.SliceData(x).impl
	return
}

// A Builtin represents a specific use of a built-in function, e.g. len.
//
// Builtins are immutable values.  Builtins do not have addresses.
//
// `fn` indicates the function: one of the built-in functions from the
// Go spec (excluding "make" and "new").
func (b Builder) BuiltinCall(fn string, args ...Expr) (ret Expr) {
	switch fn {
	case "len":
		if len(args) == 1 {
			arg := args[0]
			switch arg.kind {
			case vkSlice:
				return b.SliceLen(arg)
			case vkString:
				return b.StringLen(arg)
			case vkChan:
				return b.InlineCall(b.Pkg.rtFunc("ChanLen"), arg)
			case vkMap:
				return b.InlineCall(b.Pkg.rtFunc("MapLen"), arg)
			}
		}
	case "cap":
		if len(args) == 1 {
			arg := args[0]
			switch arg.kind {
			case vkSlice:
				return b.SliceCap(arg)
			case vkChan:
				return b.InlineCall(b.Pkg.rtFunc("ChanCap"), arg)
			}
		}
	case "append":
		if len(args) == 2 {
			src := args[0]
			if src.kind == vkSlice {
				elem := args[1]
				switch elem.kind {
				case vkSlice:
					etSize := b.Prog.SizeOf(b.Prog.Elem(elem.Type))
					ret.Type = src.Type
					ret.impl = b.InlineCall(b.Pkg.rtFunc("SliceAppend"),
						src, b.SliceData(elem), b.SliceLen(elem), b.Prog.Val(int(etSize))).impl
					return
				case vkString:
					etSize := b.Prog.SizeOf(b.Prog.Byte())
					ret.Type = src.Type
					ret.impl = b.InlineCall(b.Pkg.rtFunc("SliceAppend"),
						src, b.StringData(elem), b.StringLen(elem), b.Prog.Val(int(etSize))).impl
					return
				default:
					etSize := b.Prog.SizeOf(elem.Type)
					ret.Type = src.Type
					ret.impl = b.InlineCall(b.Pkg.rtFunc("SliceAppend"),
						src, elem, b.Const(constant.MakeInt64(1), b.Prog.Int()), b.Prog.Val(int(etSize))).impl
					return
				}
			}
		}
	case "copy":
		if len(args) == 2 {
			dst := args[0]
			if dst.kind == vkSlice {
				src := args[1]
				prog := b.Prog
				etSize := prog.Val(int(prog.SizeOf(prog.Elem(dst.Type))))
				switch src.kind {
				case vkSlice:
					return b.InlineCall(b.Pkg.rtFunc("SliceCopy"), dst, b.SliceData(src), b.SliceLen(src), etSize)
				case vkString:
					return b.InlineCall(b.Pkg.rtFunc("SliceCopy"), dst, b.StringData(src), b.StringLen(src), etSize)
				}
			}
		}
	case "close":
		if len(args) == 1 {
			arg := args[0]
			switch arg.kind {
			case vkChan:
				return b.InlineCall(b.Pkg.rtFunc("ChanClose"), arg)
			}
		}
	case "recover":
		return b.Recover()
	case "ssa:deferstack":
		return b.DeferStack()
	case "print", "println":
		return b.PrintEx(fn == "println", args...)
	case "complex":
		return b.Complex(args[0], args[1])
	case "real":
		return b.getField(args[0], 0)
	case "imag":
		return b.getField(args[0], 1)
	case "String": // unsafe.String
		return b.unsafeString(args[0].impl, args[1].impl)
	case "Slice": // unsafe.Slice
		size := b.fitIntSize(args[1])
		return b.unsafeSlice(args[0], size.impl, size.impl)
	case "StringData":
		return b.StringData(args[0]) // TODO(xsw): check return type
	case "SliceData":
		return b.SliceData(args[0]) // TODO(xsw): check return type
	case "delete":
		if len(args) == 2 && args[0].kind == vkMap {
			m := args[0]
			t := b.abiType(m.raw.Type)
			ptr := b.mapKeyPtr(args[1])
			b.Call(b.Pkg.rtFunc("MapDelete"), t, m, ptr)
			return
		}
	case "clear":
		if len(args) == 1 {
			arg := args[0]
			switch arg.kind {
			case vkMap:
				m := arg
				t := b.abiType(m.raw.Type)
				b.Call(b.Pkg.rtFunc("MapClear"), t, m)
				return
			case vkSlice:
				s := arg
				t := b.abiType(s.raw.Type)
				b.Call(b.Pkg.rtFunc("SliceClear"), t, s)
				return
			}
		}
	case "min":
		if len(args) > 0 {
			return b.compareSelect(token.LSS, args[0], args[1:]...)
		}
	case "max":
		if len(args) > 0 {
			return b.compareSelect(token.GTR, args[0], args[1:]...)
		}
	case "Add":
		return b.Advance(args[0], args[1])
	case "Sizeof":
		// instance of generic function
		return b.Prog.Val(int(b.Prog.SizeOf(args[0].Type)))
	case "Alignof":
		// instance of generic function
		return b.Prog.Val(int(b.Prog.td.ABITypeAlignment(args[0].ll)))
	case "Offsetof":
		// instance of generic function
		if load := args[0].impl.IsALoadInst(); !load.IsNil() {
			if gep := load.Operand(0).IsAGetElementPtrInst(); !gep.IsNil() {
				typ := gep.GEPSourceElementType()
				offset := gep.Operand(2).IsAConstantInt()
				if typ.TypeKind() == llvm.StructTypeKind && !offset.IsNil() {
					return b.Prog.Val(int(b.Prog.td.ElementOffset(typ, int(offset.SExtValue()))))
				}
			}
		}
		panic("invalid argument for unsafe.Offsetof: must be a selector expression")
	case "panic":
		b.Panic(args[0])
		return
	}

	panic("todo: " + fn)
}

func (p Program) tyPrintf() *types.Signature {
	if p.printfTy == nil {
		pchar := types.NewPointer(types.Typ[types.Int8])
		params := types.NewTuple(types.NewVar(0, nil, "format", pchar), VArg())
		rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32]))
		p.printfTy = types.NewSignatureType(nil, nil, nil, params, rets, true)
	}
	return p.printfTy
}

func (b Builder) Printf(fmt string, args ...Expr) Expr {
	fn := b.Pkg.cFunc("printf", b.Prog.tyPrintf())
	return b.Call(fn, append([]Expr{b.CStr(fmt)}, args...)...)
}

// Println prints the arguments to stderr, followed by a newline.
func (b Builder) Println(args ...Expr) (ret Expr) {
	return b.PrintEx(true, args...)
}

// PrintEx prints the arguments to stderr.
func (b Builder) PrintEx(ln bool, args ...Expr) (ret Expr) {
	prog := b.Prog
	ret.Type = prog.Void()
	for i, arg := range args {
		if ln && i > 0 {
			b.InlineCall(b.Pkg.rtFunc("PrintByte"), prog.IntVal(' ', prog.Byte()))
		}
		var fn string
		typ := arg.Type
		switch arg.kind {
		case vkBool:
			fn = "PrintBool"
		case vkSigned:
			fn = "PrintInt"
			typ = prog.Int64()
		case vkUnsigned:
			fn = "PrintUint"
			typ = prog.Uint64()
		case vkFloat:
			fn = "PrintFloat"
			typ = prog.Float64()
		case vkSlice:
			fn = "PrintSlice"
		case vkClosure:
			arg = b.Field(arg, 0)
			fallthrough
		case vkPtr, vkFuncPtr, vkFuncDecl:
			fn = "PrintPointer"
			typ = prog.VoidPtr()
		case vkString:
			fn = "PrintString"
		case vkEface:
			fn = "PrintEface"
		case vkIface:
			fn = "PrintIface"
		case vkComplex:
			fn = "PrintComplex"
			typ = prog.Complex128()
		case vkChan:
			fn = "PrintPointer"
			typ = prog.VoidPtr()
		case vkMap:
			fn = "PrintPointer"
			typ = prog.VoidPtr()
		default:
			panic(fmt.Errorf("illegal types for operand: print %v", arg.RawType()))
		}
		if typ != arg.Type {
			arg = b.Convert(typ, arg)
		}
		b.InlineCall(b.Pkg.rtFunc(fn), arg)
	}
	if ln {
		b.InlineCall(b.Pkg.rtFunc("PrintByte"), prog.IntVal('\n', prog.Byte()))
	}
	return
}

// -----------------------------------------------------------------------------

func checkExpr(v Expr, t types.Type, b Builder) Expr {
	if st, ok := t.Underlying().(*types.Struct); ok && IsClosure(st) {
		if v.kind == vkClosure {
			return v
		}
		prog := b.Prog
		origKind := v.kind
		tclosure := prog.rawType(t)
		fnType := prog.Field(tclosure, 0)
		if v.Type != fnType {
			// Signature conversions are representationally identical.
			if v.Type.kind == vkFuncDecl || v.Type.kind == vkFuncPtr {
				v = b.ChangeType(fnType, v)
			} else {
				v = b.Convert(fnType, v)
			}
		}
		data := prog.Nil(prog.VoidPtr())
		if origKind == vkFuncDecl || origKind == vkFuncPtr {
			if sig, ok := fnType.raw.Type.(*types.Signature); ok && closureCtxParam(sig) == nil {
				v, data = b.Pkg.closureStub(b, v, sig, origKind)
			}
		}
		return b.aggregateValue(tclosure, v.impl, data.impl)
	}
	if types.Identical(v.raw.Type, t) || !types.AssignableTo(v.raw.Type, t) {
		return v
	}
	dst := b.Prog.Type(t, InGo)
	// Range and assignment lowering can produce a value whose source type is
	// assignable but not identical to the destination, so preserve the value
	// while retagging it with the destination runtime type.
	if _, ok := t.Underlying().(*types.Interface); ok {
		if _, srcIsInterface := v.raw.Type.Underlying().(*types.Interface); srcIsInterface {
			return b.ChangeInterface(dst, v)
		}
		return b.MakeInterface(dst, v)
	}
	return b.ChangeType(dst, v)
}

func needsNegativeCheck(x Expr) bool {
	if x.kind == vkSigned {
		if rv := x.impl.IsAConstantInt(); !rv.IsNil() && rv.SExtValue() >= 0 {
			return false
		}
		return true
	}
	return false
}

func llvmParamsEx(data Expr, vals []Expr, params *types.Tuple, b Builder) (ret []llvm.Value) {
	if data.IsNil() {
		return llvmParams(0, vals, params, b)
	}
	ret = llvmParams(1, vals, params, b)
	ret[0] = data.impl
	return
}

func llvmParams(base int, vals []Expr, params *types.Tuple, b Builder) (ret []llvm.Value) {
	n := params.Len()
	if n > 0 {
		ret = make([]llvm.Value, len(vals)+base)
		for idx, v := range vals {
			i := base + idx
			if i < n {
				v = checkExpr(v, params.At(i).Type(), b)
			}
			ret[i] = v.impl
		}
	}
	return
}

func llvmFields(vals []Expr, t *types.Struct, b Builder) (ret []llvm.Value) {
	n := t.NumFields()
	if n > 0 {
		ret = make([]llvm.Value, len(vals))
		for i, v := range vals {
			if i < n {
				v = checkExpr(v, t.Field(i).Type(), b)
			}
			ret[i] = v.impl
		}
	}
	return
}

// -----------------------------------------------------------------------------
