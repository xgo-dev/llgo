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

package types

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"sort"
)

// IsInterface reports whether t is an interface type (mirroring go/types.IsInterface).
func IsInterface(t Type) bool {
	if t == nil {
		return false
	}
	_, ok := t.Underlying().(*Interface)
	return ok
}

// Default returns the default "typed" type for an "untyped" type; it returns
// the incoming type for all other types (mirroring go/types.Default).
func Default(t Type) Type {
	if b, ok := Unalias(t).(*Basic); ok {
		switch b.kind {
		case UntypedBool:
			return Typ[Bool]
		case UntypedInt:
			return Typ[Int]
		case UntypedRune:
			return Typ[Rune]
		case UntypedFloat:
			return Typ[Float64]
		case UntypedComplex:
			return Typ[Complex128]
		case UntypedString:
			return Typ[String]
		}
	}
	return t
}

// Comparable reports whether values of type T are comparable
// (mirroring go/types.Comparable).
func Comparable(T Type) bool {
	if T == nil {
		return false
	}
	switch u := Unalias(T).(type) {
	case *TypeParam:
		if u.constraint == nil {
			return false
		}
		if iface, ok := u.constraint.Underlying().(*Interface); ok {
			return iface.IsComparable()
		}
		return Comparable(u.constraint)
	}
	switch u := T.Underlying().(type) {
	case *Basic:
		return u.kind != Invalid && u.kind != UntypedNil
	case *Pointer, *Chan, *Interface, *SIMDVec:
		return true
	case *Struct:
		for _, f := range u.fields {
			if f != nil && !Comparable(f.typ) {
				return false
			}
		}
		return true
	case *Array:
		return Comparable(u.elem)
	case *Union:
		for _, term := range u.terms {
			if term == nil || !Comparable(term.typ) {
				return false
			}
		}
		return len(u.terms) > 0
	default:
		return false
	}
}

// WriteExpr writes the representation of x to buf (mirroring go/types.WriteExpr).
func WriteExpr(buf *bytes.Buffer, x ast.Expr) {
	if x == nil {
		return
	}
	_ = printer.Fprint(buf, token.NewFileSet(), x)
}

// ExprString returns the (possibly shortened) string representation for x
// (mirroring go/types.ExprString).
func ExprString(x ast.Expr) string {
	var buf bytes.Buffer
	WriteExpr(&buf, x)
	return buf.String()
}

// SelectionString returns the string form of s (mirroring go/types.SelectionString).
func SelectionString(s *Selection, qf Qualifier) string {
	if s == nil {
		return "<nil>"
	}
	var k string
	switch s.kind {
	case FieldVal:
		k = "field "
	case MethodVal:
		k = "method "
	case MethodExpr:
		k = "method expr "
	default:
		k = fmt.Sprintf("selection(%d) ", s.kind)
	}
	var buf bytes.Buffer
	buf.WriteString(k)
	buf.WriteByte('(')
	WriteType(&buf, s.recv, qf)
	buf.WriteString(") ")
	if s.obj != nil {
		buf.WriteString(s.obj.Name())
	}
	switch s.kind {
	case FieldVal:
		buf.WriteByte(' ')
		WriteType(&buf, s.typ, qf)
	case MethodVal, MethodExpr:
		if sig, ok := s.typ.(*Signature); ok {
			WriteSignature(&buf, sig, qf)
		} else {
			buf.WriteByte(' ')
			WriteType(&buf, s.typ, qf)
		}
	}
	return buf.String()
}

// LookupFieldOrMethod looks up a field or method with given package and name
// in T (mirroring go/types.LookupFieldOrMethod).
func LookupFieldOrMethod(T Type, addressable bool, pkg *Package, name string) (obj Object, index []int, indirect bool) {
	if T == nil {
		return nil, nil, false
	}
	obj, index, indirect = lookupFieldOrMethod(T, addressable, pkg, name)
	// Handle pointer-receiver method on non-addressable value receiver:
	// signal indirect=true with obj=nil matching go/types contract.
	if obj == nil && !addressable {
		if _, isPtr := Unalias(T).(*Pointer); !isPtr {
			if m, _, _ := lookupFieldOrMethod(NewPointer(T), true, pkg, name); m != nil {
				if _, isFunc := m.(*Func); isFunc {
					return nil, nil, true
				}
			}
		}
	}
	return obj, index, indirect
}

