// LITTEST
package main

type inner struct {
	x int
}

type outer struct {
	y int
	inner
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/methodthunk.(*InnerInt).M"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/methodthunk.InnerInt", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func (*inner) M() {}

type InnerInt struct {
	X int
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/methodthunk.(*OuterInt).M"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/methodthunk.OuterInt", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testrt/methodthunk.(*InnerInt).M"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
type OuterInt struct {
	Y int
	InnerInt
}

func (i *InnerInt) M() int {
	return i.X
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/methodthunk.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/methodthunk.(*outer).M$thunk", ptr null }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$31N-NdXOzvOy55m3NGAY_hdZ_NIdtCAe5V7uk7-a5HU", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/methodthunk.(*InnerInt).M$thunk", ptr null }, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$ygdobeQSbhO1hSbeWA66ORl_cNKHor-iD8MqRFtuWHg", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// CHECK-NEXT:   %5 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$31N-NdXOzvOy55m3NGAY_hdZ_NIdtCAe5V7uk7-a5HU", ptr %4)
// CHECK-NEXT:   br i1 %5, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 47 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// CHECK-NEXT:   %9 = load { ptr, ptr }, ptr %8, align 8
// CHECK-NEXT:   %10 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %9, 0
// CHECK-NEXT:   %11 = insertvalue { { ptr, ptr }, i1 } %10, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %12 = phi { { ptr, ptr }, i1 } [ %11, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %13 = extractvalue { { ptr, ptr }, i1 } %12, 0
// CHECK-NEXT:   %14 = extractvalue { { ptr, ptr }, i1 } %12, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %3, 0
// CHECK-NEXT:   %17 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$31N-NdXOzvOy55m3NGAY_hdZ_NIdtCAe5V7uk7-a5HU", ptr %16)
// CHECK-NEXT:   br i1 %17, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %3, 1
// CHECK-NEXT:   %19 = load { ptr, ptr }, ptr %18, align 8
// CHECK-NEXT:   %20 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %19, 0
// CHECK-NEXT:   %21 = insertvalue { { ptr, ptr }, i1 } %20, i1 true, 1
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   %22 = phi { { ptr, ptr }, i1 } [ %21, %_llgo_6 ], [ zeroinitializer, %_llgo_7 ]
// CHECK-NEXT:   %23 = extractvalue { { ptr, ptr }, i1 } %22, 0
// CHECK-NEXT:   %24 = extractvalue { { ptr, ptr }, i1 } %22, 1
// CHECK-NEXT:   %25 = extractvalue { ptr, ptr } %23, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br i1 %24, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }
func main() {
	var v1 any = (*outer).M
	var v2 any = (*InnerInt).M
	f1, ok := v1.(func(*outer))
	println(f1, ok)
	f2, ok := v2.(func(*outer))
	println(f2, ok)
	if ok {
		panic("type assertion should have failed but succeeded")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/methodthunk.(*outer).M"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (m *outer) M() {}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/methodthunk.(*outer).M$thunk"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/methodthunk.(*outer).M"(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/methodthunk.(*outer).M$thunk"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/methodthunk.(*outer).M$thunk"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/methodthunk.(*InnerInt).M$thunk"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call i64 @"{{.*}}/cl/_testrt/methodthunk.(*InnerInt).M"(ptr %0)
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/methodthunk.(*InnerInt).M$thunk"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testrt/methodthunk.(*InnerInt).M$thunk"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
