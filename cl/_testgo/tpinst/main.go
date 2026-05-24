// LITTEST
package main

// CHECK-LINE: @6 = private unnamed_addr constant [5 x i8] c"value", align 1
// CHECK-LINE: @9 = private unnamed_addr constant [5 x i8] c"error", align 1
// CHECK-LINE: @16 = private unnamed_addr constant [22 x i8] c"interface{value() int}", align 1

type M[T interface{}] struct {
	v T
}

type I[T interface{}] interface {
	Value() T
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpinst.demo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Jvxc0PCI_drlfK7S5npMGdZkQLeRkQ_x2e2CifPE6w8", ptr @"*_llgo_{{.*}}/cl/_testgo/tpinst.M[int]")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %0, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %5)
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %5, 0
// CHECK-NEXT:   %8 = getelementptr ptr, ptr %7, i64 3
// CHECK-NEXT:   %9 = load ptr, ptr %8, align 8
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } undef, ptr %9, 0
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } %10, ptr %6, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %11, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %11, 0
// CHECK-NEXT:   %14 = call i64 %13(ptr %12)
// CHECK-NEXT:   %15 = icmp ne i64 %14, 100
// CHECK-NEXT:   br i1 %15, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %16, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %19 = icmp eq ptr %18, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %19)
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %18, i32 0, i32 0
// CHECK-NEXT:   store double 1.001000e+02, ptr %20, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$2dxw6yZ6V86Spb7J0dTDIoWqg7ba7UDXlAlpJv3-HLk", ptr @"*_llgo_{{.*}}/cl/_testgo/tpinst.M[float64]")
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %21, 0
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %22, ptr %18, 1
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %23, 0
// CHECK-NEXT:   %26 = getelementptr ptr, ptr %25, i64 3
// CHECK-NEXT:   %27 = load ptr, ptr %26, align 8
// CHECK-NEXT:   %28 = insertvalue { ptr, ptr } undef, ptr %27, 0
// CHECK-NEXT:   %29 = insertvalue { ptr, ptr } %28, ptr %24, 1
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %29, 1
// CHECK-NEXT:   %31 = extractvalue { ptr, ptr } %29, 0
// CHECK-NEXT:   %32 = call double %31(ptr %30)
// CHECK-NEXT:   %33 = fcmp une double %32, 1.001000e+02
// CHECK-NEXT:   br i1 %33, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %5)
// CHECK-NEXT:   %37 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"{{.*}}/cl/_testgo/tpinst.iface$2sV9fFeqOv1SzesvwIdhTqCFzDT8ZX5buKUSAoHNSww", ptr %36)
// CHECK-NEXT:   br i1 %37, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %38, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %39)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %40 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %5, 1
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/tpinst.iface$2sV9fFeqOv1SzesvwIdhTqCFzDT8ZX5buKUSAoHNSww", ptr %36)
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %41, 0
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %42, ptr %40, 1
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %43)
// CHECK-NEXT:   %45 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %43, 0
// CHECK-NEXT:   %46 = getelementptr ptr, ptr %45, i64 3
// CHECK-NEXT:   %47 = load ptr, ptr %46, align 8
// CHECK-NEXT:   %48 = insertvalue { ptr, ptr } undef, ptr %47, 0
// CHECK-NEXT:   %49 = insertvalue { ptr, ptr } %48, ptr %44, 1
// CHECK-NEXT:   %50 = extractvalue { ptr, ptr } %49, 1
// CHECK-NEXT:   %51 = extractvalue { ptr, ptr } %49, 0
// CHECK-NEXT:   %52 = call i64 %51(ptr %50)
// CHECK-NEXT:   %53 = icmp ne i64 %52, 100
// CHECK-NEXT:   br i1 %53, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %36, %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 22 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 5 })
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
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tpinst.(*M[int]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   ret i64 %3
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
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load double, ptr %2, align 8
// CHECK-NEXT:   ret double %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce double @"{{.*}}/cl/_testgo/tpinst.(*M[float64]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load double, ptr %2, align 8
// CHECK-NEXT:   ret double %3
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
