// LITTEST darwin/arm64 linux/amd64 windows/arm64 windows/amd64
// Scope: arch (arm64, amd64)
package main

import _ "unsafe" // for go:linkname

type (
	CDouble = float64
	CInt    = int32
)

//go:linkname cSqrt C.sqrt
func cSqrt(x CDouble) CDouble

// llgo:link cAbs C.abs
func cAbs(x CInt) CInt { return 0 }

// llgo:cdecl
type CCallback func(CInt) CInt

type Fn func(int) int

//go:noinline
func callCInt(fn func(CInt) CInt, x CInt) CInt {
	return fn(x)
}

type S struct {
	v int
}

func (s S) Inc(x int) int {
	return s.v + x
}

func (s *S) Add(x int) int {
	return s.v + x
}

func callCallback(cb CCallback, v CInt) CInt {
	return cb(v)
}

func globalAdd(x, y int) int {
	return x + y
}

func main() {
	nf := makeNoFree()
	wf := makeWithFree(3)
	_ = nf(1)
	_ = wf(2)

	g := globalAdd
	_ = g(1, 2)

	s := &S{v: 5}
	mv := s.Add
	_ = mv(7)
	me := (*S).Add
	_ = me(s, 8)

	var i interface{ Add(int) int } = s
	im := i.Add
	_ = im(9)

	cs := cSqrt
	_ = cs(4)
	_ = callCInt(cAbs, -3)

	cb := CCallback(func(x CInt) CInt { return x + 1 })
	_ = callCallback(cb, 7)
}

func makeNoFree() Fn {
	return func(x int) int { return x + 1 }
}

func makeWithFree(base int) Fn {
	return func(x int) int { return x + base }
}

// CHECK-DAG: [[ABS_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = {{(linkonce_odr|private)}} constant { ptr, ptr } { ptr @abs, ptr null }
// CHECK-DAG: [[NO_FREE_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = {{(linkonce_odr|private)}} constant { ptr, ptr } { ptr @"main.makeNoFree$1", ptr null }
// CHECK-LABEL: define i32 @main.callCInt(ptr %0, i32 %1){{.*}} {
// CHECK: [[C_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[C_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[C_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[C_ENV:%[0-9]+]] = load ptr, ptr [[C_ENV_FIELD]]
// CHECK: [[C_CODE_RAW_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %0
// CHECK-NEXT: [[C_CODE_RAW_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[C_CODE_RAW_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[C_CODE_RAW:%[0-9]+]] = load ptr, ptr [[C_CODE_RAW_FIELD]]
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[C_CODE_RAW]])
// ARM64-NEXT: [[C_RESULT:%[0-9]+]] = call i32 %__llgo_funcval_code(ptr swiftself [[C_ENV]], i32 %1)
// AMD64-NEXT: [[C_RESULT:%[0-9]+]] = call i32 %__llgo_funcval_code(ptr nest [[C_ENV]], i32 %1)
// CHECK: ret i32 [[C_RESULT]]

