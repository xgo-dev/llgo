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
// CHECK: [[RECT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT: [[RECT_WIDTH_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr [[RECT]], i32 0, i32 0
// CHECK-NEXT: [[RECT_HEIGHT_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr [[RECT]], i32 0, i32 1
// CHECK-NEXT: store double %0, ptr [[RECT_WIDTH_FIELD]]
// CHECK-NEXT: store double %1, ptr [[RECT_HEIGHT_FIELD]]
// CHECK-NEXT: ret ptr [[RECT]]

func NewRectangle(width, height float64) *Rectangle {
	return &Rectangle{Width: width, Height: height}
}

// CHECK-LABEL: define double @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).Area"(ptr %0){{.*}} {
// CHECK: [[AREA_WIDTH_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[AREA_WIDTH:%[0-9]+]] = load double, ptr [[AREA_WIDTH_FIELD]]
// CHECK-NEXT: [[AREA_HEIGHT_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT: [[AREA_HEIGHT:%[0-9]+]] = load double, ptr [[AREA_HEIGHT_FIELD]]
// CHECK-NEXT: [[AREA_VALUE:%[0-9]+]] = fmul double [[AREA_WIDTH]], [[AREA_HEIGHT]]
// CHECK-NEXT: ret double [[AREA_VALUE]]

func (r *Rectangle) Area() float64 { return r.Width * r.Height }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).GetID"(ptr %0){{.*}} {
// CHECK: [[GET_ID_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT: [[GET_ID_VALUE:%[0-9]+]] = load i64, ptr [[GET_ID_FIELD]]
// CHECK-NEXT: ret i64 [[GET_ID_VALUE]]

func (r *Rectangle) GetID() int { return r.id }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).setID"(ptr %0, i64 %1){{.*}} {
// CHECK: [[SET_ID_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 2
// CHECK-NEXT: store i64 %1, ptr [[SET_ID_FIELD]]

func (r *Rectangle) setID(id int) { r.id = id }

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testdata/geometry1370.(*Rectangle).validate"(ptr %0){{.*}} {
// CHECK: [[VALID_WIDTH_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[VALID_WIDTH:%[0-9]+]] = load double, ptr [[VALID_WIDTH_FIELD]]
// CHECK-NEXT: [[VALID_WIDTH_OK:%[0-9]+]] = fcmp ogt double [[VALID_WIDTH]], 0.000000e+00
// CHECK: [[VALID_HEIGHT_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Rectangle", ptr %0, i32 0, i32 1
// CHECK-NEXT: [[VALID_HEIGHT:%[0-9]+]] = load double, ptr [[VALID_HEIGHT_FIELD]]
// CHECK-NEXT: [[VALID_HEIGHT_OK:%[0-9]+]] = fcmp ogt double [[VALID_HEIGHT]], 0.000000e+00
// CHECK: [[VALID_RESULT:%[0-9]+]] = phi i1 [ false, %{{.*}} ], [ [[VALID_HEIGHT_OK]], %{{.*}} ]
// CHECK-NEXT: ret i1 [[VALID_RESULT]]

func (r *Rectangle) validate() bool { return r.Width > 0 && r.Height > 0 }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/geometry1370.RegisterShape"(%"{{.*}}iface" %0, i64 %1){{.*}} {
// CHECK: [[SHAPE_DATA:%[0-9]+]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %0)
// CHECK-NEXT: [[SHAPE_ITAB:%[0-9]+]] = extractvalue %"{{.*}}iface" %0, 0
// CHECK-NEXT: [[SETID_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[SHAPE_ITAB]], i64 4
// CHECK-NEXT: [[SETID_CODE:%[0-9]+]] = load ptr, ptr [[SETID_SLOT]]
// CHECK-NEXT: [[SETID_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[SETID_CODE]], 0
// CHECK-NEXT: [[SETID_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[SETID_PAIR0]], ptr [[SHAPE_DATA]], 1
// CHECK-NEXT: [[CALL_RECEIVER:%[0-9]+]] = extractvalue { ptr, ptr } [[SETID_PAIR]], 1
// CHECK-NEXT: [[CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[SETID_PAIR]], 0
// CHECK-NEXT: call void [[CALL_CODE]](ptr [[CALL_RECEIVER]], i64 %1)

func RegisterShape(s Shape, id int) {
	s.setID(id)
}
