// LITTEST
package main

type T struct {
	s string
}

func (t T) Invoke() int {
	println("invoke", t.s)
	return 0
}

func (t *T) Method() {}

type T1 int

func (t T1) Invoke() int {
	println("invoke1", t)
	return 1
}

type T2 float64

func (t T2) Invoke() int {
	println("invoke2", t)
	return 2
}

type T3 int8

func (t *T3) Invoke() int {
	println("invoke3", *t)
	return 3
}

type T4 [1]int

func (t T4) Invoke() int {
	println("invoke4", t[0])
	return 4
}

type T5 struct {
	n int
}

func (t T5) Invoke() int {
	println("invoke5", t.n)
	return 5
}

type T6 func() int

func (t T6) Invoke() int {
	println("invoke6", t())
	return 6
}

type I interface {
	Invoke() int
}

func invoke(i I) {
	println(i.Invoke())
}

func main() {
	var t = T{"hello"}
	var t1 = T1(100)
	var t2 = T2(100.1)
	var t3 = T3(127)
	var t4 = T4{200}
	var t5 = T5{300}
	var t6 = T6(func() int { return 400 })

	invoke(t)

	invoke(&t)

	invoke(t1)

	invoke(&t1)

	invoke(t2)

	invoke(&t2)

	invoke(&t3)

	invoke(t4)

	invoke(&t4)

	invoke(t5)

	invoke(&t5)

	invoke(t6)

	invoke(&t6)

	var m M
	var i I = m

	println(i, m)

	m = &t

	invoke(m)

	var a any = T{"world"}

	invoke(a.(I))

	invoke(a.(interface{}).(interface{ Invoke() int }))

	//panic
	//invoke(nil)
}

type M interface {
	Invoke() int
	Method()
}

// CHECK-LABEL: define i64 @main.T.Invoke(%main.T %0){{.*}} {
// CHECK: store %main.T %0, ptr [[T_INVOKE_ADDR:%[0-9]+]]
// CHECK: [[T_INVOKE_FIELD:%[0-9]+]] = getelementptr inbounds %main.T, ptr [[T_INVOKE_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[T_INVOKE_STRING:%[0-9]+]] = load %"{{.*}}String", ptr [[T_INVOKE_FIELD]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[T_INVOKE_STRING]])
// CHECK: ret i64 0

// CHECK-LABEL: define i64 @"main.(*T).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T_NIL]],{{.*}})
// CHECK: %[[T_VALUE:[0-9]+]] = load %main.T, ptr %0
// CHECK: %[[T_RESULT:[0-9]+]] = call i64 @main.T.Invoke(%main.T %[[T_VALUE]])
// CHECK: ret i64 %[[T_RESULT]]

// CHECK-LABEL: define i64 @main.T1.Invoke(i64 %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK: ret i64 1

// CHECK-LABEL: define i64 @"main.(*T1).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T1_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T1_NIL]],{{.*}})
// CHECK: %[[T1_VALUE:[0-9]+]] = load i64, ptr %0
// CHECK: call i64 @main.T1.Invoke(i64 %[[T1_VALUE]])

// CHECK-LABEL: define i64 @main.T2.Invoke(double %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %0)
// CHECK: ret i64 2

// CHECK-LABEL: define i64 @"main.(*T2).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T2_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T2_NIL]],{{.*}})
// CHECK: %[[T2_VALUE:[0-9]+]] = load double, ptr %0
// CHECK: call i64 @main.T2.Invoke(double %[[T2_VALUE]])

// T3 only has a pointer receiver, so its wrapper dereferences the pointer and
// preserves the signed int8 conversion.
// CHECK-LABEL: define i64 @"main.(*T3).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T3_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %[[T3_NIL]])
// CHECK: %[[T3_VALUE:[0-9]+]] = load i8, ptr %0
// CHECK: %[[T3_WIDE:[0-9]+]] = sext i8 %[[T3_VALUE]] to i64
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[T3_WIDE]])

// CHECK-LABEL: define i64 @main.T4.Invoke([1 x i64] %0){{.*}} {
// CHECK: store [1 x i64] %0, ptr [[T4_INVOKE_ADDR:%[0-9]+]]
// CHECK: [[T4_ITEM_PTR:%[0-9]+]] = getelementptr inbounds i64, ptr [[T4_INVOKE_ADDR]], i64 0
// CHECK-NEXT: [[T4_ITEM:%[0-9]+]] = load i64, ptr [[T4_ITEM_PTR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[T4_ITEM]])
// CHECK: ret i64 4

