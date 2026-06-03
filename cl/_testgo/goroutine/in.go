// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}goroutine.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 1)
	// CHECK: store i1 false, ptr %0, align 1
	// CHECK: call ptr @"{{.*}}AllocU"(i64 16)
	// CHECK: call i32 @"{{.*}}CreateThread"(ptr %3, ptr null, ptr @"{{.*}}goroutine._llgo_routine$1", ptr %1)
	done := false
	go println("hello")
	go func(s string) {
		// CHECK: call ptr @"{{.*}}AllocU"(i64 8)
		// CHECK: { ptr @"{{.*}}goroutine.main$1", ptr undef }
		// CHECK: call ptr @"{{.*}}AllocU"(i64 32)
		// CHECK: call i32 @"{{.*}}CreateThread"(ptr %11, ptr null, ptr @"{{.*}}goroutine._llgo_routine$2", ptr %8)
		// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @2, i64 1 })
		// CHECK: ret void
		// CHECK-LABEL: define void @"{{.*}}goroutine.main$1"(ptr %0, %"{{.*}}String" %1){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" %1)
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
		// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
		// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
		// CHECK-NEXT:   store i1 true, ptr %3, align 1
		// CHECK-NEXT:   ret void
		println(s)
		done = true
	}("Hello, goroutine")
	for !done {
		print(".")
	}
}

// CHECK-LABEL: define ptr @"{{.*}}goroutine._llgo_routine$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { %"{{.*}}String" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}String" } %1, 0
// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" %2)
// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
// CHECK-NEXT:   ret ptr null

// CHECK-LABEL: define ptr @"{{.*}}goroutine._llgo_routine$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr }, %"{{.*}}String" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr }, %"{{.*}}String" } %1, 0
// CHECK-NEXT:   %3 = extractvalue { { ptr, ptr }, %"{{.*}}String" } %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   call void %5(ptr %4, %"{{.*}}String" %3)
// CHECK-NEXT:   ret ptr null
