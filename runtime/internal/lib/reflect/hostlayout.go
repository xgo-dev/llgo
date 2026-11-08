package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

// A defined marker type retains HostLayout's private field identity. A struct
// containing a marked record, or an array of markers, does not inherit it.
func isHostLayoutMarker(t Type) bool {
	st := t.common().StructType()
	if st == nil || len(st.Fields) != 1 {
		return false
	}
	// The public Field method allocates its Index slice. Inspect the immutable
	// descriptor directly so the StructOf pre-scan does not allocate.
	f := &st.Fields[0]
	body := f.Typ.StructType()
	if f.Name_ != "_" || body == nil || len(body.Fields) != 0 {
		return false
	}
	marker := toRType(f.Typ)
	if marker.Name() != "hostLayout" {
		return false
	}
	pkg := marker.PkgPath()
	return pkg == "structs" || pkg == "github.com/xgo-dev/llgo/runtime/internal/structs"
}

// Ask the compiler for native scalar field alignment without changing the Go
// scalar's own alignment or width. These can differ on ARM and Windows/386.
func hostLayoutFieldAlign(t *abi.Type) uint8 {
	switch Kind(t.Kind()) {
	case Array:
		return hostLayoutFieldAlign(t.Elem())
	case Int, Uint:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V int
		}{}))
	case Int16, Uint16:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V int16
		}{}))
	case Int32, Uint32:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V int32
		}{}))
	case Int64, Uint64:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V int64
		}{}))
	case Float32:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V float32
		}{}))
	case Float64:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V float64
		}{}))
	case Complex64:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V complex64
		}{}))
	case Complex128:
		return uint8(unsafe.Alignof(struct {
			_ structs.HostLayout
			V complex128
		}{}))
	}
	// Nested structs own their layout. Pointers, uintptrs, callbacks and other
	// Go values retain their existing storage width and alignment.
	// In particular, do not reconstruct a nested struct from its fields here:
	// fixing ordinary ARM aggregate metadata belongs to #2810, not to applying
	// a parent's marker to a different Go type.
	return t.Align_
}
