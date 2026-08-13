// LITTEST
package main

import (
	"unsafe"
)

type N[T any] struct {
	n1 T
	n2 T
}

type M[T any] struct {
	m0 T
	m1 int32
	m2 N[T]
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[BOOL_M:%.*]] = call ptr @"{{.*}}AllocZ"(i64 12)
// CHECK-NEXT: call void @"main.(*M[bool]).check"(ptr [[BOOL_M]], i64 1, i64 8, i64 1)
// CHECK: [[INT64_M:%.*]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK-NEXT: call void @"main.(*M[int64]).check"(ptr [[INT64_M]], i64 8, i64 16, i64 8)
func main() {
	m1 := M[bool]{}
	m1.check(1, 8, 1)
	m2 := M[int64]{}
	m2.check(8, 16, 8)
}

// Each instantiation folds Alignof/Offsetof to its concrete layout while still
// addressing the instantiated fields used by the expressions.
// CHECK-LABEL: define linkonce void @"main.(*M[bool]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK: [[BOOL_M2_ALIGN:%.*]] = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT: load %"main.N[bool]", ptr [[BOOL_M2_ALIGN]], align 1
// CHECK-NEXT: [[BOOL_ALIGN_BAD:%.*]] = icmp ne i64 1, %1
// CHECK: [[BOOL_M2_OFFSET:%.*]] = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT: load %"main.N[bool]", ptr [[BOOL_M2_OFFSET]], align 1
// CHECK-NEXT: [[BOOL_OFFSET1_BAD:%.*]] = icmp ne i64 8, %2
// CHECK: [[BOOL_M2:%.*]] = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT: [[BOOL_N2:%.*]] = getelementptr inbounds %"main.N[bool]", ptr [[BOOL_M2]], i32 0, i32 1
// CHECK-NEXT: load i1, ptr [[BOOL_N2]], align 1
// CHECK-NEXT: [[BOOL_OFFSET2_BAD:%.*]] = icmp ne i64 1, %3
// CHECK: ret void
func (m *M[T]) check(align, offset1, offset2 uintptr) {
	if v := unsafe.Alignof(m.m2); v != align {
		println("have", v, "want", align)
		panic("unsafe.Alignof error")
	}
	if v := unsafe.Offsetof(m.m2); v != offset1 {
		println("have", v, "want", offset1)
		panic("unsafe.Offsetof error")
	}
	if v := unsafe.Offsetof(m.m2.n2); v != offset2 {
		println("have", v, "want", offset2)
		panic("unsafe.Offsetof error")
	}
}

// CHECK-LABEL: define linkonce void @"main.(*M[int64]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK: [[INT_M2_ALIGN:%.*]] = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT: load %"main.N[int64]", ptr [[INT_M2_ALIGN]], align 8
// CHECK-NEXT: [[INT_ALIGN_BAD:%.*]] = icmp ne i64 8, %1
// CHECK: [[INT_M2_OFFSET:%.*]] = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT: load %"main.N[int64]", ptr [[INT_M2_OFFSET]], align 8
// CHECK-NEXT: [[INT_OFFSET1_BAD:%.*]] = icmp ne i64 16, %2
// CHECK: [[INT_M2:%.*]] = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT: [[INT_N2:%.*]] = getelementptr inbounds %"main.N[int64]", ptr [[INT_M2]], i32 0, i32 1
// CHECK-NEXT: load i64, ptr [[INT_N2]], align 8
// CHECK-NEXT: [[INT_OFFSET2_BAD:%.*]] = icmp ne i64 8, %3
// CHECK: ret void
