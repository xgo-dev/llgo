// LITTEST
package main

type Point struct {
	x float64
	y float64
}

type MyPoint = Point

func (p *MyPoint) Move(dx, dy float64) {
	p.x += dx
	p.y += dy
}

func (p *Point) Scale(factor float64) {
	p.x *= factor
	p.y *= factor
}

func main() {
	pt := &MyPoint{1, 2}
	pt.Scale(2)
	pt.Move(3, 4)
	println(pt.x, pt.y)
}

// Type aliases must resolve to the canonical Point receiver and float64 fields.
// CHECK-LABEL: define void @"main.(*Point).Move"(
// CHECK-SAME: ptr %[[P:[0-9]+]], double %[[DX:[0-9]+]], double %[[DY:[0-9]+]])
// CHECK: %[[X:[0-9]+]] = getelementptr inbounds nuw %main.Point, ptr %[[P]], i32 0, i32 0
// CHECK: %[[VX:[0-9]+]] = load double, ptr %[[X]]
// CHECK: %[[NX:[0-9]+]] = fadd double %[[VX]], %[[DX]]
// CHECK: store double %[[NX]], ptr
// CHECK: %[[Y:[0-9]+]] = getelementptr inbounds nuw %main.Point, ptr %[[P]], i32 0, i32 1
// CHECK: %[[VY:[0-9]+]] = load double, ptr %[[Y]]
// CHECK: %[[NY:[0-9]+]] = fadd double %[[VY]], %[[DY]]
// CHECK: store double %[[NY]], ptr
// CHECK-LABEL: define void @"main.(*Point).Scale"(
// CHECK-SAME: ptr %[[P:[0-9]+]], double %[[FACTOR:[0-9]+]])
// CHECK: %[[X:[0-9]+]] = getelementptr inbounds nuw %main.Point, ptr %[[P]], i32 0, i32 0
// CHECK: %[[VX:[0-9]+]] = load double, ptr %[[X]]
// CHECK: %[[NX:[0-9]+]] = fmul double %[[VX]], %[[FACTOR]]
// CHECK: store double %[[NX]], ptr
// CHECK: %[[Y:[0-9]+]] = getelementptr inbounds nuw %main.Point, ptr %[[P]], i32 0, i32 1
// CHECK: %[[VY:[0-9]+]] = load double, ptr %[[Y]]
// CHECK: %[[NY:[0-9]+]] = fmul double %[[VY]], %[[FACTOR]]
// CHECK: store double %[[NY]], ptr
// CHECK-LABEL: define void @main.main()
// CHECK: call void @"main.(*Point).Scale"(ptr %[[POINT:[0-9]+]], double 2.000000e+00)
// CHECK: call void @"main.(*Point).Move"(ptr %[[POINT]], double 3.000000e+00, double 4.000000e+00)
