//go:build llgo

// LLGo extends package structs with explicit storage alignment markers.
//
// Place a zero-sized marker at the beginning of a struct:
//
//	type Block struct {
//		_ Align16
//		Data [24]byte
//	}
//
// With LLGo, Block has alignment 16, size 32, and Data at offset zero.
// Alignment propagates through value fields and arrays, including zero-length
// arrays. The struct's alignment is the maximum of its fields' requirements;
// individual data fields retain their own alignment. Markers in the middle or
// at the end obey ordinary field padding and zero-sized tail rules.
//
// Markers are comparable and preserve their meaning through aliases and defined
// types. They do not select host layout, a calling convention, SIMD operations,
// atomic access, or a device memory allocator. They may be combined with
// HostLayout. These markers are LLGo extensions and are not available in the
// standard library of other Go compilers.
//
// LLGo currently rejects map keys and values with explicit alignment above eight
// bytes and explicit alignment at native by-value call boundaries. Use pointers
// at these boundaries. Reflection's libffi calls also reject alignment above
// eight bytes; ordinary reflection allocation and field access are supported.
package structs

// Align8 gives a containing struct a minimum alignment of 8 bytes.
type Align8 struct{ _ align8 }

// Align16 gives a containing struct a minimum alignment of 16 bytes.
type Align16 struct{ _ align16 }

// Align32 gives a containing struct a minimum alignment of 32 bytes.
type Align32 struct{ _ align32 }

// Align64 gives a containing struct a minimum alignment of 64 bytes.
type Align64 struct{ _ align64 }

// The compiler recognizes these private identities. Keeping them in the public
// markers' underlying structs preserves alignment through new defined types.
type align8 struct{}
type align16 struct{}
type align32 struct{}
type align64 struct{}
