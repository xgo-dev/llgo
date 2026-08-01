// LITTEST
package main

type Point struct {
	x float64
	y float64
}

type MyPoint = Point

// CHECK-LABEL: define void @"main.(*Point).Move"(ptr %0, double %1, double %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = load double, ptr %3, align 8
// CHECK-NEXT:   %6 = fadd double %5, %1
// CHECK-NEXT:   %7 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = load double, ptr %8, align 8
// CHECK-NEXT:   %11 = fadd double %10, %2
// CHECK-NEXT:   %12 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %11, ptr %12, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (p *MyPoint) Move(dx, dy float64) {
	p.x += dx
	p.y += dy
}

// CHECK-LABEL: define void @"main.(*Point).Scale"(ptr %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = load double, ptr %2, align 8
// CHECK-NEXT:   %5 = fmul double %4, %1
// CHECK-NEXT:   %6 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %5, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = load double, ptr %7, align 8
// CHECK-NEXT:   %10 = fmul double %9, %1
// CHECK-NEXT:   %11 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %10, ptr %11, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (p *Point) Scale(factor float64) {
	p.x *= factor
	p.y *= factor
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK: call void @"main.(*Point).Scale"(ptr %0, double 2.000000e+00)
	// CHECK: call void @"main.(*Point).Move"(ptr %0, double 3.000000e+00, double 4.000000e+00)
	// CHECK: call void @"{{.*}}PrintFloat"(double %4)
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double %6)
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK-NEXT: ret void
	pt := &MyPoint{1, 2}
	pt.Scale(2)
	pt.Move(3, 4)
	println(pt.x, pt.y)
}
