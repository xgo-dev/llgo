// LITTEST
package main

type M[T interface{}] struct {
	v T
}

type I[T interface{}] interface {
	Value() T
}

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

func main() {
	demo()
}

func (pt *M[T]) Value() T {
	return pt.v
}

func (pt *M[T]) value() T {
	return pt.v
}

// CHECK-LABEL: define void @main.demo(){{.*}} {
// CHECK: %[[INT_OBJ:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: %[[INT_ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.M[int]")
// CHECK: %[[INT_IFACE0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %[[INT_ITAB]], 0
// CHECK: %[[INT_IFACE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %[[INT_IFACE0]], ptr %[[INT_OBJ]], 1
// CHECK: %[[INT_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[INT_IFACE]])
// CHECK: %[[INT_IFACE_TYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %[[INT_IFACE]], 0
// CHECK: %[[INT_METHOD_SLOT:[0-9]+]] = getelementptr ptr, ptr %[[INT_IFACE_TYPE]], i64 3
// CHECK: %[[INT_METHOD:[0-9]+]] = load ptr, ptr %[[INT_METHOD_SLOT]]
// CHECK: %[[INT_CALL0:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[INT_METHOD]], 0
// CHECK: %[[INT_CALL:[0-9]+]] = insertvalue { ptr, ptr } %[[INT_CALL0]], ptr %[[INT_DATA]], 1
// CHECK: %[[INT_CALL_DATA:[0-9]+]] = extractvalue { ptr, ptr } %[[INT_CALL]], 1
// CHECK: %[[INT_CALL_CODE:[0-9]+]] = extractvalue { ptr, ptr } %[[INT_CALL]], 0
// CHECK: %[[INT_RESULT:[0-9]+]] = call i64 %[[INT_CALL_CODE]](ptr %[[INT_CALL_DATA]])
// CHECK: %[[INT_BAD:[0-9]+]] = icmp ne i64 %[[INT_RESULT]], 100
// CHECK: br i1 %[[INT_BAD]]
// CHECK: %[[FLOAT_OBJ:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: %[[FLOAT_ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.M[float64]")
// CHECK: %[[FLOAT_IFACE0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %[[FLOAT_ITAB]], 0
// CHECK: %[[FLOAT_IFACE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %[[FLOAT_IFACE0]], ptr %[[FLOAT_OBJ]], 1
// CHECK: %[[FLOAT_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[FLOAT_IFACE]])
// CHECK: %[[FLOAT_IFACE_TYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %[[FLOAT_IFACE]], 0
// CHECK: getelementptr ptr, ptr %[[FLOAT_IFACE_TYPE]], i64 3
// CHECK: insertvalue { ptr, ptr } %{{[0-9]+}}, ptr %[[FLOAT_DATA]], 1
// CHECK: %[[FLOAT_RESULT:[0-9]+]] = call double %{{[0-9]+}}(ptr %{{[0-9]+}})
// CHECK: %[[FLOAT_BAD:[0-9]+]] = fcmp une double %[[FLOAT_RESULT]], 1.001000e+02
// CHECK: br i1 %[[FLOAT_BAD]]
// CHECK: %[[INT_TYPE:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %[[INT_IFACE]])
// CHECK: %[[IMPLEMENTS:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"{{.*}}/cl/_testgo/tpinst.iface${{[-A-Za-z0-9_]+}}", ptr %[[INT_TYPE]])
// CHECK: br i1 %[[IMPLEMENTS]]
// CHECK: %[[PRIVATE_ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/tpinst.iface${{[-A-Za-z0-9_]+}}", ptr %[[INT_TYPE]])

// CHECK-LABEL: define linkonce i64 @"main.(*M[int]).Value"(ptr %0){{.*}} {
// CHECK: [[INT_VALUE_FIELD:%[0-9]+]] = getelementptr inbounds %"main.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[INT_VALUE:%[0-9]+]] = load i64, ptr [[INT_VALUE_FIELD]]
// CHECK-NEXT: ret i64 [[INT_VALUE]]

// CHECK-LABEL: define linkonce i64 @"main.(*M[int]).value"(ptr %0){{.*}} {
// CHECK: [[INT_PRIVATE_FIELD:%[0-9]+]] = getelementptr inbounds %"main.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[INT_PRIVATE_VALUE:%[0-9]+]] = load i64, ptr [[INT_PRIVATE_FIELD]]
// CHECK-NEXT: ret i64 [[INT_PRIVATE_VALUE]]

// CHECK-LABEL: define linkonce double @"main.(*M[float64]).Value"(ptr %0){{.*}} {
// CHECK: [[FLOAT_VALUE_FIELD:%[0-9]+]] = getelementptr inbounds %"main.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[FLOAT_VALUE:%[0-9]+]] = load double, ptr [[FLOAT_VALUE_FIELD]]
// CHECK-NEXT: ret double [[FLOAT_VALUE]]

// CHECK-LABEL: define linkonce double @"main.(*M[float64]).value"(ptr %0){{.*}} {
// CHECK: [[FLOAT_PRIVATE_FIELD:%[0-9]+]] = getelementptr inbounds %"main.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[FLOAT_PRIVATE_VALUE:%[0-9]+]] = load double, ptr [[FLOAT_PRIVATE_FIELD]]
// CHECK-NEXT: ret double [[FLOAT_PRIVATE_VALUE]]
