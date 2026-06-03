//go:build go1.26
// +build go1.26

package reflect

import (
	"iter"

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
	Align() int
	FieldAlign() int
	Method(int) Method
	Methods() iter.Seq[Method]
	MethodByName(string) (Method, bool)
	NumMethod() int
	Name() string
	PkgPath() string
	Size() uintptr
	String() string
	Kind() Kind
	Implements(u Type) bool
	AssignableTo(u Type) bool
	ConvertibleTo(u Type) bool
	Comparable() bool
	Bits() int
	ChanDir() ChanDir
	IsVariadic() bool
	Elem() Type
	Field(i int) StructField
	Fields() iter.Seq[StructField]
	FieldByIndex(index []int) StructField
	FieldByName(name string) (StructField, bool)
	FieldByNameFunc(match func(string) bool) (StructField, bool)
	In(i int) Type
	Ins() iter.Seq[Type]
	Key() Type
	Len() int
	NumField() int
	NumIn() int
	NumOut() int
	Out(i int) Type
	Outs() iter.Seq[Type]
	OverflowComplex(x complex128) bool
	OverflowFloat(x float64) bool
	OverflowInt(x int64) bool
	OverflowUint(x uint64) bool
	CanSeq() bool
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

func (t *rtype) Methods() iter.Seq[Method] {
	return func(yield func(Method) bool) {
		for i := 0; i < t.NumMethod(); i++ {
			if !yield(t.Method(i)) {
				return
			}
		}
	}
}

func (t *rtype) Fields() iter.Seq[StructField] {
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField) bool) {
		for i := 0; i < t.NumField(); i++ {
			if !yield(t.Field(i)) {
				return
			}
		}
	}
}

func (t *rtype) Ins() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Ins of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := 0; i < t.NumIn(); i++ {
			if !yield(t.In(i)) {
				return
			}
		}
	}
}

func (t *rtype) Outs() iter.Seq[Type] {
	if t.Kind() != Func {
		panic("reflect: Outs of non-func type " + t.String())
	}
	return func(yield func(Type) bool) {
		for i := 0; i < t.NumOut(); i++ {
			if !yield(t.Out(i)) {
				return
			}
		}
	}
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
