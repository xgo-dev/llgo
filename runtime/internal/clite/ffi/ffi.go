package ffi

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
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
	Size      uintptr
	Alignment uint16
	Type      uint16
	Elements  **Type
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
	Abi      c.Uint
	NArgs    c.Uint
	ArgTypes **Type
	RType    *Type
	Bytes    c.Uint
	Flags    c.Uint
	Extra    c.Uint
}

//llgo:type C
type ClosureFunc func(cif *Cif, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer)

func add(ptr unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) + offset)
}

func Index(args *unsafe.Pointer, i uintptr) unsafe.Pointer {
	return (*(*unsafe.Pointer)(add(unsafe.Pointer(args), i*unsafe.Sizeof(0))))
}
