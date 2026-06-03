// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/sync/atomic"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibc/atomic.main"(){{.*}} {
func main() {
	var v int64

	// CHECK: store atomic i64 100, ptr %0 seq_cst, align 8
	atomic.Store(&v, 100)
	// CHECK: %1 = load atomic i64, ptr %0 seq_cst, align 8
	c.Printf(c.Str("store: %ld\n"), atomic.Load(&v))
	// CHECK: %3 = atomicrmw add ptr %0, i64 1 seq_cst, align 8
	ret := atomic.Add(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)

	// CHECK: %6 = cmpxchg ptr %0, i64 100, i64 102 seq_cst seq_cst, align 8
	ret, _ = atomic.CompareAndExchange(&v, 100, 102)
	c.Printf(c.Str("ret: %ld vs 100, v: %ld\n"), ret, v)

	// CHECK: %11 = cmpxchg ptr %0, i64 101, i64 102 seq_cst seq_cst, align 8
	ret, _ = atomic.CompareAndExchange(&v, 101, 102)
	c.Printf(c.Str("ret: %ld vs 101, v: %ld\n"), ret, v)

	// CHECK: %16 = atomicrmw sub ptr %0, i64 1 seq_cst, align 8
	ret = atomic.Sub(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)
}
