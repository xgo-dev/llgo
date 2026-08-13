// LITTEST
package main

// Tests of interface conversions and type assertions.

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

// CHECK-LABEL: define void @main.C1.f(%main.C1 %0){{.*}} {
// CHECK: ret void

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

// CHECK-LABEL: define void @"main.(*C1).f"(ptr %0){{.*}} {
// CHECK: [[C1_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[C1_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: [[C1_DEREF_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 [[C1_DEREF_NIL]])
// CHECK: call void @main.C1.f(%main.C1 zeroinitializer)

// CHECK-LABEL: define void @main.C2.f(%main.C2 %0){{.*}} {
// CHECK: ret void

// CHECK-LABEL: define void @main.C2.g(%main.C2 %0){{.*}} {
// CHECK: ret void

// CHECK-LABEL: define void @"main.(*C2).f"(ptr %0){{.*}} {
// CHECK: [[C2_F_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[C2_F_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: call void @main.C2.f(%main.C2 zeroinitializer)

// CHECK-LABEL: define void @"main.(*C2).g"(ptr %0){{.*}} {
// CHECK: [[C2_G_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[C2_G_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: call void @main.C2.g(%main.C2 zeroinitializer)

// CHECK-LABEL: define void @main.main(){{.*}} {
// A nil empty-interface assertion is known false; non-empty nil assertions test IfaceType.
// CHECK: br i1 false, label %{{.*}}, label %{{.*}}
// CHECK: [[NIL_I1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" zeroinitializer)
// CHECK: [[NIL_I1_OK:%.*]] = icmp ne ptr [[NIL_I1_TYPE]], null
// CHECK: br i1 [[NIL_I1_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[NIL_I2_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" zeroinitializer)
// CHECK: [[NIL_I2_OK:%.*]] = icmp ne ptr [[NIL_I2_TYPE]], null
// CHECK: br i1 [[NIL_I2_OK]], label %{{.*}}, label %{{.*}}
// Converting nil I2 to I1 derives the target itab from the nil dynamic type.
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" zeroinitializer)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" zeroinitializer)
// CHECK: [[NIL_DYNAMIC_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" zeroinitializer)
// CHECK: [[NIL_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface${{[-A-Za-z0-9_]+}}", ptr [[NIL_DYNAMIC_TYPE]])
// C1 implements I0 and I1, but its dynamic type is tested against I2.
// CHECK: [[C1_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK: store %main.C1 zeroinitializer, ptr [[C1_DATA]]
// CHECK: [[C1_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.C1)
// CHECK: [[C1_IFACE_0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[C1_ITAB]], 0
// CHECK: [[C1_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[C1_IFACE_0]], ptr [[C1_DATA]], 1
// CHECK: [[C1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C1_IFACE]])
// CHECK: [[C1_IS_I0:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.I0, ptr [[C1_TYPE]])
// CHECK: br i1 [[C1_IS_I0]], label %{{.*}}, label %{{.*}}
// CHECK: [[C1_I1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C1_IFACE]])
// CHECK: [[C1_IS_I1:%.*]] = icmp ne ptr [[C1_I1_TYPE]], null
// CHECK: br i1 [[C1_IS_I1]], label %{{.*}}, label %{{.*}}
// CHECK: [[C1_I2_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C1_IFACE]])
// CHECK: [[C1_IS_I2:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.I2, ptr [[C1_I2_TYPE]])
// CHECK: br i1 [[C1_IS_I2]], label %{{.*}}, label %{{.*}}
// C2 follows the same assertion pipeline, including its positive I2 test.
// CHECK: [[C2_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK: store %main.C2 zeroinitializer, ptr [[C2_DATA]]
// CHECK: [[C2_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.C2)
// CHECK: [[C2_IFACE_0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[C2_ITAB]], 0
// CHECK: [[C2_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[C2_IFACE_0]], ptr [[C2_DATA]], 1
// CHECK: [[C2_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C2_IFACE]])
// CHECK: [[C2_IS_I0:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.I0, ptr [[C2_TYPE]])
// CHECK: br i1 [[C2_IS_I0]], label %{{.*}}, label %{{.*}}
// CHECK: [[C2_I1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C2_IFACE]])
// CHECK: [[C2_IS_I1:%.*]] = icmp ne ptr [[C2_I1_TYPE]], null
// CHECK: br i1 [[C2_IS_I1]], label %{{.*}}, label %{{.*}}
// CHECK: [[C2_I2_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[C2_IFACE]])
// CHECK: [[C2_IS_I2:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.I2, ptr [[C2_I2_TYPE]])
// CHECK: br i1 [[C2_IS_I2]], label %{{.*}}, label %{{.*}}
// The final conversions rebuild an eface from the same C1 payload before nil comparison.
// CHECK: [[FINAL_C1_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.C1)
// CHECK: [[FINAL_C1_IFACE_0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[FINAL_C1_ITAB]], 0
// CHECK: [[FINAL_C1_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[FINAL_C1_IFACE_0]], ptr [[FINAL_C1_DATA:%.*]], 1
// CHECK: [[FINAL_C1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[FINAL_C1_IFACE]])
// CHECK: [[FINAL_C1_PAYLOAD:%.*]] = extractvalue %"{{.*}}iface" [[FINAL_C1_IFACE]], 1
// CHECK: [[FINAL_EFACE_0:%.*]] = insertvalue %"{{.*}}eface" undef, ptr [[FINAL_C1_TYPE]], 0
// CHECK: [[FINAL_EFACE:%.*]] = insertvalue %"{{.*}}eface" [[FINAL_EFACE_0]], ptr [[FINAL_C1_PAYLOAD]], 1
// CHECK: [[I0_IS_NIL:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}eface" [[FINAL_EFACE]], %"{{.*}}eface" zeroinitializer)
// CHECK: br i1 [[I0_IS_NIL]], label %{{.*}}, label %{{.*}}
// CHECK: [[FINAL_I1_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[FINAL_C1_IFACE]])
// CHECK: [[FINAL_I1_PAYLOAD:%.*]] = extractvalue %"{{.*}}iface" [[FINAL_C1_IFACE]], 1
// CHECK: [[FINAL_I1_EFACE_0:%.*]] = insertvalue %"{{.*}}eface" undef, ptr [[FINAL_I1_TYPE]], 0
// CHECK: [[FINAL_I1_EFACE:%.*]] = insertvalue %"{{.*}}eface" [[FINAL_I1_EFACE_0]], ptr [[FINAL_I1_PAYLOAD]], 1
// CHECK: [[I1_IS_NIL:%.*]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}eface" [[FINAL_I1_EFACE]], %"{{.*}}eface" %{{.*}})
// CHECK: br i1 [[I1_IS_NIL]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" { ptr @{{.*}}, i64 4 })
// CHECK: ret void
// The successful C2-to-I2 assertion preserves C2's payload under a target itab.
// CHECK: extractvalue %"{{.*}}iface" [[C2_IFACE]], 1
// CHECK: [[C2_I2_PAYLOAD:%.*]] = extractvalue %"{{.*}}iface" [[C2_IFACE]], 1
// CHECK: [[C2_TARGET_ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/ifaceconv.iface${{[-A-Za-z0-9_]+}}", ptr [[C2_I2_TYPE]])
// CHECK: [[C2_TARGET_IFACE_0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[C2_TARGET_ITAB]], 0
// CHECK: [[C2_TARGET_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[C2_TARGET_IFACE_0]], ptr [[C2_I2_PAYLOAD]], 1
// CHECK: insertvalue { %"{{.*}}iface", i1 } %{{.*}}, i1 true, 1
