//go:build 386 && go1.27

package atomic

import "unsafe"

// Own exactly the complete atomic_386.s group. The SDK's five existing Go
// definitions remain unchanged; uintptr operations use the actual 32-bit width.

//go:linkname atomicLoad32 llgo.atomicLoad
func atomicLoad32(*uint32) uint32

//go:linkname atomicLoad64 llgo.atomicLoad
func atomicLoad64(*uint64) uint64

//go:linkname atomicStore8 llgo.atomicStore
func atomicStore8(*uint8, uint8)

//go:linkname atomicStore32 llgo.atomicStore
func atomicStore32(*uint32, uint32)

//go:linkname atomicStore64 llgo.atomicStore
func atomicStore64(*uint64, uint64)

//go:linkname atomicStorePointer llgo.atomicStore
func atomicStorePointer(*unsafe.Pointer, unsafe.Pointer)

//go:linkname atomicCAS32 llgo.atomicCmpXchgOK
func atomicCAS32(*uint32, uint32, uint32) bool

//go:linkname atomicCAS64 llgo.atomicCmpXchgOK
func atomicCAS64(*uint64, uint64, uint64) bool

//go:linkname atomicCASPointer llgo.atomicCmpXchgOK
func atomicCASPointer(*unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) bool

//go:linkname atomicXchg8 llgo.atomicXchg
func atomicXchg8(*uint8, uint8) uint8

//go:linkname atomicXchg32 llgo.atomicXchg
func atomicXchg32(*uint32, uint32) uint32

//go:linkname atomicXchg64 llgo.atomicXchg
func atomicXchg64(*uint64, uint64) uint64

//go:linkname atomicAdd32 llgo.atomicAddReturnNew
func atomicAdd32(*uint32, uint32) uint32

//go:linkname atomicAdd64 llgo.atomicAddReturnNew
func atomicAdd64(*uint64, uint64) uint64

//go:linkname atomicAnd8 llgo.atomicAnd
func atomicAnd8(*uint8, uint8) uint8

//go:linkname atomicAnd32 llgo.atomicAnd
func atomicAnd32(*uint32, uint32) uint32

//go:linkname atomicAnd64 llgo.atomicAnd
func atomicAnd64(*uint64, uint64) uint64

//go:linkname atomicOr8 llgo.atomicOr
func atomicOr8(*uint8, uint8) uint8

//go:linkname atomicOr32 llgo.atomicOr
func atomicOr32(*uint32, uint32) uint32

//go:linkname atomicOr64 llgo.atomicOr
func atomicOr64(*uint64, uint64) uint64

func atomicCheckAlign64(ptr unsafe.Pointer) {
	if uintptr(ptr)&7 != 0 {
		panicUnaligned()
	}
}

func Load64(ptr *uint64) uint64 {
	atomicCheckAlign64(unsafe.Pointer(ptr))
	return atomicLoad64(ptr)
}

func Loadint32(ptr *int32) int32 {
	return int32(atomicLoad32((*uint32)(unsafe.Pointer(ptr))))
}

func Loadint64(ptr *int64) int64 {
	return int64(Load64((*uint64)(unsafe.Pointer(ptr))))
}

func Loaduintptr(ptr *uintptr) uintptr {
	return uintptr(atomicLoad32((*uint32)(unsafe.Pointer(ptr))))
}

func Loaduint(ptr *uint) uint {
	return uint(atomicLoad32((*uint32)(unsafe.Pointer(ptr))))
}

func Store8(ptr *uint8, value uint8) {
	atomicStore8(ptr, value)
}

func Store(ptr *uint32, value uint32) {
	atomicStore32(ptr, value)
}

func StoreRel(ptr *uint32, value uint32) {
	Store(ptr, value)
}

func Store64(ptr *uint64, value uint64) {
	atomicCheckAlign64(unsafe.Pointer(ptr))
	atomicStore64(ptr, value)
}

func Storeint32(ptr *int32, value int32) {
	atomicStore32((*uint32)(unsafe.Pointer(ptr)), uint32(value))
}

func Storeint64(ptr *int64, value int64) {
	Store64((*uint64)(unsafe.Pointer(ptr)), uint64(value))
}

func Storeuintptr(ptr *uintptr, value uintptr) {
	atomicStore32((*uint32)(unsafe.Pointer(ptr)), uint32(value))
}

