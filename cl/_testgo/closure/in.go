// LITTEST
package main

type T func(n int)

// CHECK-LABEL: define void @"{{.*}}closure.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK: store %"{{.*}}String" { ptr @0, i64 3 }, ptr %0, align 8
	// CHECK: call ptr @"{{.*}}AllocU"(i64 8)
	// CHECK: { ptr @"{{.*}}closure.main$2", ptr undef }
	// CHECK: call void @"__llgo_stub.{{.*}}closure.main$1"(ptr null, i64 100)
	// CHECK: call void %7(ptr %6, i64 200)
	// CHECK: ret void
	var env string = "env"
	var v1 T = func(i int) {
		// CHECK-LABEL: define void @"{{.*}}closure.main$1"(i64 %0){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @1, i64 4 })
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
		// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %0)
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
		// CHECK-NEXT:   ret void
		println("func", i)
	}
	var v2 T = func(i int) {
		// CHECK-LABEL: define void @"{{.*}}closure.main$2"(ptr %0, i64 %1){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
		// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
		// CHECK-NEXT:   %4 = load %"{{.*}}String", ptr %3, align 8
		// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @2, i64 7 })
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
		// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %1)
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
		// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" %4)
		// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
		// CHECK-NEXT:   ret void
		println("closure", i, env)
	}
	v1(100)
	v2(200)
}
