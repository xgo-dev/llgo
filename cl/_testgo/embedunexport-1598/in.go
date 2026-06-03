// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/embedunexport"

// Wrapped embeds *embedunexport.Base to implement embedunexport.Object
type Wrapped struct {
	*embedunexport.Base
}

// CHECK-LABEL: define %"{{.*}}String" @"{{.*}}embedunexport-1598.Wrapped.Name"(%"{{.*}}embedunexport-1598.Wrapped" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}embedunexport-1598.Wrapped", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}embedunexport-1598.Wrapped" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}embedunexport-1598.Wrapped", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}String" @"{{.*}}embedunexport.(*Base).Name"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}String" %4

// CHECK-LABEL: define void @"{{.*}}embedunexport-1598.Wrapped.setName"(%"{{.*}}embedunexport-1598.Wrapped" %0, %"{{.*}}String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}embedunexport-1598.Wrapped", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}embedunexport-1598.Wrapped" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}embedunexport-1598.Wrapped", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}embedunexport.(*Base).setName"(ptr %4, %"{{.*}}String" %1)
// CHECK-NEXT:   ret void

// CHECK-LABEL: define %"{{.*}}String" @"{{.*}}embedunexport-1598.(*Wrapped).Name"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}embedunexport-1598.Wrapped", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}String" @"{{.*}}embedunexport.(*Base).Name"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}String" %3

// CHECK-LABEL: define void @"{{.*}}embedunexport-1598.(*Wrapped).setName"(ptr %0, %"{{.*}}String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}embedunexport-1598.Wrapped", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}embedunexport.(*Base).setName"(ptr %3, %"{{.*}}String" %1)
// CHECK-NEXT:   ret void

// CHECK-LABEL: define void @"{{.*}}embedunexport-1598.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}embedunexport.NewBase"(%"{{.*}}String" { ptr @0, i64 4 })
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: call ptr @"{{.*}}NewItab"(ptr @"{{.*}}embedunexport.iface{{.*}}", ptr @"{{.*}}embedunexport-1598.Wrapped")
	// CHECK: call void @"{{.*}}embedunexport.Use"(%"{{.*}}iface" %5)
	// CHECK: call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %5)
	// CHECK: call %"{{.*}}String" %13(ptr %12)
	// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" %14)
	// CHECK: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK: ret void
	base := embedunexport.NewBase("test")
	wrapped := &Wrapped{Base: base}

	// This should work: calling unexported method through interface
	var obj embedunexport.Object = wrapped
	embedunexport.Use(obj)

	println(obj.Name())
}
