# Explicit alignment in structs (LLGo extension)

`Align8`, `Align16`, `Align32`, and `Align64` are zero-sized, comparable markers.
Their numbers are **bytes**. Use a marker as the first field of a struct:

```go
import "structs"

type Block struct {
    _    structs.Align16
    Data [24]byte
}
```

LLGo gives `Block` alignment 16, size 32, and `Data` offset 0. Arrays and slice
elements have a 32-byte stride. The requirement is preserved in stack and global
storage, heap allocations, slice growth, interface boxes, channel buffers, and
reflection allocations. Aliases and types defined from `Block` or a marker keep
the requirement. Anonymous embedding of a marker is also supported.

Alignment is the maximum of the fields' requirements. Multiple markers therefore
select the strongest requirement. An aligned value field raises its containing
struct's alignment; a pointer or slice referring to aligned values does not.
Markers can introduce ordinary padding when placed between fields. A marker at
the end also follows Go's zero-sized tail-field rule. Prefer putting them first.

## Individual fields

Aligning a struct does not align every field to the same boundary. On a target
where `uint64` fields need only four-byte alignment, the `Count` below can still
have offset 4:

```go
type Record struct {
    _     structs.Align8
    Flag  uint32
    Count uint64
}
```

Wrap the field when it needs its own stronger requirement:

```go
type AlignedUint64 struct {
    _     structs.Align8
    Value uint64
}

type Record struct {
    Flag  uint32
    Count AlignedUint64
}
```

This guarantees that `Count.Value` is eight-byte aligned. It does not make
accesses atomic; use the appropriate atomic operations separately.

## Host layout and native interfaces

Combine a marker with `structs.HostLayout` when matching an explicitly aligned
C record. The properties are independent: `HostLayout` selects field layout,
while the alignment marker raises the record's storage requirement. Neither
changes a Go closure into a C callback.

```go
type Descriptor struct {
    _       structs.HostLayout
    _       structs.Align64
    Address uint64
    Length  uint32
}
```

Pass pointers to these records to native functions. The current implementation
rejects explicit alignment at native by-value parameter and return boundaries,
including callbacks. Native callers supplying pointers must provide storage
with the declared layout and alignment. The marker does not select a device
memory allocator or guarantee a particular cache-line size.

## Current limits

- Map keys and values with explicit alignment above eight bytes are rejected,
  including enclosing structs and zero-length arrays. Pointers remain supported.
- Reflective calls through libffi reject types with alignment above eight bytes.
  Basic reflection, including `New`, `StructOf`, `MakeSlice`, and `MakeChan`,
  supports aligned storage.
- These markers extend the standard-library `structs` package in LLGo. Other
  Go compilers do not provide `structs.Align8/16/32/64`. Source using these names
  requires LLGo (or separate implementation files selected by build tags).

No marker reduces natural alignment, packs fields, changes comparability, or
turns a scalar aggregate into a SIMD vector.
