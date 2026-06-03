//go:build go1.23 && !go1.26
// +build go1.23,!go1.26

package reflect

import (
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

	// Align returns the alignment in bytes of a value of
	// this type when allocated in memory.
	Align() int

	// FieldAlign returns the alignment in bytes of a value of
	// this type when used as a field in a struct.
	FieldAlign() int

	// Method returns the i'th method in the type's method set.
	// It panics if i is not in the range [0, NumMethod()).
	//
	// For a non-interface type T or *T, the returned Method's Type and Func
	// fields describe a function whose first argument is the receiver,
	// and only exported methods are accessible.
	//
	// For an interface type, the returned Method's Type field gives the
	// method signature, without a receiver, and the Func field is nil.
	//
	// Methods are sorted in lexicographic order.
	Method(int) Method

	// MethodByName returns the method with that name in the type's
	// method set and a boolean indicating if the method was found.
	//
	// For a non-interface type T or *T, the returned Method's Type and Func
	// fields describe a function whose first argument is the receiver.
	//
	// For an interface type, the returned Method's Type field gives the
	// method signature, without a receiver, and the Func field is nil.
	MethodByName(string) (Method, bool)

	// NumMethod returns the number of methods accessible using Method.
	//
	// For a non-interface type, it returns the number of exported methods.
	//
	// For an interface type, it returns the number of exported and unexported methods.
	NumMethod() int

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

	// String returns a string representation of the type.
	// The string representation may use shortened package names
	// (e.g., base64 instead of "encoding/base64") and is not
	// guaranteed to be unique among types. To test for type identity,
	// compare the Types directly.
	String() string

	// Kind returns the specific kind of this type.
	Kind() Kind

	// Implements reports whether the type implements the interface type u.
	Implements(u Type) bool

	// AssignableTo reports whether a value of the type is assignable to type u.
	AssignableTo(u Type) bool

	// ConvertibleTo reports whether a value of the type is convertible to type u.
	// Even if ConvertibleTo returns true, the conversion may still panic.
	// For example, a slice of type []T is convertible to *[N]T,
	// but the conversion will panic if its length is less than N.
	ConvertibleTo(u Type) bool

	// Comparable reports whether values of this type are comparable.
	// Even if Comparable returns true, the comparison may still panic.
	// For example, values of interface type are comparable,
	// but the comparison will panic if their dynamic type is not comparable.
	Comparable() bool

	// Methods applicable only to some types, depending on Kind.
	// The methods allowed for each kind are:
	//
	//	Int*, Uint*, Float*, Complex*: Bits
	//	Array: Elem, Len
	//	Chan: ChanDir, Elem
	//	Func: In, NumIn, Out, NumOut, IsVariadic.
	//	Map: Key, Elem
	//	Pointer: Elem
	//	Slice: Elem
	//	Struct: Field, FieldByIndex, FieldByName, FieldByNameFunc, NumField

	// Bits returns the size of the type in bits.
	// It panics if the type's Kind is not one of the
	// sized or unsized Int, Uint, Float, or Complex kinds.
	Bits() int

	// ChanDir returns a channel type's direction.
	// It panics if the type's Kind is not Chan.
	ChanDir() ChanDir

	// IsVariadic reports whether a function type's final input parameter
	// is a "..." parameter. If so, t.In(t.NumIn() - 1) returns the parameter's
	// implicit actual type []T.
	//
	// For concreteness, if t represents func(x int, y ... float64), then
	//
	//	t.NumIn() == 2
	//	t.In(0) is the reflect.Type for "int"
	//	t.In(1) is the reflect.Type for "[]float64"
	//	t.IsVariadic() == true
	//
	// IsVariadic panics if the type's Kind is not Func.
	IsVariadic() bool

	// Elem returns a type's element type.
	// It panics if the type's Kind is not Array, Chan, Map, Pointer, or Slice.
	Elem() Type

	// Field returns a struct type's i'th field.
	// It panics if the type's Kind is not Struct.
	// It panics if i is not in the range [0, NumField()).
	Field(i int) StructField

	// FieldByIndex returns the nested field corresponding
	// to the index sequence. It is equivalent to calling Field
	// successively for each index i.
	// It panics if the type's Kind is not Struct.
	FieldByIndex(index []int) StructField

	// FieldByName returns the struct field with the given name
	// and a boolean indicating if the field was found.
	// If the returned field is promoted from an embedded struct,
	// then Offset in the returned StructField is the offset in
	// the embedded struct.
	FieldByName(name string) (StructField, bool)

	// FieldByNameFunc returns the struct field with a name
	// that satisfies the match function and a boolean indicating if
	// the field was found.
	//
	// FieldByNameFunc considers the fields in the struct itself
	// and then the fields in any embedded structs, in breadth first order,
	// stopping at the shallowest nesting depth containing one or more
	// fields satisfying the match function. If multiple fields at that depth
	// satisfy the match function, they cancel each other
	// and FieldByNameFunc returns no match.
	// This behavior mirrors Go's handling of name lookup in
	// structs containing embedded fields.
	//
	// If the returned field is promoted from an embedded struct,
	// then Offset in the returned StructField is the offset in
	// the embedded struct.
	FieldByNameFunc(match func(string) bool) (StructField, bool)

	// In returns the type of a function type's i'th input parameter.
	// It panics if the type's Kind is not Func.
	// It panics if i is not in the range [0, NumIn()).
	In(i int) Type

	// Key returns a map type's key type.
	// It panics if the type's Kind is not Map.
	Key() Type

	// Len returns an array type's length.
	// It panics if the type's Kind is not Array.
	Len() int

	// NumField returns a struct type's field count.
	// It panics if the type's Kind is not Struct.
	NumField() int

	// NumIn returns a function type's input parameter count.
	// It panics if the type's Kind is not Func.
	NumIn() int

	// NumOut returns a function type's output parameter count.
	// It panics if the type's Kind is not Func.
	NumOut() int

	// Out returns the type of a function type's i'th output parameter.
	// It panics if the type's Kind is not Func.
	// It panics if i is not in the range [0, NumOut()).
	Out(i int) Type

	// OverflowComplex reports whether the complex128 x cannot be represented by type t.
	// It panics if t's Kind is not Complex64 or Complex128.
	OverflowComplex(x complex128) bool

	// OverflowFloat reports whether the float64 x cannot be represented by type t.
	// It panics if t's Kind is not Float32 or Float64.
	OverflowFloat(x float64) bool

	// OverflowInt reports whether the int64 x cannot be represented by type t.
	// It panics if t's Kind is not Int, Int8, Int16, Int32, or Int64.
	OverflowInt(x int64) bool

	// OverflowUint reports whether the uint64 x cannot be represented by type t.
	// It panics if t's Kind is not Uint, Uintptr, Uint8, Uint16, Uint32, or Uint64.
	OverflowUint(x uint64) bool

	// CanSeq reports whether a [Value] with this type can be iterated over using [Value.Seq].
	CanSeq() bool

	// CanSeq2 reports whether a [Value] with this type can be iterated over using [Value.Seq2].
	CanSeq2() bool

	common() *abi.Type
	uncommon() *uncommonType
}