type embeddedType struct {
	typ       Type
	index     []int
	indirect  bool
	multiples bool
}

func derefType(typ Type) (Type, bool) {
	if p, ok := Unalias(typ).(*Pointer); ok {
		return p.elem, true
	}
	return typ, false
}

func lookupFieldOrMethod(T Type, addressable bool, pkg *Package, name string) (obj Object, index []int, indirect bool) {
	if name == "_" {
		return nil, nil, false
	}
	typ, isPtr := derefType(T)
	// Named pointer types cannot have methods/fields accessed through deref.
	if isPtr {
		if _, ok := Unalias(typ).(*Pointer); ok {
			return nil, nil, false
		}
	}

	current := []embeddedType{{typ: typ, indirect: isPtr}}
	visited := make(map[Type]bool)

	for len(current) > 0 {
		var next []embeddedType
		var found Object
		var foundIndex []int
		var foundIndirect bool
		foundCount := 0

		for _, e := range current {
			base := Unalias(e.typ)
			if visited[base] {
				continue
			}
			visited[base] = true

			// Check declared methods on *Named first.
			if n, ok := base.(*Named); ok {
				for i := 0; i < n.NumMethods(); i++ {
					m := n.Method(i)
					if !matchObj(m, pkg, name) {
						continue
					}
					if !e.indirect && !addressable && hasPointerRecv(m) {
						continue
					}
					if e.multiples {
						return nil, concatIndex(e.index, i), false
					}
					foundCount++
					found = m
					foundIndex = concatIndex(e.index, i)
					foundIndirect = e.indirect
				}
			}

			switch u := base.Underlying().(type) {
			case *Interface:
				for i := 0; i < u.NumMethods(); i++ {
					m := u.Method(i)
					if !matchObj(m, pkg, name) {
						continue
					}
					if e.multiples {
						return nil, concatIndex(e.index, i), false
					}
					foundCount++
					found = m
					foundIndex = concatIndex(e.index, i)
					foundIndirect = true
				}
			case *Struct:
				for i, f := range u.fields {
					if f == nil {
						continue
					}
					if matchObj(f, pkg, name) {
						if e.multiples {
							return nil, concatIndex(e.index, i), false
						}
						foundCount++
						found = f
						foundIndex = concatIndex(e.index, i)
						foundIndirect = e.indirect
						continue
					}
					if f.embedded {
						ft, ptr := derefType(f.typ)
						next = append(next, embeddedType{
							typ:      ft,
							index:    concatIndex(e.index, i),
							indirect: e.indirect || ptr,
						})
					}
				}
			}
		}

		if foundCount == 1 {
			return found, foundIndex, foundIndirect
		}
		if foundCount > 1 {
			return nil, foundIndex, false
		}
		current = consolidateEmbedded(next)
	}
	return nil, nil, false
}

func consolidateEmbedded(list []embeddedType) []embeddedType {
	if len(list) <= 1 {
		return list
	}
	out := make([]embeddedType, 0, len(list))
	seen := make(map[Type]int, len(list))
	for _, e := range list {
		base := Unalias(e.typ)
		if idx, dup := seen[base]; dup {
			out[idx].multiples = true
		} else {
			seen[base] = len(out)
			out = append(out, e)
		}
	}
	return out
}

func matchObj(obj Object, pkg *Package, name string) bool {
	if obj == nil || obj.Name() != name {
		return false
	}
	if token.IsExported(name) {
		return true
	}
	return obj.Pkg() == pkg
}

func hasPointerRecv(m *Func) bool {
	if m == nil {
		return false
	}
	sig := m.Signature()
	if sig == nil || sig.recv == nil {
		return false
	}
	_, ok := Unalias(sig.recv.typ).(*Pointer)
	return ok
}

