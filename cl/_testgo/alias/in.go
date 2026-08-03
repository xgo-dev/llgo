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
// CHECK-NEXT:   br i1 %4, label %5, label %6
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 6:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %7 = load double, ptr %3, align 8
// CHECK-NEXT:   %8 = fadd double %7, %1
// CHECK-NEXT:   %9 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %8, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %11 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %11, label %12, label %13
// CHECK-EMPTY:
// CHECK-NEXT: 12:                                               ; preds = %6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 13:                                               ; preds = %6
// CHECK-NEXT:   %14 = load double, ptr %10, align 8
// CHECK-NEXT:   %15 = fadd double %14, %2
// CHECK-NEXT:   %16 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %15, ptr %16, align 8
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
// CHECK-NEXT:   br i1 %3, label %4, label %5
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %6 = load double, ptr %2, align 8
// CHECK-NEXT:   %7 = fmul double %6, %1
// CHECK-NEXT:   %8 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %7, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %10, label %11, label %12
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 12:                                               ; preds = %5
// CHECK-NEXT:   %13 = load double, ptr %9, align 8
// CHECK-NEXT:   %14 = fmul double %13, %1
// CHECK-NEXT:   %15 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %14, ptr %15, align 8
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
