//go:build tinygo.riscv32

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// RV32IMC (ESP32-C3/S31) has no A extension, so LLVM lowers atomic operations
// to the sized __atomic_*_4 libcalls. compiler-rt's atomic.c cannot serve this
// target: its spinlocks are themselves built on compare-exchange, which would
// recurse into the same libcall. Provide the 4-byte builtins with GCC's
// calling conventions (values by value, load returns the value) and
// single-hart semantics: plain loads and stores, and RMW sequences that
// complete without safepoints. ISR-driven preemption between the load and
// store of an RMW sequence is not covered, matching the rest of this target's
// baseline scheduler guarantees.

func atomicCas32(p *uint32, old, new uint32) bool {
	if cur := *p; cur == old {
		*p = new
		return true
	}
	return false
}

// 8-byte atomics on RV32 need a lock. One static word is enough for this
// target: the scheduler is single-hart, and the lock is only held across the
// inline RMW sequence, never across a call that could reach the same code.
var atomicLock8 uint32

func atomicLock8Acquire() {
	for !atomicCas32(&atomicLock8, 0, 1) {
	}
}

func atomicLock8Release() {
	atomicLock8 = 0
}

func atomicLoad64(p *uint64) uint64 {
	atomicLock8Acquire()
	v := *p
	atomicLock8Release()
	return v
}

func atomicStore64(p *uint64, v uint64) {
	atomicLock8Acquire()
	*p = v
	atomicLock8Release()
}

func atomicRmw64(p *uint64, val uint64, op func(old, val uint64) uint64) uint64 {
	atomicLock8Acquire()
	old := *p
	*p = op(old, val)
	atomicLock8Release()
	return old
}

func atomicCas64(p *uint64, expected *uint64, desired uint64) bool {
	atomicLock8Acquire()
	ok := false
	if *p == *expected {
		*p = desired
		ok = true
	} else {
		*expected = *p
	}
	atomicLock8Release()
	return ok
}

func atomicRmw32(p *uint32, val uint32, op func(old, val uint32) uint32) uint32 {
	for {
		old := *p
		if atomicCas32(p, old, op(old, val)) {
			return old
		}
	}
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
func __atomic_compare_exchange_4(p *uint32, expected *uint32, desired uint32, _ c.Int, _, _ c.Int) bool {
	if atomicCas32(p, *expected, desired) {
		return true
	}
	*expected = *p
	return false
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
func __atomic_compare_exchange_8(p *uint64, expected *uint64, desired uint64, _ c.Int, _, _ c.Int) bool {
	return atomicCas64(p, expected, desired)
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
