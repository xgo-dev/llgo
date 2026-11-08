package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

// A defined marker type retains HostLayout's private field identity. A struct
// containing a marked record, or an array of markers, does not inherit it.
func isHostLayoutMarker(t Type) bool {
	if t.Kind() != Struct || t.NumField() != 1 {
		return false
	}
	f := t.Field(0)
	if f.Name != "_" || f.Type.Kind() != Struct || f.Type.NumField() != 0 || f.Type.Name() != "hostLayout" {
		return false
	}
	pkg := f.Type.PkgPath()
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
	return t.Align_
}
