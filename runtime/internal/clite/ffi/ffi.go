package ffi

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

const (
	Void = iota
	Int
	Float
	Double
	LongDouble
	Uint8
	Sint8
	Uint16
	Sint16
	Uint32
	Sint32
	Uint64
	Sint64
	Struct
	Pointer
	Complex
)

const (
	OK = iota
	BAD_TYPEDEF
	BAD_ABI
	BAD_ARGTYPE
)

type Type struct {
	_         structs.HostLayout
	Size      SizeT
	Alignment uint16
	Type      uint16
	Elements  typeArray
}

/*typedef struct {
  ffi_abi abi;
  unsigned nargs;
  ffi_type **arg_types;
  ffi_type *rtype;
  unsigned bytes;
  unsigned flags;
#ifdef FFI_EXTRA_CIF_FIELDS
  FFI_EXTRA_CIF_FIELDS;
#endif
} ffi_cif;
*/

type Cif struct {
	_        structs.HostLayout
	Abi      c.Uint
	NArgs    c.Uint
	ArgTypes typeArray
	RType    typePointer
	Bytes    c.Uint
	Flags    c.Uint
	Extra    c.Uint
}

//llgo:cdecl
type ClosureFunc func(cif *Cif, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer)

// NewType describes the libffi storage type. The caller must retain typed roots
// for elements: the wasm32 address slots themselves are not Go GC pointers.
func NewType(size uintptr, alignment, kind uint16, elements **Type) Type {
	if uintptr(SizeT(size)) != size {
		panic("ffi: type size exceeds native size_t")
	}
	return Type{Size: SizeT(size), Alignment: alignment, Type: kind, Elements: storeTypeArray(elements)}
}

func (t *Type) ByteSize() uintptr { return uintptr(t.Size) }

// Element reads libffi's physical pointer array, which may use narrower slots
// than a Go []*Type. Use this instead of dereferencing Elements as a Go array.
func (t *Type) Element(i uintptr) *Type {
	if t == nil {
		return nil
	}
	elements := loadTypeArray(t.Elements)
	if elements == nil {
		return nil
	}
	return (*Type)(Index((*unsafe.Pointer)(unsafe.Pointer(elements)), i))
}

func (c *Cif) ReturnType() *Type { return loadTypePointer(c.RType) }