func StoreReluintptr(ptr *uintptr, value uintptr) {
	Storeuintptr(ptr, value)
}

func StorepNoWB(ptr unsafe.Pointer, value unsafe.Pointer) {
	atomicStorePointer((*unsafe.Pointer)(ptr), value)
}

func Cas(ptr *uint32, old, next uint32) bool {
	return atomicCAS32(ptr, old, next)
}

func CasRel(ptr *uint32, old, next uint32) bool {
	return Cas(ptr, old, next)
}

func Cas64(ptr *uint64, old, next uint64) bool {
	atomicCheckAlign64(unsafe.Pointer(ptr))
	return atomicCAS64(ptr, old, next)
}

func Casint32(ptr *int32, old, next int32) bool {
	return atomicCAS32((*uint32)(unsafe.Pointer(ptr)), uint32(old), uint32(next))
}

func Casint64(ptr *int64, old, next int64) bool {
	return Cas64((*uint64)(unsafe.Pointer(ptr)), uint64(old), uint64(next))
}

func Casuintptr(ptr *uintptr, old, next uintptr) bool {
	return atomicCAS32((*uint32)(unsafe.Pointer(ptr)), uint32(old), uint32(next))
}

func Casp1(ptr *unsafe.Pointer, old, next unsafe.Pointer) bool {
	return atomicCASPointer(ptr, old, next)
}

func Xadd(ptr *uint32, delta int32) uint32 {
	return atomicAdd32(ptr, uint32(delta))
}

func Xadd64(ptr *uint64, delta int64) uint64 {
	atomicCheckAlign64(unsafe.Pointer(ptr))
	return atomicAdd64(ptr, uint64(delta))
}

func Xaddint32(ptr *int32, delta int32) int32 {
	return int32(atomicAdd32((*uint32)(unsafe.Pointer(ptr)), uint32(delta)))
}

func Xaddint64(ptr *int64, delta int64) int64 {
	return int64(Xadd64((*uint64)(unsafe.Pointer(ptr)), delta))
}

func Xadduintptr(ptr *uintptr, delta uintptr) uintptr {
	return uintptr(atomicAdd32((*uint32)(unsafe.Pointer(ptr)), uint32(delta)))
}

func Xchg8(ptr *uint8, next uint8) uint8 {
	return atomicXchg8(ptr, next)
}

func Xchg(ptr *uint32, next uint32) uint32 {
	return atomicXchg32(ptr, next)
}

func Xchg64(ptr *uint64, next uint64) uint64 {
	atomicCheckAlign64(unsafe.Pointer(ptr))
	return atomicXchg64(ptr, next)
}

func Xchgint32(ptr *int32, next int32) int32 {
	return int32(atomicXchg32((*uint32)(unsafe.Pointer(ptr)), uint32(next)))
}

func Xchgint64(ptr *int64, next int64) int64 {
	return int64(Xchg64((*uint64)(unsafe.Pointer(ptr)), uint64(next)))
}

func Xchguintptr(ptr *uintptr, next uintptr) uintptr {
	return uintptr(atomicXchg32((*uint32)(unsafe.Pointer(ptr)), uint32(next)))
}

func And8(ptr *uint8, value uint8) {
	atomicAnd8(ptr, value)
}

func Or8(ptr *uint8, value uint8) {
	atomicOr8(ptr, value)
}

func And(ptr *uint32, value uint32) {
	atomicAnd32(ptr, value)
}

func Or(ptr *uint32, value uint32) {
	atomicOr32(ptr, value)
}

func And32(ptr *uint32, value uint32) uint32 {
	return atomicAnd32(ptr, value)
}

func Or32(ptr *uint32, value uint32) uint32 {
	return atomicOr32(ptr, value)
}

func And64(ptr *uint64, value uint64) uint64 {
	return atomicAnd64(ptr, value)
}

func Or64(ptr *uint64, value uint64) uint64 {
	return atomicOr64(ptr, value)
}

func Anduintptr(ptr *uintptr, value uintptr) uintptr {
	return uintptr(atomicAnd32((*uint32)(unsafe.Pointer(ptr)), uint32(value)))
}

func Oruintptr(ptr *uintptr, value uintptr) uintptr {
	return uintptr(atomicOr32((*uint32)(unsafe.Pointer(ptr)), uint32(value)))
}