// CHECK-LABEL: define i32 @main.callCallback(ptr %0, i32 %1){{.*}} {
// CHECK: [[CB_RESULT:%[0-9]+]] = call i32 %0(i32 %1)
// CHECK: ret i32 [[CB_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// Immediately invoked methods keep one combined header/environment on the stack.
// CHECK: [[IFACE_METHOD_ENV_ALLOCATION:%[0-9]+]] = alloca { { ptr, ptr }, { %"{{.*}}iface" } }, align 8
// CHECK: [[METHOD_ENV_ALLOCATION:%[0-9]+]] = alloca { { ptr, ptr }, { ptr } }, align 8
// A closure without free variables carries a null environment.
// CHECK: [[NO_FREE:%.*]] = call ptr @main.makeNoFree()
// CHECK: [[WITH_FREE:%.*]] = call ptr @main.makeWithFree(i64 3)
// CHECK: [[NO_FREE_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[NO_FREE]]
// CHECK-NEXT: [[NO_FREE_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[NO_FREE_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[NO_FREE_ENV:%.*]] = load ptr, ptr [[NO_FREE_ENV_FIELD]]
// CHECK: [[NO_FREE_CODE_RAW_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[NO_FREE]]
// CHECK-NEXT: [[NO_FREE_CODE_RAW_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[NO_FREE_CODE_RAW_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[NO_FREE_CODE_RAW:%.*]] = load ptr, ptr [[NO_FREE_CODE_RAW_FIELD]]
// CHECK: [[NO_FREE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[NO_FREE_CODE_RAW]])
// ARM64: call i64 [[NO_FREE_CODE]](ptr swiftself [[NO_FREE_ENV]], i64 1)
// AMD64: call i64 [[NO_FREE_CODE]](ptr nest [[NO_FREE_ENV]], i64 1)
// The free-variable closure is invoked with the environment returned by makeWithFree.
// CHECK: [[WITH_FREE_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[WITH_FREE]]
// CHECK-NEXT: [[WITH_FREE_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[WITH_FREE_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[WITH_FREE_ENV:%.*]] = load ptr, ptr [[WITH_FREE_ENV_FIELD]]
// CHECK: [[WITH_FREE_CODE_RAW_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[WITH_FREE]]
// CHECK-NEXT: [[WITH_FREE_CODE_RAW_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[WITH_FREE_CODE_RAW_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[WITH_FREE_CODE_RAW:%.*]] = load ptr, ptr [[WITH_FREE_CODE_RAW_FIELD]]
// CHECK: [[WITH_FREE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[WITH_FREE_CODE_RAW]])
// ARM64: call i64 [[WITH_FREE_CODE]](ptr swiftself [[WITH_FREE_ENV]], i64 2)
// AMD64: call i64 [[WITH_FREE_CODE]](ptr nest [[WITH_FREE_ENV]], i64 2)
// CHECK: call i64 @main.globalAdd(i64 1, i64 2)
// A bound pointer method stores the receiver in the closure environment.
// CHECK: [[S:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocAtomicU"(i64 8)
// CHECK-NEXT: call void @llvm.memset.p0.i64(ptr [[S]], i8 0, i64 8, i1 false)
// CHECK-NEXT: [[S_FIELD:%[0-9]+]] = getelementptr inbounds nuw %main.S, ptr [[S]], i32 0, i32 0
// CHECK-NEXT: store i64 5, ptr [[S_FIELD]]
// CHECK: call void @llvm.memset.p0.i64(ptr [[METHOD_ENV_ALLOCATION]], i8 0, i64 24, i1 false)
// CHECK-NEXT: [[METHOD_FN:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[METHOD_ENV_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[METHOD_ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[METHOD_ENV_ALLOCATION]], i32 0, i32 1
// CHECK: [[METHOD_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr }, ptr [[METHOD_ENV]], i32 0, i32 0
// CHECK: store ptr [[S]], ptr [[METHOD_ENV_SLOT]]
// CHECK: [[METHOD_FN_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[METHOD_FN]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.(*S).Add$bound", ptr [[METHOD_FN_CODE_SLOT]]
// CHECK-NEXT: [[METHOD_FN_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[METHOD_FN]], i32 0, i32 1
// CHECK-NEXT: store ptr [[METHOD_ENV]], ptr [[METHOD_FN_ENV_SLOT]]
// CHECK: [[METHOD_CALL_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[METHOD_FN]]
// CHECK-NEXT: [[METHOD_CALL_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[METHOD_CALL_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[METHOD_CALL_ENV:%.*]] = load ptr, ptr [[METHOD_CALL_ENV_FIELD]]
// CHECK-NOT: call ptr asm
// ARM64: call i64 @"main.(*S).Add$bound"(ptr swiftself [[METHOD_CALL_ENV]], i64 7)
// AMD64: call i64 @"main.(*S).Add$bound"(ptr nest [[METHOD_CALL_ENV]], i64 7)
// A method expression uses the receiver as an ordinary first argument.
// CHECK: call i64 @"main.(*S).Add$thunk"(ptr [[S]], i64 8)
// The interface method value keeps the same interface payload and checks it is non-nil.
// CHECK: [[IFACE:%.*]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[S]], 1
// CHECK: [[IFACE_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[IFACE]])
// CHECK: [[IFACE_OK:%.*]] = icmp ne ptr [[IFACE_TYPE]], null
// CHECK: br i1 [[IFACE_OK]], label %{{.*}}, label %{{.*}}
// CHECK: call void @llvm.memset.p0.i64(ptr [[IFACE_METHOD_ENV_ALLOCATION]], i8 0, i64 32, i1 false)
// CHECK-NEXT: [[IFACE_METHOD:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { %"{{.*}}iface" } }, ptr [[IFACE_METHOD_ENV_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[IFACE_METHOD_ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { %"{{.*}}iface" } }, ptr [[IFACE_METHOD_ENV_ALLOCATION]], i32 0, i32 1
// CHECK: [[IFACE_METHOD_SLOT:%.*]] = getelementptr inbounds nuw { %"{{.*}}iface" }, ptr [[IFACE_METHOD_ENV]], i32 0, i32 0
// CHECK: store %"{{.*}}iface" [[IFACE]], ptr [[IFACE_METHOD_SLOT]]
// CHECK: [[IFACE_METHOD_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[IFACE_METHOD]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.interface{Add(int) int}.Add$bound", ptr [[IFACE_METHOD_CODE_SLOT]]
// CHECK-NEXT: [[IFACE_METHOD_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[IFACE_METHOD]], i32 0, i32 1
// CHECK-NEXT: store ptr [[IFACE_METHOD_ENV]], ptr [[IFACE_METHOD_ENV_SLOT]]
// CHECK: [[IFACE_CALL_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[IFACE_METHOD]]
// CHECK-NEXT: [[IFACE_CALL_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[IFACE_CALL_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[IFACE_CALL_ENV:%.*]] = load ptr, ptr [[IFACE_CALL_ENV_FIELD]]
// CHECK-NOT: call ptr asm
// ARM64: call i64 @"main.interface{Add(int) int}.Add$bound"(ptr swiftself [[IFACE_CALL_ENV]], i64 9)
// AMD64: call i64 @"main.interface{Add(int) int}.Add$bound"(ptr nest [[IFACE_CALL_ENV]], i64 9)
// CHECK: call double @sqrt(double 4.000000e+00)
// CHECK: call i32 @main.callCInt(ptr [[ABS_DESCRIPTOR]], i32 -3)
// CHECK: call i32 @main.callCallback(ptr @"main.main$1", i32 7)

