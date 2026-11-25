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

// Package ssa defines LLGo's pure-Go, backend-independent mid-level SSA IR.
//
// Status: Phase 1 foundational IR (experimental; instruction set will be
// finalized as Phase 2 connects the Go frontend lowering and LLVM backend).
//
// Following golang.org/x/tools/go/ssa conventions (Package, Function,
// BasicBlock, Parameter, Instruction, Value, Pos, Extract, DebugRef), package
// ssa sits between the Go frontend and codegen backends (e.g. internal/llvmgen)
// and models LLGo-specific concepts as first-class instructions and types:
// closures (MakeClosure/CallClosure), native C/Stdcall calls (CallCFunc),
// explicit GoPtr vs CPtr pointer conversions (PtrCast), GC hooks (Alloc with
// NoScan, WriteBarrier, Safepoint, GCRoot, KeepAlive), and Go
// exception/concurrency primitives (Defer, RunDefers, Panic, Recover, Go).
//
// It requires only Go 1.20+ with zero CGo, LLVM, or x/tools dependencies.
// PackageSummary provides a deterministic bootstrap summary representation;
// compact binary encoding lives in the upcoming github.com/xgo-dev/llgo/exportdata
// package.
package ssa

import (
	"fmt"
	"go/constant"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

// ValueID uniquely identifies an SSA value within a Function.
type ValueID uint32

// Package is the top-level compilation unit of LLGo mid-level SSA IR.
type Package struct {
	Path    string
	Name    string
	Target  target.Spec
	Kind    directive.PackageKind
	Imports []string
	Exports map[string]string // go symbol -> exported C/Wasm symbol
	Globals []*Global
	Funcs   []*Function
}

// Global describes a package-level variable or constant symbol.
type Global struct {
	Name       string
	LinkName   string
	Type       types.Type
	Locality   directive.Locality
	Constant   bool
	LinkOnce   bool
	External   bool
	HasPtrData bool
}

// Function describes a function or method in mid-level SSA IR.
type Function struct {
	Pkg        *Package
	Name       string
	LinkName   string
	Sig        *types.Signature
	Params     []*Parameter
	Blocks     []*BasicBlock
	LinkOnce   bool
	NoInline   bool
	HasEnv     bool
	ExportName string
	WasmImport *directive.WasmImport

	nextID ValueID
}

// IsDeclaration reports whether the function has no basic blocks.
func (f *Function) IsDeclaration() bool {
	return len(f.Blocks) == 0
}

// BasicBlock is a single basic block in a Function's CFG.
type BasicBlock struct {
	Func   *Function
	Index  int
	Label  string
	Instrs []Instruction
}

// Parent returns the Function to which b belongs.
func (b *BasicBlock) Parent() *Function {
	return b.Func
}

// Terminator returns the block's final instruction if it is a Terminator.
func (b *BasicBlock) Terminator() Terminator {
	if len(b.Instrs) == 0 {
		return nil
	}
	t, _ := b.Instrs[len(b.Instrs)-1].(Terminator)
	return t
}

// Value is any typed SSA value usable as an operand (Parameter, Const,
// GlobalRef, or value-producing Instruction).
type Value interface {
	ID() ValueID
	Type() types.Type
	Name() string
	valMarker()
}

// Instruction is an instruction inside a BasicBlock.
type Instruction interface {
	Block() *BasicBlock
	Parent() *Function
	Pos() token.Pos
	setBlock(*BasicBlock)
	setPos(token.Pos)
	Operands() []Value
	String() string
	instrMarker()
}

// Terminator marks instructions that end a basic block.
type Terminator interface {
	Instruction
	Successors() []*BasicBlock
	termMarker()
}

type baseInstr struct {
	blk *BasicBlock
	pos token.Pos
}

func (i *baseInstr) Block() *BasicBlock     { return i.blk }
func (i *baseInstr) Pos() token.Pos         { return i.pos }
func (i *baseInstr) setBlock(b *BasicBlock) { i.blk = b }
func (i *baseInstr) setPos(pos token.Pos)   { i.pos = pos }
func (i *baseInstr) Parent() *Function {
	if i.blk == nil {
		return nil
	}
	return i.blk.Func
}
func (*baseInstr) instrMarker() {}

type baseValue struct {
	id  ValueID
	typ types.Type
}

func (v *baseValue) ID() ValueID      { return v.id }
func (v *baseValue) Type() types.Type { return v.typ }
func (v *baseValue) Name() string     { return fmt.Sprintf("v%d", v.id) }
func (*baseValue) valMarker()         {}

// Parameter is a formal function parameter.
type Parameter struct {
	baseValue
	ParamName string
	Index     int
}

func (p *Parameter) Name() string {
	if p.ParamName != "" {
		return fmt.Sprintf("%%%s", p.ParamName)
	}
	return fmt.Sprintf("%%p%d", p.Index)
}

// Const is a typed compile-time constant value (matching go/ssa.Const: a nil
// Value represents the typed zero/nil value of Type()).
type Const struct {
	baseValue
	Value constant.Value
}

func (c *Const) Name() string {
	if c.Value != nil {
		return c.Value.ExactString()
	}
	if c.typ != nil {
		return "zero:" + c.typ.String()
	}
	return "nil"
}

// GlobalRef references the address of a package Global.
type GlobalRef struct {
	baseValue
	Global *Global
}

func (g *GlobalRef) Name() string { return "@" + g.Global.Name }

// Phi merges SSA values from predecessor blocks.
type Phi struct {
	baseInstr
	baseValue
	Edges []PhiEdge
}

// PhiEdge pairs a predecessor block with its incoming SSA value.
type PhiEdge struct {
	Pred *BasicBlock
	Val  Value
}

func (p *Phi) Operands() []Value {
	ops := make([]Value, len(p.Edges))
	for i, e := range p.Edges {
		ops[i] = e.Val
	}
	return ops
}

func (p *Phi) String() string {
	parts := make([]string, len(p.Edges))
	for i, e := range p.Edges {
		parts[i] = fmt.Sprintf("[%s: %s]", e.Pred.Label, e.Val.Name())
	}
	return fmt.Sprintf("%s = phi %s %s", p.Name(), p.typ.String(), strings.Join(parts, ", "))
}

// BinOp computes a binary arithmetic, bitwise, or comparison operation.
type BinOp struct {
	baseInstr
	baseValue
	Op   token.Token
	X, Y Value
}

func (i *BinOp) Operands() []Value { return []Value{i.X, i.Y} }
func (i *BinOp) String() string {
	return fmt.Sprintf("%s = binop %s %s, %s : %s", i.Name(), i.Op, i.X.Name(), i.Y.Name(), i.typ)
}

// UnOp computes a unary operation (-, ^, !, etc.).
type UnOp struct {
	baseInstr
	baseValue
	Op token.Token
	X  Value
}

func (i *UnOp) Operands() []Value { return []Value{i.X} }
func (i *UnOp) String() string {
	return fmt.Sprintf("%s = unop %s %s : %s", i.Name(), i.Op, i.X.Name(), i.typ)
}

// Convert converts X to target type.
type Convert struct {
	baseInstr
	baseValue
	X Value
}

func (i *Convert) Operands() []Value { return []Value{i.X} }
func (i *Convert) String() string {
	return fmt.Sprintf("%s = convert %s to %s", i.Name(), i.X.Name(), i.typ)
}

// PtrCast explicitly converts between GoPtr and NativePtr address spaces.
type PtrCast struct {
	baseInstr
	baseValue
	X Value
}

func (i *PtrCast) Operands() []Value { return []Value{i.X} }
func (i *PtrCast) String() string {
	return fmt.Sprintf("%s = ptrcast %s to %s", i.Name(), i.X.Name(), i.typ)
}

// Alloc allocates storage on the stack or heap with explicit GC NoScan metadata.
type Alloc struct {
	baseInstr
	baseValue
	Elem   types.Type
	Heap   bool
	NoScan bool
	Align  int64
}

func (i *Alloc) Operands() []Value { return nil }
func (i *Alloc) String() string {
	space := "stack"
	if i.Heap {
		space = "heap"
	}
	scan := "scan"
	if i.NoScan {
		scan = "noscan"
	}
	return fmt.Sprintf("%s = alloc.%s.%s %s", i.Name(), space, scan, i.Elem)
}

// Load loads a value from pointer Addr.
type Load struct {
	baseInstr
	baseValue
	Addr   Value
	Atomic bool
}

func (i *Load) Operands() []Value { return []Value{i.Addr} }
func (i *Load) String() string {
	return fmt.Sprintf("%s = load %s : %s", i.Name(), i.Addr.Name(), i.typ)
}

// Store writes Val to pointer Addr.
type Store struct {
	baseInstr
	Addr   Value
	Val    Value
	Atomic bool
}

func (i *Store) Operands() []Value { return []Value{i.Addr, i.Val} }
func (i *Store) String() string {
	return fmt.Sprintf("store %s, %s", i.Val.Name(), i.Addr.Name())
}

// BarrierKind classifies a GC write barrier operation.
type BarrierKind uint8

const (
	BarrierStore BarrierKind = iota
	BarrierMove
	BarrierBulk
)

func (k BarrierKind) String() string {
	switch k {
	case BarrierMove:
		return "move"
	case BarrierBulk:
		return "bulk"
	default:
		return "store"
	}
}

// WriteBarrier records a GC pointer store through an explicit barrier hook,
// allowing backends to lower to direct stores (BDWGC) or precise barriers
// (future custom/moving/concurrent GC).
type WriteBarrier struct {
	baseInstr
	Dst  Value
	Val  Value
	Kind BarrierKind
}

func (i *WriteBarrier) Operands() []Value { return []Value{i.Dst, i.Val} }
func (i *WriteBarrier) String() string {
	return fmt.Sprintf("gc.writebarrier.%s %s, %s", i.Kind, i.Dst.Name(), i.Val.Name())
}

// Safepoint marks a GC/preemption poll point with its live Go pointer roots.
type Safepoint struct {
	baseInstr
	LiveRoots []Value
}

func (i *Safepoint) Operands() []Value { return i.LiveRoots }
func (i *Safepoint) String() string {
	names := make([]string, len(i.LiveRoots))
	for idx, r := range i.LiveRoots {
		names[idx] = r.Name()
	}
	return fmt.Sprintf("gc.safepoint [%s]", strings.Join(names, ", "))
}

// GCRoot registers or releases a stack slot as a precise GC root.
type GCRoot struct {
	baseInstr
	Slot Value
	Live bool
}

func (i *GCRoot) Operands() []Value { return []Value{i.Slot} }
func (i *GCRoot) String() string {
	if i.Live {
		return fmt.Sprintf("gc.root.enter %s", i.Slot.Name())
	}
	return fmt.Sprintf("gc.root.leave %s", i.Slot.Name())
}

// KeepAlive prevents premature collection of Val before this program point.
type KeepAlive struct {
	baseInstr
	Val Value
}

func (i *KeepAlive) Operands() []Value { return []Value{i.Val} }
func (i *KeepAlive) String() string    { return fmt.Sprintf("gc.keepalive %s", i.Val.Name()) }

// FieldAddr computes the address of struct field FieldIndex of Base.
type FieldAddr struct {
	baseInstr
	baseValue
	Base       Value
	FieldIndex int
}

func (i *FieldAddr) Operands() []Value { return []Value{i.Base} }
func (i *FieldAddr) String() string {
	return fmt.Sprintf("%s = fieldaddr %s.#%d : %s", i.Name(), i.Base.Name(), i.FieldIndex, i.typ)
}

// IndexAddr computes the address of element Index of slice/array Base.
type IndexAddr struct {
	baseInstr
	baseValue
	Base  Value
	Index Value
}

func (i *IndexAddr) Operands() []Value { return []Value{i.Base, i.Index} }
func (i *IndexAddr) String() string {
	return fmt.Sprintf("%s = indexaddr %s[%s] : %s", i.Name(), i.Base.Name(), i.Index.Name(), i.typ)
}

// Extract extracts element Index from a tuple-typed multi-result Value.
type Extract struct {
	baseInstr
	baseValue
	Tuple Value
	Index int
}

func (i *Extract) Operands() []Value { return []Value{i.Tuple} }
func (i *Extract) String() string {
	return fmt.Sprintf("%s = extract %s #%d : %s", i.Name(), i.Tuple.Name(), i.Index, i.typ)
}

// DebugRef associates a source-level Object with its SSA Value (or address
// when IsAddr is true) for DWARF debug info and GC stack maps.
type DebugRef struct {
	baseInstr
	X      Value
	Object types.Object
	IsAddr bool
}

func (i *DebugRef) Operands() []Value { return []Value{i.X} }
func (i *DebugRef) String() string {
	name := "<anon>"
	if i.Object != nil {
		name = i.Object.Name()
	}
	kind := "val"
	if i.IsAddr {
		kind = "addr"
	}
	return fmt.Sprintf("debugref.%s %s = %s", kind, name, i.X.Name())
}

// MakeClosure constructs a first-class Closure value from a target function
// symbol and optional heap/stack environment pointer.
type MakeClosure struct {
	baseInstr
	baseValue
	FnName string
	Env    Value // nil for stateless closures
}

func (i *MakeClosure) Operands() []Value {
	if i.Env == nil {
		return nil
	}
	return []Value{i.Env}
}
func (i *MakeClosure) String() string {
	env := "nil"
	if i.Env != nil {
		env = i.Env.Name()
	}
	return fmt.Sprintf("%s = makeclosure @%s, %s : %s", i.Name(), i.FnName, env, i.typ)
}

// CallDirect invokes a statically resolved symbol under an explicit CallConv.
type CallDirect struct {
	baseInstr
	baseValue
	Callee   string
	CallConv types.CallConv
	Args     []Value
}

func (i *CallDirect) Operands() []Value { return i.Args }
func (i *CallDirect) String() string {
	return fmt.Sprintf("%s = call.%s @%s(%s)", i.Name(), i.CallConv, i.Callee, formatValues(i.Args))
}

// CallClosure invokes a first-class Closure value ({codePtr, envPtr}).
type CallClosure struct {
	baseInstr
	baseValue
	Closure Value
	Args    []Value
}

func (i *CallClosure) Operands() []Value {
	ops := make([]Value, 0, 1+len(i.Args))
	ops = append(ops, i.Closure)
	return append(ops, i.Args...)
}
func (i *CallClosure) String() string {
	return fmt.Sprintf("%s = call.closure %s(%s)", i.Name(), i.Closure.Name(), formatValues(i.Args))
}

// CallCFunc invokes a native C or Stdcall function pointer directly.
type CallCFunc struct {
	baseInstr
	baseValue
	Callee   Value
	CallConv types.CallConv
	Args     []Value
}

func (i *CallCFunc) Operands() []Value {
	ops := make([]Value, 0, 1+len(i.Args))
	ops = append(ops, i.Callee)
	return append(ops, i.Args...)
}
func (i *CallCFunc) String() string {
	return fmt.Sprintf("%s = call.%s %s(%s)", i.Name(), i.CallConv, i.Callee.Name(), formatValues(i.Args))
}

// CallInterface performs dynamic interface dispatch, preserving the interface
// method name and slot index for Go-side DCE and LLVM VFE/LTO metadata.
type CallInterface struct {
	baseInstr
	baseValue
	Iface       Value
	Method      string
	MethodIndex int
	Args        []Value
}

func (i *CallInterface) Operands() []Value {
	ops := make([]Value, 0, 1+len(i.Args))
	ops = append(ops, i.Iface)
	return append(ops, i.Args...)
}
func (i *CallInterface) String() string {
	return fmt.Sprintf("%s = call.iface %s.%s#%d(%s)", i.Name(), i.Iface.Name(), i.Method, i.MethodIndex, formatValues(i.Args))
}

// Go spawns a new OS-backed goroutine (1:1 threading model).
type Go struct {
	baseInstr
	Callee Value
	Args   []Value
}

func (i *Go) Operands() []Value {
	ops := make([]Value, 0, 1+len(i.Args))
	ops = append(ops, i.Callee)
	return append(ops, i.Args...)
}
func (i *Go) String() string {
	return fmt.Sprintf("go %s(%s)", i.Callee.Name(), formatValues(i.Args))
}

// Defer registers a deferred call.
type Defer struct {
	baseInstr
	Callee Value
	Args   []Value
	InLoop bool
}

func (i *Defer) Operands() []Value {
	ops := make([]Value, 0, 1+len(i.Args))
	ops = append(ops, i.Callee)
	return append(ops, i.Args...)
}
func (i *Defer) String() string {
	return fmt.Sprintf("defer %s(%s)", i.Callee.Name(), formatValues(i.Args))
}

// RunDefers executes pending deferred calls before returning.
type RunDefers struct {
	baseInstr
}

func (i *RunDefers) Operands() []Value { return nil }
func (i *RunDefers) String() string    { return "rundefers" }

// Panic raises a Go panic value.
type Panic struct {
	baseInstr
	Val Value
}

func (i *Panic) Operands() []Value         { return []Value{i.Val} }
func (i *Panic) Successors() []*BasicBlock { return nil }
func (i *Panic) termMarker()               {}
func (i *Panic) String() string            { return fmt.Sprintf("panic %s", i.Val.Name()) }

// Recover captures an active panic inside a deferred frame.
type Recover struct {
	baseInstr
	baseValue
}

func (i *Recover) Operands() []Value { return nil }
func (i *Recover) String() string    { return fmt.Sprintf("%s = recover : %s", i.Name(), i.typ) }

// Jump unconditionally transfers control to Target.
type Jump struct {
	baseInstr
	Target *BasicBlock
}

func (i *Jump) Operands() []Value         { return nil }
func (i *Jump) Successors() []*BasicBlock { return []*BasicBlock{i.Target} }
func (i *Jump) termMarker()               {}
func (i *Jump) String() string            { return fmt.Sprintf("jump %s", i.Target.Label) }

// Branch transfers control based on boolean Cond.
type Branch struct {
	baseInstr
	Cond  Value
	True  *BasicBlock
	False *BasicBlock
}

func (i *Branch) Operands() []Value         { return []Value{i.Cond} }
func (i *Branch) Successors() []*BasicBlock { return []*BasicBlock{i.True, i.False} }
func (i *Branch) termMarker()               {}
func (i *Branch) String() string {
	return fmt.Sprintf("branch %s ? %s : %s", i.Cond.Name(), i.True.Label, i.False.Label)
}

// Return returns zero or more values from the current Function.
type Return struct {
	baseInstr
	Results []Value
}

func (i *Return) Operands() []Value         { return i.Results }
func (i *Return) Successors() []*BasicBlock { return nil }
func (i *Return) termMarker()               {}
func (i *Return) String() string {
	if len(i.Results) == 0 {
		return "return"
	}
	return fmt.Sprintf("return %s", formatValues(i.Results))
}

// Unreachable marks control flow that cannot be reached at runtime.
type Unreachable struct {
	baseInstr
}

func (i *Unreachable) Operands() []Value         { return nil }
func (i *Unreachable) Successors() []*BasicBlock { return nil }
func (i *Unreachable) termMarker()               {}
func (i *Unreachable) String() string            { return "unreachable" }

func formatValues(vals []Value) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = v.Name()
	}
	return strings.Join(parts, ", ")
}

// String renders a readable textual dump of the package IR.
func (p *Package) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s (%s) target=%s\n", p.Name, p.Path, p.Target.Triple)
	for _, g := range p.Globals {
		fmt.Fprintf(&b, "global @%s : %s", g.Name, g.Type)
		if g.Locality != directive.LocalityNone {
			fmt.Fprintf(&b, " locality=%s", g.Locality)
		}
		b.WriteByte('\n')
	}
	for _, fn := range p.Funcs {
		fmt.Fprintf(&b, "func @%s %s", fn.Name, fn.Sig)
		if fn.IsDeclaration() {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(" {\n")
		for _, blk := range fn.Blocks {
			fmt.Fprintf(&b, "%s:\n", blk.Label)
			for _, ins := range blk.Instrs {
				fmt.Fprintf(&b, "  %s\n", ins.String())
			}
		}
		b.WriteString("}\n")
	}
	return b.String()
}
