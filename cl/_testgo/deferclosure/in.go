// LITTEST darwin/arm64 linux/amd64
package main

// Cover the distinct deferred function-value forms in this test without
// pinning the defer-node layout or basic blocks.
// CHECK-DAG: [[METHOD_FIRST_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = {{(linkonce_odr|private)}} constant { ptr, ptr } { ptr @"main.testDeferMethodLiteral$1", ptr null }
// CHECK-DAG: [[METHOD_SECOND_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = {{(linkonce_odr|private)}} constant { ptr, ptr } { ptr @"main.testDeferMethodLiteral$2", ptr null }
// CHECK-DAG: [[FIELD_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = {{(linkonce_odr|private)}} constant { ptr, ptr } { ptr @"main.testDeferFieldAccess$1", ptr null }
// CHECK-LABEL: define void @"main.(*Handler).SetHandler"(ptr %0, ptr %1){{.*}} {
// CHECK: [[HANDLER_FIELD:%.*]] = getelementptr inbounds nuw %main.Handler, ptr %0, i32 0, i32 0
// CHECK: store ptr %1, ptr [[HANDLER_FIELD]]

// CHECK-LABEL: define void @"main.(*Processor).SetCallback"(ptr %0, ptr %1){{.*}} {
// CHECK: [[PROCESSOR_FIELD:%.*]] = getelementptr inbounds nuw %main.Processor, ptr %0, i32 0, i32 0
// CHECK: store ptr %1, ptr [[PROCESSOR_FIELD]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.testDeferMethodLiteral()
// CHECK: call void @main.testDeferClosureValue()
// CHECK: call void @main.testDeferStructClosure()
// CHECK: call void @main.testDeferFieldAccess()

// CHECK-LABEL: define void @main.testDeferClosureValue(){{.*}} {
// The captured 42 is stored in the closure environment before its descriptor is deferred.
// CHECK: [[VALUE_ENV_ALLOCATION:%[0-9]+]] = alloca { { ptr, ptr }, { i64 } }
// CHECK: [[VALUE_CAPTURE:%.*]] = alloca i64
// CHECK: store volatile i64 42, ptr [[VALUE_CAPTURE]]
// CHECK-NEXT: [[VALUE_COPY:%.*]] = load i64, ptr [[VALUE_CAPTURE]]
// CHECK: [[VALUE_FN:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[VALUE_ENV_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[VALUE_ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[VALUE_ENV_ALLOCATION]], i32 0, i32 1
// CHECK: [[VALUE_ENV_CAPTURE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { i64 }, ptr [[VALUE_ENV]], i32 0, i32 0
// CHECK-NEXT: store i64 [[VALUE_COPY]], ptr [[VALUE_ENV_CAPTURE_SLOT]]
// CHECK: [[VALUE_FN_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[VALUE_FN]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.testDeferClosureValue$1", ptr [[VALUE_FN_CODE_SLOT]]
// CHECK-NEXT: [[VALUE_FN_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[VALUE_FN]], i32 0, i32 1
// CHECK-NEXT: store ptr [[VALUE_ENV]], ptr [[VALUE_FN_ENV_SLOT]]
// CHECK: [[VALUE_PREV_DEFER:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store ptr [[VALUE_FN]], ptr %{{.*}}
// When unwound, the stack defer record is removed without freeing its storage,
// and its function descriptor is invoked with the captured environment.
// CHECK: [[VALUE_NODE_DATA:%.*]] = load { ptr, i64, ptr }, ptr [[VALUE_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[VALUE_RUN_FN:%.*]] = extractvalue { ptr, i64, ptr } [[VALUE_NODE_DATA]], 2
// CHECK-NOT: /runtime/internal/runtime.FreeDeferNode
// CHECK: [[VALUE_RUN_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[VALUE_RUN_FN]]
// CHECK-NEXT: [[VALUE_RUN_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[VALUE_RUN_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[VALUE_RUN_ENV:%.*]] = load ptr, ptr [[VALUE_RUN_ENV_FIELD]]
// CHECK: [[VALUE_RUN_CODE_RAW_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[VALUE_RUN_FN]]
// CHECK-NEXT: [[VALUE_RUN_CODE_RAW_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[VALUE_RUN_CODE_RAW_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[VALUE_RUN_CODE_RAW:%.*]] = load ptr, ptr [[VALUE_RUN_CODE_RAW_FIELD]]
// CHECK: [[VALUE_RUN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[VALUE_RUN_CODE_RAW]])
// ARM64: call void [[VALUE_RUN_CODE]](ptr swiftself [[VALUE_RUN_ENV]])
// AMD64: call void [[VALUE_RUN_CODE]](ptr nest [[VALUE_RUN_ENV]])

