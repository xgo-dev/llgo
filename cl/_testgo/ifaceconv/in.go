// LITTEST
package main

// Tests of interface conversions and type assertions.

// CHECK: {{^}}@0 = private unnamed_addr constant [46 x i8] c"{{.*}}/cl/_testgo/ifaceconv.C1", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [1 x i8] c"f", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [46 x i8] c"{{.*}}/cl/_testgo/ifaceconv.C2", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [1 x i8] c"g", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [21 x i8] c"nil i0.(I0) succeeded", align 1{{$}}
// CHECK: {{^}}@11 = private unnamed_addr constant [21 x i8] c"nil i1.(I1) succeeded", align 1{{$}}
// CHECK: {{^}}@14 = private unnamed_addr constant [21 x i8] c"nil i2.(I2) succeeded", align 1{{$}}
// CHECK: {{^}}@17 = private unnamed_addr constant [17 x i8] c"C1 i1.(I0) failed", align 1{{$}}
// CHECK: {{^}}@19 = private unnamed_addr constant [17 x i8] c"C1 i1.(I1) failed", align 1{{$}}
// CHECK: {{^}}@20 = private unnamed_addr constant [20 x i8] c"C1 i1.(I2) succeeded", align 1{{$}}
// CHECK: {{^}}@22 = private unnamed_addr constant [17 x i8] c"C2 i1.(I0) failed", align 1{{$}}
// CHECK: {{^}}@23 = private unnamed_addr constant [17 x i8] c"C2 i1.(I1) failed", align 1{{$}}
// CHECK: {{^}}@24 = private unnamed_addr constant [17 x i8] c"C2 i1.(I2) failed", align 1{{$}}
// CHECK: {{^}}@25 = private unnamed_addr constant [17 x i8] c"C1 I0(i1) was nil", align 1{{$}}
// CHECK: {{^}}@26 = private unnamed_addr constant [17 x i8] c"C1 I1(i1) was nil", align 1{{$}}
// CHECK: {{^}}@27 = private unnamed_addr constant [4 x i8] c"pass", align 1{{$}}

type I0 interface {
}
type I1 interface {
	f()
}
type I2 interface {
	f()
	g()
}

type C0 struct{}
type C1 struct{}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.C1.f"(%"{{.*}}/cl/_testgo/ifaceconv.C1" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func (C1) f() {}

type C2 struct{}

func (C2) f() {}
func (C2) g() {}

