//go:build arm && baremetal

package atomic

import "unsafe"

type Int32 struct {
	noCopy noCopy
	value  int32
}

//go:nosplit
func (i *Int32) Load() int32 { return Loadint32(&i.value) }

//go:nosplit
func (i *Int32) Store(value int32) { Storeint32(&i.value, value) }

//go:nosplit
func (i *Int32) CompareAndSwap(old, new int32) bool { return Casint32(&i.value, old, new) }

//go:nosplit
func (i *Int32) Swap(new int32) int32 { return Xchgint32(&i.value, new) }

//go:nosplit
func (i *Int32) Add(delta int32) int32 { return Xaddint32(&i.value, delta) }

type Int64 struct {
	noCopy noCopy
	_      align64
	value  int64
}

//go:nosplit
func (i *Int64) Load() int64 { return Loadint64(&i.value) }

//go:nosplit
func (i *Int64) Store(value int64) { Storeint64(&i.value, value) }

//go:nosplit
func (i *Int64) CompareAndSwap(old, new int64) bool { return Casint64(&i.value, old, new) }

//go:nosplit
func (i *Int64) Swap(new int64) int64 { return Xchgint64(&i.value, new) }

//go:nosplit
func (i *Int64) Add(delta int64) int64 { return Xaddint64(&i.value, delta) }

type Uint8 struct {
	noCopy noCopy
	value  uint8
}

//go:nosplit
func (u *Uint8) Load() uint8 { return Load8(&u.value) }

//go:nosplit
func (u *Uint8) Store(value uint8) { Store8(&u.value, value) }

//go:nosplit
func (u *Uint8) And(value uint8) { And8(&u.value, value) }

//go:nosplit
func (u *Uint8) Or(value uint8) { Or8(&u.value, value) }

type Bool struct {
	u Uint8
}

//go:nosplit
func (b *Bool) Load() bool { return b.u.Load() != 0 }

//go:nosplit
func (b *Bool) Store(value bool) {
	s := uint8(0)
	if value {
		s = 1
	}
	b.u.Store(s)
}

type Uint32 struct {
	noCopy noCopy
	value  uint32
}

//go:nosplit
func (u *Uint32) Load() uint32 { return Load(&u.value) }

//go:nosplit
func (u *Uint32) LoadAcquire() uint32 { return LoadAcq(&u.value) }

//go:nosplit
func (u *Uint32) Store(value uint32) { Store(&u.value, value) }

//go:nosplit
func (u *Uint32) StoreRelease(value uint32) { StoreRel(&u.value, value) }

//go:nosplit
func (u *Uint32) CompareAndSwap(old, new uint32) bool { return Cas(&u.value, old, new) }

//go:nosplit
func (u *Uint32) CompareAndSwapRelease(old, new uint32) bool { return CasRel(&u.value, old, new) }

//go:nosplit
func (u *Uint32) Swap(value uint32) uint32 { return Xchg(&u.value, value) }

//go:nosplit
func (u *Uint32) And(value uint32) { And(&u.value, value) }

//go:nosplit
func (u *Uint32) Or(value uint32) { Or(&u.value, value) }

//go:nosplit
func (u *Uint32) Add(delta int32) uint32 { return Xadd(&u.value, delta) }

type Uint64 struct {
	noCopy noCopy
	_      align64
	value  uint64
}

//go:nosplit
func (u *Uint64) Load() uint64 { return Load64(&u.value) }

//go:nosplit
func (u *Uint64) Store(value uint64) { Store64(&u.value, value) }

//go:nosplit
func (u *Uint64) CompareAndSwap(old, new uint64) bool { return Cas64(&u.value, old, new) }

//go:nosplit
func (u *Uint64) Swap(value uint64) uint64 { return Xchg64(&u.value, value) }

//go:nosplit
func (u *Uint64) Add(delta int64) uint64 { return Xadd64(&u.value, delta) }

type Uintptr struct {
	noCopy noCopy
	value  uintptr
}

//go:nosplit
func (u *Uintptr) Load() uintptr { return Loaduintptr(&u.value) }

//go:nosplit
func (u *Uintptr) LoadAcquire() uintptr { return LoadAcquintptr(&u.value) }

//go:nosplit
func (u *Uintptr) Store(value uintptr) { Storeuintptr(&u.value, value) }

//go:nosplit
func (u *Uintptr) StoreRelease(value uintptr) { StoreReluintptr(&u.value, value) }

//go:nosplit
func (u *Uintptr) CompareAndSwap(old, new uintptr) bool { return Casuintptr(&u.value, old, new) }

//go:nosplit
func (u *Uintptr) Swap(value uintptr) uintptr { return Xchguintptr(&u.value, value) }

//go:nosplit
func (u *Uintptr) Add(delta uintptr) uintptr { return Xadduintptr(&u.value, delta) }

type Float64 struct {
	u Uint64
}

//go:nosplit
func (f *Float64) Load() float64 {
	r := f.u.Load()
	return *(*float64)(unsafe.Pointer(&r))
}

//go:nosplit
func (f *Float64) Store(value float64) {
	f.u.Store(*(*uint64)(unsafe.Pointer(&value)))
}

type UnsafePointer struct {
	noCopy noCopy
	value  unsafe.Pointer
}

//go:nosplit
func (u *UnsafePointer) Load() unsafe.Pointer { return Loadp(unsafe.Pointer(&u.value)) }

//go:nosplit
func (u *UnsafePointer) StoreNoWB(value unsafe.Pointer) { StorepNoWB(unsafe.Pointer(&u.value), value) }

func (u *UnsafePointer) Store(value unsafe.Pointer) { storePointer(&u.value, value) }

//go:linkname storePointer
func storePointer(ptr *unsafe.Pointer, new unsafe.Pointer)

//go:nosplit
func (u *UnsafePointer) CompareAndSwapNoWB(old, new unsafe.Pointer) bool {
	return Casp1(&u.value, old, new)
}

func (u *UnsafePointer) CompareAndSwap(old, new unsafe.Pointer) bool {
	return casPointer(&u.value, old, new)
}

//go:linkname casPointer
func casPointer(ptr *unsafe.Pointer, old, new unsafe.Pointer) bool

type Pointer[T any] struct {
	u UnsafePointer
}

//go:nosplit
func (p *Pointer[T]) Load() *T { return (*T)(p.u.Load()) }

//go:nosplit
func (p *Pointer[T]) StoreNoWB(value *T) { p.u.StoreNoWB(unsafe.Pointer(value)) }

//go:nosplit
func (p *Pointer[T]) Store(value *T) { p.u.Store(unsafe.Pointer(value)) }

//go:nosplit
func (p *Pointer[T]) CompareAndSwapNoWB(old, new *T) bool {
	return p.u.CompareAndSwapNoWB(unsafe.Pointer(old), unsafe.Pointer(new))
}

func (p *Pointer[T]) CompareAndSwap(old, new *T) bool {
	return p.u.CompareAndSwap(unsafe.Pointer(old), unsafe.Pointer(new))
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type align64 struct{}