func concatIndex(list []int, i int) []int {
	out := make([]int, len(list)+1)
	copy(out, list)
	out[len(list)] = i
	return out
}

// MissingMethod returns (nil, false) if V implements T, otherwise the first
// missing or mismatched method required by T (mirroring go/types.MissingMethod).
func MissingMethod(V Type, T *Interface, static bool) (method *Func, wrongType bool) {
	if T == nil || V == nil {
		return nil, false
	}
	T.resolve()
	if T.NumMethods() == 0 {
		return nil, false
	}
	if vIface, ok := V.Underlying().(*Interface); ok {
		vIface.resolve()
		for i := 0; i < T.NumMethods(); i++ {
			tm := T.Method(i)
			var vm *Func
			for j := 0; j < vIface.NumMethods(); j++ {
				cand := vIface.Method(j)
				if cand.Id() == tm.Id() {
					vm = cand
					break
				}
			}
			if vm == nil {
				if static {
					return tm, false
				}
				continue
			}
			if !Identical(vm.typ, tm.typ) {
				return tm, true
			}
		}
		return nil, false
	}

	for i := 0; i < T.NumMethods(); i++ {
		tm := T.Method(i)
		obj, _, indirect := LookupFieldOrMethod(V, false, tm.Pkg(), tm.Name())
		fn, ok := obj.(*Func)
		if !ok || fn == nil {
			if indirect {
				return tm, true
			}
			return tm, false
		}
		if !Identical(fn.typ, tm.typ) {
			return tm, true
		}
	}
	return nil, false
}

// Implements reports whether type V implements interface T
// (mirroring go/types.Implements).
func Implements(V Type, T *Interface) bool {
	if T == nil || V == nil {
		return false
	}
	T.resolve()
	if !T.IsMethodSet() {
		return false
	}
	m, _ := MissingMethod(V, T, true)
	return m == nil
}

// Satisfies reports whether type V satisfies the type constraint T
// (mirroring go/types.Satisfies).
func Satisfies(V Type, T *Interface) bool {
	if T == nil || V == nil {
		return false
	}
	T.resolve()
	if m, _ := MissingMethod(V, T, true); m != nil {
		return false
	}
	if T.IsComparable() && !Comparable(V) {
		return false
	}
	if T.IsMethodSet() {
		return true
	}
	// Check embedded type terms / unions.
	for i := 0; i < T.NumEmbeddeds(); i++ {
		if !satisfiesEmbedded(V, T.EmbeddedType(i)) {
			return false
		}
	}
	return true
}

func satisfiesEmbedded(V Type, emb Type) bool {
	if emb == nil {
		return true
	}
	if iface, ok := emb.Underlying().(*Interface); ok {
		return Satisfies(V, iface)
	}
	switch u := Unalias(emb).(type) {
	case *Union:
		for _, term := range u.terms {
			if termMatches(V, term) {
				return true
			}
		}
		return false
	default:
		return Identical(V, emb)
	}
}

func termMatches(V Type, term *Term) bool {
	if term == nil || term.typ == nil {
		return false
	}
	if term.tilde {
		return Identical(V.Underlying(), term.typ.Underlying())
	}
	return Identical(V, term.typ)
}

