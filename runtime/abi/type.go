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

package abi

import (
	"unsafe"
)

// IsExported reports whether name starts with an upper-case letter.
func IsExported(name string) bool {
	if len(name) > 0 {
		c := name[0]
		return 'A' <= c && c <= 'Z'
	}
	return false
}

// -----------------------------------------------------------------------------

// Type is the runtime representation of a Go type.
//
// Type is also referenced implicitly
// (in the form of expressions involving constants and arch.PtrSize)
// in cmd/compile/internal/reflectdata/reflect.go
// and cmd/link/internal/ld/decodesym.go
// (e.g. data[2*arch.PtrSize+4] references the TFlag field)
// unsafe.OffsetOf(Type{}.TFlag) cannot be used directly in those
// places because it varies with cross compilation and experiments.
type Type struct {
	Size_       uintptr
	PtrBytes    uintptr // number of (prefix) bytes in the type that can contain pointers
	Hash        uint32  // hash of type; avoids computation in hash tables
	TFlag       TFlag   // extra type information flags
	Align_      uint8   // alignment of variable with this type
	FieldAlign_ uint8   // alignment of struct field with this type
	Kind_       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
	// GCData stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, GCData is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	GCData     *byte
	Str_       string // string form
	PtrToThis_ *Type  // type for pointer to this type, may be nil
}

// A Kind represents the specific kind of type that a Type represents.
// The zero Kind is not a valid kind.
type Kind uint

const (
	Invalid Kind = iota
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
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

const (
	// TODO (khr, drchase) why aren't these in TFlag?  Investigate, fix if possible.
	KindDirectIface = 1 << 5
	KindGCProg      = 1 << 6 // Type.gc points to GC program
	KindMask        = (1 << 5) - 1
)

// String returns the name of k.
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return kindNames[0]
}

var kindNames = []string{
	Invalid:       "invalid",
	Bool:          "bool",
	Int:           "int",
	Int8:          "int8",
	Int16:         "int16",
	Int32:         "int32",
	Int64:         "int64",
	Uint:          "uint",
	Uint8:         "uint8",
	Uint16:        "uint16",
	Uint32:        "uint32",
	Uint64:        "uint64",
	Uintptr:       "uintptr",
	Float32:       "float32",
	Float64:       "float64",
	Complex64:     "complex64",
	Complex128:    "complex128",
	Array:         "array",
	Chan:          "chan",
	Func:          "func",
	Interface:     "interface",
	Map:           "map",
	Pointer:       "ptr",
	Slice:         "slice",
	String:        "string",
	Struct:        "struct",
	UnsafePointer: "unsafe.Pointer",
}

// TFlag is used by a Type to signal what extra type information is
// available in the memory directly following the Type value.
type TFlag uint8

const (
	// TFlagUncommon means that there is a data with a type, UncommonType,
	// just beyond the shared-per-type common data.  That is, the data
	// for struct types will store their UncommonType at one offset, the
	// data for interface types will store their UncommonType at a different
	// offset.  UncommonType is always accessed via a pointer that is computed
	// using trust-us-we-are-the-implementors pointer arithmetic.
	//
	// For example, if t.Kind() == Struct and t.tflag&TFlagUncommon != 0,
	// then t has UncommonType data and it can be accessed as:
	//
	//	type structTypeUncommon struct {
	//		structType
	//		u UncommonType
	//	}
	//	u := &(*structTypeUncommon)(unsafe.Pointer(t)).u
	TFlagUncommon TFlag = 1 << 0

	// TFlagExtraStar means the name in the str field has an
	// extraneous '*' prefix. This is because for most types T in
	// a program, the type *T also exists and reusing the str data
	// saves binary size.
	TFlagExtraStar TFlag = 1 << 1

	// TFlagNamed means the type has a name.
	TFlagNamed TFlag = 1 << 2

	// TFlagRegularMemory means that equal and hash functions can treat
	// this type as a single region of t.size bytes.
	TFlagRegularMemory TFlag = 1 << 3

	// TFlagVariadic means a funcType with variadic parameters
	TFlagVariadic TFlag = 1 << 4

	// TFflagClosure means the structType is a closure
	TFlagClosure TFlag = 1 << 5

	// TFlagGCMaskOnDemand means that the GC pointer bitmask will be
	// computed on demand at runtime instead of being precomputed at
	// compile time. If this flag is set, the GCData field effectively
	// has type **byte instead of *byte. The runtime will store a
	// pointer to the GC pointer bitmask in *GCData.
	TFlagGCMaskOnDemand TFlag = 1 << 7
)

