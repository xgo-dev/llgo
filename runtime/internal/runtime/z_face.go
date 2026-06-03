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

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
)

type eface struct {
	_type *_type
	data  unsafe.Pointer
}

type iface struct {
	tab  *itab
	data unsafe.Pointer
}

type interfacetype = abi.InterfaceType

// layout of Itab known to compilers
// allocated in non-garbage-collected memory
// Needs to be in sync with
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.WriteTabs.
type itab struct {
	inter *interfacetype
	_type *_type
	hash  uint32     // copy of _type.hash. Used for type switches.
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}

// -----------------------------------------------------------------------------

type (
	Eface = eface
	Iface = iface
	Itab  = itab
)

type Imethod = abi.Imethod
type Method = abi.Method
type FuncType = abi.FuncType
type InterfaceType = abi.InterfaceType

// ToEface converts an iface to an eface.
func ToEface(i Iface) Eface {
	return Eface{i.tab._type, i.data}
}

// -----------------------------------------------------------------------------

const (
	pointerSize = unsafe.Sizeof(uintptr(0))
	itabHdrSize = unsafe.Sizeof(itab{}) - pointerSize
)

var itabTable struct {
	mutex
	entries []*Itab
}

func init() {
	(*sync.Mutex)(&itabTable.mutex).Init(nil)
}

type mutex sync.Mutex

func (m *mutex) Lock() {
	(*sync.Mutex)(m).Lock()
}

func (m *mutex) Unlock() {
	(*sync.Mutex)(m).Unlock()
}

func findItab(inter *InterfaceType, typ *Type) *Itab {
	itabTable.Lock()
	for _, i := range itabTable.entries {
		if i.inter == inter && i._type == typ {
			itabTable.Unlock()
			return i
		}
	}
	itabTable.Unlock()
	return nil
}

func addItab(i *Itab) {
	itabTable.Lock()
	itabTable.entries = append(itabTable.entries, i)
	itabTable.Unlock()
}

// NewItab returns a new itab.
func NewItab(inter *InterfaceType, typ *Type) *Itab {
	if typ == nil {
		return nil
	}
	if i := findItab(inter, typ); i != nil {
		return i
	}
	n := len(inter.Methods)
	size := itabHdrSize + uintptr(n)*pointerSize
	ptr := AllocU(size)

	ret := (*itab)(ptr)
	ret.inter = inter
	ret._type = typ
	ret.hash = typ.Hash

	u := typ.Uncommon()
	if u == nil {
		ret.fun[0] = 0
	} else {
		data := (*uintptr)(c.Advance(ptr, int(itabHdrSize)))
		mthds := u.Methods()
		for i, m := range inter.Methods {
			fn, matched := findMethod(mthds, m)
			if !matched {
				ret.fun[0] = 0
				break
			}
			if fn == nil {
				// matched but no function pointer (e.g. stripped/unreachable after DCE);
				// keep itab entry with a placeholder so the itab stays intact
				// and only panics on call, not at assertion time.
				fn = abi.Text(uintptr(0))
			}
			*c.Advance(data, i) = uintptr(fn)
		}
	}
	if ret.fun[0] != 0 {
		addItab(ret)
	}
	return ret
}

// findMethod searches for an interface method in the type's method set.
// Returns (fn, matched) where:
//   - fn: the function pointer (may be nil even when matched=true for promoted unexported methods)
//   - matched: true if a method with matching name and type signature exists
func findMethod(mthds []abi.Method, im abi.Imethod) (abi.Text, bool) {
	imName := im.Name_
	for _, m := range mthds {
		mName := m.Name_
		if mName >= imName {
			if mName == imName && m.Mtyp_ == im.Typ_ {
				return m.Ifn_, true
			}
			break
		}
	}
	return nil, false
}

func IfaceType(i iface) *abi.Type {
	if i.tab == nil {
		return nil
	}
	return i.tab._type
}

func IfacePtrData(i iface) unsafe.Pointer {
	if i.tab == nil {
		panic(errorString("invalid memory address or nil pointer dereference").Error())
	}
	if DirectIfaceData(i.tab._type) {
		// For direct-iface values, i.data holds the value bits (not a stable
		// pointer). Interface method calls use the "one-word receiver" ABI which
		// expects a pointer to the receiver value. We must return a stable
		// address, not the address of a stack copy of i.
		p := AllocU(pointerSize)
		*(*unsafe.Pointer)(p) = i.data
		return p
	}
	return i.data
}

