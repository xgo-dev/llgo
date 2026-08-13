// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//llgo:type C
type T func()

type M struct {
	fn T
	v  int
}

type N struct {
	fn func()
	v  int
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// Sizeof, Alignof, and Offsetof are compile-time constants. Each of the ten
// source comparisons must fold to the non-panic edge.
// CHECK-COUNT-10: br i1 false

// unsafe.String checks pointer arithmetic overflow and uses the resulting
// pointer/length pair as the string value.
// CHECK: %[[STRING_OVERFLOW:[0-9]+]] = icmp ult i64 add (i64 ptrtoint (ptr @[[CSTR:[0-9]+]] to i64), i64 2), ptrtoint (ptr @[[CSTR]] to i64)
// CHECK: %[[STRING_INVALID:[0-9]+]] = and i1 true, %[[STRING_OVERFLOW]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 %[[STRING_INVALID]], %"{{.*}}/runtime/internal/runtime.String" {{.*}})
// CHECK: %[[STRING_EQ:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" { ptr @[[CSTR]], i64 3 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK: %[[STRING_NE:[0-9]+]] = xor i1 %[[STRING_EQ]], true
// CHECK: br i1 %[[STRING_NE]]

// unsafe.StringData starts from the same backing pointer.
// CHECK: load i8, ptr @[[CSTR]]

// unsafe.Slice validates pointer/length overflow, then constructs a slice whose
// data and length are the values consumed by ordinary bounds checks.
// CHECK: %[[ARRAY:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: %[[ELEM0:[0-9]+]] = getelementptr inbounds i64, ptr %[[ARRAY]], i64 0
// CHECK: %[[ELEM1:[0-9]+]] = getelementptr inbounds i64, ptr %[[ARRAY]], i64 1
// CHECK: store i64 1, ptr %[[ELEM0]]
// CHECK: store i64 2, ptr %[[ELEM1]]
// CHECK: %[[BASE:[0-9]+]] = getelementptr inbounds i64, ptr %[[ARRAY]], i64 0
// CHECK: %[[BASE_INT:[0-9]+]] = ptrtoint ptr %[[BASE]] to i64
// CHECK: %[[SLICE_END:[0-9]+]] = add i64 %[[BASE_INT]], 15
// CHECK: %[[SLICE_OVERFLOW:[0-9]+]] = icmp ult i64 %[[SLICE_END]], %[[BASE_INT]]
// CHECK: %[[SLICE_INVALID:[0-9]+]] = and i1 true, %[[SLICE_OVERFLOW]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 %[[SLICE_INVALID]], %"{{.*}}/runtime/internal/runtime.String" {{.*}})
// CHECK: %[[SLICE_PTR:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %[[BASE]], 0
// CHECK: %[[SLICE_LEN:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE_PTR]], i64 2, 1
// CHECK: %[[SLICE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE_LEN]], i64 2, 2
// CHECK: %[[DATA0:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 0
// CHECK: %[[LEN0:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 1
// CHECK: %[[OOB0:[0-9]+]] = icmp uge i64 0, %[[LEN0]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %[[OOB0]], i64 0, i1 true, i64 %[[LEN0]])
// CHECK: getelementptr inbounds i64, ptr %[[DATA0]], i64 0
// The remaining StringData loads may be printed in predecessor block order,
// but must still use the captured string backing pointer.
// CHECK: load i8, ptr getelementptr inbounds (i8, ptr @[[CSTR]], i64 2)
// CHECK: load i8, ptr getelementptr inbounds (i8, ptr @[[CSTR]], i64 1)

// unsafe.SliceData reads from the constructed slice data pointer.
// CHECK: %[[SLICE_DATA:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 0
// CHECK: load i64, ptr %[[SLICE_DATA]]

// CHECK: %[[DATA1:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 0
// CHECK: %[[LEN1:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 1
// CHECK: %[[OOB1:[0-9]+]] = icmp uge i64 1, %[[LEN1]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %[[OOB1]], i64 1, i1 true, i64 %[[LEN1]])
// CHECK: getelementptr inbounds i64, ptr %[[DATA1]], i64 1

// unsafe.Add(nil, 1) remains a byte-wise pointer addition.
// CHECK: icmp ne i64 ptrtoint (ptr getelementptr (i8, ptr null, i64 1) to i64), 1

func main() {
	if unsafe.Sizeof(*(*T)(nil)) != unsafe.Sizeof(0) {
		panic("error")
	}
	if unsafe.Sizeof(*(*M)(nil)) != unsafe.Sizeof([2]int{}) {
		panic("error")
	}
	// TODO(lijie): inconsistent with golang
	if unsafe.Sizeof(*(*N)(nil)) != unsafe.Sizeof([3]int{}) {
		panic("error")
	}

	if unsafe.Alignof(*(*T)(nil)) != unsafe.Alignof(0) {
		panic("error")
	}
	if unsafe.Alignof(*(*M)(nil)) != unsafe.Alignof([2]int{}) {
		panic("error")
	}
	if unsafe.Alignof(*(*N)(nil)) != unsafe.Alignof([3]int{}) {
		panic("error")
	}

	if unsafe.Offsetof(M{}.fn) != 0 {
		panic("error")
	}
	if unsafe.Offsetof(M{}.v) != unsafe.Sizeof(int(0)) {
		panic("error")
	}
	if unsafe.Offsetof(N{}.fn) != 0 {
		panic("error")
	}
	// TODO(lijie): inconsistent with golang
	if unsafe.Offsetof(N{}.v) != unsafe.Sizeof([2]int{}) {
		panic("error")
	}

	s := unsafe.String((*byte)(unsafe.Pointer(c.Str("abc"))), 3)
	if s != "abc" {
		panic("error")
	}

	p := unsafe.StringData(s)
	arr := (*[3]byte)(unsafe.Pointer(p))
	if arr[0] != 'a' || arr[1] != 'b' || arr[2] != 'c' {
		panic("error")
	}

	intArr := [2]int{1, 2}
	pi := &intArr[0]
	intSlice := unsafe.Slice(pi, 2)
	if intSlice[0] != 1 || intSlice[1] != 2 {
		panic("error")
	}

	pi = unsafe.SliceData(intSlice)
	if *pi != 1 {
		panic("error")
	}

	if uintptr(unsafe.Add(unsafe.Pointer(nil), 1)) != 1 {
		panic("error")
	}

}
