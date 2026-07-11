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

package cl

import (
	"go/constant"
	"go/types"
	"sort"
	"strings"

	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Static folding is an optimization. Keep sparse large arrays in the package
// initializer instead of materializing every zero element as an LLVM constant.
const maxStaticInitArrayElements = 1 << 16

type staticInitPathElem struct {
	index int
}

type staticInitStore struct {
	store *ssa.Store
	path  []staticInitPathElem
	value *ssa.Const
}

type staticInitCandidate struct {
	stores  []staticInitStore
	invalid bool
}

type staticInitNode struct {
	value    *ssa.Const
	children map[int]*staticInitNode
}

func (p *context) collectStaticGlobalInits(pkg *ssa.Package) {
	initFn := pkg.Func("init")
	if initFn == nil || initFn.Synthetic != "package initializer" {
		return
	}

	globals := make(map[*ssa.Global]none)
	for name, member := range pkg.Members {
		if _, skip := p.skips[name]; skip {
			continue
		}
		if strings.HasSuffix(name, "init$guard") {
			continue
		}
		if global, ok := member.(*ssa.Global); ok {
			if isCgoFuncPtrVar(global.Name()) {
				continue
			}
			globalName, vtype, define := p.varName(global.Pkg.Pkg, global)
			if !define || vtype != goVar {
				continue
			}
			if _, rewritten := p.rewriteValue(globalName); rewritten {
				continue
			}
			globals[global] = none{}
		}
	}
	if len(globals) == 0 {
		return
	}

	candidates := make(map[*ssa.Global]*staticInitCandidate)
	candidateOf := func(global *ssa.Global) *staticInitCandidate {
		candidate := candidates[global]
		if candidate == nil {
			candidate = new(staticInitCandidate)
			candidates[global] = candidate
		}
		return candidate
	}

	for _, block := range initFn.Blocks {
		for _, instr := range block.Instrs {
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}
			global := staticInitRootGlobal(store.Addr)
			if global == nil {
				continue
			}
			if _, ok := globals[global]; !ok {
				continue
			}

			path, ok := staticInitStorePath(store.Addr)
			value, isConst := store.Val.(*ssa.Const)
			candidate := candidateOf(global)
			if !ok || !isConst {
				candidate.invalid = true
				continue
			}
			candidate.stores = append(candidate.stores, staticInitStore{
				store: store,
				path:  path,
				value: value,
			})
		}
	}

	orderedGlobals := make([]*ssa.Global, 0, len(candidates))
	for global := range candidates {
		orderedGlobals = append(orderedGlobals, global)
	}
	sort.Slice(orderedGlobals, func(i, j int) bool {
		return p.globalFullName(orderedGlobals[i]) < p.globalFullName(orderedGlobals[j])
	})

	for _, global := range orderedGlobals {
		candidate := candidates[global]
		if candidate.invalid || len(candidate.stores) == 0 {
			continue
		}
		init, ok := p.buildStaticGlobalInit(global, candidate.stores)
		if !ok {
			continue
		}
		if p.staticGlobalInits == nil {
			p.staticGlobalInits = make(map[*ssa.Global]llssa.Expr)
			p.staticInitStores = make(map[*ssa.Store]none)
		}
		p.staticGlobalInits[global] = init
		for _, store := range candidate.stores {
			p.staticInitStores[store.store] = none{}
		}
	}
}

func staticInitRootGlobal(addr ssa.Value) *ssa.Global {
	switch addr := addr.(type) {
	case *ssa.Global:
		return addr
	case *ssa.FieldAddr:
		return staticInitRootGlobal(addr.X)
	case *ssa.IndexAddr:
		return staticInitRootGlobal(addr.X)
	default:
		return nil
	}
}

func staticInitStorePath(addr ssa.Value) ([]staticInitPathElem, bool) {
	switch addr := addr.(type) {
	case *ssa.Global:
		return nil, true
	case *ssa.FieldAddr:
		path, ok := staticInitStorePath(addr.X)
		if !ok {
			return nil, false
		}
		return append(path, staticInitPathElem{index: addr.Field}), true
	case *ssa.IndexAddr:
		path, ok := staticInitStorePath(addr.X)
		if !ok {
			return nil, false
		}
		index, ok := staticInitConstIndex(addr.Index)
		if !ok {
			return nil, false
		}
		return append(path, staticInitPathElem{index: index}), true
	default:
		return nil, false
	}
}

