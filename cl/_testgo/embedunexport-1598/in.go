// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/embedunexport"

// Wrapped embeds *embedunexport.Base to implement embedunexport.Object

// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"test", align 1

type Wrapped struct {
	*embedunexport.Base
}

func main() {
	base := embedunexport.NewBase("test")
	wrapped := &Wrapped{Base: base}

	// This should work: calling unexported method through interface
	var obj embedunexport.Object = wrapped
	embedunexport.Use(obj)

	println(obj.Name())
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.Name"(%"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testdata/embedunexport.(*Base).Name"(ptr %4)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %5
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.setName"(%"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/embedunexport.(*Base).setName"(ptr %5, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).Name"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testdata/embedunexport.(*Base).Name"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).setName"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/embedunexport.(*Base).setName"(ptr %4, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/embedunexport-1598.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/embedunexport-1598.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/embedunexport-1598.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/embedunexport.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/embedunexport-1598.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testdata/embedunexport.NewBase"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped", ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testdata/embedunexport.iface$gGW7PSocDeRlTvk5kuSew8C-TZ8OXQrGkMlj2EUlZ9E", ptr @"*_llgo_{{.*}}/cl/_testgo/embedunexport-1598.Wrapped")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %1, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/embedunexport.Use"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" %14(ptr %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testdata/embedunexport.(*Base).Name"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testdata/embedunexport.(*Base).Name"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testdata/embedunexport.(*Base).setName"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testdata/embedunexport.(*Base).setName"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.Name"(ptr %0, %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.Name"(%"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.setName"(ptr %0, %"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped.setName"(%"{{.*}}/cl/_testgo/embedunexport-1598.Wrapped" %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).Name"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).Name"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).setName"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/embedunexport-1598.(*Wrapped).setName"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
