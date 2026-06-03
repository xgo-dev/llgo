// LITTEST
package main

type M[T interface{}] struct {
	v T
}

type I[T interface{}] interface {
	Value() T
}

// CHECK-LABEL: define void @"{{.*}}tpinst.demo"(){{.*}} {
func demo() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: getelementptr inbounds %"{{.*}}tpinst.M[int]", ptr %0, i32 0, i32 0
	// CHECK: store i64 100, ptr %1, align 8
	// CHECK: call ptr @"{{.*}}NewItab"(ptr @"{{.*}}iface{{.*}}", ptr @"*_llgo_{{.*}}tpinst.M[int]")
	var v1 I[int] = &M[int]{100}

	if v1.Value() != 100 {
		// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %{{[0-9]+}})
		// CHECK: getelementptr ptr, ptr %{{[0-9]+}}, i64 3
		// CHECK: call i64 %{{[0-9]+}}(ptr %{{[0-9]+}})
		// CHECK: icmp ne i64 %{{[0-9]+}}, 100
		// CHECK: call ptr @"{{.*}}AllocU"(i64 16)
		// CHECK: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr %{{[0-9]+}}, align 8
		// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" %{{[0-9]+}})
		panic("error")
	}

	// CHECK: call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: getelementptr inbounds %"{{.*}}tpinst.M[float64]", ptr %{{[0-9]+}}, i32 0, i32 0
	// CHECK: store double 1.001000e+02, ptr %{{[0-9]+}}, align 8
	// CHECK: call ptr @"{{.*}}NewItab"(ptr @"{{.*}}iface{{.*}}", ptr @"*_llgo_{{.*}}tpinst.M[float64]")
	var v2 I[float64] = &M[float64]{100.1}

	if v2.Value() != 100.1 {
		// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %{{[0-9]+}})
		// CHECK: getelementptr ptr, ptr %{{[0-9]+}}, i64 3
		// CHECK: call double %{{[0-9]+}}(ptr %{{[0-9]+}})
		// CHECK: fcmp une double %{{[0-9]+}}, 1.001000e+02
		// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" %{{[0-9]+}})
		panic("error")
	}

	if v1.(interface{ value() int }).value() != 100 {
		// CHECK: call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" %{{[0-9]+}})
		// CHECK: call i1 @"{{.*}}Implements"(ptr @"{{.*}}tpinst.iface{{.*}}", ptr %{{[0-9]+}})
		// CHECK: call ptr @"{{.*}}NewItab"(ptr @"{{.*}}tpinst.iface{{.*}}", ptr %{{[0-9]+}})
		// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %{{[0-9]+}})
		// CHECK: getelementptr ptr, ptr %{{[0-9]+}}, i64 3
		// CHECK: call i64 %{{[0-9]+}}(ptr %{{[0-9]+}})
		// CHECK: icmp ne i64 %{{[0-9]+}}, 100
		// CHECK: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 95 }, ptr %{{[0-9]+}}, align 8
		// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" %{{[0-9]+}})
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}tpinst.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}tpinst.demo"()
// CHECK-NEXT:   ret void
func main() {
	demo()
}

// CHECK-LABEL: define linkonce i64 @"{{.*}}tpinst.(*M[int]).Value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
func (pt *M[T]) Value() T {
	return pt.v
}

// CHECK-LABEL: define linkonce i64 @"{{.*}}tpinst.(*M[int]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}tpinst.M[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
func (pt *M[T]) value() T {
	return pt.v
}

// CHECK-LABEL: define linkonce double @"{{.*}}tpinst.(*M[float64]).Value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   ret double %2

// CHECK-LABEL: define linkonce double @"{{.*}}tpinst.(*M[float64]).value"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}tpinst.M[float64]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   ret double %2
