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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/alias.(*Point).Move"(ptr %0, double %1, double %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load double, ptr %4, align 8
// CHECK-NEXT:   %6 = fadd double %5, %1
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %6, ptr %8, align 8
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %11 = load double, ptr %10, align 8
// CHECK-NEXT:   %12 = fadd double %11, %2
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %12, ptr %14, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/alias.(*Point).Scale"(ptr %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load double, ptr %3, align 8
// CHECK-NEXT:   %5 = fmul double %4, %1
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store double %5, ptr %7, align 8
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %10 = load double, ptr %9, align 8
// CHECK-NEXT:   %11 = fmul double %10, %1
// CHECK-NEXT:   %12 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double %11, ptr %13, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/alias.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/alias.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/alias.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/alias.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store double 1.000000e+00, ptr %2, align 8
// CHECK-NEXT:   store double 2.000000e+00, ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/alias.(*Point).Scale"(ptr %0, double 2.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/alias.(*Point).Move"(ptr %0, double 3.000000e+00, double 4.000000e+00)
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load double, ptr %6, align 8
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/alias.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %10 = load double, ptr %9, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