func (t *rtype) OverflowComplex(x complex128) bool {
	k := t.Kind()
	switch k {
	case Complex64:
		return overflowFloat32(real(x)) || overflowFloat32(imag(x))
	case Complex128:
		return false
	}
	panic("reflect: OverflowComplex of non-complex type " + t.String())
}

func (t *rtype) OverflowFloat(x float64) bool {
	k := t.Kind()
	switch k {
	case Float32:
		return overflowFloat32(x)
	case Float64:
		return false
	}
	panic("reflect: OverflowFloat of non-float type " + t.String())
}

func (t *rtype) OverflowInt(x int64) bool {
	k := t.Kind()
	switch k {
	case Int, Int8, Int16, Int32, Int64:
		bitSize := t.Size() * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic("reflect: OverflowInt of non-int type " + t.String())
}

func (t *rtype) OverflowUint(x uint64) bool {
	k := t.Kind()
	switch k {
	case Uint, Uintptr, Uint8, Uint16, Uint32, Uint64:
		bitSize := t.Size() * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic("reflect: OverflowUint of non-uint type " + t.String())
}

func (t *rtype) CanSeq() bool {
	switch t.Kind() {
	case Int8, Int16, Int32, Int64, Int, Uint8, Uint16, Uint32, Uint64, Uint, Uintptr, Array, Slice, Chan, String, Map:
		return true
	case Func:
		return canRangeFunc(&t.t)
	case Pointer:
		return t.Elem().Kind() == Array
	}
	return false
}

func canRangeFunc(t *abi.Type) bool {
	if t.Kind() != abi.Func {
		return false
	}
	f := t.FuncType()
	if len(f.In) != 1 || len(f.Out) != 0 {
		return false
	}
	y := f.In[0]
	if y.Kind() != abi.Func {
		return false
	}
	yield := y.FuncType()
	return len(yield.In) == 1 && len(yield.Out) == 1 && yield.Out[0].Kind() == abi.Bool
}

func (t *rtype) CanSeq2() bool {
	switch t.Kind() {
	case Array, Slice, String, Map:
		return true
	case Func:
		return canRangeFunc2(&t.t)
	case Pointer:
		return t.Elem().Kind() == Array
	}
	return false
}

func canRangeFunc2(t *abi.Type) bool {
	if t.Kind() != abi.Func {
		return false
	}
	f := t.FuncType()
	if len(f.In) != 1 || len(f.Out) != 0 {
		return false
	}
	y := f.In[0]
	if y.Kind() != abi.Func {
		return false
	}
	yield := y.FuncType()
	return len(yield.In) == 2 && len(yield.Out) == 1 && yield.Out[0].Kind() == abi.Bool
}