// CHECK-LABEL: define i32 @"main.main$1"(i32 %0){{.*}} {
// CHECK: [[CALLBACK_RESULT:%[0-9]+]] = add i32 %0, 1
// CHECK: ret i32 [[CALLBACK_RESULT]]

// CHECK-LABEL: define ptr @main.makeNoFree(){{.*}} {
// CHECK:   ret ptr [[NO_FREE_DESCRIPTOR]]

// CHECK-LABEL: define i64 @"main.makeNoFree$1"(i64 %0){{.*}} {
// CHECK: [[NO_FREE_RESULT:%[0-9]+]] = add i64 %0, 1
// CHECK: ret i64 [[NO_FREE_RESULT]]

// CHECK-LABEL: define ptr @main.makeWithFree(i64 %0){{.*}} {
// CHECK: [[BASE_ADDR:%[0-9]+]] = alloca i64
// CHECK: store i64 %0, ptr [[BASE_ADDR]]
// CHECK-NEXT: [[BASE_VALUE:%[0-9]+]] = load i64, ptr [[BASE_ADDR]]
// CHECK: [[WITH_FREE_ENV_OUT_ALLOCATION:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocAtomicU"(i64 24)
// CHECK-NEXT: [[WITH_FREE_FN:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[WITH_FREE_ENV_OUT_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[WITH_FREE_ENV_OUT:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[WITH_FREE_ENV_OUT_ALLOCATION]], i32 0, i32 1
// CHECK-NEXT: [[WITH_FREE_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { i64 }, ptr [[WITH_FREE_ENV_OUT]], i32 0, i32 0
// CHECK-NEXT: store i64 [[BASE_VALUE]], ptr [[WITH_FREE_ENV_SLOT]]
// CHECK: [[WITH_FREE_FN_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[WITH_FREE_FN]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.makeWithFree$1", ptr [[WITH_FREE_FN_CODE_SLOT]]
// CHECK-NEXT: [[WITH_FREE_FN_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[WITH_FREE_FN]], i32 0, i32 1
// CHECK-NEXT: store ptr [[WITH_FREE_ENV_OUT]], ptr [[WITH_FREE_FN_ENV_SLOT]]
// Named function conversion preserves the descriptor identity.
// CHECK-NEXT: ret ptr [[WITH_FREE_FN]]

