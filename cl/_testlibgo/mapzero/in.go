// LITTEST
package main

import "fmt"

// CHECK-LINE: @13 = private unnamed_addr constant [5 x i8] c"panic", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/mapzero.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testlibgo/mapzero.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testlibgo/mapzero.init$guard", align 1
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   store i64 0, ptr @"{{.*}}/cl/_testlibgo/mapzero.a", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

var a = 0

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/mapzero.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8, i64 196, align 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/mapzero.main", %_llgo_2), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %9 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %11 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %13 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %14, align 8
// CHECK-NEXT:   %15 = call i32 @sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/mapzero.main", %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %17 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testlibgo/mapzero.main$1"()
// CHECK-NEXT:   %18 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %18, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %19)
// CHECK-NEXT:   %20 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %20, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %21 = load i64, ptr @"{{.*}}/cl/_testlibgo/mapzero.a", align 8
// CHECK-NEXT:   %22 = icmp slt i64 %21, 0
// CHECK-NEXT:   %23 = icmp uge i64 %21, 0
// CHECK-NEXT:   %24 = or i1 %23, %22
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %24, i64 %21, i1 true, i64 0)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %25, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_int", ptr null, ptr %25)
// CHECK-NEXT:   %27 = load i64, ptr %26, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %27)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/mapzero.main", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/mapzero.main", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %28 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %28, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {

	// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/mapzero.main$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
	// CHECK-NEXT:   %1 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
	// CHECK-NEXT:   %2 = xor i1 %1, true
	// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
	// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
	// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %3, i64 0
	// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 5 }, ptr %5, align 8
	// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %6, ptr %4, align 8
	// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3, 0
	// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, i64 1, 1
	// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, i64 1, 2
	// CHECK-NEXT:   %10 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %9)
	// CHECK-NEXT:   br label %_llgo_2
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	defer func() {
		if recover() != nil {
			fmt.Println("panic")
		}
	}()
	m := [0]map[int]int{}[a][0]
	print(m)
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
