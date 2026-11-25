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
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/types"
)

type anInstruction struct {
	block *BasicBlock
	pos   token.Pos
}

func (i *anInstruction) Block() *BasicBlock { return i.block }
func (i *anInstruction) Pos() token.Pos     { return i.pos }
func (i *anInstruction) Parent() *Function {
	if i.block == nil {
		return nil
	}
	return i.block.parent
}
func (i *anInstruction) Referrers() *[]Instruction { return nil }
func (i *anInstruction) setBlock(b *BasicBlock)    { i.block = b }
func (i *anInstruction) setPos(p token.Pos)        { i.pos = p }

type register struct {
	anInstruction
	num       int
	typ       types.Type
	referrers []Instruction
}

func (r *register) Name() string              { return fmt.Sprintf("t%d", r.num) }
func (r *register) Type() types.Type          { return r.typ }
func (r *register) Referrers() *[]Instruction { return &r.referrers }
func (r *register) setType(t types.Type)      { r.typ = t }
func (r *register) setNum(n int)              { r.num = n }

type instrSetter interface {
	setBlock(*BasicBlock)
	setPos(token.Pos)
}

type regSetter interface {
	setType(types.Type)
	setNum(int)
}

// --- All 37 standard x/tools/go/ssa concrete Instruction types ---

// Alloc yields the address of a new variable of type T (*T).
type Alloc struct {
	register
	Comment string
	Heap    bool
	NoScan  bool // LLGo extension: GC pointer-free allocation hint
}

func (v *Alloc) Operands(rands []*Value) []*Value { return rands }
func (v *Alloc) String() string {
	op := "local"
	if v.Heap {
		op = "new"
	}
	var elem types.Type
	if ptr, ok := v.typ.(*types.Pointer); ok {
		elem = ptr.Elem()
	} else {
		elem = v.typ
	}
	if v.Comment != "" {
		return fmt.Sprintf("%s %s (%s)", op, elem, v.Comment)
	}
	return fmt.Sprintf("%s %s", op, elem)
}

// BinOp yields the result of binary operation X Op Y.
type BinOp struct {
	register
	Op   token.Token
	X, Y Value
}

func (v *BinOp) Operands(rands []*Value) []*Value { return append(rands, &v.X, &v.Y) }
func (v *BinOp) String() string                   { return fmt.Sprintf("%s %s %s", valName(v.X), v.Op, valName(v.Y)) }

// UnOp yields the result of unary operation Op X.
type UnOp struct {
	register
	Op      token.Token
	X       Value
	CommaOk bool
}

func (v *UnOp) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *UnOp) String() string {
	if v.CommaOk {
		return fmt.Sprintf("%s%s,ok", v.Op, valName(v.X))
	}
	return fmt.Sprintf("%s%s", v.Op, valName(v.X))
}

// Call represents a function or method call.
type Call struct {
	register
	Call CallCommon
}

func (v *Call) Common() *CallCommon              { return &v.Call }
func (v *Call) Value() *Call                     { return v }
func (v *Call) Operands(rands []*Value) []*Value { return v.Call.Operands(rands) }
func (v *Call) String() string                   { return v.Call.String() }

// ChangeInterface constructs a value of one interface type from another.
type ChangeInterface struct {
	register
	X Value
}

