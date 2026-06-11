// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [3 x i8] c"env", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [4 x i8] c"func", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [7 x i8] c"closure", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/closure.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/closure.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T func(n int)

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 }, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closure.main$2", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = alloca %"{{.*}}/cl/_testgo/closure.T", align 8
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testgo/closure.T", ptr %4, align 8
// CHECK-NEXT:   call void @"__llgo_stub.{{.*}}/cl/_testgo/closure.main$1"(ptr null, i64 100)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/cl/_testgo/closure.T" %5, 1
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/cl/_testgo/closure.T" %5, 0
// CHECK-NEXT:   call void %7(ptr %6, i64 200)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	var env string = "env"

	// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure.main$1"(i64 %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	var v1 T = func(i int) {
		println("func", i)
	}

	// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure.main$2"(ptr %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
	// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 7 })
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %4)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	var v2 T = func(i int) {
		println("closure", i, env)
	}
	v1(100)
	v2(200)
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/closure.main$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/closure.main$1"(i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
