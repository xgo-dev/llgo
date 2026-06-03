// LITTEST
package main

var seed = 40

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/litdemo.add"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/litdemo.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/litdemo.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/litdemo.init$guard", align 1
// CHECK-NEXT:   store i64 40, ptr @"{{.*}}/cl/_testrt/litdemo.seed", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func add(x, y int) int {
	return x + y
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/litdemo.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i64, ptr @"{{.*}}/cl/_testrt/litdemo.seed", align 8
// CHECK-NEXT:   %1 = call i64 @"{{.*}}/cl/_testrt/litdemo.main$1"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {

	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/litdemo.main$1"(i64 %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = call i64 @"{{.*}}/cl/_testrt/litdemo.add"(i64 %0, i64 2)
	// CHECK-NEXT:   ret i64 %1
	// CHECK-NEXT: }

	plusTwo := func(v int) int {
		return add(v, 2)
	}
	println(plusTwo(seed))
}