// AssignableTo reports whether a value of type V is assignable to a variable of
// type T (mirroring go/types.AssignableTo, respecting LLGo CallConv and PtrKind).
func AssignableTo(V, T Type) bool {
	if V == nil || T == nil {
		return false
	}
	Vu := Unalias(V)
	Tu := Unalias(T)
	if Identical(Vu, Tu) {
		return true
	}
	// Untyped nil is assignable to pointer, slice, map, chan, func, interface.
	if b, ok := Vu.(*Basic); ok && b.kind == UntypedNil {
		switch Tu.Underlying().(type) {
		case *Pointer, *Slice, *Map, *Chan, *Signature, *Interface:
			return true
		}
		return false
	}
	// Untyped constants assignable to matching typed basic types.
	if vb, ok := Vu.(*Basic); ok && vb.info&IsUntyped != 0 {
		if tb, ok := Tu.Underlying().(*Basic); ok {
			switch vb.kind {
			case UntypedBool:
				return tb.info&IsBoolean != 0
			case UntypedInt, UntypedRune:
				return tb.info&(IsInteger|IsFloat|IsComplex) != 0
			case UntypedFloat:
				return tb.info&(IsFloat|IsComplex) != 0
			case UntypedComplex:
				return tb.info&IsComplex != 0
			case UntypedString:
				return tb.info&IsString != 0
			}
		}
		if iface, ok := Tu.Underlying().(*Interface); ok && iface.Empty() {
			return true
		}
		return false
	}
	// Identical underlying types when at least one is not a defined (*Named) type.
	_, vNamed := Vu.(*Named)
	_, tNamed := Tu.(*Named)
	if (!vNamed || !tNamed) && Identical(Vu.Underlying(), Tu.Underlying()) {
		return true
	}
	// Bidirectional chan assignable to directional chan with identical element type.
	if vc, ok := Vu.Underlying().(*Chan); ok && vc.dir == SendRecv {
		if tc, ok := Tu.Underlying().(*Chan); ok && (!vNamed || !tNamed) && Identical(vc.elem, tc.elem) {
			return true
		}
	}
	// Value assignable to interface T if V implements T.
	if ti, ok := Tu.Underlying().(*Interface); ok && ti.IsMethodSet() {
		return Implements(V, ti)
	}
	return false
}

// ConvertibleTo reports whether a value of type V is convertible to type T
// (mirroring go/types.ConvertibleTo).
func ConvertibleTo(V, T Type) bool {
	if V == nil || T == nil {
		return false
	}
	if AssignableTo(V, T) {
		return true
	}
	Vu := V.Underlying()
	Tu := T.Underlying()
	if IdenticalIgnoreTags(Vu, Tu) {
		return true
	}
	// Pointers to types with identical underlying types ignoring tags (when neither is defined).
	if vp, ok := Vu.(*Pointer); ok {
		if tp, ok := Tu.(*Pointer); ok && vp.kind == tp.kind {
			_, vNamed := Unalias(V).(*Named)
			_, tNamed := Unalias(T).(*Named)
			if (!vNamed || !tNamed) && IdenticalIgnoreTags(vp.elem.Underlying(), tp.elem.Underlying()) {
				return true
			}
		}
	}
	vb, vBasic := Vu.(*Basic)
	tb, tBasic := Tu.(*Basic)
	if vBasic && tBasic {
		// Numeric conversions (int/float <-> int/float, complex <-> complex).
		if vb.info&(IsInteger|IsFloat) != 0 && tb.info&(IsInteger|IsFloat) != 0 {
			return true
		}
		if vb.info&IsComplex != 0 && tb.info&IsComplex != 0 {
			return true
		}
		// Integer -> string.
		if vb.info&IsInteger != 0 && tb.info&IsString != 0 {
			return true
		}
		// unsafe.Pointer <-> uintptr.
		if (vb.kind == UnsafePointer && tb.kind == Uintptr) || (vb.kind == Uintptr && tb.kind == UnsafePointer) {
			return true
		}
	}
	// Pointer <-> unsafe.Pointer.
	if _, ok := Vu.(*Pointer); ok && tBasic && tb.kind == UnsafePointer {
		return true
	}
	if vBasic && vb.kind == UnsafePointer {
		if _, ok := Tu.(*Pointer); ok {
			return true
		}
	}
	// string <-> []byte / []rune.
	if vBasic && vb.info&IsString != 0 {
		if sl, ok := Tu.(*Slice); ok {
			if eb, ok := sl.elem.Underlying().(*Basic); ok && (eb.kind == Uint8 || eb.kind == Int32) {
				return true
			}
		}
	}
	if tBasic && tb.info&IsString != 0 {
		if sl, ok := Vu.(*Slice); ok {
			if eb, ok := sl.elem.Underlying().(*Basic); ok && (eb.kind == Uint8 || eb.kind == Int32) {
				return true
			}
		}
	}
	// []T -> [N]T or *[N]T (Go 1.17 / Go 1.20 slice-to-array conversions).
	if vsl, ok := Vu.(*Slice); ok {
		if tarr, ok := Tu.(*Array); ok && Identical(vsl.elem, tarr.elem) {
			return true
		}
		if tp, ok := Tu.(*Pointer); ok {
			if tarr, ok := tp.elem.Underlying().(*Array); ok && Identical(vsl.elem, tarr.elem) {
				return true
			}
		}
	}
	return false
}

