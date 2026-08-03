// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/embedunexport"

// Wrapped embeds *embedunexport.Base to implement embedunexport.Object
type Wrapped struct {
	*embedunexport.Base
}

// CHECK-LABEL: define %"{{.*}}String" @main.Wrapped.Name(%main.Wrapped %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %main.Wrapped, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %main.Wrapped %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Wrapped, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}String" @"{{.*}}embedunexport.(*Base).Name"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}String" %4

// CHECK-LABEL: define void @main.Wrapped.setName(%main.Wrapped %0, %"{{.*}}String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %main.Wrapped, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %main.Wrapped %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.Wrapped, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}embedunexport.(*Base).setName"(ptr %4, %"{{.*}}String" %1)
// CHECK-NEXT:   ret void

// CHECK-LABEL: define %"{{.*}}String" @"main.(*Wrapped).Name"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %1, label %2, label %3
// CHECK-EMPTY:
// CHECK-NEXT: 2:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %4 = getelementptr inbounds %main.Wrapped, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testdata/embedunexport.(*Base).Name"(ptr %5)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %6
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.(*Wrapped).setName"(ptr %0, %"{{.*}}String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = getelementptr inbounds %main.Wrapped, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/embedunexport.(*Base).setName"(ptr %6, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}embedunexport.NewBase"(%"{{.*}}String" { ptr @0, i64 4 })
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: call ptr @"{{.*}}NewItab"(ptr @"{{.*}}embedunexport.iface{{.*}}", ptr @"*_llgo_main.Wrapped")
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
