//go:build tinygo.riscv32

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// RV32IMC (ESP32-C3/S31) has no A extension, so LLVM lowers every atomic
// operation to the __atomic_* libcalls collected here. compiler-rt's atomic.c
// cannot serve this target: it is excluded from the ESP rtlib, and its
// spinlocks would deadlock when an interrupt interrupts a holder. All RMW
// paths below therefore run with interrupts masked through the CSR helpers
// in _wrap/atomic_riscv32.c. Single-hart semantics: with interrupts masked,
// each RMW sequence is atomic against interrupt handlers on this hart;
// plain sized loads and stores are single RV32 instructions and need no
// masking. Nothing here protects against other harts (this runtime is
// single-hart) or non-maskable interrupts, matching the rest of this
// target's baseline scheduler guarantees.

//go:linkname llgoAtomicIrqSave C.llgoAtomicIrqSave
func llgoAtomicIrqSave() uint32

//go:linkname llgoAtomicIrqRestore C.llgoAtomicIrqRestore
func llgoAtomicIrqRestore(saved uint32)

func atomicCopyBytes(dst, src *byte, size uint32) {
	c.Memmove(unsafe.Pointer(dst), unsafe.Pointer(src), uintptr(size))
}

func atomicEqualBytes(a, b *byte, size uint32) bool {
	return c.Memcmp(unsafe.Pointer(a), unsafe.Pointer(b), uintptr(size)) == 0
}

func atomicRmw32(p *uint32, val uint32, op func(old, val uint32) uint32) uint32 {
	saved := llgoAtomicIrqSave()
	old := *p
	*p = op(old, val)
	llgoAtomicIrqRestore(saved)
	return old
}

// 64-bit accesses are two RV32 instructions; every path masks interrupts so
// an interrupt handler cannot observe or interleave a torn value.
func atomicLoad64(p *uint64) uint64 {
	saved := llgoAtomicIrqSave()
	v := *p
	llgoAtomicIrqRestore(saved)
	return v
}

func atomicStore64(p *uint64, v uint64) {
	saved := llgoAtomicIrqSave()
	*p = v
	llgoAtomicIrqRestore(saved)
}

func atomicRmw64(p *uint64, val uint64, op func(old, val uint64) uint64) uint64 {
	saved := llgoAtomicIrqSave()
	old := *p
	*p = op(old, val)
	llgoAtomicIrqRestore(saved)
	return old
}

func atomicRmw8(p *uint8, val uint8, op func(old, val uint8) uint8) uint8 {
	saved := llgoAtomicIrqSave()
	old := *p
	*p = op(old, val)
	llgoAtomicIrqRestore(saved)
	return old
}

func atomicRmw16(p *uint16, val uint16, op func(old, val uint16) uint16) uint16 {
	saved := llgoAtomicIrqSave()
	old := *p
	*p = op(old, val)
	llgoAtomicIrqRestore(saved)
	return old
}

// Sized libcalls. LLVM AtomicExpand passes the success and failure memory
// models positionally; single-hart masking makes the models indistinguishable
// here, so they are accepted and ignored.

//export __atomic_load_1
func __atomic_load_1(p *uint8, _ c.Int) uint8 {
	return *p
}

//export __atomic_store_1
func __atomic_store_1(p *uint8, val uint8, _ c.Int) {
	*p = val
}

//export __atomic_exchange_1
func __atomic_exchange_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(_, val uint8) uint8 { return val })
}

//export __atomic_compare_exchange_1
func __atomic_compare_exchange_1(p *uint8, expected *uint8, desired uint8, _, _ c.Int) bool {
	saved := llgoAtomicIrqSave()
	cur := *p
	ok := cur == *expected
	if ok {
		*p = desired
	} else {
		*expected = cur
	}
	llgoAtomicIrqRestore(saved)
	return ok
}

//export __atomic_fetch_add_1
func __atomic_fetch_add_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return old + val })
}

//export __atomic_fetch_sub_1
func __atomic_fetch_sub_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return old - val })
}

//export __atomic_fetch_and_1
func __atomic_fetch_and_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return old & val })
}

//export __atomic_fetch_or_1
func __atomic_fetch_or_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return old | val })
}

//export __atomic_fetch_xor_1
func __atomic_fetch_xor_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return old ^ val })
}

//export __atomic_fetch_nand_1
func __atomic_fetch_nand_1(p *uint8, val uint8, _ c.Int) uint8 {
	return atomicRmw8(p, val, func(old, val uint8) uint8 { return ^(old & val) })
}

//export __atomic_load_2
func __atomic_load_2(p *uint16, _ c.Int) uint16 {
	return *p
}

//export __atomic_store_2
func __atomic_store_2(p *uint16, val uint16, _ c.Int) {
	*p = val
}

//export __atomic_exchange_2
func __atomic_exchange_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(_, val uint16) uint16 { return val })
}

//export __atomic_compare_exchange_2
func __atomic_compare_exchange_2(p *uint16, expected *uint16, desired uint16, _, _ c.Int) bool {
	saved := llgoAtomicIrqSave()
	cur := *p
	ok := cur == *expected
	if ok {
		*p = desired
	} else {
		*expected = cur
	}
	llgoAtomicIrqRestore(saved)
	return ok
}

