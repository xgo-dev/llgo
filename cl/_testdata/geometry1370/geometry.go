// LITTEST
package geometry1370

type Shape interface {
	Area() float64
	validate() bool
	setID(int)
}

type Rectangle struct {
	Width, Height float64
	id            int
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testdata/geometry1370.NewRectangle"(double %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store double %0, ptr %3, align 8
// CHECK-NEXT:   store double %1, ptr %4, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

func NewRectangle(width, height float64) *Rectangle {
	return &Rectangle{Width: width, Height: height}
}

// CHECK-LABEL: define double @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).Area"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = load double, ptr %1, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %7, label %8, label %9
// CHECK-EMPTY:
// CHECK-NEXT: 8:                                                ; preds = %4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %4
// CHECK-NEXT:   %10 = load double, ptr %6, align 8
// CHECK-NEXT:   %11 = fmul double %5, %10
// CHECK-NEXT:   ret double %11
// CHECK-NEXT: }

func (r *Rectangle) Area() float64 { return r.Width * r.Height }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).GetID"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

func (r *Rectangle) GetID() int { return r.id }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).setID"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func (r *Rectangle) setID(id int) { r.id = id }

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).validate"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %6, label %7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %7
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testdata/geometry1370.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %4, label %10, label %11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %11, %7
// CHECK-NEXT:   %5 = phi i1 [ false, %7 ], [ %13, %11 ]
// CHECK-NEXT:   ret i1 %5
// CHECK-EMPTY:
// CHECK-NEXT: 6:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 7:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %8 = load double, ptr %1, align 8
// CHECK-NEXT:   %9 = fcmp ogt double %8, 0.000000e+00
// CHECK-NEXT:   br i1 %9, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %_llgo_1
// CHECK-NEXT:   %12 = load double, ptr %3, align 8
// CHECK-NEXT:   %13 = fcmp ogt double %12, 0.000000e+00
// CHECK-NEXT:   br label %_llgo_2
// CHECK-NEXT: }

func (r *Rectangle) validate() bool { return r.Width > 0 && r.Height > 0 }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/geometry1370.RegisterShape"(%"{{.*}}/runtime/internal/runtime.iface" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %4 = getelementptr ptr, ptr %3, i64 4
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } %6, ptr %2, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %7, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %7, 0
// CHECK-NEXT:   call void %9(ptr %8, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/geometry1370.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/geometry1370.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/geometry1370.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func RegisterShape(s Shape, id int) {
	s.setID(id)
}
