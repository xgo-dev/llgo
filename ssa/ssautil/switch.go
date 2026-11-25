// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssautil

import (
	"bytes"
	"fmt"
	"go/token"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// A ConstCase represents a single constant comparison.
// It is part of a Switch.
type ConstCase struct {
	Block *ssa.BasicBlock // block performing the comparison
	Body  *ssa.BasicBlock // body of the case
	Value *ssa.Const      // case comparand
}

// A TypeCase represents a single type assertion.
// It is part of a Switch.
type TypeCase struct {
	Block   *ssa.BasicBlock // block performing the type assert
	Body    *ssa.BasicBlock // body of the case
	Type    types.Type      // case type
	Binding ssa.Value       // value bound by this case
}

// A Switch is a logical high-level control flow operation
// (a multiway branch) discovered by analysis of a CFG containing
// only if/else chains. It is not part of the ssa.Instruction set.
type Switch struct {
	Start      *ssa.BasicBlock // block containing start of if/else chain
	X          ssa.Value       // the switch operand
	ConstCases []ConstCase     // ordered list of constant comparisons
	TypeCases  []TypeCase      // ordered list of type assertions
	Default    *ssa.BasicBlock // successor if all comparisons fail
}

func (sw *Switch) String() string {
	var buf bytes.Buffer
	if sw.ConstCases != nil {
		fmt.Fprintf(&buf, "switch %s {\n", sw.X.Name())
		for _, c := range sw.ConstCases {
			fmt.Fprintf(&buf, "case %s: %s\n", c.Value, c.Body.Instrs[0])
		}
	} else {
		fmt.Fprintf(&buf, "switch %s.(type) {\n", sw.X.Name())
		for _, c := range sw.TypeCases {
			fmt.Fprintf(&buf, "case %s %s: %s\n",
				c.Binding.Name(), c.Type, c.Body.Instrs[0])
		}
	}
	if sw.Default != nil {
		fmt.Fprintf(&buf, "default: %s\n", sw.Default.Instrs[0])
	}
	fmt.Fprintf(&buf, "}")
	return buf.String()
}

// Switches examines the control-flow graph of fn and returns the
// set of inferred value and type switches.
func Switches(fn *ssa.Function) []Switch {
	var switches []Switch
	seen := make(map[*ssa.BasicBlock]bool)
	for _, b := range fn.DomPreorder() {
		if x, k := isComparisonBlock(b); x != nil {
			sw := Switch{Start: b, X: x}
			valueSwitch(&sw, k, seen)
			if len(sw.ConstCases) > 1 {
				switches = append(switches, sw)
			}
		}

		if y, x, T := isTypeAssertBlock(b); y != nil {
			sw := Switch{Start: b, X: x}
			typeSwitch(&sw, y, T, seen)
			if len(sw.TypeCases) > 1 {
				switches = append(switches, sw)
			}
		}
	}
	return switches
}

func valueSwitch(sw *Switch, k *ssa.Const, seen map[*ssa.BasicBlock]bool) {
	b := sw.Start
	x := sw.X
	for x == sw.X {
		if seen[b] {
			break
		}
		seen[b] = true

		sw.ConstCases = append(sw.ConstCases, ConstCase{
			Block: b,
			Body:  b.Succs[0],
			Value: k,
		})
		b = b.Succs[1]
		if len(b.Instrs) > 2 {
			break
		}
		if len(b.Preds) != 1 {
			break
		}
		x, k = isComparisonBlock(b)
	}
	sw.Default = b
}

func typeSwitch(sw *Switch, y ssa.Value, T types.Type, seen map[*ssa.BasicBlock]bool) {
	b := sw.Start
	x := sw.X
	for x == sw.X {
		if seen[b] {
			break
		}
		seen[b] = true

		sw.TypeCases = append(sw.TypeCases, TypeCase{
			Block:   b,
			Body:    b.Succs[0],
			Type:    T,
			Binding: y,
		})
		b = b.Succs[1]
		if len(b.Instrs) > 4 {
			break
		}
		if len(b.Preds) != 1 {
			break
		}
		y, x, T = isTypeAssertBlock(b)
	}
	sw.Default = b
}

func isComparisonBlock(b *ssa.BasicBlock) (v ssa.Value, k *ssa.Const) {
	if n := len(b.Instrs); n >= 2 {
		if i, ok := b.Instrs[n-1].(*ssa.If); ok {
			if binop, ok := i.Cond.(*ssa.BinOp); ok && binop.Block() == b && binop.Op == token.EQL {
				if k, ok := binop.Y.(*ssa.Const); ok {
					return binop.X, k
				}
				if k, ok := binop.X.(*ssa.Const); ok {
					return binop.Y, k
				}
			}
		}
	}
	return
}

func isTypeAssertBlock(b *ssa.BasicBlock) (y, x ssa.Value, T types.Type) {
	if n := len(b.Instrs); n >= 4 {
		if i, ok := b.Instrs[n-1].(*ssa.If); ok {
			if ext1, ok := i.Cond.(*ssa.Extract); ok && ext1.Block() == b && ext1.Index == 1 {
				if ta, ok := ext1.Tuple.(*ssa.TypeAssert); ok && ta.Block() == b {
					if ext0, ok := b.Instrs[n-3].(*ssa.Extract); ok {
						return ext0, ta.X, ta.AssertedType
					}
				}
			}
		}
	}
	return
}