// CHECK-LABEL: define i64 @"main.(*T4).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T4_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T4_NIL]],{{.*}})
// CHECK: %[[T4_VALUE:[0-9]+]] = load [1 x i64], ptr %0
// CHECK: call i64 @main.T4.Invoke([1 x i64] %[[T4_VALUE]])

// CHECK-LABEL: define i64 @main.T5.Invoke(%main.T5 %0){{.*}} {
// CHECK: store %main.T5 %0, ptr [[T5_INVOKE_ADDR:%[0-9]+]]
// CHECK: [[T5_FIELD:%[0-9]+]] = getelementptr inbounds %main.T5, ptr [[T5_INVOKE_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[T5_ITEM:%[0-9]+]] = load i64, ptr [[T5_FIELD]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[T5_ITEM]])
// CHECK: ret i64 5

// CHECK-LABEL: define i64 @"main.(*T5).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T5_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T5_NIL]],{{.*}})
// CHECK: %[[T5_VALUE:[0-9]+]] = load %main.T5, ptr %0
// CHECK: call i64 @main.T5.Invoke(%main.T5 %[[T5_VALUE]])

// CHECK-LABEL: define i64 @main.T6.Invoke(%main.T6 %0){{.*}} {
// CHECK: %[[T6_DATA:[0-9]+]] = extractvalue %main.T6 %0, 1
// CHECK: %[[T6_CODE:[0-9]+]] = extractvalue %main.T6 %0, 0
// CHECK: %[[T6_CALL:__llgo_funcval_code]] = call ptr asm "", "=r,0"(ptr %[[T6_CODE]])
// CHECK: call i64 %[[T6_CALL]](ptr {{(nest|swiftself)}} %[[T6_DATA]])

// CHECK-LABEL: define i64 @"main.(*T6).Invoke"(ptr %0){{.*}} {
// CHECK: %[[T6_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[T6_NIL]],{{.*}})
// CHECK: %[[T6_LOADED:[0-9]+]] = load %main.T6, ptr %0
// CHECK: call i64 @main.T6.Invoke(%main.T6 %[[T6_LOADED]])

// CHECK-LABEL: define void @main.invoke(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: %[[INVOKE_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: %[[INVOKE_TYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK: %[[INVOKE_SLOT:[0-9]+]] = getelementptr ptr, ptr %[[INVOKE_TYPE]], i64 3
// CHECK: %[[INVOKE_METHOD:[0-9]+]] = load ptr, ptr %[[INVOKE_SLOT]]
// CHECK: %[[INVOKE_PAIR0:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[INVOKE_METHOD]], 0
// CHECK: %[[INVOKE_PAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[INVOKE_PAIR0]], ptr %[[INVOKE_DATA]], 1
// CHECK: %[[INVOKE_CALL_DATA:[0-9]+]] = extractvalue { ptr, ptr } %[[INVOKE_PAIR]], 1
// CHECK: %[[INVOKE_CALL_CODE:[0-9]+]] = extractvalue { ptr, ptr } %[[INVOKE_PAIR]], 0
// CHECK: %[[INVOKE_RESULT:[0-9]+]] = call i64 %[[INVOKE_CALL_CODE]](ptr %[[INVOKE_CALL_DATA]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[INVOKE_RESULT]])

// main exercises both value and pointer method sets for every named type.
// Capture the first pair completely, then retain descriptor checks for the
// remaining matrix.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: store %main.T6 { ptr @"main.main$1", ptr null }, ptr %{{[0-9]+}}
// CHECK: %[[T_VALUE_ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T)
// CHECK: %[[T_VALUE_IF0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %[[T_VALUE_ITAB]], 0
// CHECK: %[[T_VALUE_IFACE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %[[T_VALUE_IF0]], ptr %{{[0-9]+}}, 1
// CHECK: call void @main.invoke(%"{{.*}}/runtime/internal/runtime.iface" %[[T_VALUE_IFACE]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T1)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T1")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T2)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T2")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T3")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T4)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T4")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T5)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T5")
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.T6)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.T6")
// CHECK: %[[ASSERT_TYPE:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"
// CHECK: %[[ASSERT_ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr %[[ASSERT_TYPE]])

// CHECK-LABEL: define i64 @"main.main$1"(){{.*}} {
// CHECK: ret i64 400
