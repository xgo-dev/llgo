// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/embedunexport"

// Wrapped embeds *embedunexport.Base to implement embedunexport.Object
type Wrapped struct {
	*embedunexport.Base
}

// CHECK-LABEL: define %"{{.*}}String" @main.Wrapped.Name(%main.Wrapped %0){{.*}} {
// CHECK: [[VALUE_NAME_ADDR:%[0-9]+]] = alloca %main.Wrapped
// CHECK: store %main.Wrapped %0, ptr [[VALUE_NAME_ADDR]]
// CHECK: [[VALUE_NAME_FIELD:%[0-9]+]] = getelementptr inbounds %main.Wrapped, ptr [[VALUE_NAME_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[VALUE_NAME_BASE:%[0-9]+]] = load ptr, ptr [[VALUE_NAME_FIELD]]
// CHECK-NEXT: [[VALUE_NAME_RESULT:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}embedunexport.(*Base).Name"(ptr [[VALUE_NAME_BASE]])
// CHECK-NEXT: ret %"{{.*}}String" [[VALUE_NAME_RESULT]]

// CHECK-LABEL: define void @"main.Wrapped.github.com/goplus/llgo/cl/_testdata/embedunexport.setName"(%main.Wrapped %0, %"{{.*}}String" %1){{.*}} {
// CHECK: [[VALUE_SET_ADDR:%[0-9]+]] = alloca %main.Wrapped
// CHECK: store %main.Wrapped %0, ptr [[VALUE_SET_ADDR]]
// CHECK: [[VALUE_SET_FIELD:%[0-9]+]] = getelementptr inbounds %main.Wrapped, ptr [[VALUE_SET_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[VALUE_SET_BASE:%[0-9]+]] = load ptr, ptr [[VALUE_SET_FIELD]]
// CHECK-NEXT: call void @"{{.*}}embedunexport.(*Base).setName"(ptr [[VALUE_SET_BASE]], %"{{.*}}String" %1)

// CHECK-LABEL: define %"{{.*}}String" @"main.(*Wrapped).Name"(ptr %0){{.*}} {
// CHECK: [[PTR_NAME_FIELD:%[0-9]+]] = getelementptr inbounds %main.Wrapped, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[PTR_NAME_BASE:%[0-9]+]] = load ptr, ptr [[PTR_NAME_FIELD]]
// CHECK-NEXT: [[PTR_NAME_RESULT:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}embedunexport.(*Base).Name"(ptr [[PTR_NAME_BASE]])
// CHECK-NEXT: ret %"{{.*}}String" [[PTR_NAME_RESULT]]

// CHECK-LABEL: define void @"main.(*Wrapped).github.com/goplus/llgo/cl/_testdata/embedunexport.setName"(ptr %0, %"{{.*}}String" %1){{.*}} {
// CHECK: [[PTR_SET_FIELD:%[0-9]+]] = getelementptr inbounds %main.Wrapped, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[PTR_SET_BASE:%[0-9]+]] = load ptr, ptr [[PTR_SET_FIELD]]
// CHECK-NEXT: call void @"{{.*}}embedunexport.(*Base).setName"(ptr [[PTR_SET_BASE]], %"{{.*}}String" %1)

func main() {
	// CHECK: [[EMBEDDED_BASE:%[0-9]+]] = call ptr @"{{.*}}embedunexport.NewBase"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 4 })
	// CHECK-NEXT: [[WRAPPED_PTR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
	// CHECK-NEXT: [[WRAPPED_FIELD:%[0-9]+]] = getelementptr inbounds %main.Wrapped, ptr [[WRAPPED_PTR]], i32 0, i32 0
	// CHECK-NEXT: store ptr [[EMBEDDED_BASE]], ptr [[WRAPPED_FIELD]]
	// CHECK: [[WRAPPED_ITAB:%[0-9]+]] = call ptr @"{{.*}}NewItab"(ptr @"{{.*}}embedunexport.iface{{.*}}", ptr @"*_llgo_main.Wrapped")
	// CHECK-NEXT: [[WRAPPED_IFACE0:%[0-9]+]] = insertvalue %"{{.*}}iface" undef, ptr [[WRAPPED_ITAB]], 0
	// CHECK-NEXT: [[WRAPPED_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" [[WRAPPED_IFACE0]], ptr [[WRAPPED_PTR]], 1
	// CHECK-NEXT: call void @"{{.*}}embedunexport.Use"(%"{{.*}}iface" [[WRAPPED_IFACE]])
	// CHECK-NEXT: [[WRAPPED_DATA:%[0-9]+]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[WRAPPED_IFACE]])
	// CHECK-NEXT: [[WRAPPED_TABLE:%[0-9]+]] = extractvalue %"{{.*}}iface" [[WRAPPED_IFACE]], 0
	// CHECK-NEXT: [[WRAPPED_NAME_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[WRAPPED_TABLE]], i64 3
	// CHECK-NEXT: [[WRAPPED_NAME_METHOD:%[0-9]+]] = load ptr, ptr [[WRAPPED_NAME_SLOT]]
	// CHECK: [[WRAPPED_NAME_FUNC0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[WRAPPED_NAME_METHOD]], 0
	// CHECK-NEXT: [[WRAPPED_NAME_FUNC:%[0-9]+]] = insertvalue { ptr, ptr } [[WRAPPED_NAME_FUNC0]], ptr [[WRAPPED_DATA]], 1
	// CHECK-NEXT: [[WRAPPED_CALL_DATA:%[0-9]+]] = extractvalue { ptr, ptr } [[WRAPPED_NAME_FUNC]], 1
	// CHECK-NEXT: [[WRAPPED_CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[WRAPPED_NAME_FUNC]], 0
	// CHECK-NEXT: [[WRAPPED_NAME:%[0-9]+]] = call %"{{.*}}String" [[WRAPPED_CALL_CODE]](ptr [[WRAPPED_CALL_DATA]])
	// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[WRAPPED_NAME]])
	base := embedunexport.NewBase("test")
	wrapped := &Wrapped{Base: base}

	// This should work: calling unexported method through interface
	var obj embedunexport.Object = wrapped
	embedunexport.Use(obj)

	println(obj.Name())
}
