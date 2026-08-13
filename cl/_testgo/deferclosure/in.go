// LITTEST
package main

// Cover the distinct deferred function-value forms in this test without
// pinning the defer-node layout or basic blocks.
// CHECK-LABEL: define void @"main.(*Handler).SetHandler"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[HANDLER_FIELD:%.*]] = getelementptr inbounds %main.Handler, ptr %0, i32 0, i32 0
// CHECK: store { ptr, ptr } %1, ptr [[HANDLER_FIELD]]

// CHECK-LABEL: define void @"main.(*Processor).SetCallback"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[PROCESSOR_FIELD:%.*]] = getelementptr inbounds %main.Processor, ptr %0, i32 0, i32 0
// CHECK: store { ptr, ptr } %1, ptr [[PROCESSOR_FIELD]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.testDeferMethodLiteral()
// CHECK: call void @main.testDeferClosureValue()
// CHECK: call void @main.testDeferStructClosure()
// CHECK: call void @main.testDeferFieldAccess()

// CHECK-LABEL: define void @main.testDeferClosureValue(){{.*}} {
// The captured 42 is stored in the closure environment before the function pair is deferred.
// CHECK: [[VALUE_CAPTURE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: store i64 42, ptr [[VALUE_CAPTURE]]
// CHECK: [[VALUE_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: store ptr [[VALUE_CAPTURE]], ptr %{{.*}}
// CHECK: [[VALUE_FN:%.*]] = insertvalue { ptr, ptr } { ptr @"main.testDeferClosureValue$1", ptr undef }, ptr [[VALUE_ENV]], 1
// CHECK: [[VALUE_PREV_DEFER:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store { ptr, ptr } [[VALUE_FN]], ptr %{{.*}}
// When unwound, the deferred function pair is removed, freed, and invoked with its environment.
// CHECK: [[VALUE_NODE_DATA:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[VALUE_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[VALUE_RUN_FN:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[VALUE_NODE_DATA]], 2
// CHECK: call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr [[VALUE_NODE]])
// CHECK: [[VALUE_RUN_ENV:%.*]] = extractvalue { ptr, ptr } [[VALUE_RUN_FN]], 1
// CHECK: [[VALUE_RUN_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[VALUE_RUN_FN]], 0
// CHECK: [[VALUE_RUN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[VALUE_RUN_CODE_RAW]])
// CHECK: call void [[VALUE_RUN_CODE]](ptr {{(nest|swiftself)}} [[VALUE_RUN_ENV]])

// CHECK-LABEL: define void @"main.testDeferClosureValue$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[VALUE_BODY_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[VALUE_BODY_ADDR:%.*]] = extractvalue { ptr } [[VALUE_BODY_ENV]], 0
// CHECK: [[VALUE_BODY_X:%.*]] = load i64, ptr [[VALUE_BODY_ADDR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[VALUE_BODY_X]])

// CHECK-LABEL: define void @main.testDeferFieldAccess(){{.*}} {
// The field value, rather than its declaration, is the pair placed into the defer record.
// CHECK: [[FIELD_HOLDER:%.*]] = alloca %main.FuncHolder
// CHECK: [[FIELD_STORE_SLOT:%.*]] = getelementptr inbounds %main.FuncHolder, ptr [[FIELD_HOLDER]], i32 0, i32 0
// CHECK: store volatile { ptr, ptr } { ptr @"main.testDeferFieldAccess$1", ptr null }, ptr [[FIELD_STORE_SLOT]]
// CHECK: [[FIELD_LOAD_SLOT:%.*]] = getelementptr inbounds %main.FuncHolder, ptr [[FIELD_HOLDER]], i32 0, i32 0
// CHECK: [[FIELD_FN:%.*]] = load volatile { ptr, ptr }, ptr [[FIELD_LOAD_SLOT]]
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store { ptr, ptr } [[FIELD_FN]], ptr %{{.*}}
// CHECK: [[FIELD_NODE_DATA:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[FIELD_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[FIELD_RUN_FN:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[FIELD_NODE_DATA]], 2
// CHECK: call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr [[FIELD_NODE]])
// CHECK: [[FIELD_RECOVER_CODE:%.*]] = extractvalue { ptr, ptr } [[FIELD_RUN_FN]], 0
// CHECK: [[FIELD_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr [[FIELD_RECOVER_CODE]])
// CHECK: [[FIELD_RUN_ENV:%.*]] = extractvalue { ptr, ptr } [[FIELD_RUN_FN]], 1
// CHECK: [[FIELD_RUN_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[FIELD_RUN_FN]], 0
// CHECK: [[FIELD_RUN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[FIELD_RUN_CODE_RAW]])
// CHECK: call void [[FIELD_RUN_CODE]](ptr {{(nest|swiftself)}} [[FIELD_RUN_ENV]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[FIELD_RECOVER]])

