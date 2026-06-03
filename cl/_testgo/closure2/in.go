// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}closure2.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: store i64 1, ptr %0, align 8
	// CHECK: call ptr @"{{.*}}AllocU"(i64 8)
	// CHECK: { ptr @"{{.*}}closure2.main$1", ptr undef }
	// CHECK: call { ptr, ptr } %5(ptr %4, i64 1)
	// CHECK: call void %8(ptr %7, i64 2)
	// CHECK: ret void
	x := 1
	f := func(i int) func(int) {
		// CHECK-LABEL: define { ptr, ptr } @"{{.*}}closure2.main$1"(ptr %0, i64 %1){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
		// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
		// CHECK-NEXT:   %4 = call ptr @"{{.*}}AllocU"(i64 8)
		// CHECK-NEXT:   %5 = getelementptr inbounds { ptr }, ptr %4, i32 0, i32 0
		// CHECK-NEXT:   store ptr %3, ptr %5, align 8
		// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } { ptr @"{{.*}}closure2.main$1$1", ptr undef }, ptr %4, 1
		// CHECK-NEXT:   ret { ptr, ptr } %6
		return func(i int) {
			// CHECK-LABEL: define void @"{{.*}}closure2.main$1$1"(ptr %0, i64 %1){{.*}} {
			// CHECK-NEXT: _llgo_0:
			// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
			// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
			// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
			// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @0, i64 7 })
			// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
			// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %1)
			// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
			// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %4)
			// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
			// CHECK-NEXT:   ret void
			println("closure", i, x)
		}
	}
	f(1)(2)
}