func (v *ChangeInterface) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *ChangeInterface) String() string {
	return fmt.Sprintf("change interface %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// ChangeType constructs a value of one type from an identically-represented type.
type ChangeType struct {
	register
	X Value
}

func (v *ChangeType) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *ChangeType) String() string {
	return fmt.Sprintf("changetype %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// Convert converts X to v.Type().
type Convert struct {
	register
	X Value
}

func (v *Convert) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *Convert) String() string {
	return fmt.Sprintf("convert %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// MultiConvert converts between values whose type-parameter sets may match multiple ways.
type MultiConvert struct {
	register
	X Value
}

func (v *MultiConvert) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *MultiConvert) String() string {
	return fmt.Sprintf("multiconvert %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// SliceToArrayPointer converts a slice X to an array pointer.
type SliceToArrayPointer struct {
	register
	X Value
}

func (v *SliceToArrayPointer) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *SliceToArrayPointer) String() string {
	return fmt.Sprintf("slice to array pointer %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// DebugRef associates a source AST expression with its SSA value or address.
type DebugRef struct {
	anInstruction
	Expr   ast.Expr
	IsAddr bool
	X      Value

	object types.Object
}

func (d *DebugRef) Object() types.Object             { return d.object }
func (d *DebugRef) Operands(rands []*Value) []*Value { return append(rands, &d.X) }
func (d *DebugRef) String() string {
	kind := "val"
	if d.IsAddr {
		kind = "addr"
	}
	return fmt.Sprintf("; %s %s", kind, valName(d.X))
}

// Defer pushes a call onto the current function's defer stack.
type Defer struct {
	anInstruction
	Call       CallCommon
	DeferStack Value
}

func (d *Defer) Common() *CallCommon { return &d.Call }
func (d *Defer) Value() *Call        { return nil }
func (d *Defer) Operands(rands []*Value) []*Value {
	rands = d.Call.Operands(rands)
	if d.DeferStack != nil {
		rands = append(rands, &d.DeferStack)
	}
	return rands
}
func (d *Defer) String() string { return "defer " + d.Call.String() }

// Extract yields component Index of tuple Tuple.
type Extract struct {
	register
	Tuple Value
	Index int
}

func (v *Extract) Operands(rands []*Value) []*Value { return append(rands, &v.Tuple) }
func (v *Extract) String() string                   { return fmt.Sprintf("extract %s #%d", valName(v.Tuple), v.Index) }

// Field yields field Field of struct value X.
type Field struct {
	register
	X     Value
	Field int
}

func (v *Field) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *Field) String() string                   { return fmt.Sprintf("%s.#%d", valName(v.X), v.Field) }

// FieldAddr yields the address of field Field of *struct X.
type FieldAddr struct {
	register
	X     Value
	Field int
}

func (v *FieldAddr) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *FieldAddr) String() string                   { return fmt.Sprintf("&%s.#%d", valName(v.X), v.Field) }

// Go creates a new goroutine calling Call.
type Go struct {
	anInstruction
	Call CallCommon
}

func (g *Go) Common() *CallCommon              { return &g.Call }
func (g *Go) Value() *Call                     { return nil }
func (g *Go) Operands(rands []*Value) []*Value { return g.Call.Operands(rands) }
func (g *Go) String() string                   { return "go " + g.Call.String() }

// If transfers control to Block().Succs[0] if Cond is true, else Succs[1].
type If struct {
	anInstruction
	Cond Value
}

func (s *If) Operands(rands []*Value) []*Value { return append(rands, &s.Cond) }
func (s *If) String() string {
	if s.block != nil && len(s.block.Succs) >= 2 {
		return fmt.Sprintf("if %s goto %s else %s", valName(s.Cond), s.block.Succs[0], s.block.Succs[1])
	}
	return fmt.Sprintf("if %s", valName(s.Cond))
}

// Index yields element Index of array/string X.
type Index struct {
	register
	X     Value
	Index Value
}

func (v *Index) Operands(rands []*Value) []*Value { return append(rands, &v.X, &v.Index) }
func (v *Index) String() string                   { return fmt.Sprintf("%s[%s]", valName(v.X), valName(v.Index)) }

// IndexAddr yields the address of element Index of slice/*array X.
type IndexAddr struct {
	register
	X     Value
	Index Value
}

func (v *IndexAddr) Operands(rands []*Value) []*Value { return append(rands, &v.X, &v.Index) }
func (v *IndexAddr) String() string                   { return fmt.Sprintf("&%s[%s]", valName(v.X), valName(v.Index)) }

// Jump unconditionally transfers control to Block().Succs[0].
type Jump struct {
	anInstruction
}

func (*Jump) Operands(rands []*Value) []*Value { return rands }
func (s *Jump) String() string {
	if s.block != nil && len(s.block.Succs) >= 1 {
		return fmt.Sprintf("jump %s", s.block.Succs[0])
	}
	return "jump"
}

// Lookup yields element Index of map/string X.
type Lookup struct {
	register
	X       Value
	Index   Value
	CommaOk bool
}

func (v *Lookup) Operands(rands []*Value) []*Value { return append(rands, &v.X, &v.Index) }
func (v *Lookup) String() string {
	if v.CommaOk {
		return fmt.Sprintf("%s[%s],ok", valName(v.X), valName(v.Index))
	}
	return fmt.Sprintf("%s[%s]", valName(v.X), valName(v.Index))
}

// MakeChan allocates a channel with buffer size Size.
type MakeChan struct {
	register
	Size Value
}

func (v *MakeChan) Operands(rands []*Value) []*Value { return append(rands, &v.Size) }
func (v *MakeChan) String() string                   { return fmt.Sprintf("make %s %s", v.typ, valName(v.Size)) }

// MakeClosure yields a closure value binding Fn with Bindings.
type MakeClosure struct {
	register
	Fn       Value
	Bindings []Value
}

func (v *MakeClosure) Operands(rands []*Value) []*Value {
	rands = append(rands, &v.Fn)
	for i := range v.Bindings {
		rands = append(rands, &v.Bindings[i])
	}
	return rands
}
func (v *MakeClosure) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "make closure %s [", valName(v.Fn))
	for i, bind := range v.Bindings {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(valName(bind))
	}
	b.WriteByte(']')
	return b.String()
}

// MakeInterface constructs an interface value from concrete value X.
type MakeInterface struct {
	register
	X Value
}

func (v *MakeInterface) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *MakeInterface) String() string {
	return fmt.Sprintf("make %s <- %s (%s)", v.typ, valType(v.X), valName(v.X))
}

// MakeMap allocates a map with optional initial capacity Reserve.
type MakeMap struct {
	register
	Reserve Value
}

func (v *MakeMap) Operands(rands []*Value) []*Value {
	if v.Reserve != nil {
		return append(rands, &v.Reserve)
	}
	return rands
}
func (v *MakeMap) String() string {
	if v.Reserve != nil {
		return fmt.Sprintf("make %s %s", v.typ, valName(v.Reserve))
	}
	return fmt.Sprintf("make %s", v.typ)
}

// MakeSlice allocates a slice with length Len and capacity Cap.
type MakeSlice struct {
	register
	Len Value
	Cap Value
}

func (v *MakeSlice) Operands(rands []*Value) []*Value { return append(rands, &v.Len, &v.Cap) }
func (v *MakeSlice) String() string {
	return fmt.Sprintf("make %s %s %s", v.typ, valName(v.Len), valName(v.Cap))
}

// MapUpdate updates Map[Key] = Value.
type MapUpdate struct {
	anInstruction
	Map   Value
	Key   Value
	Value Value
}

func (s *MapUpdate) Operands(rands []*Value) []*Value { return append(rands, &s.Map, &s.Key, &s.Value) }
func (s *MapUpdate) String() string {
	return fmt.Sprintf("%s[%s] = %s", valName(s.Map), valName(s.Key), valName(s.Value))
}

// Next advances iterator Iter produced by Range.
type Next struct {
	register
	Iter     Value
	IsString bool
}

func (v *Next) Operands(rands []*Value) []*Value { return append(rands, &v.Iter) }
func (v *Next) String() string                   { return fmt.Sprintf("next %s", valName(v.Iter)) }

// Panic initiates a panic with value X.
type Panic struct {
	anInstruction
	X Value
}

func (s *Panic) Operands(rands []*Value) []*Value { return append(rands, &s.X) }
func (s *Panic) String() string                   { return fmt.Sprintf("panic %s", valName(s.X)) }

// Phi selects Edges[i] corresponding to predecessor Block().Preds[i].
type Phi struct {
	register
	Comment string
	Edges   []Value
}

func (v *Phi) Operands(rands []*Value) []*Value {
	for i := range v.Edges {
		rands = append(rands, &v.Edges[i])
	}
	return rands
}
func (v *Phi) String() string {
	var b strings.Builder
	b.WriteString("phi [")
	for i, e := range v.Edges {
		if i > 0 {
			b.WriteString(", ")
		}
		if v.block != nil && i < len(v.block.Preds) {
			fmt.Fprintf(&b, "%s: %s", v.block.Preds[i], valName(e))
		} else {
			b.WriteString(valName(e))
		}
	}
	b.WriteByte(']')
	if v.Comment != "" {
		fmt.Fprintf(&b, " #%s", v.Comment)
	}
	return b.String()
}

// Range creates an iterator over map or string X.
type Range struct {
	register
	X Value
}

func (v *Range) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *Range) String() string                   { return fmt.Sprintf("range %s", valName(v.X)) }

