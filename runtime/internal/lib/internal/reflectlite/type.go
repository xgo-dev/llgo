// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflectlite implements lightweight version of reflect, not using
// any package except for "runtime", "unsafe", and "internal/abi"
package reflectlite

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// Type is the representation of a Go type.
//
// Not all methods apply to all kinds of types. Restrictions,
// if any, are noted in the documentation for each method.
// Use the Kind method to find out the kind of type before
// calling kind-specific methods. Calling a method
// inappropriate to the kind of type causes a run-time panic.
//
// Type values are comparable, such as with the == operator,
// so they can be used as map keys.
// Two Type values are equal if they represent identical types.
type Type interface {
	// Methods applicable to all types.

	// Name returns the type's name within its package for a defined type.
	// For other (non-defined) types it returns the empty string.
	Name() string

	// PkgPath returns a defined type's package path, that is, the import path
	// that uniquely identifies the package, such as "encoding/base64".
	// If the type was predeclared (string, error) or not defined (*T, struct{},
	// []int, or A where A is an alias for a non-defined type), the package path
	// will be the empty string.
	PkgPath() string

	// Size returns the number of bytes needed to store
	// a value of the given type; it is analogous to unsafe.Sizeof.
	Size() uintptr

	// Kind returns the specific kind of this type.
	Kind() Kind

	// Implements reports whether the type implements the interface type u.
	Implements(u Type) bool

	// AssignableTo reports whether a value of the type is assignable to type u.
	AssignableTo(u Type) bool

	// Comparable reports whether values of this type are comparable.
	Comparable() bool

	// String returns a string representation of the type.
	// The string representation may use shortened package names
	// (e.g., base64 instead of "encoding/base64") and is not
	// guaranteed to be unique among types. To test for type identity,
	// compare the Types directly.
	String() string

	// Elem returns a type's element type.
	// It panics if the type's Kind is not Ptr.
	Elem() Type

	common() *abi.Type
	uncommon() *uncommonType
}

/*
 * These data structures are known to the compiler (../../cmd/internal/reflectdata/reflect.go).
 * A few are known to ../runtime/type.go to convey to debuggers.
 * They are also known to ../runtime/type.go.
 */

// A Kind represents the specific kind of type that a Type represents.
// The zero Kind is not a valid kind.
type Kind = abi.Kind

const Ptr = abi.Pointer

const (
	// Import-and-export these constants as necessary
	Interface = abi.Interface
	Slice     = abi.Slice
	String    = abi.String
	Struct    = abi.Struct
)

type rtype struct {
	*abi.Type
}

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type uncommonType = abi.UncommonType

// arrayType represents a fixed array type.
type arrayType = abi.ArrayType

// chanType represents a channel type.
type chanType = abi.ChanType

type funcType = abi.FuncType

type interfaceType = abi.InterfaceType

// mapType represents a map type.
type mapType struct {
	rtype
	Key    *abi.Type // map key type
	Elem   *abi.Type // map element (value) type
	Bucket *abi.Type // internal bucket structure
	// function for hashing keys (ptr to key, seed) -> hash
	Hasher     func(unsafe.Pointer, uintptr) uintptr
	KeySize    uint8  // size of key slot
	ValueSize  uint8  // size of value slot
	BucketSize uint16 // size of bucket
	Flags      uint32
}

// ptrType represents a pointer type.
type ptrType = abi.PtrType

// sliceType represents a slice type.
type sliceType = abi.SliceType

// structType represents a struct type.
type structType = abi.StructType

func (t rtype) uncommon() *uncommonType {
	return t.Uncommon()
}

func (t rtype) String() string {
	return t.Type.String()
}

func (t rtype) common() *abi.Type { return t.Type }

func (t rtype) exportedMethods() []abi.Method {
	ut := t.uncommon()
	if ut == nil {
		return nil
	}
	return ut.ExportedMethods()
}

func (t rtype) NumMethod() int {
	tt := t.Type.InterfaceType()
	if tt != nil {
		return tt.NumMethod()
	}
	return len(t.exportedMethods())
}

func (t rtype) PkgPath() string {
	if t.TFlag&abi.TFlagNamed == 0 {
		return ""
	}
	if t.Kind() == abi.UnsafePointer {
		return "unsafe"
	}
	ut := t.uncommon()
	if ut == nil {
		return ""
	}
	return ut.PkgPath_
}

func (t rtype) Name() string {
	if !t.HasName() {
		return ""
	}
	s := t.String()
	i := len(s) - 1
	sqBrackets := 0
	for i >= 0 && (s[i] != '.' || sqBrackets != 0) {
		switch s[i] {
		case ']':
			sqBrackets++
		case '[':
			sqBrackets--
		}
		i--
	}
	return s[i+1:]
}

func toRType(t *abi.Type) rtype {
	return rtype{t}
}

func elem(t *abi.Type) *abi.Type {
	et := t.Elem()
	if et != nil {
		return et
	}
	panic("reflect: Elem of invalid type " + toRType(t).String())
}

func (t rtype) Elem() Type {
	return toType(elem(t.common()))
}

func (t rtype) In(i int) Type {
	tt := t.Type.FuncType()
	if tt == nil {
		panic("reflect: In of non-func type")
	}
	return toType(tt.In[i])
}

func (t rtype) Key() Type {
	tt := t.Type.MapType()
	if tt == nil {
		panic("reflect: Key of non-map type")
	}
	return toType(tt.Key)
}

