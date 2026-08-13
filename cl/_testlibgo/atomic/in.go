// LITTEST
package main

import (
	"sync/atomic"
)

// Cover the LLVM operations and orderings selected for the four atomic APIs;
// the runtime golden checks their returned values.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ATOMIC:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT: store atomic i64 100, ptr [[ATOMIC]] seq_cst
// CHECK-NEXT: [[LOADED:%[0-9]+]] = load atomic i64, ptr [[ATOMIC]] seq_cst
// CHECK: call void @"{{.*}}PrintInt"(i64 [[LOADED]])
// CHECK: [[ADD_OLD:%[0-9]+]] = atomicrmw add ptr [[ATOMIC]], i64 1 seq_cst
// CHECK-NEXT: [[ADD_RESULT:%[0-9]+]] = add i64 [[ADD_OLD]], 1
// CHECK-NEXT: [[ADD_VALUE:%[0-9]+]] = load i64, ptr [[ATOMIC]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ADD_RESULT]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ADD_VALUE]])
// CHECK: [[CAS0_PAIR:%[0-9]+]] = cmpxchg ptr [[ATOMIC]], i64 100, i64 102 seq_cst seq_cst
// CHECK-NEXT: [[CAS0_OK:%[0-9]+]] = extractvalue { i64, i1 } [[CAS0_PAIR]], 1
// CHECK-NEXT: [[CAS0_VALUE:%[0-9]+]] = load i64, ptr [[ATOMIC]]
// CHECK: call void @"{{.*}}PrintBool"(i1 [[CAS0_OK]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[CAS0_VALUE]])
// CHECK: [[CAS1_PAIR:%[0-9]+]] = cmpxchg ptr [[ATOMIC]], i64 101, i64 102 seq_cst seq_cst
// CHECK-NEXT: [[CAS1_OK:%[0-9]+]] = extractvalue { i64, i1 } [[CAS1_PAIR]], 1
// CHECK-NEXT: [[CAS1_VALUE:%[0-9]+]] = load i64, ptr [[ATOMIC]]
// CHECK: call void @"{{.*}}PrintBool"(i1 [[CAS1_OK]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[CAS1_VALUE]])
// CHECK: [[SUB_OLD:%[0-9]+]] = atomicrmw add ptr [[ATOMIC]], i64 -1 seq_cst
// CHECK-NEXT: [[SUB_RESULT:%[0-9]+]] = add i64 [[SUB_OLD]], -1
// CHECK-NEXT: [[SUB_VALUE:%[0-9]+]] = load i64, ptr [[ATOMIC]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[SUB_RESULT]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[SUB_VALUE]])

func main() {
	var v int64

	atomic.StoreInt64(&v, 100)
	println("store:", atomic.LoadInt64(&v))

	ret := atomic.AddInt64(&v, 1)
	println("ret:", ret, "v:", v)

	swp := atomic.CompareAndSwapInt64(&v, 100, 102)
	println("swp:", swp, "v:", v)

	swp = atomic.CompareAndSwapInt64(&v, 101, 102)
	println("swp:", swp, "v:", v)

	ret = atomic.AddInt64(&v, -1)
	println("ret:", ret, "v:", v)
}