// -----------------------------------------------------------------------------

// ArrayType represents a fixed array type.
type ArrayType struct {
	Type
	Elem  *Type // array element type
	Slice *Type // slice type
	Len   uintptr
}

type SliceType struct {
	Type
	Elem *Type // slice element type
}

func (t *Type) Key() *Type {
	if t.Kind() == Map {
		return (*MapType)(unsafe.Pointer(t)).Key
	}
	return nil
}

type PtrType struct {
	Type
	Elem *Type // pointer element (pointed at) type
}

type ChanDir int

const (
	RecvDir    ChanDir = 1 << iota         // <-chan
	SendDir                                // chan<-
	BothDir            = RecvDir | SendDir // chan
	InvalidDir ChanDir = 0
)

// ChanType represents a channel type
type ChanType struct {
	Type
	Elem *Type
	Dir  ChanDir
}

// funcType represents a function type.
type FuncType struct {
	Type
	In  []*Type
	Out []*Type
}

// Variadic reports whether the function type is variadic.
func (p *FuncType) Variadic() bool {
	return p.TFlag&TFlagVariadic != 0
}

type StructField struct {
	Name_  string  // name is always non-empty
	Typ    *Type   // type of field
	Offset uintptr // byte offset of field

	Tag_      string
	Embedded_ bool
}

// Embedded reports whether the field is embedded.
func (f *StructField) Embedded() bool {
	return f.Embedded_
}

// Exported reports whether the field is exported.
func (f *StructField) Exported() bool {
	return IsExported(f.Name_)
}

type StructType struct {
	Type
	PkgPath_ string
	Fields   []StructField
}

type InterfaceType struct {
	Type
	PkgPath_ string    // import path
	Methods  []Imethod // sorted by hash
}

type Text = unsafe.Pointer // TODO(xsw): to be confirmed

// Method on non-interface type
type Method struct {
	Name_ string    // name of method
	Mtyp_ *FuncType // method type (without receiver)
	Ifn_  Text      // fn used in interface call (one-word receiver)
	Tfn_  Text      // fn used for normal method call
}

// Exported reports whether the method is exported.
func (p *Method) Exported() bool {
	return lastDot(p.Name_) == -1
}

// Name returns the tag string for method.
func (p *Method) Name() string {
	_, name := splitName(p.Name_)
	return name
}

// PkgPath returns the pkgpath string for method, or empty if there is none.
func (p *Method) PkgPath() string {
	pkg, _ := splitName(p.Name_)
	return pkg
}

// UncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type UncommonType struct {
	PkgPath_ string // import path; empty for built-in types like int, string
	Mcount   uint16 // number of methods
	Xcount   uint16 // number of exported methods
	Moff     uint32 // offset from this uncommontype to [mcount]Method
}

