// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testgo/genericembediface/streamlib"
)

type Request struct{}
type Response struct{}

type ReflectionServer interface {
	ServerReflectionInfo(streamlib.BidiStreamingServer[Request, Response]) error
}

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

func main() {
	_ = handler(server{}, stream{})
	println("pass")
}

// The concrete stream method is reached through a generic struct embedding an interface.
// CHECK: [[CONTEXT:@[0-9]+]] = private unnamed_addr constant [7 x i8] c"Context"

// handler asserts srv to ReflectionServer, wraps stream as the instantiated generic
// BidiStreamingServer, and invokes ServerReflectionInfo through the asserted itab.
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.handler(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK: [[H_DYN_TYPE:%.*]] = extractvalue %"{{.*}}eface" %0, 0
// CHECK: [[H_IMPLEMENTS:%.*]] = call i1 @"{{.*}}Implements"(ptr @_llgo_main.ReflectionServer, ptr [[H_DYN_TYPE]])
// CHECK: br i1 [[H_IMPLEMENTS]], label %{{.*}}, label %{{.*}}
// CHECK: [[H_DYN_DATA:%.*]] = extractvalue %"{{.*}}eface" %0, 1
// CHECK: [[H_REF_ITAB:%.*]] = call ptr @"{{.*}}NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr [[H_DYN_TYPE]])
// CHECK: [[H_REF_IFACE0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[H_REF_ITAB]], 0
// CHECK: [[H_REF_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[H_REF_IFACE0]], ptr [[H_DYN_DATA]], 1
// CHECK: [[H_GENERIC_STREAM:%.*]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK: [[H_STREAM_FIELD:%.*]] = getelementptr inbounds %"{{.*}}GenericServerStream[main.Request,main.Response]", ptr [[H_GENERIC_STREAM]], i32 0, i32 0
// CHECK: store %"{{.*}}iface" %1, ptr [[H_STREAM_FIELD]]
// CHECK: [[H_STREAM_ITAB:%.*]] = call ptr @"{{.*}}NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_{{.*}}GenericServerStream[main.Request,main.Response]")
// CHECK: [[H_STREAM_IFACE0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[H_STREAM_ITAB]], 0
// CHECK: [[H_STREAM_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[H_STREAM_IFACE0]], ptr [[H_GENERIC_STREAM]], 1
// CHECK: [[H_SERVER_DATA:%.*]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[H_REF_IFACE]])
// CHECK: [[H_SERVER_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[H_REF_IFACE]], 0
// CHECK: [[H_METHOD_SLOT:%.*]] = getelementptr ptr, ptr [[H_SERVER_ITAB]], i64 3
// CHECK: [[H_METHOD:%.*]] = load ptr, ptr [[H_METHOD_SLOT]]
// CHECK: [[H_CALL0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[H_METHOD]], 0
// CHECK: [[H_CALL:%.*]] = insertvalue { ptr, ptr } [[H_CALL0]], ptr [[H_SERVER_DATA]], 1
// CHECK: [[H_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[H_CALL]], 1
// CHECK: [[H_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[H_CALL]], 0
// CHECK: [[H_RESULT:%.*]] = call %"{{.*}}iface" [[H_CALL_FN]](ptr [[H_CALL_DATA]], %"{{.*}}iface" [[H_STREAM_IFACE]])
// CHECK: ret %"{{.*}}iface" [[H_RESULT]]
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[H_DYN_TYPE]], ptr @_llgo_main.ReflectionServer)
// CHECK-NEXT: unreachable

// main supplies the value implementations of ReflectionServer and ServerStream.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[M_SERVER:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_main.server, ptr undef }, ptr {{%.*}}, 1
// CHECK: [[M_STREAM_ITAB:%.*]] = call ptr @"{{.*}}NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @_llgo_main.stream)
// CHECK: [[M_STREAM0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[M_STREAM_ITAB]], 0
// CHECK: [[M_STREAM:%.*]] = insertvalue %"{{.*}}iface" [[M_STREAM0]], ptr {{%.*}}, 1
// CHECK: call %"{{.*}}iface" @main.handler(%"{{.*}}eface" [[M_SERVER]], %"{{.*}}iface" [[M_STREAM]])

// CHECK-LABEL: define %"{{.*}}iface" @main.server.ServerReflectionInfo(%main.server %0, %"{{.*}}iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT: ret %"{{.*}}iface" zeroinitializer

// The pointer wrapper preserves the stream argument while forwarding to the value method.
// CHECK-LABEL: define %"{{.*}}iface" @"main.(*server).ServerReflectionInfo"(ptr %0, %"{{.*}}iface" %1){{.*}} {
// CHECK: [[SERVER_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}PanicWrapNilPointer"(i1 [[SERVER_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" {{.*}})
// CHECK: [[SERVER_DEREF_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}AssertNilDeref"(i1 [[SERVER_DEREF_NIL]])
// CHECK: [[SERVER_RESULT:%.*]] = call %"{{.*}}iface" @main.server.ServerReflectionInfo(%main.server zeroinitializer, %"{{.*}}iface" %1)
// CHECK: ret %"{{.*}}iface" [[SERVER_RESULT]]

