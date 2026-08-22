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
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Static folding is an optimization. Keep sparse large non-byte arrays in the
// package initializer instead of materializing every zero element as an LLVM
// constant. Byte arrays use a compact LLVM string constant and are not capped.
const maxStaticInitArrayElements = 1 << 16

type staticInitPathElem struct {
	index int
}

type staticInitStore struct {
	store    *ssa.Store
	path     []staticInitPathElem
	value    *ssa.Const
	function *ssa.Function
	slice    *staticSliceInit
	pointer  *staticPointerInit
}

type staticInitCandidate struct {
	stores  []staticInitStore
	slice   *staticSliceInit
	instrs  []ssa.Instruction
	invalid bool
}

type staticSliceInit struct {
	store  *ssa.Store
	slice  *ssa.Slice
	alloc  *ssa.Alloc
	array  *types.Array
	stores []staticInitStore
	instrs []ssa.Instruction
}

type staticPointerInit struct {
	alloc  *ssa.Alloc
	stores []staticInitStore
	instrs []ssa.Instruction
}

type staticInitNode struct {
	value    *ssa.Const
	function *ssa.Function
	slice    *staticSliceInit
	pointer  *staticPointerInit
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
			if info, ok := p.resolveLocality(global.Pkg.Pkg, llssa.FullName(global.Pkg.Pkg, global.Name())); ok && info.Locality != llssa.LocalityNone {
				// Local initializers must remain executable so they can populate the
				// current context rather than a process-wide LLVM initializer.
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

			candidate := candidateOf(global)
			path, ok := staticInitStorePath(store.Addr)
			if !ok {
				candidate.invalid = true
				continue
			}
			if len(path) == 0 {
				if slice, ok := staticSliceInitOf(store); ok {
					if candidate.slice != nil || len(candidate.stores) != 0 {
						candidate.invalid = true
					} else {
						candidate.slice = slice
					}
					continue
				}
			}
			if value, isConst := store.Val.(*ssa.Const); isConst {
				candidate.stores = append(candidate.stores, staticInitStore{
					store: store,
					path:  path,
					value: value,
				})
			} else if function, ok := staticInitFunctionOf(store.Val, store); ok {
				candidate.stores = append(candidate.stores, staticInitStore{
					store:    store,
					path:     path,
					function: function,
				})
				if closure, ok := store.Val.(*ssa.MakeClosure); ok {
					candidate.instrs = append(candidate.instrs, closure)
				}
			} else if slice, ok := staticSliceInitOfVisited(store, make(map[*ssa.Alloc]bool)); ok {
				candidate.stores = append(candidate.stores, staticInitStore{
					store: store,
					path:  path,
					slice: slice,
				})
				candidate.instrs = append(candidate.instrs, slice.instrs...)
			} else if pointer, ok := staticPointerInitOfVisited(store, make(map[*ssa.Alloc]bool)); ok {
				candidate.stores = append(candidate.stores, staticInitStore{
					store:   store,
					path:    path,
					pointer: pointer,
				})
				candidate.instrs = append(candidate.instrs, pointer.instrs...)
			} else if unop, ok := store.Val.(*ssa.UnOp); ok && unop.Op == token.MUL {
				if alloc, ok := unop.X.(*ssa.Alloc); ok && !alloc.Heap {
					if !collectAllocStores(alloc, unop, store, path, &candidate.stores, &candidate.instrs, make(map[*ssa.Alloc]bool)) {
						candidate.invalid = true
						continue
					}
					candidate.instrs = append(candidate.instrs, store)
				} else {
					candidate.invalid = true
					continue
				}
			} else {
				candidate.invalid = true
				continue
			}
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
		if candidate.invalid || (len(candidate.stores) == 0 && candidate.slice == nil) {
			continue
		}
		var init llssa.Expr
		var ok bool
		if candidate.slice != nil {
			init, ok = p.buildStaticSliceInit(global, candidate.slice)
		} else {
			init, ok = p.buildStaticGlobalInit(global, candidate.stores)
		}
		if !ok {
			continue
		}
		if p.staticGlobalInits == nil {
			p.staticGlobalInits = make(map[*ssa.Global]llssa.Expr)
			p.staticInitStores = make(map[*ssa.Store]none)
			p.staticInitInstrs = make(map[ssa.Instruction]none)
		}
		p.staticGlobalInits[global] = init
		if candidate.slice != nil {
			for _, instr := range candidate.slice.instrs {
				p.staticInitInstrs[instr] = none{}
			}
		}
		for _, instr := range candidate.instrs {
			p.staticInitInstrs[instr] = none{}
		}
		for _, store := range candidate.stores {
			p.staticInitStores[store.store] = none{}
		}
	}
}