func (t rtype) Len() int {
	tt := t.Type.ArrayType()
	if tt == nil {
		panic("reflect: Len of non-array type")
	}
	return int(tt.Len)
}

func (t rtype) NumField() int {
	tt := t.Type.StructType()
	if tt == nil {
		panic("reflect: NumField of non-struct type")
	}
	return len(tt.Fields)
}

func (t rtype) NumIn() int {
	tt := t.Type.FuncType()
	if tt == nil {
		panic("reflect: NumIn of non-func type")
	}
	return len(tt.In)
}

func (t rtype) NumOut() int {
	tt := t.Type.FuncType()
	if tt == nil {
		panic("reflect: NumOut of non-func type")
	}
	return len(tt.Out)
}

func (t rtype) Out(i int) Type {
	tt := t.Type.FuncType()
	if tt == nil {
		panic("reflect: Out of non-func type")
	}
	return toType(tt.Out[i])
}

// add returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// TypeOf returns the reflection Type that represents the dynamic type of i.
// If i is a nil interface value, TypeOf returns nil.
func TypeOf(i any) Type {
	eface := *(*emptyInterface)(unsafe.Pointer(&i))
	return toType(eface.typ)
}

func (t rtype) Implements(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.Implements")
	}
	if u.Kind() != Interface {
		panic("reflect: non-interface type passed to Type.Implements")
	}
	return implements(u.common(), t.common())
}

func (t rtype) AssignableTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo")
	}
	uu := u.common()
	tt := t.common()
	return directlyAssignable(uu, tt) || implements(uu, tt)
}

func (t rtype) Comparable() bool {
	return t.Equal != nil
}

// implements reports whether the type V implements the interface type T.
//
//go:linkname implements github.com/goplus/llgo/runtime/internal/runtime.Implements
func implements(T, V *abi.Type) bool

// directlyAssignable reports whether a value x of type V can be directly
// assigned (using memmove) to a value of type T.
// https://golang.org/doc/go_spec.html#Assignability
// Ignoring the interface rules (implemented elsewhere)
// and the ideal constant rules (no ideal constants at run time).
func directlyAssignable(T, V *abi.Type) bool {
	// x's type V is identical to T?
	if T == V {
		return true
	}

	// Otherwise at least one of T and V must not be defined
	// and they must have the same kind.
	if T.HasName() && V.HasName() || T.Kind() != V.Kind() {
		return false
	}

	// x's type T and V must  have identical underlying types.
	return haveIdenticalUnderlyingType(T, V, true)
}

func haveIdenticalType(T, V *abi.Type, cmpTags bool) bool {
	if cmpTags {
		return T == V
	}

	if toRType(T).Name() != toRType(V).Name() || T.Kind() != V.Kind() {
		return false
	}

	return haveIdenticalUnderlyingType(T, V, false)
}

func haveIdenticalUnderlyingType(T, V *abi.Type, cmpTags bool) bool {
	if T == V {
		return true
	}

	kind := T.Kind()
	if kind != V.Kind() {
		return false
	}

	// Non-composite types of equal kind have same underlying type
	// (the predefined instance of the type).
	if abi.Bool <= kind && kind <= abi.Complex128 || kind == abi.String || kind == abi.UnsafePointer {
		return true
	}

	// Composite types.
	switch kind {
	case abi.Array:
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case abi.Chan:
		// Special case:
		// x is a bidirectional channel value, T is a channel type,
		// and x's type V and T have identical element types.
		if V.ChanDir() == abi.BothDir && haveIdenticalType(T.Elem(), V.Elem(), cmpTags) {
			return true
		}

		// Otherwise continue test for identical underlying type.
		return V.ChanDir() == T.ChanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case abi.Func:
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if len(t.Out) != len(v.Out) || len(t.In) != len(v.In) {
			return false
		}
		for i := 0; i < len(t.In); i++ {
			if !haveIdenticalType(t.In[i], v.In[i], cmpTags) {
				return false
			}
		}
		for i := 0; i < len(t.Out); i++ {
			if !haveIdenticalType(t.Out[i], v.Out[i], cmpTags) {
				return false
			}
		}
		return true

	case Interface:
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.Methods) == 0 && len(v.Methods) == 0 {
			return true
		}
		// Might have the same methods but still
		// need a run time conversion.
		return false

	case abi.Map:
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Ptr, abi.Slice:
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case abi.Struct:
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.Fields) != len(v.Fields) {
			return false
		}
		if t.PkgPath_ != v.PkgPath_ {
			return false
		}
		for i := range t.Fields {
			tf := &t.Fields[i]
			vf := &v.Fields[i]
			if tf.Name_ != vf.Name_ {
				return false
			}
			if !haveIdenticalType(tf.Typ, vf.Typ, cmpTags) {
				return false
			}
			if cmpTags && tf.Tag_ != vf.Tag_ {
				return false
			}
			if tf.Offset != vf.Offset {
				return false
			}
			if tf.Embedded() != vf.Embedded() {
				return false
			}
		}
		return true
	}

	return false
}

// toType converts from a *rtype to a Type that can be returned
// to the client of package reflect. In gc, the only concern is that
// a nil *rtype must be replaced by a nil Type, but in gccgo this
// function takes care of ensuring that multiple *rtype for the same
// type are coalesced into a single Type.
func toType(t *abi.Type) Type {
	if t == nil {
		return nil
	}
	return toRType(t)
}

// ifaceIndir reports whether t is stored indirectly in an interface value.
func ifaceIndir(t *abi.Type) bool {
	return t.Kind_&abi.KindDirectIface == 0
}