// CHECK-LABEL: define %"{{.*}}String" @main.stream.Context(%main.stream %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT: ret %"{{.*}}String" { ptr [[CONTEXT]], i64 7 }

// CHECK-LABEL: define %"{{.*}}String" @"main.(*stream).Context"(ptr %0){{.*}} {
// CHECK: [[STREAM_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}PanicWrapNilPointer"(i1 [[STREAM_NIL]], %"{{.*}}String" {{.*}}, %"{{.*}}String" { ptr [[CONTEXT]], i64 7 })
// CHECK: [[STREAM_DEREF_NIL:%.*]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}AssertNilDeref"(i1 [[STREAM_DEREF_NIL]])
// CHECK: [[STREAM_RESULT:%.*]] = call %"{{.*}}String" @main.stream.Context(%main.stream zeroinitializer)
// CHECK: ret %"{{.*}}String" [[STREAM_RESULT]]

// Both instantiated forwarding methods load the embedded ServerStream method and
// preserve its receiver across the recover-frame alias used for interface calls.
// CHECK-LABEL: define linkonce %"{{.*}}String" @"{{.*}}/streamlib.(*GenericServerStream[main.Request,main.Response]).Context"(ptr %0){{.*}} {
// CHECK: [[GP_FIELD:%.*]] = getelementptr inbounds %"{{.*}}GenericServerStream[main.Request,main.Response]", ptr %0, i32 0, i32 0
// CHECK: [[GP_IFACE:%.*]] = load %"{{.*}}iface", ptr [[GP_FIELD]]
// CHECK: [[GP_DATA:%.*]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[GP_IFACE]])
// CHECK: [[GP_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[GP_IFACE]], 0
// CHECK: [[GP_SLOT:%.*]] = getelementptr ptr, ptr [[GP_ITAB]], i64 3
// CHECK: [[GP_METHOD:%.*]] = load ptr, ptr [[GP_SLOT]]
// CHECK: [[GP_CALL0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[GP_METHOD]], 0
// CHECK: [[GP_CALL:%.*]] = insertvalue { ptr, ptr } [[GP_CALL0]], ptr [[GP_DATA]], 1
// CHECK: [[GP_ALIAS_FN:%.*]] = extractvalue { ptr, ptr } [[GP_CALL]], 0
// CHECK: [[GP_RECOVER:%.*]] = call ptr @"{{.*}}StartRecoverFrameAlias"(ptr @"{{.*}}/streamlib.(*GenericServerStream[main.Request,main.Response]).Context", ptr [[GP_ALIAS_FN]])
// CHECK: [[GP_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[GP_CALL]], 1
// CHECK: [[GP_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[GP_CALL]], 0
// CHECK: [[GP_RESULT:%.*]] = call %"{{.*}}String" [[GP_CALL_FN]](ptr [[GP_CALL_DATA]])
// CHECK: call void @"{{.*}}EndRecoverFrameAlias"(ptr [[GP_RECOVER]])
// CHECK: ret %"{{.*}}String" [[GP_RESULT]]

// CHECK-LABEL: define linkonce %"{{.*}}String" @"{{.*}}/streamlib.GenericServerStream[main.Request,main.Response].Context"(%"{{.*}}GenericServerStream[main.Request,main.Response]" %0){{.*}} {
// CHECK: [[GV_FIELD:%.*]] = getelementptr inbounds %"{{.*}}GenericServerStream[main.Request,main.Response]", ptr {{%.*}}, i32 0, i32 0
// CHECK: [[GV_IFACE:%.*]] = load %"{{.*}}iface", ptr [[GV_FIELD]]
// CHECK: [[GV_DATA:%.*]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" [[GV_IFACE]])
// CHECK: [[GV_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[GV_IFACE]], 0
// CHECK: [[GV_SLOT:%.*]] = getelementptr ptr, ptr [[GV_ITAB]], i64 3
// CHECK: [[GV_METHOD:%.*]] = load ptr, ptr [[GV_SLOT]]
// CHECK: [[GV_CALL0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[GV_METHOD]], 0
// CHECK: [[GV_CALL:%.*]] = insertvalue { ptr, ptr } [[GV_CALL0]], ptr [[GV_DATA]], 1
// CHECK: [[GV_ALIAS_FN:%.*]] = extractvalue { ptr, ptr } [[GV_CALL]], 0
// CHECK: [[GV_RECOVER:%.*]] = call ptr @"{{.*}}StartRecoverFrameAlias"(ptr @"{{.*}}/streamlib.GenericServerStream[main.Request,main.Response].Context", ptr [[GV_ALIAS_FN]])
// CHECK: [[GV_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[GV_CALL]], 1
// CHECK: [[GV_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[GV_CALL]], 0
// CHECK: [[GV_RESULT:%.*]] = call %"{{.*}}String" [[GV_CALL_FN]](ptr [[GV_CALL_DATA]])
// CHECK: call void @"{{.*}}EndRecoverFrameAlias"(ptr [[GV_RECOVER]])
// CHECK: ret %"{{.*}}String" [[GV_RESULT]]
