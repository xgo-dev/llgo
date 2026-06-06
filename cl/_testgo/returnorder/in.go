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
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = icmp ne i64 %7, 2
// CHECK-NEXT:   br i1 %8, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %13, i64 0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %12, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %15, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %16, ptr %14, align 8
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %13, i64 1
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %3, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %18, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %19, ptr %17, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %13, 0
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 2, 1
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %21, i64 2, 2
// CHECK-NEXT:   %23 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 46 }, %"{{.*}}/runtime/internal/runtime.Slice" %22)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %23, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %24, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %25)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %26 = icmp ne i64 %3, 2
// CHECK-NEXT:   br i1 %26, label %_llgo_1, label %_llgo_2
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
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %4, align 8
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %8 = load i64, ptr %7, align 8
// CHECK-NEXT:   ret i64 %8
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