//export __atomic_fetch_add_2
func __atomic_fetch_add_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return old + val })
}

//export __atomic_fetch_sub_2
func __atomic_fetch_sub_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return old - val })
}

//export __atomic_fetch_and_2
func __atomic_fetch_and_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return old & val })
}

//export __atomic_fetch_or_2
func __atomic_fetch_or_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return old | val })
}

//export __atomic_fetch_xor_2
func __atomic_fetch_xor_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return old ^ val })
}

//export __atomic_fetch_nand_2
func __atomic_fetch_nand_2(p *uint16, val uint16, _ c.Int) uint16 {
	return atomicRmw16(p, val, func(old, val uint16) uint16 { return ^(old & val) })
}

//export __atomic_load_4
func __atomic_load_4(p *uint32, _ c.Int) uint32 {
	return *p
}

//export __atomic_store_4
func __atomic_store_4(p *uint32, val uint32, _ c.Int) {
	*p = val
}

//export __atomic_exchange_4
func __atomic_exchange_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(_, val uint32) uint32 { return val })
}

//export __atomic_compare_exchange_4
func __atomic_compare_exchange_4(p *uint32, expected *uint32, desired uint32, _, _ c.Int) bool {
	saved := llgoAtomicIrqSave()
	cur := *p
	ok := cur == *expected
	if ok {
		*p = desired
	} else {
		*expected = cur
	}
	llgoAtomicIrqRestore(saved)
	return ok
}

//export __atomic_fetch_add_4
func __atomic_fetch_add_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return old + val })
}

//export __atomic_fetch_sub_4
func __atomic_fetch_sub_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return old - val })
}

//export __atomic_fetch_and_4
func __atomic_fetch_and_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return old & val })
}

//export __atomic_fetch_or_4
func __atomic_fetch_or_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return old | val })
}

//export __atomic_fetch_xor_4
func __atomic_fetch_xor_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return old ^ val })
}

//export __atomic_fetch_nand_4
func __atomic_fetch_nand_4(p *uint32, val uint32, _ c.Int) uint32 {
	return atomicRmw32(p, val, func(old, val uint32) uint32 { return ^(old & val) })
}

//export __atomic_load_8
func __atomic_load_8(p *uint64, _ c.Int) uint64 {
	return atomicLoad64(p)
}

//export __atomic_store_8
func __atomic_store_8(p *uint64, val uint64, _ c.Int) {
	atomicStore64(p, val)
}

//export __atomic_exchange_8
func __atomic_exchange_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(_, val uint64) uint64 { return val })
}

//export __atomic_compare_exchange_8
func __atomic_compare_exchange_8(p *uint64, expected *uint64, desired uint64, _, _ c.Int) bool {
	saved := llgoAtomicIrqSave()
	cur := *p
	ok := cur == *expected
	if ok {
		*p = desired
	} else {
		*expected = cur
	}
	llgoAtomicIrqRestore(saved)
	return ok
}

//export __atomic_fetch_add_8
func __atomic_fetch_add_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return old + val })
}

//export __atomic_fetch_sub_8
func __atomic_fetch_sub_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return old - val })
}

//export __atomic_fetch_and_8
func __atomic_fetch_and_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return old & val })
}

//export __atomic_fetch_or_8
func __atomic_fetch_or_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return old | val })
}

//export __atomic_fetch_xor_8
func __atomic_fetch_xor_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return old ^ val })
}

//export __atomic_fetch_nand_8
func __atomic_fetch_nand_8(p *uint64, val uint64, _ c.Int) uint64 {
	return atomicRmw64(p, val, func(old, val uint64) uint64 { return ^(old & val) })
}

// Size-generalized libcalls. Clang's frontend emits these for atomic
// operations whose size is not a power-of-two word size, passing the size as
// the first argument; the CAS form takes no weak flag. Buffers are caller
// allocated and naturally aligned for the object being moved.

//export __atomic_load
func __atomic_load(size uint32, src, dst *byte, _ c.Int) {
	saved := llgoAtomicIrqSave()
	atomicCopyBytes(dst, src, size)
	llgoAtomicIrqRestore(saved)
}

//export __atomic_store
func __atomic_store(size uint32, dst, src *byte, _ c.Int) {
	saved := llgoAtomicIrqSave()
	atomicCopyBytes(dst, src, size)
	llgoAtomicIrqRestore(saved)
}

//export __atomic_exchange
func __atomic_exchange(size uint32, obj, desired, ret *byte, _ c.Int) {
	saved := llgoAtomicIrqSave()
	atomicCopyBytes(ret, obj, size)
	atomicCopyBytes(obj, desired, size)
	llgoAtomicIrqRestore(saved)
}

//export __atomic_compare_exchange
func __atomic_compare_exchange(size uint32, obj, expected, desired *byte, _, _ c.Int) bool {
	saved := llgoAtomicIrqSave()
	ok := atomicEqualBytes(obj, expected, size)
	if ok {
		atomicCopyBytes(obj, desired, size)
	} else {
		atomicCopyBytes(expected, obj, size)
	}
	llgoAtomicIrqRestore(saved)
	return ok
}

//export __atomic_clear
func __atomic_clear(p *byte, _ c.Int) {
	*p = 0
}
