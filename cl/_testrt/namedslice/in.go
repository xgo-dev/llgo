// LITTEST
package main

// CHECK-LINE: @3 = private unnamed_addr constant [9 x i8] c"bad slice", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/namedslice.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/namedslice.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/namedslice.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type MyBytes []byte

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/namedslice.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/namedslice.MyBytes", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// CHECK-NEXT:   %3 = icmp eq ptr %2, @"_llgo_{{.*}}/cl/_testrt/namedslice.MyBytes"
// CHECK-NEXT:   br i1 %3, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 9 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", i1 } undef, %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", i1 } %8, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %10 = phi { %"{{.*}}/runtime/internal/runtime.Slice", i1 } [ %9, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %11 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", i1 } %10, 0
// CHECK-NEXT:   %12 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", i1 } %10, 1
// CHECK-NEXT:   br i1 %12, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

func main() {
	var i any = MyBytes{}
	_, ok := i.(MyBytes)
	if !ok {
		panic("bad slice")
	}
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
