// LITTEST
package embedunexport

// Object is an interface with both exported and unexported methods
type Object interface {
	Name() string
	setName(string)
}

// Base implements Object
type Base struct {
	name string
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}.(*Base).Name"(ptr %0){{.*}} {
// CHECK: [[NAME_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Base", ptr %0, i32 0, i32 0
// CHECK-NEXT: [[NAME_VALUE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[NAME_FIELD]]
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.String" [[NAME_VALUE]]

func (b *Base) Name() string {
	return b.name
}

// CHECK-LABEL: define void @"{{.*}}.(*Base).setName"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK: [[SET_NAME_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Base", ptr %0, i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.String" %1, ptr [[SET_NAME_FIELD]]

func (b *Base) setName(name string) {
	b.name = name
}

// CHECK-LABEL: define ptr @"{{.*}}.NewBase"(%"{{.*}}.String" %0){{.*}} {
// CHECK: [[NEW_BASE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: [[NEW_BASE_NAME:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Base", ptr [[NEW_BASE]], i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.String" %0, ptr [[NEW_BASE_NAME]]
// CHECK-NEXT: ret ptr [[NEW_BASE]]
func NewBase(name string) *Base {
	return &Base{name: name}
}

// Use calls the unexported method through interface
// CHECK-LABEL: define void @"{{.*}}.Use"(%"{{.*}}.iface" %0){{.*}} {
// CHECK: %[[OBJ_DATA:[0-9]+]] = call ptr @"{{.*}}.IfacePtrData"(%"{{.*}}.iface" %0)
// CHECK: insertvalue { ptr, ptr } %{{[0-9]+}}, ptr %[[OBJ_DATA]], 1
func Use(obj Object) {
	obj.setName("modified")
}
