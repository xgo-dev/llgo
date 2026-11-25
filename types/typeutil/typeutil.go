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

// Package typeutil defines utilities for github.com/xgo-dev/llgo/types,
// mirroring golang.org/x/tools/go/types/typeutil in pure Go 1.20 with zero
// x/tools or LLVM dependencies.
package typeutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"hash/maphash"
	"sync"

	"github.com/xgo-dev/llgo/types"
)

var globalSeed = maphash.MakeSeed()

// Hasher maps LLGo types to uint32 hash values consistent with types.Identical
// (mirroring golang.org/x/tools/go/types/typeutil.Hasher).
type Hasher struct{}

// MakeHasher returns a stateless Hasher.
func MakeHasher() Hasher { return Hasher{} }

// Hash computes a hash value for t such that types.Identical(x, y) => Hash(x) == Hash(y).
func (Hasher) Hash(t types.Type) uint32 {
	var h maphash.Hash
	h.SetSeed(globalSeed)
	var th types.Hasher
	th.Hash(&h, t)
	sum := h.Sum64()
	return uint32(sum ^ (sum >> 32))
}

type entry struct {
	key   types.Type
	value interface{}
}

// Map is a hash-table-based mapping from LLGo types.Type to arbitrary values
// using types.Identical equivalence (mirroring typeutil.Map).
// A nil *Map is a valid empty map. Read-only operations are concurrency-safe.
type Map struct {
	table map[uint32][]entry
	len   int
}

// SetHasher is a no-op retained for source compatibility with x/tools typeutil.Map.
func (m *Map) SetHasher(Hasher) {}

// Delete removes the entry with key, returning true if it was present.
func (m *Map) Delete(key types.Type) bool {
	if m == nil || m.table == nil {
		return false
	}
	h := Hasher{}.Hash(key)
	bucket := m.table[h]
	for i, e := range bucket {
		if e.key != nil && types.Identical(key, e.key) {
			last := len(bucket) - 1
			bucket[i] = bucket[last]
			bucket[last] = entry{}
			if last == 0 {
				delete(m.table, h)
			} else {
				m.table[h] = bucket[:last]
			}
			m.len--
			return true
		}
	}
	return false
}

// At returns the map entry for key, or nil if absent.
func (m *Map) At(key types.Type) interface{} {
	if m == nil || m.table == nil {
		return nil
	}
	for _, e := range m.table[Hasher{}.Hash(key)] {
		if e.key != nil && types.Identical(key, e.key) {
			return e.value
		}
	}
	return nil
}

// Set sets the map entry for key to value and returns the previous entry, if any.
func (m *Map) Set(key types.Type, value interface{}) (prev interface{}) {
	if m.table == nil {
		m.table = make(map[uint32][]entry)
	}
	h := Hasher{}.Hash(key)
	bucket := m.table[h]
	for i, e := range bucket {
		if e.key != nil && types.Identical(key, e.key) {
			prev = e.value
			bucket[i].value = value
			return prev
		}
	}
	m.table[h] = append(bucket, entry{key: key, value: value})
	m.len++
	return nil
}

// Len returns the number of map entries.
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	return m.len
}

// Iterate calls f on each entry in the map in unspecified order.
func (m *Map) Iterate(f func(key types.Type, value interface{})) {
	if m == nil || m.table == nil {
		return
	}
	for _, bucket := range m.table {
		for _, e := range bucket {
			if e.key != nil {
				f(e.key, e.value)
			}
		}
	}
}

// Keys returns a new slice containing the map keys.
func (m *Map) Keys() []types.Type {
	if m == nil || m.len == 0 {
		return nil
	}
	keys := make([]types.Type, 0, m.len)
	m.Iterate(func(key types.Type, _ interface{}) {
		keys = append(keys, key)
	})
	return keys
}