// ARM64-LABEL: define hidden void @"main.testDeferClosureValue$1"(ptr swiftself %0){{.*}} {
// AMD64-LABEL: define hidden void @"main.testDeferClosureValue$1"(ptr nest %0){{.*}} {
// CHECK: [[VALUE_BODY_ADDR:%.*]] = getelementptr inbounds nuw { i64 }, ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK: [[VALUE_BODY_X:%.*]] = load i64, ptr [[VALUE_BODY_ADDR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[VALUE_BODY_X]])

// CHECK-LABEL: define void @main.testDeferFieldAccess(){{.*}} {
// The field value, rather than its declaration, is the descriptor placed into the defer record.
// CHECK: [[FIELD_HOLDER:%.*]] = alloca %main.FuncHolder
// CHECK: [[FIELD_STORE_SLOT:%.*]] = getelementptr inbounds nuw %main.FuncHolder, ptr [[FIELD_HOLDER]], i32 0, i32 0
// CHECK: store volatile ptr [[FIELD_DESCRIPTOR]], ptr [[FIELD_STORE_SLOT]]
// CHECK: [[FIELD_LOAD_SLOT:%.*]] = getelementptr inbounds nuw %main.FuncHolder, ptr [[FIELD_HOLDER]], i32 0, i32 0
// CHECK: [[FIELD_FN:%.*]] = load volatile ptr, ptr [[FIELD_LOAD_SLOT]]
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store ptr [[FIELD_FN]], ptr %{{.*}}
// CHECK: [[FIELD_NODE_DATA:%.*]] = load { ptr, i64, ptr }, ptr [[FIELD_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[FIELD_RUN_FN:%.*]] = extractvalue { ptr, i64, ptr } [[FIELD_NODE_DATA]], 2
// CHECK-NOT: /runtime/internal/runtime.FreeDeferNode
// CHECK: [[FIELD_RECOVER_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FIELD_RUN_FN]]
// CHECK-NEXT: [[FIELD_RECOVER_CODE_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FIELD_RECOVER_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[FIELD_RECOVER_CODE:%.*]] = load ptr, ptr [[FIELD_RECOVER_CODE_FIELD]]
// CHECK: [[FIELD_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr [[FIELD_RECOVER_CODE]])
// CHECK: [[FIELD_RUN_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FIELD_RUN_FN]]
// CHECK-NEXT: [[FIELD_RUN_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FIELD_RUN_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[FIELD_RUN_ENV:%.*]] = load ptr, ptr [[FIELD_RUN_ENV_FIELD]]
// CHECK: [[FIELD_RUN_CODE_RAW_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FIELD_RUN_FN]]
// CHECK-NEXT: [[FIELD_RUN_CODE_RAW_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FIELD_RUN_CODE_RAW_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[FIELD_RUN_CODE_RAW:%.*]] = load ptr, ptr [[FIELD_RUN_CODE_RAW_FIELD]]
// CHECK: [[FIELD_RUN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[FIELD_RUN_CODE_RAW]])
// ARM64: call void [[FIELD_RUN_CODE]](ptr swiftself [[FIELD_RUN_ENV]])
// AMD64: call void [[FIELD_RUN_CODE]](ptr nest [[FIELD_RUN_ENV]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[FIELD_RECOVER]])

// CHECK-LABEL: define void @"main.testDeferFieldAccess$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" { ptr @{{.*}}, i64 19 })

// CHECK-LABEL: define void @main.testDeferMethodLiteral(){{.*}} {
// CHECK: [[METHOD_HANDLER:%.*]] = alloca %main.Handler
// CHECK: call void @"main.(*Handler).SetHandler"(ptr [[METHOD_HANDLER]], ptr [[METHOD_FIRST_DESCRIPTOR]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// The deferred receiver and second literal are recorded independently.
// CHECK: store ptr [[METHOD_HANDLER]], ptr [[METHOD_RECEIVER_SLOT:%.*]]
// CHECK: store ptr [[METHOD_SECOND_DESCRIPTOR]], ptr [[METHOD_DESCRIPTOR_SLOT:%.*]]
// CHECK: [[METHOD_NODE_DATA:%.*]] = load { ptr, i64, ptr, ptr }, ptr [[METHOD_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[METHOD_RECEIVER:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[METHOD_NODE_DATA]], 2
// CHECK: [[METHOD_ARGUMENT:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[METHOD_NODE_DATA]], 3
// CHECK-NOT: /runtime/internal/runtime.FreeDeferNode
// The original descriptor is passed through the node without reboxing.
// CHECK-NOT: /runtime/internal/runtime.AllocU
// CHECK-NEXT: call void @"main.(*Handler).SetHandler"(ptr [[METHOD_RECEIVER]], ptr [[METHOD_ARGUMENT]])

// CHECK-LABEL: define void @"main.testDeferMethodLiteral$2"(i64 %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)