func staticInitConstIndex(v ssa.Value) (int, bool) {
	c, ok := v.(*ssa.Const)
	if !ok {
		return 0, false
	}
	index, exact := constant.Int64Val(c.Value)
	if !exact || index < 0 || int64(int(index)) != index {
		return 0, false
	}
	return int(index), true
}

func (p *context) buildStaticGlobalInit(global *ssa.Global, stores []staticInitStore) (llssa.Expr, bool) {
	ptr, ok := global.Type().(*types.Pointer)
	if !ok {
		return llssa.Expr{}, false
	}

	root := new(staticInitNode)
	for _, store := range stores {
		if !root.add(store.path, store.value) {
			return llssa.Expr{}, false
		}
	}
	return p.buildStaticInitExpr(ptr.Elem(), root)
}

func (n *staticInitNode) add(path []staticInitPathElem, value *ssa.Const) bool {
	if len(path) == 0 {
		if n.value != nil || len(n.children) != 0 {
			return false
		}
		n.value = value
		return true
	}
	if n.value != nil {
		return false
	}
	head := path[0]
	child := n.children[head.index]
	if child == nil {
		if n.children == nil {
			n.children = make(map[int]*staticInitNode)
		}
		child = new(staticInitNode)
		n.children[head.index] = child
	}
	return child.add(path[1:], value)
}

func (p *context) buildStaticInitExpr(typ types.Type, node *staticInitNode) (llssa.Expr, bool) {
	lltyp := p.type_(typ, llssa.InGo)
	if node == nil {
		return p.prog.Zero(lltyp), true
	}
	if node.value != nil {
		return p.staticConstExpr(node.value, lltyp)
	}

	switch u := typ.Underlying().(type) {
	case *types.Struct:
		values := make([]llssa.Expr, u.NumFields())
		for i := range values {
			child := node.children[i]
			value, ok := p.buildStaticInitExpr(u.Field(i).Type(), child)
			if !ok {
				return llssa.Expr{}, false
			}
			values[i] = value
		}
		if !staticInitChildrenInRange(node, u.NumFields()) {
			return llssa.Expr{}, false
		}
		return p.prog.ConstStruct(lltyp, values), true
	case *types.Array:
		if u.Len() > maxStaticInitArrayElements {
			return llssa.Expr{}, false
		}
		n := int(u.Len())
		values := make([]llssa.Expr, n)
		for i := range values {
			child := node.children[i]
			value, ok := p.buildStaticInitExpr(u.Elem(), child)
			if !ok {
				return llssa.Expr{}, false
			}
			values[i] = value
		}
		if !staticInitChildrenInRange(node, n) {
			return llssa.Expr{}, false
		}
		return p.prog.ConstArray(lltyp, values), true
	default:
		if len(node.children) == 0 {
			return p.prog.Zero(lltyp), true
		}
		return llssa.Expr{}, false
	}
}

func staticInitChildrenInRange(node *staticInitNode, n int) bool {
	for index := range node.children {
		if index < 0 || index >= n {
			return false
		}
	}
	return true
}

func (p *context) staticConstExpr(c *ssa.Const, typ llssa.Type) (llssa.Expr, bool) {
	if c.Value == nil {
		return p.prog.Zero(typ), true
	}
	raw := typ.RawType().Underlying()
	basic, ok := raw.(*types.Basic)
	if !ok {
		return llssa.Expr{}, false
	}
	switch kind := basic.Kind(); {
	case kind == types.Bool:
		return p.prog.BoolVal(constant.BoolVal(c.Value)), true
	case kind >= types.Int && kind <= types.Int64:
		v, exact := constant.Int64Val(constant.ToInt(c.Value))
		if !exact {
			return llssa.Expr{}, false
		}
		return p.prog.IntVal(uint64(v), typ), true
	case kind >= types.Uint && kind <= types.Uintptr:
		v, exact := constant.Uint64Val(constant.ToInt(c.Value))
		if !exact {
			return llssa.Expr{}, false
		}
		return p.prog.IntVal(v, typ), true
	case kind == types.Float32 || kind == types.Float64:
		v, _ := constant.Float64Val(constant.ToFloat(c.Value))
		return p.prog.FloatVal(v, typ), true
	case kind == types.String:
		return p.pkg.ConstString(constant.StringVal(c.Value)), true
	case kind == types.Complex64 || kind == types.Complex128:
		v := constant.ToComplex(c.Value)
		re, _ := constant.Float64Val(constant.Real(v))
		im, _ := constant.Float64Val(constant.Imag(v))
		return p.prog.ComplexVal(complex(re, im), typ), true
	default:
		return llssa.Expr{}, false
	}
}