// ARM64-LABEL: define hidden i64 @"main.makeWithFree$1"(ptr swiftself %0, i64 %1){{.*}} {
// AMD64-LABEL: define hidden i64 @"main.makeWithFree$1"(ptr nest %0, i64 %1){{.*}} {
// CHECK: [[FREE_ADDR:%[0-9]+]] = getelementptr inbounds nuw { i64 }, ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT: [[FREE_VALUE:%[0-9]+]] = load i64, ptr [[FREE_ADDR]]
// CHECK-NEXT: [[FREE_RESULT:%[0-9]+]] = add i64 %1, [[FREE_VALUE]]
// CHECK: ret i64 [[FREE_RESULT]]

// ARM64-LABEL: define hidden i64 @"main.(*S).Add$bound"(ptr swiftself %0, i64 %1){{.*}} {
// AMD64-LABEL: define hidden i64 @"main.(*S).Add$bound"(ptr nest %0, i64 %1){{.*}} {
// CHECK: [[BOUND_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[BOUND_RECEIVER:%[0-9]+]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK-NEXT: [[BOUND_RESULT:%[0-9]+]] = call i64 @"main.(*S).Add"(ptr [[BOUND_RECEIVER]], i64 %1)
// CHECK: ret i64 [[BOUND_RESULT]]

// CHECK-LABEL: define i64 @"main.(*S).Add$thunk"(ptr %0, i64 %1){{.*}} {
// CHECK: [[THUNK_RESULT:%[0-9]+]] = call i64 @"main.(*S).Add"(ptr %0, i64 %1)
// CHECK: ret i64 [[THUNK_RESULT]]

// ARM64-LABEL: define hidden i64 @"main.interface{Add(int) int}.Add$bound"(ptr swiftself %0, i64 %1){{.*}} {
// AMD64-LABEL: define hidden i64 @"main.interface{Add(int) int}.Add$bound"(ptr nest %0, i64 %1){{.*}} {
// CHECK: [[BOUND_IFACE_ENV:%[0-9]+]] = load { %"{{.*}}iface" }, ptr %0
// CHECK: [[BOUND_IFACE:%[0-9]+]] = extractvalue { %"{{.*}}iface" } [[BOUND_IFACE_ENV]], 0
// CHECK: [[BOUND_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrDataBuffer"(%"{{.*}}iface" [[BOUND_IFACE]], ptr %{{[0-9]+}})
// CHECK: [[BOUND_ITAB:%.*]] = extractvalue %"{{.*}}iface" [[BOUND_IFACE]], 0
// CHECK: [[BOUND_SLOT:%.*]] = getelementptr ptr, ptr [[BOUND_ITAB]], i64 3
// CHECK: [[BOUND_CODE:%.*]] = load ptr, ptr [[BOUND_SLOT]]
// CHECK: [[BOUND_PAIR_CODE:%.*]] = insertvalue { ptr, ptr } undef, ptr [[BOUND_CODE]], 0
// CHECK: [[BOUND_PAIR:%.*]] = insertvalue { ptr, ptr } [[BOUND_PAIR_CODE]], ptr [[BOUND_DATA]], 1
// CHECK: [[RECOVER_CODE:%.*]] = extractvalue { ptr, ptr } [[BOUND_PAIR]], 0
// CHECK: [[RECOVER:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrameAlias"(ptr @"main.interface{Add(int) int}.Add$bound", ptr [[RECOVER_CODE]])
// CHECK: [[IFACE_CALL_DATA:%.*]] = extractvalue { ptr, ptr } [[BOUND_PAIR]], 1
// CHECK: [[IFACE_CALL_CODE:%.*]] = extractvalue { ptr, ptr } [[BOUND_PAIR]], 0
// CHECK: [[IFACE_BOUND_RESULT:%.*]] = call i64 [[IFACE_CALL_CODE]](ptr [[IFACE_CALL_DATA]], i64 %1)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrameAlias"(ptr [[RECOVER]])
// CHECK: ret i64 [[IFACE_BOUND_RESULT]]