// Return returns Results from the enclosing Function.
type Return struct {
	anInstruction
	Results []Value
}

func (s *Return) Operands(rands []*Value) []*Value {
	for i := range s.Results {
		rands = append(rands, &s.Results[i])
	}
	return rands
}
func (s *Return) String() string {
	if len(s.Results) == 0 {
		return "return"
	}
	parts := make([]string, len(s.Results))
	for i, r := range s.Results {
		parts[i] = valName(r)
	}
	return "return " + strings.Join(parts, ", ")
}

// RunDefers pops and invokes all deferred calls on the current function's stack.
type RunDefers struct {
	anInstruction
}

func (*RunDefers) Operands(rands []*Value) []*Value { return rands }
func (*RunDefers) String() string                   { return "rundefers" }

// Select executes a Go select statement across States.
type Select struct {
	register
	States   []*SelectState
	Blocking bool
}

func (v *Select) Operands(rands []*Value) []*Value {
	for _, st := range v.States {
		rands = append(rands, &st.Chan)
		if st.Send != nil {
			rands = append(rands, &st.Send)
		}
	}
	return rands
}
func (v *Select) String() string {
	kind := "nonblocking"
	if v.Blocking {
		kind = "blocking"
	}
	parts := make([]string, len(v.States))
	for i, st := range v.States {
		if st.Dir == types.SendOnly {
			parts[i] = fmt.Sprintf("send %s %s", valName(st.Chan), valName(st.Send))
		} else {
			parts[i] = fmt.Sprintf("recv %s", valName(st.Chan))
		}
	}
	return fmt.Sprintf("select %s [%s]", kind, strings.Join(parts, ", "))
}