// collectAllocStores recursively traces store instructions made to an alloc,
// recording constant stores into out and tracking intermediate instructions for suppression.
// terminal must be the exact load or full-slice value consumed only by terminalStore;
// any other escaping use rejects the fold. The visited map guards against cyclic pointer graphs.
// A false result may leave partial entries in out and instrs; callers must discard the entire
// candidate on failure.
func collectAllocStores(alloc *ssa.Alloc, terminal ssa.Value, terminalStore *ssa.Store, basePath []staticInitPathElem, out *[]staticInitStore, instrs *[]ssa.Instruction, visited map[*ssa.Alloc]bool) bool {
	if visited[alloc] {
		return false
	}
	switch terminal := terminal.(type) {
	case *ssa.UnOp:
		if terminal.Op != token.MUL || terminal.X != alloc {
			return false
		}
	case *ssa.Slice:
		if terminal.X != alloc || terminal.Low != nil || terminal.High != nil || terminal.Max != nil {
			return false
		}
	default:
		return false
	}
	if terminalStore == nil || terminalStore.Val != terminal {
		return false
	}
	terminalRefs, ok := nonDebugReferrers(terminal)
	if !ok || len(terminalRefs) != 1 || terminalRefs[0] != terminalStore {
		return false
	}
	visited[alloc] = true
	*instrs = append(*instrs, alloc)

	refs, ok := nonDebugReferrers(alloc)
	if !ok {
		return false
	}
	seenTerminal := false
	for _, ref := range refs {
		switch ref := ref.(type) {
		case *ssa.Slice:
			if ref != terminal || seenTerminal {
				return false
			}
			seenTerminal = true
			*instrs = append(*instrs, ref)
		case *ssa.UnOp:
			if ref != terminal || seenTerminal {
				return false
			}
			seenTerminal = true
			*instrs = append(*instrs, ref)
		case *ssa.FieldAddr:
			if !collectAddrStores(ref, alloc, basePath, out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		case *ssa.IndexAddr:
			if !collectAddrStores(ref, alloc, basePath, out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		case *ssa.Store:
			if ref.Addr != alloc {
				return false
			}
			if !handleStoreVal(ref, appendStaticInitPath(basePath, nil), out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		default:
			return false
		}
	}
	return seenTerminal
}

// collectAddrStores recursively visits field/index address projections derived from rootAlloc,
// recording constant stores and intermediate instructions.
func collectAddrStores(addr ssa.Value, rootAlloc *ssa.Alloc, basePath []staticInitPathElem, out *[]staticInitStore, instrs *[]ssa.Instruction, visited map[*ssa.Alloc]bool) bool {
	switch addr := addr.(type) {
	case *ssa.FieldAddr:
	case *ssa.IndexAddr:
		if _, ok := staticInitConstIndex(addr.Index); !ok {
			return false
		}
	default:
		return false
	}
	refs, ok := nonDebugReferrers(addr)
	if !ok {
		return false
	}
	seenStore := false
	for _, ref := range refs {
		switch ref := ref.(type) {
		case *ssa.FieldAddr:
			if !collectAddrStores(ref, rootAlloc, basePath, out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		case *ssa.IndexAddr:
			if !collectAddrStores(ref, rootAlloc, basePath, out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		case *ssa.Store:
			if ref.Addr != addr || seenStore {
				return false
			}
			seenStore = true
			subPath, ok := staticInitStorePathToAlloc(addr, rootAlloc)
			if !ok {
				return false
			}
			if !handleStoreVal(ref, appendStaticInitPath(basePath, subPath), out, instrs, visited) {
				return false
			}
			*instrs = append(*instrs, ref)
		default:
			return false
		}
	}
	return true
}

// appendStaticInitPath concatenates base and sub paths into a newly allocated slice
// to avoid slice-aliasing hazards when branching across multiple struct fields or array elements.
func appendStaticInitPath(base, sub []staticInitPathElem) []staticInitPathElem {
	res := make([]staticInitPathElem, len(base)+len(sub))
	copy(res, base)
	copy(res[len(base):], sub)
	return res
}

// handleStoreVal inspects a store value, appending constant stores directly to out (while the caller
// tracks the store instruction in instrs for compilation suppression) or recursing into inner nested
// local allocs reached through pointer indirection (*ssa.UnOp).
func handleStoreVal(store *ssa.Store, fullPath []staticInitPathElem, out *[]staticInitStore, instrs *[]ssa.Instruction, visited map[*ssa.Alloc]bool) bool {
	if val, ok := store.Val.(*ssa.Const); ok {
		*out = append(*out, staticInitStore{
			store: store,
			path:  fullPath,
			value: val,
		})
		return true
	}
	if function, ok := staticInitFunctionOf(store.Val, store); ok {
		*out = append(*out, staticInitStore{
			store:    store,
			path:     fullPath,
			function: function,
		})
		if closure, ok := store.Val.(*ssa.MakeClosure); ok {
			*instrs = append(*instrs, closure)
		}
		return true
	}
	if slice, ok := staticSliceInitOfVisited(store, visited); ok {
		*out = append(*out, staticInitStore{
			store: store,
			path:  fullPath,
			slice: slice,
		})
		*instrs = append(*instrs, slice.instrs...)
		return true
	}
	if pointer, ok := staticPointerInitOfVisited(store, visited); ok {
		*out = append(*out, staticInitStore{
			store:   store,
			path:    fullPath,
			pointer: pointer,
		})
		*instrs = append(*instrs, pointer.instrs...)
		return true
	}
	if unop, ok := store.Val.(*ssa.UnOp); ok && unop.Op == token.MUL {
		if innerAlloc, ok := unop.X.(*ssa.Alloc); ok && !innerAlloc.Heap {
			return collectAllocStores(innerAlloc, unop, store, fullPath, out, instrs, visited)
		}
	}
	return false
}

func staticPointerInitOfVisited(store *ssa.Store, visited map[*ssa.Alloc]bool) (*staticPointerInit, bool) {
	if store == nil || store.Block() == nil {
		return nil, false
	}
	alloc, ok := store.Val.(*ssa.Alloc)
	if !ok || alloc.Parent() != store.Parent() || visited[alloc] {
		return nil, false
	}
	ptr, ok := alloc.Type().Underlying().(*types.Pointer)
	if !ok || staticInitZeroSized(ptr.Elem()) {
		return nil, false
	}
	refs, ok := nonDebugReferrers(alloc)
	if !ok {
		return nil, false
	}

	visited[alloc] = true
	ret := &staticPointerInit{alloc: alloc, instrs: []ssa.Instruction{store, alloc}}
	seenTerminal := false
	for _, ref := range refs {
		switch ref := ref.(type) {
		case *ssa.FieldAddr:
			if !collectAddrStores(ref, alloc, nil, &ret.stores, &ret.instrs, visited) {
				return nil, false
			}
			ret.instrs = append(ret.instrs, ref)
		case *ssa.IndexAddr:
			if !collectAddrStores(ref, alloc, nil, &ret.stores, &ret.instrs, visited) {
				return nil, false
			}
			ret.instrs = append(ret.instrs, ref)
		case *ssa.Store:
			if ref == store && ref.Val == alloc {
				if seenTerminal {
					return nil, false
				}
				seenTerminal = true
				continue
			}
			if ref.Addr != alloc || !handleStoreVal(ref, nil, &ret.stores, &ret.instrs, visited) {
				return nil, false
			}
			ret.instrs = append(ret.instrs, ref)
		default:
			return nil, false
		}
	}
	if !seenTerminal {
		return nil, false
	}
	return ret, true
}

func staticInitFunctionOf(value ssa.Value, terminalStore *ssa.Store) (*ssa.Function, bool) {
	switch value := value.(type) {
	case *ssa.Function:
		if value.Parent() == nil && len(value.FreeVars) == 0 {
			return value, true
		}
	case *ssa.MakeClosure:
		if len(value.Bindings) != 0 {
			return nil, false
		}
		function, ok := value.Fn.(*ssa.Function)
		if !ok || function.Parent() != nil || len(function.FreeVars) != 0 {
			return nil, false
		}
		refs, ok := nonDebugReferrers(value)
		if !ok || len(refs) != 1 || refs[0] != terminalStore {
			return nil, false
		}
		return function, true
	}
	return nil, false
}

// staticInitStorePathToAlloc resolves the nested path elements from an address expression
// back to the root target alloc.
func staticInitStorePathToAlloc(addr ssa.Value, target *ssa.Alloc) ([]staticInitPathElem, bool) {
	switch addr := addr.(type) {
	case *ssa.Alloc:
		if addr == target {
			return nil, true
		}
		return nil, false
	case *ssa.FieldAddr:
		path, ok := staticInitStorePathToAlloc(addr.X, target)
		if !ok {
			return nil, false
		}
		return append(path, staticInitPathElem{index: addr.Field}), true
	case *ssa.IndexAddr:
		path, ok := staticInitStorePathToAlloc(addr.X, target)
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

func staticSliceInitOf(store *ssa.Store) (*staticSliceInit, bool) {
	return staticSliceInitOfVisited(store, make(map[*ssa.Alloc]bool))
}

func staticSliceInitOfVisited(store *ssa.Store, visited map[*ssa.Alloc]bool) (*staticSliceInit, bool) {
	if store == nil || store.Block() == nil {
		return nil, false
	}
	slice, ok := store.Val.(*ssa.Slice)
	if !ok || slice.Low != nil || slice.High != nil || slice.Max != nil {
		return nil, false
	}
	alloc, ok := slice.X.(*ssa.Alloc)
	if !ok || alloc.Parent() != store.Parent() {
		return nil, false
	}
	ptr, ok := alloc.Type().Underlying().(*types.Pointer)
	if !ok {
		return nil, false
	}
	array, ok := ptr.Elem().Underlying().(*types.Array)
	if !ok || array.Len() == 0 || !staticInitArraySizeAllowed(array) || staticInitZeroSized(array.Elem()) {
		return nil, false
	}

	ret := &staticSliceInit{
		store: store, slice: slice, alloc: alloc, array: array,
		instrs: []ssa.Instruction{store},
	}

	if !collectAllocStores(alloc, slice, store, nil, &ret.stores, &ret.instrs, visited) {
		return nil, false
	}
	return ret, true
}

func staticInitArraySizeAllowed(array *types.Array) bool {
	if array.Len() < 0 || int64(int(array.Len())) != array.Len() {
		return false
	}
	if basic, ok := array.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.Uint8 {
		return true
	}
	return array.Len() <= maxStaticInitArrayElements
}

func staticInitZeroSized(typ types.Type) bool {
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Array:
		return typ.Len() == 0 || staticInitZeroSized(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if !staticInitZeroSized(typ.Field(i).Type()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (p *context) buildStaticSliceInit(global *ssa.Global, init *staticSliceInit) (llssa.Expr, bool) {
	if global == nil || global.Object() == nil {
		return llssa.Expr{}, false
	}
	sliceType, ok := global.Type().(*types.Pointer)
	if !ok {
		return llssa.Expr{}, false
	}
	return p.buildStaticSliceValue(p.globalFullName(global)+"$data", sliceType.Elem(), init)
}

func (p *context) buildStaticSliceValue(name string, typ types.Type, init *staticSliceInit) (llssa.Expr, bool) {
	sliceType, ok := typ.Underlying().(*types.Slice)
	if !ok || !types.Identical(sliceType.Elem(), init.array.Elem()) {
		return llssa.Expr{}, false
	}
	n := int(init.array.Len())
	elemType := init.array.Elem()
	elemStores := make(map[int][]staticInitStore, n)
	for _, s := range init.stores {
		if len(s.path) == 0 {
			return llssa.Expr{}, false
		}
		idx := s.path[0].index
		if idx < 0 || idx >= n {
			return llssa.Expr{}, false
		}
		elemStores[idx] = append(elemStores[idx], staticInitStore{
			store:    s.store,
			path:     s.path[1:],
			value:    s.value,
			function: s.function,
			slice:    s.slice,
			pointer:  s.pointer,
		})
	}
	values := make([]llssa.Expr, n)
	for i := range values {
		stores := elemStores[i]
		root := new(staticInitNode)
		for _, store := range stores {
			if !root.addStore(store) {
				return llssa.Expr{}, false
			}
		}
		var ok bool
		values[i], ok = p.buildStaticInitExprNamed(elemType, root, name+"$"+strconv.Itoa(i))
		if !ok {
			return llssa.Expr{}, false
		}
	}
	return p.pkg.ConstSlice(name, p.type_(typ, llssa.InGo), values), true
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
	if !ok || c.Value == nil || c.Value.Kind() != constant.Int {
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
		if !root.addStore(store) {
			return llssa.Expr{}, false
		}
	}
	return p.buildStaticInitExprNamed(ptr.Elem(), root, p.globalFullName(global)+"$data")
}

func (n *staticInitNode) addStore(store staticInitStore) bool {
	leaf := &staticInitNode{
		value: store.value, function: store.function,
		slice: store.slice, pointer: store.pointer,
	}
	if !leaf.hasLeaf() {
		return false
	}
	return n.addLeaf(store.path, leaf)
}

func (n *staticInitNode) add(path []staticInitPathElem, value *ssa.Const) bool {
	return n.addLeaf(path, &staticInitNode{value: value})
}

func (n *staticInitNode) addLeaf(path []staticInitPathElem, leaf *staticInitNode) bool {
	if len(path) == 0 {
		if n.hasLeaf() || len(n.children) != 0 {
			return false
		}
		n.value, n.function = leaf.value, leaf.function
		n.slice, n.pointer = leaf.slice, leaf.pointer
		return true
	}
	if n.hasLeaf() {
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
	return child.addLeaf(path[1:], leaf)
}

func (n *staticInitNode) hasLeaf() bool {
	return n.value != nil || n.function != nil || n.slice != nil || n.pointer != nil
}

func (p *context) buildStaticInitExpr(typ types.Type, node *staticInitNode) (llssa.Expr, bool) {
	return p.buildStaticInitExprNamed(typ, node, "__llgo.staticinit$data")
}

func (p *context) buildStaticInitExprNamed(typ types.Type, node *staticInitNode, name string) (llssa.Expr, bool) {
	lltyp := p.type_(typ, llssa.InGo)
	if node == nil {
		return p.prog.Zero(lltyp), true
	}
	if node.value != nil {
		return p.staticConstExpr(node.value, lltyp)
	}
	if node.function != nil {
		if _, ok := typ.Underlying().(*types.Signature); !ok {
			return llssa.Expr{}, false
		}
		// compileFunction is required here rather than funcOf: source declarations
		// carrying //llgo:env must be materialized with their environment ABI before
		// NeedsEnv is checked. Such functions then fail closed to runtime init.
		function, _, ftype := p.compileFunction(node.function)
		if function == nil || ftype != goFunc || function.NeedsEnv() {
			return llssa.Expr{}, false
		}
		return p.prog.ConstStruct(lltyp, []llssa.Expr{
			function.Expr,
			p.prog.Nil(p.prog.VoidPtr()),
		}), true
	}
	if node.slice != nil {
		return p.buildStaticSliceValue(name, typ, node.slice)
	}
	if node.pointer != nil {
		return p.buildStaticPointerValue(name, typ, node.pointer)
	}

	switch u := typ.Underlying().(type) {
	case *types.Struct:
		values := make([]llssa.Expr, u.NumFields())
		for i := range values {
			child := node.children[i]
			if u.Field(i).Name() == "_" {
				child = nil
			}
			value, ok := p.buildStaticInitExprNamed(u.Field(i).Type(), child, name+"$"+strconv.Itoa(i))
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
		if !staticInitArraySizeAllowed(u) {
			return llssa.Expr{}, false
		}
		if value, ok := staticInitByteArray(node, u); ok {
			return p.prog.ConstByteArray(lltyp, value), true
		}
		n := int(u.Len())
		values := make([]llssa.Expr, n)
		for i := range values {
			child := node.children[i]
			value, ok := p.buildStaticInitExprNamed(u.Elem(), child, name+"$"+strconv.Itoa(i))
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

func (p *context) buildStaticPointerValue(name string, typ types.Type, init *staticPointerInit) (llssa.Expr, bool) {
	ptr, ok := typ.Underlying().(*types.Pointer)
	if !ok {
		return llssa.Expr{}, false
	}
	allocPtr, ok := init.alloc.Type().Underlying().(*types.Pointer)
	if !ok || !types.Identical(ptr.Elem(), allocPtr.Elem()) {
		return llssa.Expr{}, false
	}
	root := new(staticInitNode)
	for _, store := range init.stores {
		if !root.addStore(store) {
			return llssa.Expr{}, false
		}
	}
	value, ok := p.buildStaticInitExprNamed(ptr.Elem(), root, name+"$value")
	if !ok {
		return llssa.Expr{}, false
	}
	data := p.pkg.NewVarEx(name, p.prog.Pointer(p.type_(ptr.Elem(), llssa.InGo)))
	data.Init(value)
	return data.Expr, true
}

func staticInitByteArray(node *staticInitNode, array *types.Array) ([]byte, bool) {
	basic, ok := array.Elem().Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.Uint8 || !staticInitChildrenInRange(node, int(array.Len())) {
		return nil, false
	}
	value := make([]byte, int(array.Len()))
	for index, child := range node.children {
		if child == nil || child.value == nil || child.value.Value == nil || child.value.Value.Kind() != constant.Int ||
			child.function != nil || child.slice != nil || child.pointer != nil || len(child.children) != 0 {
			return nil, false
		}
		v, exact := constant.Uint64Val(constant.ToInt(child.value.Value))
		if !exact || v > 0xff {
			return nil, false
		}
		value[index] = byte(v)
	}
	return value, true
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
