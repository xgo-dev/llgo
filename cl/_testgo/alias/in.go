// LITTEST
package main

type Point struct {
	x float64
	y float64
}

type MyPoint = Point

// CHECK-LABEL: define void @"main.(*Point).Move"(ptr %0, double %1, double %2){{.*}} {
// CHECK: [[MOVE_X_FIELD:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[MOVE_X:%[0-9]+]] = load double, ptr [[MOVE_X_FIELD]]
// CHECK-NEXT: [[MOVE_NEW_X:%[0-9]+]] = fadd double [[MOVE_X]], %1
// CHECK-NEXT: [[MOVE_X_STORE:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT: store double [[MOVE_NEW_X]], ptr [[MOVE_X_STORE]]
// CHECK-NEXT: [[MOVE_Y_FIELD:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[MOVE_Y:%[0-9]+]] = load double, ptr [[MOVE_Y_FIELD]]
// CHECK-NEXT: [[MOVE_NEW_Y:%[0-9]+]] = fadd double [[MOVE_Y]], %2
// CHECK-NEXT: [[MOVE_Y_STORE:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT: store double [[MOVE_NEW_Y]], ptr [[MOVE_Y_STORE]]
func (p *MyPoint) Move(dx, dy float64) {
	p.x += dx
	p.y += dy
}

// CHECK-LABEL: define void @"main.(*Point).Scale"(ptr %0, double %1){{.*}} {
// CHECK: [[SCALE_X_FIELD:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[SCALE_X:%[0-9]+]] = load double, ptr [[SCALE_X_FIELD]]
// CHECK-NEXT: [[SCALE_NEW_X:%[0-9]+]] = fmul double [[SCALE_X]], %1
// CHECK-NEXT: [[SCALE_X_STORE:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 0
// CHECK-NEXT: store double [[SCALE_NEW_X]], ptr [[SCALE_X_STORE]]
// CHECK-NEXT: [[SCALE_Y_FIELD:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[SCALE_Y:%[0-9]+]] = load double, ptr [[SCALE_Y_FIELD]]
// CHECK-NEXT: [[SCALE_NEW_Y:%[0-9]+]] = fmul double [[SCALE_Y]], %1
// CHECK-NEXT: [[SCALE_Y_STORE:%[0-9]+]] = getelementptr inbounds %main.Point, ptr %0, i32 0, i32 1
// CHECK-NEXT: store double [[SCALE_NEW_Y]], ptr [[SCALE_Y_STORE]]
func (p *Point) Scale(factor float64) {
	p.x *= factor
	p.y *= factor
}

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[POINT:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK: call void @"main.(*Point).Scale"(ptr [[POINT]], double 2.000000e+00)
	// CHECK-NEXT: call void @"main.(*Point).Move"(ptr [[POINT]], double 3.000000e+00, double 4.000000e+00)
	// CHECK: [[PRINT_X:%[0-9]+]] = load double, ptr {{.*}}
	// CHECK: [[PRINT_Y:%[0-9]+]] = load double, ptr {{.*}}
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[PRINT_X]])
	// CHECK: call void @"{{.*}}PrintFloat"(double [[PRINT_Y]])
	pt := &MyPoint{1, 2}
	pt.Scale(2)
	pt.Move(3, 4)
	println(pt.x, pt.y)
}
