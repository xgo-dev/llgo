// LITTEST
package embedunexport

// Object is an interface with both exported and unexported methods
type Object interface {
	Name() string
	setName(string)
}

// Base implements Object
// CHECK: %"{{.*}}.Base" = type { %"{{.*}}.String" }
type Base struct {
	name string
}

// CHECK: @0 = private unnamed_addr constant [8 x i8] c"modified", align 1
// CHECK: define %"{{.*}}.String" @"{{.*}}.(*Base).Name"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}.Base", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}.String", ptr %1, align 8
// CHECK-NEXT:   ret %"{{.*}}.String" %2
// CHECK-NEXT: }
func (b *Base) Name() string {
	return b.name
}

// CHECK: define void @"{{.*}}.(*Base).setName"(ptr %0, %"{{.*}}.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}.Base", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}.String" %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (b *Base) setName(name string) {
	b.name = name
}

// CHECK: define ptr @"{{.*}}.NewBase"(%"{{.*}}.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}.Base", ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}.String" %0, ptr %2, align 8
// CHECK-NEXT:   ret ptr %1
// CHECK-NEXT: }
func NewBase(name string) *Base {
	return &Base{name: name}
}

// Use calls the unexported method through interface
// CHECK: define void @"{{.*}}.Use"(%"{{.*}}.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}.IfacePtrData"(%"{{.*}}.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 4
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   call void %8(ptr %7, %"{{.*}}.String" { ptr @0, i64 8 })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func Use(obj Object) {
	obj.setName("modified")
}
