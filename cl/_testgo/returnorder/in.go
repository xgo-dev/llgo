// LITTEST
package main

import "fmt"

// CHECK-LINE: @1 = private unnamed_addr constant [46 x i8] c"return order mismatch: got (%d,%d), want (2,2)", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [2 x i8] c"ok", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/returnorder.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/returnorder.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/returnorder.init$guard", align 1
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type state struct {
	v int
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/returnorder.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testgo/returnorder.state", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = call { %"{{.*}}/cl/_testgo/returnorder.state", i64 } @"{{.*}}/cl/_testgo/returnorder.returnStateAndMut"()
// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}/cl/_testgo/returnorder.state", i64 } %1, 0
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/returnorder.state" %2, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { %"{{.*}}/cl/_testgo/returnorder.state", i64 } %1, 1
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   %7 = icmp ne i64 %6, 2
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %10 = load i64, ptr %9, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %11, i64 0
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %10, ptr %13, align 8
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %13, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %14, ptr %12, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %11, i64 1
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %3, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %16, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %15, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %11, 0
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %18, i64 2, 1
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 2, 2
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 46 }, %"{{.*}}/runtime/internal/runtime.Slice" %20)
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %21, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %22, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %24 = icmp ne i64 %3, 2
// CHECK-NEXT:   br i1 %24, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

func main() {
	a, b := returnStateAndMut()
	if a.v != 2 || b != 2 {
		panic(fmt.Sprintf("return order mismatch: got (%d,%d), want (2,2)", a.v, b))
	}
	println("ok")
}

// CHECK-LABEL: define { %"{{.*}}/cl/_testgo/returnorder.state", i64 } @"{{.*}}/cl/_testgo/returnorder.returnStateAndMut"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/returnorder.(*state).mutate"(ptr %0, i64 2)
// CHECK-NEXT:   %4 = load %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, align 8
// CHECK-NEXT:   %5 = insertvalue { %"{{.*}}/cl/_testgo/returnorder.state", i64 } undef, %"{{.*}}/cl/_testgo/returnorder.state" %4, 0
// CHECK-NEXT:   %6 = insertvalue { %"{{.*}}/cl/_testgo/returnorder.state", i64 } %5, i64 %3, 1
// CHECK-NEXT:   ret { %"{{.*}}/cl/_testgo/returnorder.state", i64 } %6
// CHECK-NEXT: }

func returnStateAndMut() (state, int) {
	x := state{v: 1}
	return x, x.mutate(2)
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/returnorder.(*state).mutate"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %3, align 8
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   ret i64 %6
// CHECK-NEXT: }

func (s *state) mutate(next int) int {
	s.v = next
	return s.v
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