func (t *UncommonType) Methods() []Method {
	if t.Mcount == 0 {
		return nil
	}
	methodsPtr := addChecked(unsafe.Pointer(t), uintptr(t.Moff), "t.mcount > 0")
	methods := make([]Method, t.Mcount)
	for i := 0; i < int(t.Mcount); i++ {
		elemPtr := addChecked(methodsPtr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
		elem := (*Method)(elemPtr)
		methods[i] = *elem
	}
	return methods
}

func (t *UncommonType) ExportedMethods() []Method {
	if t.Xcount == 0 {
		return nil
	}
	mthdsPtr := addChecked(unsafe.Pointer(t), uintptr(t.Moff), "t.xcount > 0")
	mthds := make([]Method, t.Xcount)
	for i := 0; i < int(t.Xcount); i++ {
		elemPtr := addChecked(mthdsPtr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
		elem := (*Method)(elemPtr)
		mthds[i] = *elem
	}
	return mthds
}

// Imethod represents a method on an interface type
type Imethod struct {
	Name_ string    // name of method
	Typ_  *FuncType // .(*FuncType) underneath
}

// Exported reports whether the imethod is exported.
func (p *Imethod) Exported() bool {
	return lastDot(p.Name_) == -1
}

// Name returns the tag string for imethod.
func (p *Imethod) Name() string {
	_, name := splitName(p.Name_)
	return name
}

// PkgPath returns the pkgpath string for imethod, or empty if there is none.
func (p *Imethod) PkgPath() string {
	pkg, _ := splitName(p.Name_)
	return pkg
}

func (t *Type) Kind() Kind { return Kind(t.Kind_ & KindMask) }

func (t *Type) HasName() bool {
	return t.TFlag&TFlagNamed != 0
}

func (t *Type) Pointers() bool { return t.PtrBytes != 0 }

// IfaceIndir reports whether t is stored indirectly in an interface value.
func (t *Type) IfaceIndir() bool {
	return t.Kind_&KindDirectIface == 0
}

// IsDirectIface reports whether t is stored directly in an interface value.
func (t *Type) IsDirectIface() bool {
	return t.Kind_&KindDirectIface != 0
}

// IsClosure reports whether t is closure struct
func (t *Type) IsClosure() bool {
	return t.TFlag&TFlagClosure != 0
}

// Size returns the size of data with type t.
func (t *Type) Size() uintptr { return t.Size_ }

// Align returns the alignment of data with type t.
func (t *Type) Align() int { return int(t.Align_) }

func (t *Type) FieldAlign() int { return int(t.FieldAlign_) }

// String returns string form of type t.
func (t *Type) String() string {
	if t.TFlag&TFlagExtraStar != 0 {
		return "*" + t.Str_ // TODO(xsw): misunderstand
	}
	return t.Str_
}

func (t *Type) Common() *Type {
	return t
}

type structTypeUncommon struct {
	StructType
	u UncommonType
}

// ChanDir returns the direction of t if t is a channel type, otherwise InvalidDir (0).
func (t *Type) ChanDir() ChanDir {
	if t.Kind() == Chan {
		ch := (*ChanType)(unsafe.Pointer(t))
		return ch.Dir
	}
	return InvalidDir
}

// Uncommon returns a pointer to T's "uncommon" data if there is any, otherwise nil
func (t *Type) Uncommon() *UncommonType {
	if t.TFlag&TFlagUncommon == 0 {
		return nil
	}
	switch t.Kind() {
	case Struct:
		return &(*structTypeUncommon)(unsafe.Pointer(t)).u
	case Pointer:
		type u struct {
			PtrType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Func:
		type u struct {
			FuncType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Slice:
		type u struct {
			SliceType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Array:
		type u struct {
			ArrayType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Chan:
		type u struct {
			ChanType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Map:
		type u struct {
			MapType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Interface:
		type u struct {
			InterfaceType
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	default:
		type u struct {
			Type
			u UncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	}
}

// Len returns the length of t if t is an array type, otherwise 0
func (t *Type) Len() int {
	if t.Kind() == Array {
		return int((*ArrayType)(unsafe.Pointer(t)).Len)
	}
	return 0
}

// Elem returns the element type for t if t is an array, channel, map, pointer, or slice, otherwise nil.
func (t *Type) Elem() *Type {
	switch t.Kind() {
	case Pointer:
		tt := (*PtrType)(unsafe.Pointer(t))
		return tt.Elem
	case Slice:
		tt := (*SliceType)(unsafe.Pointer(t))
		return tt.Elem
	case Map:
		tt := (*MapType)(unsafe.Pointer(t))
		return tt.Elem
	case Array:
		tt := (*ArrayType)(unsafe.Pointer(t))
		return tt.Elem
	case Chan:
		tt := (*ChanType)(unsafe.Pointer(t))
		return tt.Elem
	}
	return nil
}

// StructType returns t cast to a *StructType, or nil if its tag does not match.
func (t *Type) StructType() *StructType {
	if t.Kind() != Struct {
		return nil
	}
	return (*StructType)(unsafe.Pointer(t))
}

// MapType returns t cast to a *MapType, or nil if its tag does not match.
func (t *Type) MapType() *MapType {
	if t.Kind() != Map {
		return nil
	}
	return (*MapType)(unsafe.Pointer(t))
}

// ArrayType returns t cast to a *ArrayType, or nil if its tag does not match.
func (t *Type) ArrayType() *ArrayType {
	if t.Kind() != Array {
		return nil
	}
	return (*ArrayType)(unsafe.Pointer(t))
}

// FuncType returns t cast to a *FuncType, or nil if its tag does not match.
func (t *Type) FuncType() *FuncType {
	if t.Kind() != Func {
		return nil
	}
	return (*FuncType)(unsafe.Pointer(t))
}

// InterfaceType returns t cast to a *InterfaceType, or nil if its tag does not match.
func (t *Type) InterfaceType() *InterfaceType {
	if t.Kind() != Interface {
		return nil
	}
	return (*InterfaceType)(unsafe.Pointer(t))
}

func (t *Type) ExportedMethods() []Method {
	ut := t.Uncommon()
	if ut == nil {
		return nil
	}
	return ut.ExportedMethods()
}

func (t *Type) NumMethod() int {
	if t.Kind() == Interface {
		tt := (*InterfaceType)(unsafe.Pointer(t))
		return len(tt.Methods)
	}
	return len(t.ExportedMethods())
}

func (t *Type) GcSlice(begin, end uintptr) []byte {
	return unsafe.Slice(t.GCData, int(end))[begin:]
}

// -----------------------------------------------------------------------------

// addChecked returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func addChecked(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	_ = whySafe
	return unsafe.Pointer(uintptr(p) + x)
}

func splitName(s string) (pkg string, name string) {
	i := lastDot(s)
	if i == -1 {
		return "", s
	}
	return s[:i], s[i+1:]
}

func lastDot(s string) int {
	i := len(s) - 1
	for i >= 0 && s[i] != '.' {
		i--
	}
	return i
}

// MaxPtrmaskBytes is the maximum length of a GC ptrmask bitmap,
// which holds 1-bit entries describing where pointers are in a given type.
// Above this length, the GC information is recorded as a GC program,
// which can express repetition compactly. In either form, the
// information is used by the runtime to initialize the heap bitmap,
// and for large types (like 128 or more words), they are roughly the
// same speed. GC programs are never much larger and often more
// compact. (If large arrays are involved, they can be arbitrarily
// more compact.)
//
// The cutoff must be large enough that any allocation large enough to
// use a GC program is large enough that it does not share heap bitmap
// bytes with any other objects, allowing the GC program execution to
// assume an aligned start and not use atomic operations. In the current
// runtime, this means all malloc size classes larger than the cutoff must
// be multiples of four words. On 32-bit systems that's 16 bytes, and
// all size classes >= 16 bytes are 16-byte aligned, so no real constraint.
// On 64-bit systems, that's 32 bytes, and 32-byte alignment is guaranteed
// for size classes >= 256 bytes. On a 64-bit system, 256 bytes allocated
// is 32 pointers, the bits for which fit in 4 bytes. So MaxPtrmaskBytes
// must be >= 4.
//
// We used to use 16 because the GC programs do have some constant overhead
// to get started, and processing 128 pointers seems to be enough to
// amortize that overhead well.
//
// To make sure that the runtime's chansend can call typeBitsBulkBarrier,
// we raised the limit to 2048, so that even 32-bit systems are guaranteed to
// use bitmaps for objects up to 64 kB in size.
const MaxPtrmaskBytes = 2048

// -----------------------------------------------------------------------------