// CHECK-LABEL: define void @main.testDeferStructClosure(){{.*}} {
// CHECK: [[STRUCT_MSG:%.*]] = alloca %"{{.*}}String"
// CHECK: [[STRUCT_PROCESSOR:%.*]] = alloca %main.Processor
// CHECK: store volatile %"{{.*}}String" { ptr @{{.*}}, i64 8 }, ptr [[STRUCT_MSG]]
// CHECK-NEXT: [[STRUCT_COPY:%.*]] = load %"{{.*}}String", ptr [[STRUCT_MSG]]
// CHECK: [[STRUCT_ENV_ALLOCATION:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT: [[STRUCT_FN:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { %"{{.*}}String" } }, ptr [[STRUCT_ENV_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[STRUCT_ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { %"{{.*}}String" } }, ptr [[STRUCT_ENV_ALLOCATION]], i32 0, i32 1
// CHECK: [[STRUCT_ENV_CAPTURE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { %"{{.*}}String" }, ptr [[STRUCT_ENV]], i32 0, i32 0
// CHECK-NEXT: store %"{{.*}}String" [[STRUCT_COPY]], ptr [[STRUCT_ENV_CAPTURE_SLOT]]
// CHECK: [[STRUCT_FN_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[STRUCT_FN]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.testDeferStructClosure$1", ptr [[STRUCT_FN_CODE_SLOT]]
// CHECK-NEXT: [[STRUCT_FN_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[STRUCT_FN]], i32 0, i32 1
// CHECK-NEXT: store ptr [[STRUCT_ENV]], ptr [[STRUCT_FN_ENV_SLOT]]
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store ptr [[STRUCT_PROCESSOR]], ptr %{{.*}}
// CHECK: store ptr [[STRUCT_FN]], ptr %{{.*}}
// CHECK: [[STRUCT_NODE_DATA:%.*]] = load { ptr, i64, ptr, ptr }, ptr [[STRUCT_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[STRUCT_RUN_PROCESSOR:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[STRUCT_NODE_DATA]], 2
// CHECK: [[STRUCT_RUN_FN:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[STRUCT_NODE_DATA]], 3
// CHECK-NOT: /runtime/internal/runtime.FreeDeferNode
// CHECK: call void @"main.(*Processor).SetCallback"(ptr [[STRUCT_RUN_PROCESSOR]], ptr [[STRUCT_RUN_FN]])

// ARM64-LABEL: define hidden void @"main.testDeferStructClosure$1"(ptr swiftself %0, %"{{.*}}String" %1){{.*}} {
// AMD64-LABEL: define hidden void @"main.testDeferStructClosure$1"(ptr nest %0, %"{{.*}}String" %1){{.*}} {
// CHECK: [[STRUCT_BODY_MSG_ADDR:%.*]] = getelementptr inbounds nuw { %"{{.*}}String" }, ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK: [[STRUCT_BODY_MSG:%.*]] = load %"{{.*}}String", ptr [[STRUCT_BODY_MSG_ADDR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" %1)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[STRUCT_BODY_MSG]])

// Test deferred closure and method-value lowering.

// Type for holding a function

type Handler struct {
	fn func(int)
}

func (h *Handler) SetHandler(f func(int)) {
	h.fn = f
}

// Test case 1: Deferred method call with function literal (issue #1488)
// This triggers the temporary register name check (name[0] == '%')
func testDeferMethodLiteral() {
	var h Handler
	h.SetHandler(func(int) {})
	defer h.SetHandler(func(x int) {
		println("deferred", x)
	})
	println("before return")
}

// Test case 2: Defer a closure value directly
// This triggers the v.kind != vkFuncDecl && v.kind != vkFuncPtr branch
func testDeferClosureValue() {
	x := 42
	fn := func() {
		println("closure value", x)
	}
	defer fn()
	println("deferred closure")
}

// Test case 3: Complex scenario with closure in struct
type Processor struct {
	callback func(string)
}

func (p *Processor) SetCallback(cb func(string)) {
	p.callback = cb
}

func testDeferStructClosure() {
	var p Processor
	msg := "captured"
	// Defer a method call that takes a closure capturing a variable
	defer p.SetCallback(func(s string) {
		println(s, msg)
	})
	println("struct closure test")
}

// Test case 4: Defer a function accessed through a struct field
// This should trigger the v.kind != vkFuncDecl && v.kind != vkFuncPtr branch
// because accessing p.callback returns a value that's not a function declaration
type FuncHolder struct {
	callback func()
}

func testDeferFieldAccess() {
	var holder FuncHolder
	holder.callback = func() {
		println("callback from field")
	}
	// When we defer holder.callback directly, it's accessed as a field load
	// which might have a different value kind than vkFuncDecl/vkFuncPtr
	defer holder.callback()
	println("field access test")
}

func main() {
	testDeferMethodLiteral()
	testDeferClosureValue()
	testDeferStructClosure()
	testDeferFieldAccess()
}
