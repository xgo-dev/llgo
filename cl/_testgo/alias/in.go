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
// CHECK-NEXT:   %4 = load double, ptr %3, align 8
// CHECK-NEXT:   %5 = fadd double %4, %1
// CHECK-NEXT:   %6 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %5, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = load double, ptr %7, align 8
// CHECK-NEXT:   %9 = fadd double %8, %2
// CHECK-NEXT:   %10 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %9, ptr %10, align 8
// CHECK-NEXT:   ret void
func (p *MyPoint) Move(dx, dy float64) {
	p.x += dx
	p.y += dy
}

// CHECK-LABEL: define void @"main.(*Point).Scale"(ptr %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load double, ptr %2, align 8
// CHECK-NEXT:   %4 = fmul double %3, %1
// CHECK-NEXT:   %5 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %4, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load double, ptr %6, align 8
// CHECK-NEXT:   %8 = fmul double %7, %1
// CHECK-NEXT:   %9 = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %8, ptr %9, align 8
// CHECK-NEXT:   ret void
func (p *Point) Scale(factor float64) {
	p.x *= factor
	p.y *= factor
}

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 16, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %0 = getelementptr inbounds %main.Point, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %1 = getelementptr inbounds %main.Point, ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   store double 1.000000e+00, ptr %0, align 8
// ESCAPE-NEXT:   store double 2.000000e+00, ptr %1, align 8
// ESCAPE-NEXT:   call void @"main.(*Point).Scale"(ptr %.stack, double 2.000000e+00)
// ESCAPE-NEXT:   call void @"main.(*Point).Move"(ptr %.stack, double 3.000000e+00, double 4.000000e+00)
// ESCAPE-NEXT:   %2 = getelementptr inbounds %main.Point, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %3 = load double, ptr %2, align 8
// ESCAPE-NEXT:   %4 = getelementptr inbounds %main.Point, ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   %5 = load double, ptr %4, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %3)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %5)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

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