func main() {
	var i0 I0
	var i1 I1
	var i2 I2

	// Nil always causes a type assertion to fail, even to the
	// same type.
	if _, ok := i0.(I0); ok {
		panic("nil i0.(I0) succeeded")
	}
	if _, ok := i1.(I1); ok {
		panic("nil i1.(I1) succeeded")
	}
	if _, ok := i2.(I2); ok {
		panic("nil i2.(I2) succeeded")
	}

	// Conversions can't fail, even with nil.
	_ = I0(i0)

	_ = I0(i1)
	_ = I1(i1)

	_ = I0(i2)
	_ = I1(i2)
	_ = I2(i2)

	// Non-nil type assertions pass or fail based on the concrete type.
	i1 = C1{}
	if _, ok := i1.(I0); !ok {
		panic("C1 i1.(I0) failed")
	}
	if _, ok := i1.(I1); !ok {
		panic("C1 i1.(I1) failed")
	}
	if _, ok := i1.(I2); ok {
		panic("C1 i1.(I2) succeeded")
	}

	i1 = C2{}
	if _, ok := i1.(I0); !ok {
		panic("C2 i1.(I0) failed")
	}
	if _, ok := i1.(I1); !ok {
		panic("C2 i1.(I1) failed")
	}
	if _, ok := i1.(I2); !ok {
		panic("C2 i1.(I2) failed")
	}

	// Conversions can't fail.
	i1 = C1{}
	if I0(i1) == nil {
		panic("C1 I0(i1) was nil")
	}
	if I1(i1) == nil {
		panic("C1 I1(i1) was nil")
	}

	println("pass")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.(*C1).f"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 46 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 1 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/ifaceconv.C1.f"(%"{{.*}}/cl/_testgo/ifaceconv.C1" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.C2.f"(%"{{.*}}/cl/_testgo/ifaceconv.C2" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.C2.g"(%"{{.*}}/cl/_testgo/ifaceconv.C2" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.(*C2).f"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 46 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 1 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/ifaceconv.C2.f"(%"{{.*}}/cl/_testgo/ifaceconv.C2" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.(*C2).g"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 46 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 1 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/ifaceconv.C2.g"(%"{{.*}}/cl/_testgo/ifaceconv.C2" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/ifaceconv.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/ifaceconv.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/ifaceconv.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 false, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_25
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 21 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_25
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %3 = icmp ne ptr %2, null
// CHECK-NEXT:   br i1 %3, label %_llgo_26, label %_llgo_27
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_28
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 21 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_28
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %7 = icmp ne ptr %6, null
// CHECK-NEXT:   br i1 %7, label %_llgo_29, label %_llgo_30
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_31
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 21 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_31
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %11, ptr null, 1
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %14, ptr null, 1
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$brpgdLtIeRlPi8QUoTgPCXzlehUkncg7v9aITo-GsF4", ptr %16)
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %17, 0
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %18, ptr null, 1
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceconv.C1" zeroinitializer, ptr %20, align 1
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$brpgdLtIeRlPi8QUoTgPCXzlehUkncg7v9aITo-GsF4", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.C1")
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %21, 0
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %22, ptr %20, 1
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   %25 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.I0", ptr %24)
// CHECK-NEXT:   br i1 %25, label %_llgo_32, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_34
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 17 }, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_34
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   %29 = icmp ne ptr %28, null
// CHECK-NEXT:   br i1 %29, label %_llgo_35, label %_llgo_36
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_37
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 17 }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_37
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   %33 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.I2", ptr %32)
// CHECK-NEXT:   br i1 %33, label %_llgo_38, label %_llgo_39
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_40
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 20 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_40
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceconv.C2" zeroinitializer, ptr %36, align 1
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$brpgdLtIeRlPi8QUoTgPCXzlehUkncg7v9aITo-GsF4", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.C2")
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %37, 0
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %38, ptr %36, 1
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %39)
// CHECK-NEXT:   %41 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.I0", ptr %40)
// CHECK-NEXT:   br i1 %41, label %_llgo_41, label %_llgo_42
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_43
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 17 }, ptr %42, align 8
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %42, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %43)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_43
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %39)
// CHECK-NEXT:   %45 = icmp ne ptr %44, null
// CHECK-NEXT:   br i1 %45, label %_llgo_44, label %_llgo_45
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_46
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 17 }, ptr %46, align 8
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %46, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %47)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_46
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %39)
// CHECK-NEXT:   %49 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.I2", ptr %48)
// CHECK-NEXT:   br i1 %49, label %_llgo_47, label %_llgo_48
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_49
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 17 }, ptr %50, align 8
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %50, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %51)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_49
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/ifaceconv.C1" zeroinitializer, ptr %52, align 1
// CHECK-NEXT:   %53 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$brpgdLtIeRlPi8QUoTgPCXzlehUkncg7v9aITo-GsF4", ptr @"_llgo_{{.*}}/cl/_testgo/ifaceconv.C1")
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %53, 0
// CHECK-NEXT:   %55 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %54, ptr %52, 1
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %55)
// CHECK-NEXT:   %57 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %55, 1
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %56, 0
// CHECK-NEXT:   %59 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %58, ptr %57, 1
// CHECK-NEXT:   %60 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %59, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   br i1 %60, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %61 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 17 }, ptr %61, align 8
// CHECK-NEXT:   %62 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %61, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %62)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %55)
// CHECK-NEXT:   %64 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %55, 1
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %63, 0
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %65, ptr %64, 1
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %67, 0
// CHECK-NEXT:   %69 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %68, ptr null, 1
// CHECK-NEXT:   %70 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %66, %"{{.*}}/runtime/internal/runtime.eface" %69)
// CHECK-NEXT:   br i1 %70, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %71 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 17 }, ptr %71, align 8
// CHECK-NEXT:   %72 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %71, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %72)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_24, %_llgo_23
// CHECK-NEXT:   %73 = phi { %"{{.*}}/runtime/internal/runtime.eface", i1 } [ { %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer, i1 true }, %_llgo_23 ], [ zeroinitializer, %_llgo_24 ]
// CHECK-NEXT:   %74 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %73, 0
// CHECK-NEXT:   %75 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %73, 1
// CHECK-NEXT:   br i1 %75, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_27, %_llgo_26
// CHECK-NEXT:   %76 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ { %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, i1 true }, %_llgo_26 ], [ zeroinitializer, %_llgo_27 ]
// CHECK-NEXT:   %77 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %76, 0
// CHECK-NEXT:   %78 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %76, 1
// CHECK-NEXT:   br i1 %78, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_4
// CHECK-NEXT:   br label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_4
// CHECK-NEXT:   br label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_30, %_llgo_29
// CHECK-NEXT:   %79 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ { %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, i1 true }, %_llgo_29 ], [ zeroinitializer, %_llgo_30 ]
// CHECK-NEXT:   %80 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %79, 0
// CHECK-NEXT:   %81 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %79, 1
// CHECK-NEXT:   br i1 %81, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %82 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %23, 1
// CHECK-NEXT:   %83 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %24, 0
// CHECK-NEXT:   %84 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %83, ptr %82, 1
// CHECK-NEXT:   %85 = insertvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } undef, %"{{.*}}/runtime/internal/runtime.eface" %84, 0
// CHECK-NEXT:   %86 = insertvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %85, i1 true, 1
// CHECK-NEXT:   br label %_llgo_34
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_6
// CHECK-NEXT:   br label %_llgo_34
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_34:                                         ; preds = %_llgo_33, %_llgo_32
// CHECK-NEXT:   %87 = phi { %"{{.*}}/runtime/internal/runtime.eface", i1 } [ %86, %_llgo_32 ], [ zeroinitializer, %_llgo_33 ]
// CHECK-NEXT:   %88 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %87, 0
// CHECK-NEXT:   %89 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %87, 1
// CHECK-NEXT:   br i1 %89, label %_llgo_8, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_35:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %90 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %23, 0
// CHECK-NEXT:   %91 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %90, i1 true, 1
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_36:                                         ; preds = %_llgo_8
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_37:                                         ; preds = %_llgo_36, %_llgo_35
// CHECK-NEXT:   %92 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %91, %_llgo_35 ], [ zeroinitializer, %_llgo_36 ]
// CHECK-NEXT:   %93 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %92, 0
// CHECK-NEXT:   %94 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %92, 1
// CHECK-NEXT:   br i1 %94, label %_llgo_10, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_38:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %95 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %23, 1
// CHECK-NEXT:   %96 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$gZBF8fFlqIMZ9M6lT2VWPyc3eu5Co6j0WoKGIEgDPAw", ptr %32)
// CHECK-NEXT:   %97 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %96, 0
// CHECK-NEXT:   %98 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %97, ptr %95, 1
// CHECK-NEXT:   %99 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %98, 0
// CHECK-NEXT:   %100 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %99, i1 true, 1
// CHECK-NEXT:   br label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_39:                                         ; preds = %_llgo_10
// CHECK-NEXT:   br label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_40:                                         ; preds = %_llgo_39, %_llgo_38
// CHECK-NEXT:   %101 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %100, %_llgo_38 ], [ zeroinitializer, %_llgo_39 ]
// CHECK-NEXT:   %102 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %101, 0
// CHECK-NEXT:   %103 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %101, 1
// CHECK-NEXT:   br i1 %103, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_41:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %104 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %39, 1
// CHECK-NEXT:   %105 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %40, 0
// CHECK-NEXT:   %106 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %105, ptr %104, 1
// CHECK-NEXT:   %107 = insertvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } undef, %"{{.*}}/runtime/internal/runtime.eface" %106, 0
// CHECK-NEXT:   %108 = insertvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %107, i1 true, 1
// CHECK-NEXT:   br label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_42:                                         ; preds = %_llgo_12
// CHECK-NEXT:   br label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_43:                                         ; preds = %_llgo_42, %_llgo_41
// CHECK-NEXT:   %109 = phi { %"{{.*}}/runtime/internal/runtime.eface", i1 } [ %108, %_llgo_41 ], [ zeroinitializer, %_llgo_42 ]
// CHECK-NEXT:   %110 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %109, 0
// CHECK-NEXT:   %111 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %109, 1
// CHECK-NEXT:   br i1 %111, label %_llgo_14, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_44:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %112 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %39, 0
// CHECK-NEXT:   %113 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %112, i1 true, 1
// CHECK-NEXT:   br label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_45:                                         ; preds = %_llgo_14
// CHECK-NEXT:   br label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_46:                                         ; preds = %_llgo_45, %_llgo_44
// CHECK-NEXT:   %114 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %113, %_llgo_44 ], [ zeroinitializer, %_llgo_45 ]
// CHECK-NEXT:   %115 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %114, 0
// CHECK-NEXT:   %116 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %114, 1
// CHECK-NEXT:   br i1 %116, label %_llgo_16, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_47:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %117 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %39, 1
// CHECK-NEXT:   %118 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface$gZBF8fFlqIMZ9M6lT2VWPyc3eu5Co6j0WoKGIEgDPAw", ptr %48)
// CHECK-NEXT:   %119 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %118, 0
// CHECK-NEXT:   %120 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %119, ptr %117, 1
// CHECK-NEXT:   %121 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %120, 0
// CHECK-NEXT:   %122 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %121, i1 true, 1
// CHECK-NEXT:   br label %_llgo_49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_48:                                         ; preds = %_llgo_16
// CHECK-NEXT:   br label %_llgo_49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_49:                                         ; preds = %_llgo_48, %_llgo_47
// CHECK-NEXT:   %123 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %122, %_llgo_47 ], [ zeroinitializer, %_llgo_48 ]
// CHECK-NEXT:   %124 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %123, 0
// CHECK-NEXT:   %125 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %123, 1
// CHECK-NEXT:   br i1 %125, label %_llgo_18, label %_llgo_17
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal0"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal0"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.(*C1).f"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.(*C1).f"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.C1.f"(ptr %0, %"{{.*}}/cl/_testgo/ifaceconv.C1" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.C1.f"(%"{{.*}}/cl/_testgo/ifaceconv.C1" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.(*C2).f"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.(*C2).f"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.(*C2).g"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.(*C2).g"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.C2.f"(ptr %0, %"{{.*}}/cl/_testgo/ifaceconv.C2" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.C2.f"(%"{{.*}}/cl/_testgo/ifaceconv.C2" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/ifaceconv.C2.g"(ptr %0, %"{{.*}}/cl/_testgo/ifaceconv.C2" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/ifaceconv.C2.g"(%"{{.*}}/cl/_testgo/ifaceconv.C2" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
