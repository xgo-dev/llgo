// LITTEST
package main

import (
	"github.com/goplus/llgo/cl/_testgo/genericembediface/streamlib"
)

// CHECK: {{^}}@2 = private unnamed_addr constant [20 x i8] c"ServerReflectionInfo", align 1{{$}}
// CHECK: {{^}}@5 = private unnamed_addr constant [7 x i8] c"Context", align 1{{$}}
// CHECK: {{^}}@11 = private unnamed_addr constant [68 x i8] c"{{.*}}/cl/_testgo/genericembediface.ReflectionServer", align 1{{$}}
// CHECK: {{^}}@19 = private unnamed_addr constant [4 x i8] c"pass", align 1{{$}}
// CHECK: {{^}}@20 = private unnamed_addr constant [58 x i8] c"{{.*}}/cl/_testgo/genericembediface.server", align 1{{$}}
// CHECK: {{^}}@21 = private unnamed_addr constant [58 x i8] c"{{.*}}/cl/_testgo/genericembediface.stream", align 1{{$}}

type Request struct{}
type Response struct{}

type ReflectionServer interface {
	ServerReflectionInfo(streamlib.BidiStreamingServer[Request, Response]) error
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.handler"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/genericembediface.ReflectionServer", ptr %2)
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$ZzWqZgiYW4qfvEaEOnxMk0iM2CGUNdNCyXNPKgONU60", ptr %2)
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %6, ptr %4, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %1, ptr %9, align 8
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$v_XV1q3uiNvAZy1sSF5r_9UE2XfxcttHV0UKe3XpAeo", ptr @"*_llgo_{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]")
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %11, ptr %8, 1
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %7, 0
// CHECK-NEXT:   %15 = getelementptr ptr, ptr %14, i64 3
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } undef, ptr %16, 0
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } %17, ptr %13, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.iface" %20(ptr %19, %"{{.*}}/runtime/internal/runtime.iface" %12)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 68 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/genericembediface.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/genericembediface.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/genericembediface.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/genericembediface/streamlib.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func handler(srv any, stream streamlib.ServerStream) error {
	return srv.(ReflectionServer).ServerReflectionInfo(&streamlib.GenericServerStream[Request, Response]{ServerStream: stream})
}

type server struct{}

func (server) ServerReflectionInfo(streamlib.BidiStreamingServer[Request, Response]) error {
	return nil
}

type stream struct{}

func (stream) Context() string {
	return "Context"
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/genericembediface.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/genericembediface.server" zeroinitializer, ptr %0, align 1
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/genericembediface.server", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/genericembediface.stream" zeroinitializer, ptr %2, align 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$v_XV1q3uiNvAZy1sSF5r_9UE2XfxcttHV0UKe3XpAeo", ptr @"_llgo_{{.*}}/cl/_testgo/genericembediface.stream")
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %3, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %2, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.handler"(%"{{.*}}/runtime/internal/runtime.eface" %1, %"{{.*}}/runtime/internal/runtime.iface" %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.server.ServerReflectionInfo"(%"{{.*}}/cl/_testgo/genericembediface.server" %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

func main() {
	_ = handler(server{}, stream{})
	println("pass")
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.(*server).ServerReflectionInfo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 })
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.server.ServerReflectionInfo"(%"{{.*}}/cl/_testgo/genericembediface.server" zeroinitializer, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface.stream.Context"(%"{{.*}}/cl/_testgo/genericembediface.stream" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 7 }
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface.(*stream).Context"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 58 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 7 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface.stream.Context"(%"{{.*}}/cl/_testgo/genericembediface.stream" zeroinitializer)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface/streamlib.(*GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]).Context"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %1, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %2, 0
// CHECK-NEXT:   %5 = getelementptr ptr, ptr %4, i64 3
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } %7, ptr %3, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %8, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   %11 = call %"{{.*}}/runtime/internal/runtime.String" %10(ptr %9)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %11
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response].Context"(%"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.String" %11(ptr %10)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %12
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response].Context"(ptr %0, %"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response].Context"(%"{{.*}}/cl/_testgo/genericembediface/streamlib.GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface/streamlib.(*GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]).Context"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface/streamlib.(*GenericServerStream[{{.*}}/cl/_testgo/genericembediface.Request,{{.*}}/cl/_testgo/genericembediface.Response]).Context"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal0"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal0"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface.(*server).ServerReflectionInfo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.(*server).ServerReflectionInfo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface.server.ServerReflectionInfo"(ptr %0, %"{{.*}}/cl/_testgo/genericembediface.server" %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/genericembediface.server.ServerReflectionInfo"(%"{{.*}}/cl/_testgo/genericembediface.server" %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface.(*stream).Context"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface.(*stream).Context"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/genericembediface.stream.Context"(ptr %0, %"{{.*}}/cl/_testgo/genericembediface.stream" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/genericembediface.stream.Context"(%"{{.*}}/cl/_testgo/genericembediface.stream" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
