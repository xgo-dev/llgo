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
// CHECK: %[[INT_IFACE:[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %[[INT_OBJ:[0-9]+]], 1
// CHECK: %[[INT_RESULT:[0-9]+]] = call i64 @"main.(*M[int]).Value"(ptr %[[INT_OBJ]])
// CHECK: %[[INT_BAD:[0-9]+]] = icmp ne i64 %[[INT_RESULT]], 100
// CHECK: br i1 %[[INT_BAD]]
// CHECK: %[[FLOAT_IFACE:[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %[[FLOAT_OBJ:[0-9]+]], 1
// CHECK: %[[FLOAT_RESULT:[0-9]+]] = call double @"main.(*M[float64]).Value"(ptr %[[FLOAT_OBJ]])
// CHECK: %[[FLOAT_BAD:[0-9]+]] = fcmp une double %[[FLOAT_RESULT]], 1.001000e+02
// CHECK: br i1 %[[FLOAT_BAD]]
// CHECK: %[[INT_PAYLOAD:[0-9]+]] = extractvalue %"{{.*}}iface" %[[INT_IFACE]], 1
// CHECK: insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %[[INT_PAYLOAD]], 1
// CHECK: %[[PRIVATE_RESULT:[0-9]+]] = call i64 @"main.(*M[int]).value"(ptr %[[INT_OBJ]])
// CHECK: %[[PRIVATE_BAD:[0-9]+]] = icmp ne i64 %[[PRIVATE_RESULT]], 100
// CHECK: br i1 %[[PRIVATE_BAD]]

// CHECK-LABEL: define linkonce i64 @"main.(*M[int]).Value"(ptr %0){{.*}} {
// CHECK: [[INT_VALUE_FIELD:%[0-9]+]] = getelementptr inbounds nuw %"main.M[int]", ptr %0, i32 0, i32 0
// CHECK: [[INT_VALUE:%[0-9]+]] = load i64, ptr [[INT_VALUE_FIELD]]
// CHECK-NEXT: ret i64 [[INT_VALUE]]

// CHECK-LABEL: define linkonce i64 @"main.(*M[int]).value"(ptr %0){{.*}} {
// CHECK: [[INT_PRIVATE_FIELD:%[0-9]+]] = getelementptr inbounds nuw %"main.M[int]", ptr %0, i32 0, i32 0
// CHECK: [[INT_PRIVATE_VALUE:%[0-9]+]] = load i64, ptr [[INT_PRIVATE_FIELD]]
// CHECK-NEXT: ret i64 [[INT_PRIVATE_VALUE]]

// CHECK-LABEL: define linkonce double @"main.(*M[float64]).Value"(ptr %0){{.*}} {
// CHECK: [[FLOAT_VALUE_FIELD:%[0-9]+]] = getelementptr inbounds nuw %"main.M[float64]", ptr %0, i32 0, i32 0
// CHECK: [[FLOAT_VALUE:%[0-9]+]] = load double, ptr [[FLOAT_VALUE_FIELD]]
// CHECK-NEXT: ret double [[FLOAT_VALUE]]

// CHECK-LABEL: define linkonce double @"main.(*M[float64]).value"(ptr %0){{.*}} {
// CHECK: [[FLOAT_PRIVATE_FIELD:%[0-9]+]] = getelementptr inbounds nuw %"main.M[float64]", ptr %0, i32 0, i32 0
// CHECK: [[FLOAT_PRIVATE_VALUE:%[0-9]+]] = load double, ptr [[FLOAT_PRIVATE_FIELD]]
// CHECK-NEXT: ret double [[FLOAT_PRIVATE_VALUE]]
