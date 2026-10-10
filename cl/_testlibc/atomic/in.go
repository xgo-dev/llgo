// LITTEST
// Scope: common
package main

import _ "unsafe"

type atomicValue interface {
	~int | ~uint | ~uintptr | ~int32 | ~uint32 | ~int64 | ~uint64
}

// llgo:link atomicAdd llgo.atomicAdd
func atomicAdd[T atomicValue](ptr *T, v T) T { return v }

// llgo:link atomicSub llgo.atomicSub
func atomicSub[T atomicValue](ptr *T, v T) T { return v }

// llgo:link atomicLoad llgo.atomicLoad
func atomicLoad[T atomicValue](ptr *T) T { return *ptr }

// llgo:link atomicStore llgo.atomicStore
func atomicStore[T atomicValue](ptr *T, v T) {}

// llgo:link atomicCompareAndExchange llgo.atomicCmpXchg
func atomicCompareAndExchange[T atomicValue](ptr *T, old, new T) (T, bool) {
	return old, false
}

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	var v int64

	// All operations target the same slot and retain sequential consistency.
	// CHECK: [[V:%.*]] = call noalias dereferenceable(8) ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: store atomic i64 100, ptr [[V]] seq_cst
	atomicStore(&v, 100)
	// CHECK: [[LOADED:%.*]] = load atomic i64, ptr [[V]] seq_cst
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[LOADED]])
	printf(cstr("store: %ld\n"), atomicLoad(&v))
	// CHECK: [[ADD_OLD:%.*]] = atomicrmw add ptr [[V]], i64 1 seq_cst
	// CHECK: [[ADD_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[ADD_OLD]], i64 [[ADD_CURRENT]])
	ret := atomicAdd(&v, 1)
	printf(cstr("ret: %ld, v: %ld\n"), ret, v)

	// CHECK: [[CAS100:%.*]] = cmpxchg ptr [[V]], i64 100, i64 102 seq_cst seq_cst
	// CHECK: [[CAS100_OLD:%.*]] = extractvalue { i64, i1 } [[CAS100]], 0
	// CHECK: [[CAS100_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[CAS100_OLD]], i64 [[CAS100_CURRENT]])
	ret, _ = atomicCompareAndExchange(&v, 100, 102)
	printf(cstr("ret: %ld vs 100, v: %ld\n"), ret, v)

	// CHECK: [[CAS101:%.*]] = cmpxchg ptr [[V]], i64 101, i64 102 seq_cst seq_cst
	// CHECK: [[CAS101_OLD:%.*]] = extractvalue { i64, i1 } [[CAS101]], 0
	// CHECK: [[CAS101_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[CAS101_OLD]], i64 [[CAS101_CURRENT]])
	ret, _ = atomicCompareAndExchange(&v, 101, 102)
	printf(cstr("ret: %ld vs 101, v: %ld\n"), ret, v)

	// CHECK: [[SUB_OLD:%.*]] = atomicrmw sub ptr [[V]], i64 1 seq_cst
	// CHECK: [[SUB_CURRENT:%.*]] = load i64, ptr [[V]]
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[SUB_OLD]], i64 [[SUB_CURRENT]])
	ret = atomicSub(&v, 1)
	printf(cstr("ret: %ld, v: %ld\n"), ret, v)
}