func DirectIfaceData(typ *abi.Type) bool {
	switch typ.Kind() {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64, abi.Array, abi.Struct,
		// These are single-word values stored directly in iface.data.
		// For interface method calls, non-pointer receivers still require a
		// pointer to the stored word.
		abi.Chan, abi.Func, abi.Map, abi.UnsafePointer:
		if isDirectIface(typ) {
			return true
		}
	}
	return false
}

// MatchesClosure reports whether the type V matches the closure type T for
// type assertion purposes.
func MatchesClosure(T, V *abi.Type) bool {
	if T == V {
		return true
	} else if V == nil || !V.IsClosure() {
		return false
	}
	return T.StructType().Fields[0].Typ == V.StructType().Fields[0].Typ
}

// Implements reports whether the type V implements the interface type T.
func Implements(T, V *abi.Type) bool {
	if V == nil {
		return false
	}
	if T.Kind() != abi.Interface {
		return false
	}
	t := (*abi.InterfaceType)(unsafe.Pointer(T))

	if len(t.Methods) == 0 {
		return true
	}

	// The same algorithm applies in both cases, but the
	// method tables for an interface type and a concrete type
	// are different, so the code is duplicated.
	// In both cases the algorithm is a linear scan over the two
	// lists - T's methods and V's methods - simultaneously.
	// Since method tables are stored in a unique sorted order
	// (alphabetical, with no duplicate method names), the scan
	// through V's methods must hit a match for each of T's
	// methods along the way, or else V does not implement T.
	// This lets us run the scan in overall linear time instead of
	// the quadratic time  a naive search would require.
	// See also ../runtime/iface.go.
	if V.Kind() == abi.Interface {
		v := (*abi.InterfaceType)(unsafe.Pointer(V))
		i := 0
		for j := 0; j < len(v.Methods); j++ {
			tm := &t.Methods[i]
			vm := &v.Methods[j]
			if vm.Name_ == tm.Name_ && vm.Typ_ == tm.Typ_ {
				if i++; i >= len(t.Methods) {
					return true
				}
			}
		}
		return false
	}

	v := V.Uncommon()
	if v == nil {
		return false
	}
	i := 0
	vmethods := v.Methods()
	for j := 0; j < int(v.Mcount); j++ {
		tm := &t.Methods[i]
		vm := vmethods[j]
		if vm.Name_ == tm.Name_ && vm.Mtyp_ == tm.Typ_ {
			if i++; i >= len(t.Methods) {
				return true
			}
		}
	}
	return false
}

func EfaceEqual(v, u eface) bool {
	if v._type == nil || u._type == nil {
		return v._type == u._type
	}
	if v._type != u._type {
		return false
	}
	equal := v._type.Equal
	if equal == nil {
		panic(errorString("comparing uncomparable type " + v._type.String()))
	}
	if isDirectIface(v._type) {
		return v.data == u.data
	}
	return equal(v.data, u.data)
}

func (v eface) Kind() abi.Kind {
	if v._type == nil {
		return abi.Invalid
	}
	return v._type.Kind()
}

func (v eface) Elem() eface {
	switch v.Kind() {
	case abi.Interface:
		var i any
		tt := (*abi.InterfaceType)(unsafe.Pointer(v._type))
		if len(tt.Methods) == 0 {
			i = *(*any)(v.data)
		} else {
			i = (any)(*(*interface {
				M()
			})(v.data))
		}
		return *(*eface)(unsafe.Pointer(&i))
	case abi.Pointer:
		ptr := v.data
		if isDirectIface(v._type) {
			ptr = *(*unsafe.Pointer)(ptr)
		}
		if ptr == nil {
			return eface{}
		}
		return eface{v._type.Elem(), ptr}
	}
	panic("invalid eface elem")
}

func assertE2I(inter *interfacetype, t *_type) *itab {
	if t == nil {
		// explicit conversions require non-nil interface value.
		panic(&TypeAssertionError{nil, nil, &inter.Type, ""})
	}
	return getitab(inter, t, false)
}

func IfaceE2I(inter *interfacetype, e eface, dst *iface) {
	*dst = iface{assertE2I(inter, e._type), e.data}
}

func getitab(inter *interfacetype, typ *_type, canfail bool) *itab {
	if len(inter.Methods) == 0 {
		panic("internal error - misuse of itab")
	}

	// easy case
	if typ.TFlag&abi.TFlagUncommon == 0 {
		if canfail {
			return nil
		}
		name := inter.Methods[0].Name()
		panic(&TypeAssertionError{nil, typ, &inter.Type, name})
	}

	m := NewItab(inter, typ)

	if m.fun[0] != 0 {
		return m
	}
	if canfail {
		return nil
	}
	panic(&TypeAssertionError{concrete: typ, asserted: &inter.Type, missingMethod: ""})
}

// -----------------------------------------------------------------------------
