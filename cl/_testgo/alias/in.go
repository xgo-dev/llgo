// LITTEST
package main

type Point struct {
	x float64
	y float64
}

type MyPoint = Point

// CHECK-LABEL: define void @"{{.*}}alias.(*Point).Move"(ptr %0, double %1, double %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load double, ptr %3, align 8
// CHECK-NEXT:   %5 = fadd double %4, %1
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %5, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = load double, ptr %7, align 8
// CHECK-NEXT:   %9 = fadd double %8, %2
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %9, ptr %10, align 8
// CHECK-NEXT:   ret void
func (p *MyPoint) Move(dx, dy float64) {
	p.x += dx
	p.y += dy
}

// CHECK-LABEL: define void @"{{.*}}alias.(*Point).Scale"(ptr %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load double, ptr %2, align 8
// CHECK-NEXT:   %4 = fmul double %3, %1
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %4, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load double, ptr %6, align 8
// CHECK-NEXT:   %8 = fmul double %7, %1
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %8, ptr %9, align 8
// CHECK-NEXT:   ret void
func (p *Point) Scale(factor float64) {
	p.x *= factor
	p.y *= factor
}

// CHECK-LABEL: define void @"{{.*}}alias.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK: call void @"{{.*}}alias.(*Point).Scale"(ptr %0, double 2.000000e+00)
	// CHECK: call void @"{{.*}}alias.(*Point).Move"(ptr %0, double 3.000000e+00, double 4.000000e+00)
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