func (m *Map) toString(values bool) string {
	if m == nil {
		return "{}"
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	sep := ""
	m.Iterate(func(key types.Type, value interface{}) {
		buf.WriteString(sep)
		sep = ", "
		buf.WriteString(key.String())
		if values {
			fmt.Fprintf(&buf, ": %v", value)
		}
	})
	buf.WriteByte('}')
	return buf.String()
}

// String returns a string representation of the map's entries.
func (m *Map) String() string { return m.toString(true) }

// KeysString returns a string representation of the map's key set.
func (m *Map) KeysString() string { return m.toString(false) }

// MethodSetCache caches *types.MethodSet computations in a concurrency-safe map.
// The zero value is ready to use; a nil *MethodSetCache falls back to
// types.NewMethodSet directly.
type MethodSetCache struct {
	mu    sync.RWMutex
	named map[*types.Named]struct{ value, pointer *types.MethodSet }
	other Map
}

// MethodSet returns the method set of T (thread-safe).
func (cache *MethodSetCache) MethodSet(T types.Type) *types.MethodSet {
	if cache == nil {
		return types.NewMethodSet(T)
	}
	if T == nil {
		return types.NewMethodSet(nil)
	}
	u := types.Unalias(T)
	if n, ok := u.(*types.Named); ok {
		cache.mu.RLock()
		ms := cache.named[n].value
		cache.mu.RUnlock()
		if ms != nil {
			return ms
		}
		ms = types.NewMethodSet(T)
		cache.mu.Lock()
		if cache.named == nil {
			cache.named = make(map[*types.Named]struct{ value, pointer *types.MethodSet })
		}
		pair := cache.named[n]
		if pair.value == nil {
			pair.value = ms
			cache.named[n] = pair
		} else {
			ms = pair.value
		}
		cache.mu.Unlock()
		return ms
	}
	if p, ok := u.(*types.Pointer); ok && p.Kind() == types.GoPtr {
		if n, ok := types.Unalias(p.Elem()).(*types.Named); ok {
			cache.mu.RLock()
			ms := cache.named[n].pointer
			cache.mu.RUnlock()
			if ms != nil {
				return ms
			}
			ms = types.NewMethodSet(T)
			cache.mu.Lock()
			if cache.named == nil {
				cache.named = make(map[*types.Named]struct{ value, pointer *types.MethodSet })
			}
			pair := cache.named[n]
			if pair.pointer == nil {
				pair.pointer = ms
				cache.named[n] = pair
			} else {
				ms = pair.pointer
			}
			cache.mu.Unlock()
			return ms
		}
	}

	cache.mu.RLock()
	cached := cache.other.At(T)
	cache.mu.RUnlock()
	if ms, ok := cached.(*types.MethodSet); ok && ms != nil {
		return ms
	}
	ms := types.NewMethodSet(T)
	cache.mu.Lock()
	if existing, ok := cache.other.At(T).(*types.MethodSet); ok && existing != nil {
		ms = existing
	} else {
		cache.other.Set(T, ms)
	}
	cache.mu.Unlock()
	return ms
}

// IntuitiveMethodSet returns the intuitive method set of T: all methods callable
// on an addressable value of type T (mirroring typeutil.IntuitiveMethodSet).
func IntuitiveMethodSet(T types.Type, msets *MethodSetCache) []*types.Selection {
	if T == nil {
		return nil
	}
	isPointerToConcrete := func(t types.Type) bool {
		p, ok := types.Unalias(t).(*types.Pointer)
		return ok && !types.IsInterface(p.Elem())
	}

	var result []*types.Selection
	mset := msets.MethodSet(T)
	if types.IsInterface(T) || isPointerToConcrete(T) {
		for i := 0; i < mset.Len(); i++ {
			result = append(result, mset.At(i))
		}
		return result
	}
	pmset := msets.MethodSet(types.NewPointer(T))
	for i := 0; i < pmset.Len(); i++ {
		meth := pmset.At(i)
		if m := mset.Lookup(meth.Obj().Pkg(), meth.Obj().Name()); m != nil {
			meth = m
		}
		result = append(result, meth)
	}
	return result
}

// Dependencies returns all transitive dependencies of pkgs in topological order
// (mirroring typeutil.Dependencies).
func Dependencies(pkgs ...*types.Package) []*types.Package {
	var result []*types.Package
	seen := make(map[*types.Package]bool)
	var visit func(p *types.Package)
	visit = func(p *types.Package) {
		if p == nil || seen[p] {
			return
		}
		seen[p] = true
		for _, imp := range p.Imports() {
			visit(imp)
		}
		result = append(result, p)
	}
	for _, p := range pkgs {
		visit(p)
	}
	return result
}

// Callee returns the named target of a function call expression (function,
// method, builtin, or variable), or nil for type conversions (mirroring
// typeutil.Callee).
func Callee(info *types.Info, call *ast.CallExpr) types.Object {
	if info == nil || call == nil {
		return nil
	}
	fun := unparen(call.Fun)
	// Unwrap generic instantiation f[T] or f[T, U].
	switch x := fun.(type) {
	case *ast.IndexExpr:
		fun = unparen(x.X)
	case *ast.IndexListExpr:
		fun = unparen(x.X)
	}
	if tv, ok := info.Types[fun]; ok && tv.IsType() {
		return nil
	}
	switch x := fun.(type) {
	case *ast.Ident:
		obj := info.Uses[x]
		if _, isType := obj.(*types.TypeName); isType {
			return nil
		}
		return obj
	case *ast.SelectorExpr:
		if sel, ok := info.Selections[x]; ok {
			return sel.Obj()
		}
		obj := info.Uses[x.Sel]
		if _, isType := obj.(*types.TypeName); isType {
			return nil
		}
		return obj
	}
	return nil
}

// StaticCallee returns the statically resolved *types.Func target of call,
// or nil if call invokes a variable, builtin, interface method, or conversion
// (mirroring typeutil.StaticCallee).
func StaticCallee(info *types.Info, call *ast.CallExpr) *types.Func {
	if info == nil || call == nil {
		return nil
	}
	fun := unparen(call.Fun)
	switch x := fun.(type) {
	case *ast.IndexExpr:
		fun = unparen(x.X)
	case *ast.IndexListExpr:
		fun = unparen(x.X)
	}
	if selExpr, ok := fun.(*ast.SelectorExpr); ok {
		if sel, ok := info.Selections[selExpr]; ok && sel.Kind() == types.MethodVal {
			if types.IsInterface(sel.Recv()) {
				return nil
			}
			if fn, ok := sel.Obj().(*types.Func); ok {
				return fn
			}
			return nil
		}
	}
	obj := Callee(info, call)
	fn, _ := obj.(*types.Func)
	if fn != nil {
		if sig := fn.Signature(); sig != nil && sig.Recv() != nil && types.IsInterface(sig.Recv().Type()) {
			return nil
		}
	}
	return fn
}

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}
