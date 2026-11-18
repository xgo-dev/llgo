package ffi

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/ffi"
)

type BasicKind int

const (
	Void BasicKind = iota // type is invalid

	// predeclared types
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	String
	UnsafePointer
	Interface
	Slice

	// aliases
	Byte = Uint8
	Rune = Int32
)

const (
	_64bit   = 1 << (^uintptr(0) >> 63) / 2
	_Int     = _64bit*ffi.Sint64 + (1-_64bit)*ffi.Sint32
	_Uint    = _64bit*ffi.Uint64 + (1-_64bit)*ffi.Uint32
	_sizei   = unsafe.Sizeof(0)
	_aligni  = uint16(unsafe.Alignof(0))
	_sizeci  = unsafe.Sizeof(c.Int(0))
	_alignci = uint16(unsafe.Alignof(c.Int(0)))
	_sizes   = unsafe.Sizeof("")
	_aligns  = uint16(unsafe.Alignof(""))
)

var (
	TypeVoid       = &Type{Size: 1, Alignment: 1, Type: ffi.Void}
	TypeBool       = &Type{Size: 1, Alignment: 1, Type: ffi.Uint8}
	TypeInt8       = &Type{Size: 1, Alignment: 1, Type: ffi.Sint8}
	TypeInt16      = &Type{Size: 2, Alignment: 2, Type: ffi.Sint16}
	TypeInt32      = &Type{Size: 4, Alignment: 4, Type: ffi.Sint32}
	TypeInt64      = &Type{Size: 8, Alignment: 8, Type: ffi.Sint64}
	TypeUint8      = &Type{Size: 1, Alignment: 1, Type: ffi.Uint8}
	TypeUint16     = &Type{Size: 2, Alignment: 2, Type: ffi.Uint16}
	TypeUint32     = &Type{Size: 4, Alignment: 4, Type: ffi.Uint32}
	TypeUint64     = &Type{Size: 8, Alignment: 8, Type: ffi.Uint64}
	TypeFloat32    = &Type{Size: 4, Alignment: 4, Type: ffi.Float}
	TypeFloat64    = &Type{Size: 8, Alignment: 8, Type: ffi.Double}
	TypeComplex64  = newComplexType(TypeFloat32, 8, 4)
	TypeComplex128 = newComplexType(TypeFloat64, 16, 8)
	TypeInt        = &Type{Size: ffi.SizeT(_sizei), Alignment: _aligni, Type: _Int}
	TypeUint       = &Type{Size: ffi.SizeT(_sizei), Alignment: _aligni, Type: _Uint}
	TypeUintptr    = &Type{Size: ffi.SizeT(_sizei), Alignment: _aligni, Type: _Uint}
	TypePointer    = &Type{Size: ffi.SizeT(_sizei), Alignment: _aligni, Type: ffi.Pointer}
	TypeString     = StructOf(TypePointer, TypeInt)
	TypeInterface  = StructOf(TypePointer, TypePointer)
	TypeSlice      = StructOf(TypePointer, TypeInt, TypeInt)
	empty          = [2]*Type{TypeInt8, nil}
	typeEmpty      = newAggregateType(0, 0, ffi.Struct, empty[:])
)

var Typ = []*Type{
	Void:          TypeVoid,
	Bool:          TypeBool,
	Int:           TypeInt,
	Int8:          TypeInt8,
	Int16:         TypeInt16,
	Int32:         TypeInt32,
	Int64:         TypeInt64,
	Uint:          TypeUint,
	Uint8:         TypeUint8,
	Uint16:        TypeUint16,
	Uint32:        TypeUint32,
	Uint64:        TypeUint64,
	Uintptr:       TypeUintptr,
	Float32:       TypeFloat32,
	Float64:       TypeFloat64,
	Complex64:     TypeComplex64,
	Complex128:    TypeComplex128,
	String:        TypeString,
	UnsafePointer: TypePointer,
	Interface:     TypeInterface,
	Slice:         TypeSlice,
}

func ArrayOf(elem *Type, N int) *Type {
	if N == 0 {
		return typeEmpty
	}
	fs := make([]*Type, N+1)
	for i := 0; i < N; i++ {
		fs[i] = elem
	}
	return newAggregateType(0, 0, ffi.Struct, fs)
}

func StructOf(fields ...*Type) *Type {
	if len(fields) == 0 {
		return typeEmpty
	}
	fs := make([]*Type, len(fields)+1)
	copy(fs, fields)
	return newAggregateType(0, 0, ffi.Struct, fs)
}
