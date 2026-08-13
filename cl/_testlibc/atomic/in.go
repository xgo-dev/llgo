// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/sync/atomic"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	var v int64

	// All operations target the same slot and retain sequential consistency.
	// CHECK: [[V:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: store atomic i64 100, ptr [[V]] seq_cst
	atomic.Store(&v, 100)
	// CHECK: [[LOADED:%.*]] = load atomic i64, ptr [[V]] seq_cst
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[LOADED]])
	c.Printf(c.Str("store: %ld\n"), atomic.Load(&v))
	// CHECK: [[ADD_OLD:%.*]] = atomicrmw add ptr [[V]], i64 1 seq_cst
	// CHECK: [[ADD_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[ADD_OLD]], i64 [[ADD_CURRENT]])
	ret := atomic.Add(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)

	// CHECK: [[CAS100:%.*]] = cmpxchg ptr [[V]], i64 100, i64 102 seq_cst seq_cst
	// CHECK: [[CAS100_OLD:%.*]] = extractvalue { i64, i1 } [[CAS100]], 0
	// CHECK: [[CAS100_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[CAS100_OLD]], i64 [[CAS100_CURRENT]])
	ret, _ = atomic.CompareAndExchange(&v, 100, 102)
	c.Printf(c.Str("ret: %ld vs 100, v: %ld\n"), ret, v)

	// CHECK: [[CAS101:%.*]] = cmpxchg ptr [[V]], i64 101, i64 102 seq_cst seq_cst
	// CHECK: [[CAS101_OLD:%.*]] = extractvalue { i64, i1 } [[CAS101]], 0
	// CHECK: [[CAS101_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[CAS101_OLD]], i64 [[CAS101_CURRENT]])
	ret, _ = atomic.CompareAndExchange(&v, 101, 102)
	c.Printf(c.Str("ret: %ld vs 101, v: %ld\n"), ret, v)

	// CHECK: [[SUB_OLD:%.*]] = atomicrmw sub ptr [[V]], i64 1 seq_cst
	// CHECK: [[SUB_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[SUB_OLD]], i64 [[SUB_CURRENT]])
	ret = atomic.Sub(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)
}