// CHECK-LABEL: define void @"main.testDeferFieldAccess$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" { ptr @{{.*}}, i64 19 })

// CHECK-LABEL: define void @main.testDeferMethodLiteral(){{.*}} {
// CHECK: [[METHOD_HANDLER:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: call void @"main.(*Handler).SetHandler"(ptr [[METHOD_HANDLER]], { ptr, ptr } { ptr @"main.testDeferMethodLiteral$1", ptr null })
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// The deferred receiver and second literal are recorded independently.
// CHECK: store ptr [[METHOD_HANDLER]], ptr [[METHOD_RECEIVER_SLOT:%.*]]
// CHECK: store ptr @"main.testDeferMethodLiteral$2", ptr [[METHOD_CODE_SLOT:%.*]]
// CHECK: [[METHOD_NODE_DATA:%.*]] = load { ptr, i64, ptr, ptr }, ptr [[METHOD_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[METHOD_RECEIVER:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[METHOD_NODE_DATA]], 2
// CHECK: [[METHOD_CODE:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[METHOD_NODE_DATA]], 3
// CHECK: call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr [[METHOD_NODE]])
// CHECK: [[METHOD_PAIR_0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[METHOD_CODE]], 0
// CHECK: [[METHOD_PAIR:%.*]] = insertvalue { ptr, ptr } [[METHOD_PAIR_0]], ptr null, 1
// CHECK: call void @"main.(*Handler).SetHandler"(ptr [[METHOD_RECEIVER]], { ptr, ptr } [[METHOD_PAIR]])

// CHECK-LABEL: define void @"main.testDeferMethodLiteral$2"(i64 %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)

// CHECK-LABEL: define void @main.testDeferStructClosure(){{.*}} {
// CHECK: [[STRUCT_PROCESSOR:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: [[STRUCT_MSG:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: store %"{{.*}}String" { ptr @{{.*}}, i64 8 }, ptr [[STRUCT_MSG]]
// CHECK: [[STRUCT_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: store ptr [[STRUCT_MSG]], ptr %{{.*}}
// CHECK: [[STRUCT_FN:%.*]] = insertvalue { ptr, ptr } { ptr @"main.testDeferStructClosure$1", ptr undef }, ptr [[STRUCT_ENV]], 1
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: store ptr [[STRUCT_PROCESSOR]], ptr %{{.*}}
// CHECK: store { ptr, ptr } [[STRUCT_FN]], ptr %{{.*}}
// CHECK: [[STRUCT_NODE_DATA:%.*]] = load { ptr, i64, ptr, { ptr, ptr } }, ptr [[STRUCT_NODE:%[-A-Za-z0-9_.]+]]
// CHECK: [[STRUCT_RUN_PROCESSOR:%.*]] = extractvalue { ptr, i64, ptr, { ptr, ptr } } [[STRUCT_NODE_DATA]], 2
// CHECK: [[STRUCT_RUN_FN:%.*]] = extractvalue { ptr, i64, ptr, { ptr, ptr } } [[STRUCT_NODE_DATA]], 3
// CHECK: call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr [[STRUCT_NODE]])
// CHECK: call void @"main.(*Processor).SetCallback"(ptr [[STRUCT_RUN_PROCESSOR]], { ptr, ptr } [[STRUCT_RUN_FN]])

// CHECK-LABEL: define void @"main.testDeferStructClosure$1"(ptr {{(nest|swiftself)}} %0, %"{{.*}}String" %1){{.*}} {
// CHECK: [[STRUCT_BODY_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[STRUCT_BODY_MSG_ADDR:%.*]] = extractvalue { ptr } [[STRUCT_BODY_ENV]], 0
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
