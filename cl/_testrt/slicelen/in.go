// LITTEST
package main

import (
	"unsafe"
)

func main() {
	var s *int
	var lens uint32
	sl := unsafe.Slice(s, lens)
	slen := len(sl)
	println(slen)
	if slen > 0 {
		println("len > 0")
	}
}

// A nil pointer is valid for unsafe.Slice only because the length is zero; the
// compiler must fold all guards and the resulting length/branch consistently.
// CHECK: [[LEN_RANGE:@[0-9]+]] = private unnamed_addr constant [30 x i8] c"unsafe.Slice: len out of range"
// CHECK: [[NIL_NONZERO:@[0-9]+]] = private unnamed_addr constant [46 x i8] c"unsafe.Slice: nil pointer with non-zero length"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"{{.*}}AssertRuntimeError"(i1 false, %"{{.*}}String" { ptr [[LEN_RANGE]], i64 30 })
// CHECK-NEXT: call void @"{{.*}}AssertRuntimeError"(i1 false, %"{{.*}}String" { ptr [[NIL_NONZERO]], i64 46 })
// CHECK: call void @"{{.*}}PrintInt"(i64 0)
// CHECK: br i1 false, label %{{.*}}, label %{{.*}}