// Send sends X on channel Chan.
type Send struct {
	anInstruction
	Chan, X Value
}

func (s *Send) Operands(rands []*Value) []*Value { return append(rands, &s.Chan, &s.X) }
func (s *Send) String() string                   { return fmt.Sprintf("send %s <- %s", valName(s.Chan), valName(s.X)) }

// Slice constructs a sub-slice or substring of X[Low:High:Max].
type Slice struct {
	register
	X              Value
	Low, High, Max Value
}

func (v *Slice) Operands(rands []*Value) []*Value {
	rands = append(rands, &v.X)
	if v.Low != nil {
		rands = append(rands, &v.Low)
	}
	if v.High != nil {
		rands = append(rands, &v.High)
	}
	if v.Max != nil {
		rands = append(rands, &v.Max)
	}
	return rands
}
func (v *Slice) String() string {
	lo, hi, mx := "", "", ""
	if v.Low != nil {
		lo = valName(v.Low)
	}
	if v.High != nil {
		hi = valName(v.High)
	}
	if v.Max != nil {
		mx = ":" + valName(v.Max)
	}
	return fmt.Sprintf("slice %s[%s:%s%s]", valName(v.X), lo, hi, mx)
}

// Store writes Val into address Addr.
type Store struct {
	anInstruction
	Addr Value
	Val  Value
}

func (s *Store) Operands(rands []*Value) []*Value { return append(rands, &s.Addr, &s.Val) }
func (s *Store) String() string                   { return fmt.Sprintf("*%s = %s", valName(s.Addr), valName(s.Val)) }

// TypeAssert tests whether interface value X holds AssertedType.
type TypeAssert struct {
	register
	X            Value
	AssertedType types.Type
	CommaOk      bool
}

func (v *TypeAssert) Operands(rands []*Value) []*Value { return append(rands, &v.X) }
func (v *TypeAssert) String() string {
	if v.CommaOk {
		return fmt.Sprintf("typeassert,ok %s.(%s)", valName(v.X), v.AssertedType)
	}
	return fmt.Sprintf("typeassert %s.(%s)", valName(v.X), v.AssertedType)
}

func valType(v Value) types.Type {
	if v == nil {
		return nil
	}
	return v.Type()
}