// AssertableTo reports whether a value of interface type V can be asserted to
// have type T (mirroring go/types.AssertableTo).
func AssertableTo(V *Interface, T Type) bool {
	if V == nil || T == nil {
		return false
	}
	if _, ok := T.Underlying().(*Interface); ok {
		return true
	}
	m, _ := MissingMethod(T, V, false)
	return m == nil
}

// syntheticMethodSet collects direct and promoted methods on T sorted by Id().
func syntheticMethodSet(T Type) *MethodSet {
	if T == nil {
		return &MethodSet{}
	}
	if iface, ok := T.Underlying().(*Interface); ok {
		iface.resolve()
		list := make([]*Selection, iface.NumMethods())
		for i := 0; i < iface.NumMethods(); i++ {
			m := iface.Method(i)
			list[i] = &Selection{
				kind:  MethodVal,
				recv:  T,
				obj:   m,
				typ:   m.typ,
				index: []int{i},
			}
		}
		return &MethodSet{list: list}
	}

	typ, isPtr := derefType(T)
	if isPtr {
		if _, ok := Unalias(typ).(*Pointer); ok {
			return &MethodSet{}
		}
	}

	type mentry struct {
		sel       *Selection
		ambiguous bool
	}
	mset := make(map[string]mentry)
	current := []embeddedType{{typ: typ, indirect: isPtr}}
	visited := make(map[Type]bool)

	for len(current) > 0 {
		var next []embeddedType
		level := make(map[string]mentry)
		addMethod := func(id string, sel *Selection, multiples bool) {
			if _, exists := mset[id]; exists {
				return
			}
			if prev, dup := level[id]; dup || multiples {
				prev.ambiguous = true
				level[id] = prev
				return
			}
			level[id] = mentry{sel: sel}
		}

		for _, e := range current {
			base := Unalias(e.typ)
			if visited[base] {
				continue
			}
			visited[base] = true

			if n, ok := base.(*Named); ok {
				for i := 0; i < n.NumMethods(); i++ {
					m := n.Method(i)
					if !e.indirect && hasPointerRecv(m) {
						continue
					}
					addMethod(m.Id(), &Selection{
						kind:     MethodVal,
						recv:     T,
						obj:      m,
						typ:      m.typ,
						index:    concatIndex(e.index, i),
						indirect: e.indirect,
					}, e.multiples)
				}
			}

			switch u := base.Underlying().(type) {
			case *Interface:
				for i := 0; i < u.NumMethods(); i++ {
					m := u.Method(i)
					addMethod(m.Id(), &Selection{
						kind:     MethodVal,
						recv:     T,
						obj:      m,
						typ:      m.typ,
						index:    concatIndex(e.index, i),
						indirect: true,
					}, e.multiples)
				}
			case *Struct:
				for i, f := range u.fields {
					if f != nil && f.embedded {
						ft, ptr := derefType(f.typ)
						next = append(next, embeddedType{
							typ:      ft,
							index:    concatIndex(e.index, i),
							indirect: e.indirect || ptr,
						})
					}
				}
			}
		}
		for id, entry := range level {
			mset[id] = entry
		}
		current = consolidateEmbedded(next)
	}

	var list []*Selection
	for _, entry := range mset {
		if !entry.ambiguous && entry.sel != nil {
			list = append(list, entry.sel)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].obj.Id() < list[j].obj.Id()
	})
	return &MethodSet{list: list}
}
