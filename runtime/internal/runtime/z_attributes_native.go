//go:build !(wasm && llgo.wasm.gc.linear)

package runtime

import "unsafe"

// These typed alias declarations attach source contracts to the implementation
// symbols without changing their bodies. Linear-memory wasm GC publishes roots
// around calls, including conservative safepoints; those writes/captures are not
// covered by the contracts audited for the native/no-GC implementations. Keep
// that mode's declarations conservative until those effects can be represented.
// This uses the same build constraints and linkname attribute propagation as any
// other LLGo package; the compiler has no runtime-specific attribute policy.

//go:linkname attribute_Memhash github.com/xgo-dev/llgo/runtime/internal/runtime.Memhash
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read)
//llgo:attribute param(p) readonly captures(none)
func attribute_Memhash(p unsafe.Pointer, seed, size uintptr) uintptr

//go:linkname attribute_Memhash32 github.com/xgo-dev/llgo/runtime/internal/runtime.Memhash32
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read)
//llgo:attribute param(p) readonly captures(none)
func attribute_Memhash32(p unsafe.Pointer, seed uintptr) uintptr

//go:linkname attribute_Memhash64 github.com/xgo-dev/llgo/runtime/internal/runtime.Memhash64
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read)
//llgo:attribute param(p) readonly captures(none)
func attribute_Memhash64(p unsafe.Pointer, seed uintptr) uintptr

//go:linkname attribute_memequal github.com/xgo-dev/llgo/runtime/internal/runtime.memequal
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(argmem: read)
//llgo:attribute param(p) readonly captures(none)
//llgo:attribute param(q) readonly captures(none)
func attribute_memequal(p, q unsafe.Pointer, size uintptr) bool

//go:linkname attribute_Typedmemmove github.com/xgo-dev/llgo/runtime/internal/runtime.Typedmemmove
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(argmem: readwrite)
//llgo:attribute param(typ) readonly captures(none)
//llgo:attribute param(dst) writeonly captures(none)
//llgo:attribute param(src) readonly captures(none)
func attribute_Typedmemmove(typ *Type, dst, src unsafe.Pointer)

//go:linkname attribute_Typedmemclr github.com/xgo-dev/llgo/runtime/internal/runtime.Typedmemclr
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(argmem: readwrite)
//llgo:attribute param(typ) readonly captures(none)
//llgo:attribute param(ptr) writeonly captures(none)
func attribute_Typedmemclr(typ *Type, ptr unsafe.Pointer)

//go:linkname attribute_reflect_typedmemmove reflect.typedmemmove
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(argmem: readwrite)
//llgo:attribute param(t) readonly captures(none)
//llgo:attribute param(dst) writeonly captures(none)
//llgo:attribute param(src) readonly captures(none)
func attribute_reflect_typedmemmove(t *Type, dst, src unsafe.Pointer)

//go:linkname attribute_CStrCopy github.com/xgo-dev/llgo/runtime/internal/runtime.CStrCopy
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read, argmem: readwrite)
//llgo:attribute param(dest) returned writeonly captures(ret: address, provenance)
func attribute_CStrCopy(dest unsafe.Pointer, s String) *int8

//go:linkname attribute_StringEqual github.com/xgo-dev/llgo/runtime/internal/runtime.StringEqual
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read)
func attribute_StringEqual(x, y String) bool

//go:linkname attribute_StringLess github.com/xgo-dev/llgo/runtime/internal/runtime.StringLess
//llgo:attribute nofree nosync nounwind willreturn
//llgo:attribute memory(read)
func attribute_StringLess(x, y String) bool
