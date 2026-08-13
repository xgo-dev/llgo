// LITTEST
package main

// Zero-sized captures need no environment, while real captures and bound
// methods use LLVM's hidden closure-environment parameter.
// CHECK-LABEL: define i1 @"main.(*nilReceiver).IsNil"(ptr %0){{.*}} {
// CHECK: [[IS_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK-NEXT: ret i1 [[IS_NIL]]
// CHECK-LABEL: define { { ptr, ptr }, ptr } @main.zeroSizedAddressCapture(){{.*}} {
// CHECK: ret { { ptr, ptr }, ptr } { { ptr, ptr } { ptr @"main.zeroSizedAddressCapture$1", ptr null }, ptr @"__llgo.moduleZeroSizedAlloc$" }
// CHECK-LABEL: define ptr @"main.zeroSizedAddressCapture$1"(){{.*}} {
// CHECK: ret ptr @"__llgo.moduleZeroSizedAlloc$"
// CHECK-LABEL: define { ptr, ptr } @main.zeroSizedCapture(){{.*}} {
// CHECK: ret { ptr, ptr } { ptr @"main.zeroSizedCapture$1", ptr null }
// CHECK-LABEL: define i64 @"main.zeroSizedCapture$1"(){{.*}} {
// CHECK: br i1 false, label %{{.*}}, label %{{.*}}
// CHECK: ret i64 0
// CHECK: ret i64 42
// CHECK-LABEL: define { ptr, ptr } @main.zeroSizedPointerCapture(ptr %0){{.*}} {
// CHECK: [[ZSP_VALUE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: store ptr %0, ptr [[ZSP_VALUE]]
// CHECK-NEXT: [[ZSP_ENV:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: store ptr [[ZSP_VALUE]], ptr %{{[0-9]+}}
// CHECK: [[ZSP_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.zeroSizedPointerCapture$1", ptr undef }, ptr [[ZSP_ENV]], 1
// CHECK-NEXT: ret { ptr, ptr } [[ZSP_CLOSURE]]
// CHECK-LABEL: define i1 @"main.zeroSizedPointerCapture$1"(ptr {{(nest|swiftself)}}
// CHECK: [[ZSP_ENV_VALUE:%[0-9]+]] = load { ptr }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[ZSP_SLOT:%[0-9]+]] = extractvalue { ptr } [[ZSP_ENV_VALUE]], 0
// CHECK-NEXT: [[ZSP_POINTER:%[0-9]+]] = load ptr, ptr [[ZSP_SLOT]]
// CHECK-NEXT: [[ZSP_IS_NIL:%[0-9]+]] = icmp eq ptr [[ZSP_POINTER]], null
// CHECK-NEXT: ret i1 [[ZSP_IS_NIL]]
// CHECK-LABEL: define i1 @"main.(*nilReceiver).IsNil$bound"(ptr {{(nest|swiftself)}}
// CHECK: [[BOUND_ENV:%[0-9]+]] = load { ptr }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[BOUND_RECEIVER:%[0-9]+]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK-NEXT: [[BOUND_RESULT:%[0-9]+]] = call i1 @"main.(*nilReceiver).IsNil"(ptr [[BOUND_RECEIVER]])
// CHECK-NEXT: ret i1 [[BOUND_RESULT]]
// CHECK-LABEL: define i1 @"main.interface{IsNil() bool}.IsNil$bound"(ptr {{(nest|swiftself)}}
// CHECK: [[IB_ENV:%[0-9]+]] = load { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[IB_IFACE:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.iface" } [[IB_ENV]], 0
// CHECK-NEXT: [[IB_DATA:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" [[IB_IFACE]])
// CHECK: [[IB_METHOD:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[IB_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[IB_METHOD]], 0
// CHECK-NEXT: [[IB_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[IB_PAIR0]], ptr [[IB_DATA]], 1
// CHECK: [[IB_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[IB_PAIR]], 0
// CHECK-NEXT: [[IB_FRAME:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrameAlias"(ptr @"main.interface{IsNil() bool}.IsNil$bound", ptr [[IB_CODE]])
// CHECK: [[IB_CALL_DATA:%[0-9]+]] = extractvalue { ptr, ptr } [[IB_PAIR]], 1
// CHECK-NEXT: [[IB_CALL_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[IB_PAIR]], 0
// CHECK-NEXT: [[IB_RESULT:%[0-9]+]] = call i1 [[IB_CALL_CODE]](ptr [[IB_CALL_DATA]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrameAlias"(ptr [[IB_FRAME]])
// CHECK-NEXT: ret i1 [[IB_RESULT]]

type nilReceiver struct{}

func (p *nilReceiver) IsNil() bool {
	return p == nil
}

func zeroSizedCapture() func() int {
	captured := struct{}{}
	return func() int {
		if captured != (struct{}{}) {
			return 0
		}
		return 42
	}
}

func zeroSizedAddressCapture() (func() *struct{}, *struct{}) {
	captured := struct{}{}
	return func() *struct{} { return &captured }, &captured
}

func zeroSizedPointerCapture(pointer *struct{}) func() bool {
	return func() bool { return pointer == nil }
}

func main() {
	if zeroSizedCapture()() != 42 {
		panic("zero-sized capture")
	}
	addressClosure, address := zeroSizedAddressCapture()
	if addressClosure() != address {
		panic("zero-sized capture address")
	}
	if !zeroSizedPointerCapture(nil)() {
		panic("zero-sized pointer capture")
	}

	var receiver *nilReceiver
	method := receiver.IsNil
	if !method() {
		panic("nil receiver method value")
	}

	var typedNil interface{ IsNil() bool } = receiver
	interfaceMethod := typedNil.IsNil
	if !interfaceMethod() {
		panic("typed-nil interface method value")
	}
	println("ok")
}
