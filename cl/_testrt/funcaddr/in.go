// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//llgo:type C
type Add func(int, int) int

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/funcaddr.add"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func add(a, b int) int {
	return a + b
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/funcaddr.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store ptr @"{{.*}}/cl/_testrt/funcaddr.add", ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store ptr @"{{.*}}/cl/_testrt/funcaddr.main$1", ptr %1, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %3 = icmp eq ptr @"{{.*}}/cl/_testrt/funcaddr.add", %2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %5 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %6 = icmp eq ptr %4, %5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %7 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %8 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %9 = icmp eq ptr %7, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	var fn Add = add
	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/funcaddr.main$1"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = add i64 %0, %1
	// CHECK-NEXT:   ret i64 %2
	// CHECK-NEXT: }
	var myfn Add = func(a, b int) int {
		return a + b
	}
	println(c.Func(add) == c.Func(fn))
	println(c.Func(fn) == *(*unsafe.Pointer)(unsafe.Pointer(&fn)))
	println(c.Func(myfn) == *(*unsafe.Pointer)(unsafe.Pointer(&myfn)))
}
