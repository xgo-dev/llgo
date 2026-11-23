Type Mapping between C and Go
=====

| C type | Go Type | Note |
| ---- | ---- | ---- |
| char | c.Char | int8 |
| short | int16 | |
| int | c.Int | |
| long | c.Long | |
| long long | int64 | c.LongLong |
| unsigned char | byte | uint8 |
| unsigned short | uint16 | |
| unsigned int | c.Uint | |
| unsigned long | c.Ulong | |
| unsigned long long | uint64 | c.UlongLong |
| size_t | uintptr | |
| ssize_t | int | sizeof(int) = sizeof(uint) = sizeof(uintptr) in Go |
| float | float32 | c.Float |
| double | float64 | c.Double |
| complex float | complex64 |
| complex double | complex128 |
| void* | c.Pointer | |
| T* | *T | |
| T[] (in parameter) | *T |  |
| T[N] (in parameter) | *T | fake array |
| T[N] (in struct) | [N]T | array |
| func ptr (in parameter) | func | LLGo knows `func ptr` in C function parameters is `func ptr` not `closure` |
| func ptr (in struct) | c.Pointer | C `func ptr` not Go `func()` |
| struct | struct | |

## Reflection of native callback slices

A slice obtained with `reflect.Value.Slice` or `Slice3` from a native callback
array retains the array's element stride while used through reflection. That
stride can differ from an ordinary Go slice of the same callback type. For
example, on WASI Memory32, callback slots in a `//llgo:cdecl` record or
array occupy four bytes, while Go slice elements occupy eight bytes.

When the strides differ, `Value.Interface`, `reflect.TypeAssert` (Go 1.25 and
later), and `Value.Set` into a Go slice create a detached copy in Go storage.
The copy preserves the slice's length and capacity, including the elements
accessible by reslicing up to that capacity. Changes to the resulting Go slice
do not affect the native record or array, and changes to the original do not
affect the copy. When the strides match, these operations retain the shared
backing array, so writes remain visible in both directions.
