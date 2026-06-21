// LITTEST
package main

// CHECK: @6 = private unnamed_addr constant [5 x i8] c"value", align 1
// CHECK: @9 = private unnamed_addr constant [5 x i8] c"error", align 1
// CHECK: @16 = private unnamed_addr constant [22 x i8] c"interface{value() int}", align 1

type M[T interface{}] struct {
	v T
}

type I[T interface{}] interface {
	Value() T
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpinst.demo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Jvxc0PCI_drlfK7S5npMGdZkQLeRkQ_x2e2CifPE6w8", ptr @"*_llgo_{{.*}}/cl/_testgo/tpinst.M[int]")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   %14 = icmp ne i64 %13, 100
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %17, i32 0, i32 0
// CHECK-NEXT:   store double 1.001000e+02, ptr %18, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$2dxw6yZ6V86Spb7J0dTDIoWqg7ba7UDXlAlpJv3-HLk", ptr @"*_llgo_{{.*}}/cl/_testgo/tpinst.M[float64]")
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %19, 0
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %20, ptr %17, 1
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %21)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %21, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 3
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call double %29(ptr %28)
// CHECK-NEXT:   %31 = fcmp une double %30, 1.001000e+02
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %35 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"{{.*}}/cl/_testgo/tpinst.iface$2sV9fFeqOv1SzesvwIdhTqCFzDT8ZX5buKUSAoHNSww", ptr %34)
// CHECK-NEXT:   br i1 %35, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %38 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/tpinst.iface$2sV9fFeqOv1SzesvwIdhTqCFzDT8ZX5buKUSAoHNSww", ptr %34)
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %39, 0
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %40, ptr %38, 1
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %41)
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %41, 0
// CHECK-NEXT:   %44 = getelementptr ptr, ptr %43, i64 3
// CHECK-NEXT:   %45 = load ptr, ptr %44, align 8
// CHECK-NEXT:   %46 = insertvalue { ptr, ptr } undef, ptr %45, 0
// CHECK-NEXT:   %47 = insertvalue { ptr, ptr } %46, ptr %42, 1
// CHECK-NEXT:   %48 = extractvalue { ptr, ptr } %47, 1
// CHECK-NEXT:   %49 = extractvalue { ptr, ptr } %47, 0
// CHECK-NEXT:   %50 = call i64 %49(ptr %48)
// CHECK-NEXT:   %51 = icmp ne i64 %50, 100
// CHECK-NEXT:   br i1 %51, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %34, %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 22 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 5 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpinst.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tpinst.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tpinst.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func demo() {
	var v1 I[int] = &M[int]{100}

	if v1.Value() != 100 {
		panic("error")
	}

	var v2 I[float64] = &M[float64]{100.1}

	if v2.Value() != 100.1 {
		panic("error")
	}

	if v1.(interface{ value() int }).value() != 100 {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpinst.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/tpinst.demo"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	demo()
}

func (pt *M[T]) Value() T {
	return pt.v
}

func (pt *M[T]) value() T {
	return pt.v
}

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tpinst.(*M[int]).Value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tpinst.(*M[int]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/tpinst.(*M[int]).Value"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/tpinst.(*M[int]).Value"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/tpinst.(*M[int]).value"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/tpinst.(*M[int]).value"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce double @"{{.*}}/cl/_testgo/tpinst.(*M[float64]).Value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   ret double %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce double @"{{.*}}/cl/_testgo/tpinst.(*M[float64]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   ret double %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.f64equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.f64equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce double @"__llgo_stub.{{.*}}/cl/_testgo/tpinst.(*M[float64]).Value"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call double @"{{.*}}/cl/_testgo/tpinst.(*M[float64]).Value"(ptr %1)
// CHECK-NEXT:   ret double %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce double @"__llgo_stub.{{.*}}/cl/_testgo/tpinst.(*M[float64]).value"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call double @"{{.*}}/cl/_testgo/tpinst.(*M[float64]).value"(ptr %1)
// CHECK-NEXT:   ret double %2
// CHECK-NEXT: }
