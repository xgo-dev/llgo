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

// CHECK-LABEL: define ptr @"{{.*}}.NewRectangle"(double %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}.AllocZ"(i64 24)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}.Rectangle", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}.Rectangle", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store double %0, ptr %3, align 8
// CHECK-NEXT:   store double %1, ptr %4, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }
func NewRectangle(width, height float64) *Rectangle {
	return &Rectangle{Width: width, Height: height}
}

// CHECK-LABEL: define double @"{{.*}}geometry1370.(*Rectangle).Area"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = load double, ptr %3, align 8
// CHECK-NEXT:   %5 = fmul double %2, %4
// CHECK-NEXT:   ret double %5
// CHECK-NEXT: }
func (r *Rectangle) Area() float64 { return r.Width * r.Height }

// CHECK-LABEL: define i64 @"{{.*}}geometry1370.(*Rectangle).GetID"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func (r *Rectangle) GetID() int { return r.id }

// CHECK-LABEL: define void @"{{.*}}geometry1370.(*Rectangle).setID"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (r *Rectangle) setID(id int) { r.id = id }

// CHECK-LABEL: define i1 @"{{.*}}geometry1370.(*Rectangle).validate"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load double, ptr %1, align 8
// CHECK-NEXT:   %3 = fcmp ogt double %2, 0.000000e+00
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}geometry1370.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %5 = load double, ptr %4, align 8
// CHECK-NEXT:   %6 = fcmp ogt double %5, 0.000000e+00
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %7 = phi i1 [ false, %_llgo_0 ], [ %6, %_llgo_1 ]
// CHECK-NEXT:   ret i1 %7
// CHECK-NEXT: }
func (r *Rectangle) validate() bool { return r.Width > 0 && r.Height > 0 }

// CHECK-LABEL: define void @"{{.*}}.RegisterShape"(%"{{.*}}.iface" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}.IfacePtrData"(%"{{.*}}.iface" %0)
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}.iface" %0, 0
// CHECK-NEXT:   %4 = getelementptr ptr, ptr %3, i64 4
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } %6, ptr %2, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %7, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %7, 0
// CHECK-NEXT:   call void %9(ptr %8, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func RegisterShape(s Shape, id int) {
	s.setID(id)
}
