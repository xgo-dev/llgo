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

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
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
	// Match Go's runtime itabInitSize. Capacities stay powers of two so probing
	// can use a mask.
	itabInitSize = 512
)

// itabTableType is an open-addressed table whose slots are only filled, never
// removed. Writers hold itabTable.mutex. Hosted targets publish pointers
// atomically; baremetal uses its single-context task backend. A grown table is
// fully rehashed before publication, so readers never see a partial rehash.
type itabTableType struct {
	size    uintptr
	count   uintptr
	entries []unsafe.Pointer
}

var itabTable struct {
	mutex
	// table points to the current table. Use storeItabPointer to publish it
	// because interface conversions read it without taking the lock.
	table unsafe.Pointer
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

func itabHash(inter *InterfaceType, typ *Type) uintptr {
	return uintptr(inter.Hash ^ typ.Hash)
}

func loadItabTable() *itabTableType {
	return (*itabTableType)(loadItabPointer(&itabTable.table))
}

func newItabTable(size uintptr) *itabTableType {
	// Keep the table header and its pointer slots in one allocation.
	// The unsafe.Pointer slots use target-specific pointer loads and stores.
	bytes := unsafe.Sizeof(itabTableType{}) + size*unsafe.Sizeof(unsafe.Pointer(nil))
	// Empty slots and count must start at zero, including with the nogc allocator.
	ptr := AllocZ(bytes)
	t := (*itabTableType)(ptr)
	t.size = size
	t.entries = unsafe.Slice((*unsafe.Pointer)(unsafe.Add(ptr, unsafe.Sizeof(itabTableType{}))), int(size))
	return t
}

func (t *itabTableType) find(inter *InterfaceType, typ *Type) *Itab {
	if t == nil || t.size == 0 {
		return nil
	}
	mask := t.size - 1
	h := itabHash(inter, typ) & mask
	// Triangular probing visits every slot in a power-of-two table. The 75%
	// load limit guarantees an empty slot; bound probes if that invariant breaks.
	for step := uintptr(1); step <= t.size; step++ {
		m := (*Itab)(loadItabPointer(&t.entries[h]))
		if m == nil {
			return nil
		}
		if m.inter == inter && m._type == typ {
			return m
		}
		h = (h + step) & mask
	}
	throw("itab table is full")
	return nil
}

// add inserts an itab into a table. The caller must hold itabTable.mutex;
// t.size must be a nonzero power of two and the table must have an empty slot.
func (t *itabTableType) add(i *Itab) {
	mask := t.size - 1
	h := itabHash(i.inter, i._type) & mask
	for step := uintptr(1); step <= t.size; step++ {
		m := (*Itab)(loadItabPointer(&t.entries[h]))
		if m == nil {
			storeItabPointer(&t.entries[h], unsafe.Pointer(i))
			t.count++
			return
		}
		if m.inter == i.inter && m._type == i._type {
			return
		}
		h = (h + step) & mask
	}
	throw("itab table is full")
}

func findItab(inter *InterfaceType, typ *Type) *Itab {
	return loadItabTable().find(inter, typ)
}

func addItab(i *Itab) *Itab {
	itabTable.Lock()
	t := loadItabTable()
	if t == nil {
		t = newItabTable(itabInitSize)
		t.add(i)
		storeItabPointer(&itabTable.table, unsafe.Pointer(t))
		itabTable.Unlock()
		return i
	}
	// NewItab constructs outside the lock. Recheck under the lock so two
	// concurrent conversions still return one canonical itab pointer.
	if existing := t.find(i.inter, i._type); existing != nil {
		itabTable.Unlock()
		return existing
	}
	if t.count >= 3*(t.size/4) {
		grown := newItabTable(t.size * 2)
		// Use the same pointer access helpers as concurrent lookups.
		for index := range t.entries {
			if old := (*Itab)(loadItabPointer(&t.entries[index])); old != nil {
				grown.add(old)
			}
		}
		grown.add(i)
		storeItabPointer(&itabTable.table, unsafe.Pointer(grown))
		itabTable.Unlock()
		return i
	}
	t.add(i)
	itabTable.Unlock()
	return i
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
		// Locate the variable tail in bytes, then index it as Go uintptr slots.
		// On J32 C pointer arrays have four-byte elements while itab function
		// slots follow Go's eight-byte word model.
		data := c.Advance(ptr, int(itabHdrSize))
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
				fn = abi.Text(c.Func(unreachableMethod))
			}
			slot := unsafe.Add(data, uintptr(i)*unsafe.Sizeof(uintptr(0)))
			*(*uintptr)(slot) = uintptr(fn)
		}
	}
	if ret.fun[0] != 0 {
		return addItab(ret)
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

// unreachableMethod matches Go's runtime.unreachableMethod: method entries
// proven unreachable by link-time DCE are redirected here and should never run.
func unreachableMethod() {
	throw("unreachable method called. linker bug?")
}

func IfaceType(i iface) *abi.Type {
	if i.tab == nil {
		return nil
	}
	return i.tab._type
}

func IfacePtrData(i iface) unsafe.Pointer {
	if i.tab == nil {
		panic(errorString("invalid memory address or nil pointer dereference"))
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

func IfacePtrDataBuffer(i iface, buffer *unsafe.Pointer) unsafe.Pointer {
	if i.tab == nil {
		panic(errorString("invalid memory address or nil pointer dereference"))
	}
	if DirectIfaceData(i.tab._type) {
		*buffer = i.data
		return unsafe.Pointer(buffer)
	}
	return i.data
}

func DirectIfaceData(typ *abi.Type) bool {
	// Pointer receivers already carry their address. Other direct-interface
	// values need an addressable copy of the stored word. The descriptor flag
	// includes the single-element array and struct cases as well.
	return isDirectIface(typ) && typ.Kind() != abi.Pointer
}

// MatchesClosure reports whether the type V matches the closure type T for
// type assertion purposes.
func MatchesClosure(T, V *abi.Type) bool {
	if T == V {
		return true
	} else if T == nil || V == nil {
		return false
	}
	if T.IsClosure() {
		T = T.StructType().Fields[0].Typ
	}
	if V.IsClosure() {
		V = V.StructType().Fields[0].Typ
	}
	return identicalFuncType(T, V)
}

func identicalFuncType(T, V *abi.Type) bool {
	if T == V {
		return true
	}
	if T == nil || V == nil || T.Kind() != abi.Func || V.Kind() != abi.Func {
		return false
	}
	t := T.FuncType()
	v := V.FuncType()
	if t.Variadic() != v.Variadic() || len(t.In) != len(v.In) || len(t.Out) != len(v.Out) {
		return false
	}
	for i := range t.In {
		if t.In[i] != v.In[i] {
			return false
		}
	}
	for i := range t.Out {
		if t.Out[i] != v.Out[i] {
			return false
		}
	}
	return true
}

// interfaceImplementation reports whether V implements the interface T. On
// failure, missing is the first unmatched method in T's sorted method table,
// when such a method exists.
func interfaceImplementation(T, V *abi.Type) (missing *abi.Imethod, ok bool) {
	if T == nil || V == nil || T.Kind() != abi.Interface {
		return nil, false
	}
	t := (*abi.InterfaceType)(unsafe.Pointer(T))

	if len(t.Methods) == 0 {
		return nil, true
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
					return nil, true
				}
			}
		}
		return &t.Methods[i], false
	}

	v := V.Uncommon()
	if v == nil {
		return &t.Methods[0], false
	}
	i := 0
	for j := 0; j < int(v.Mcount); j++ {
		tm := &t.Methods[i]
		vm := v.Method(j)
		if vm.Name_ == tm.Name_ && vm.Mtyp_ == tm.Typ_ {
			if i++; i >= len(t.Methods) {
				return nil, true
			}
		}
	}
	return &t.Methods[i], false
}

// Implements reports whether the type V implements the interface type T.
func Implements(T, V *abi.Type) bool {
	_, ok := interfaceImplementation(T, V)
	return ok
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
	missing, _ := interfaceImplementation(&inter.Type, typ)
	missingMethod := ""
	if missing != nil {
		missingMethod = missing.Name()
	}
	panic(&TypeAssertionError{
		concrete:      typ,
		asserted:      &inter.Type,
		missingMethod: missingMethod,
	})
}

// -----------------------------------------------------------------------------
