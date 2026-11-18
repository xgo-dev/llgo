package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

// Aliases and defined marker types retain this exact private field type.
// Cache its canonical descriptor once instead of decoding names for each
// StructOf candidate. The private identity cannot be recreated by user types.
var hostLayoutPrivateMarker = abi.TypeFor[structs.HostLayout]().StructType().Fields[0].Typ

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
	return f.Name_ == "_" && f.Typ == hostLayoutPrivateMarker
}

// structHasHostLayout reports whether typ's compiler layout includes a
// HostLayout marker. Direct markers do not propagate as a layout policy, but
// enclosing structs and arrays inherit the marked field's size, including
// trailing zero-size protection. Match ssa.containsHostLayout so FFI tail
// padding follows the Size_ the compiler actually produced.
func structHasHostLayout(typ *abi.Type) bool {
	if typ == nil {
		return false
	}
	switch Kind(typ.Kind()) {
	case Array:
		return structHasHostLayout(typ.Elem())
	case Struct:
		st := typ.StructType()
		if st == nil {
			return false
		}
		for i := range st.Fields {
			ft := st.Fields[i].Typ
			if isHostLayoutMarker(toType(ft)) || structHasHostLayout(ft) {
				return true
			}
		}
	}
	return false
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
