// LITTEST
package main

import _ "unsafe" // for go:linkname

import "github.com/goplus/lib/c"

//go:linkname cSqrt C.sqrt
func cSqrt(x c.Double) c.Double

// llgo:link cAbs C.abs
func cAbs(x c.Int) c.Int { return 0 }

// llgo:type C
type CCallback func(c.Int) c.Int

type Fn func(int) int

//go:noinline
func callCInt(fn func(c.Int) c.Int, x c.Int) c.Int {
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

func callCallback(cb CCallback, v c.Int) c.Int {
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

	cb := CCallback(func(x c.Int) c.Int { return x + 1 })
	_ = callCallback(cb, 7)
}

func makeNoFree() Fn {
	return func(x int) int { return x + 1 }
}

func makeWithFree(base int) Fn {
	return func(x int) int { return x + base }
}

// CHECK-LABEL: define i32 @main.callCInt({ ptr, ptr } %0, i32 %1){{.*}} {
// CHECK: [[C_ENV:%[0-9]+]] = extractvalue { ptr, ptr } %0, 1
// CHECK-NEXT: [[C_CODE_RAW:%[0-9]+]] = extractvalue { ptr, ptr } %0, 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[C_CODE_RAW]])
// CHECK-NEXT: [[C_RESULT:%[0-9]+]] = call i32 %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[C_ENV]], i32 %1)
// CHECK: ret i32 [[C_RESULT]]

// CHECK-LABEL: define i32 @main.callCallback(ptr %0, i32 %1){{.*}} {
// CHECK: [[CB_RESULT:%[0-9]+]] = call i32 %0(i32 %1)
// CHECK: ret i32 [[CB_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// A closure without free variables carries a null environment.
// CHECK: [[NO_FREE:%.*]] = call %main.Fn @main.makeNoFree()
// CHECK: [[WITH_FREE:%.*]] = call %main.Fn @main.makeWithFree(i64 3)
// CHECK: [[NO_FREE_ENV:%.*]] = extractvalue %main.Fn [[NO_FREE]], 1
// CHECK: [[NO_FREE_CODE_RAW:%.*]] = extractvalue %main.Fn [[NO_FREE]], 0
// CHECK: [[NO_FREE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[NO_FREE_CODE_RAW]])
// CHECK: call i64 [[NO_FREE_CODE]](ptr {{(nest|swiftself)}} [[NO_FREE_ENV]], i64 1)
// The free-variable closure is invoked with the environment returned by makeWithFree.
// CHECK: [[WITH_FREE_ENV:%.*]] = extractvalue %main.Fn [[WITH_FREE]], 1
// CHECK: [[WITH_FREE_CODE_RAW:%.*]] = extractvalue %main.Fn [[WITH_FREE]], 0
// CHECK: [[WITH_FREE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[WITH_FREE_CODE_RAW]])
// CHECK: call i64 [[WITH_FREE_CODE]](ptr {{(nest|swiftself)}} [[WITH_FREE_ENV]], i64 2)
// CHECK: call i64 @main.globalAdd(i64 1, i64 2)
// A bound pointer method stores the receiver in the closure environment.
// CHECK: [[S:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: [[S_FIELD:%[0-9]+]] = getelementptr inbounds %main.S, ptr [[S]], i32 0, i32 0
// CHECK-NEXT: store i64 5, ptr [[S_FIELD]]
// CHECK: [[METHOD_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: [[METHOD_ENV_SLOT:%.*]] = getelementptr inbounds { ptr }, ptr [[METHOD_ENV]], i32 0, i32 0
// CHECK: store ptr [[S]], ptr [[METHOD_ENV_SLOT]]
// CHECK: [[METHOD_FN:%.*]] = insertvalue { ptr, ptr } { ptr @"main.(*S).Add$bound", ptr undef }, ptr [[METHOD_ENV]], 1
// CHECK: [[METHOD_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[METHOD_FN]], 1
// CHECK: [[METHOD_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[METHOD_FN]], 0
// CHECK: [[METHOD_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[METHOD_CODE_RAW]])
// CHECK: call i64 [[METHOD_CODE]](ptr {{(nest|swiftself)}} [[METHOD_CALL_ENV]], i64 7)
// A method expression uses the receiver as an ordinary first argument.
// CHECK: call i64 @"main.(*S).Add$thunk"(ptr [[S]], i64 8)
// The interface method value keeps the same interface payload and checks it is non-nil.
// CHECK: [[ITAB:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr {{.*}}, ptr @"*_llgo_main.S")
// CHECK: [[IFACE_ITAB:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[ITAB]], 0
// CHECK: [[IFACE:%.*]] = insertvalue %"{{.*}}iface" [[IFACE_ITAB]], ptr [[S]], 1
// CHECK: [[IFACE_TYPE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[IFACE]])
// CHECK: [[IFACE_OK:%.*]] = icmp ne ptr [[IFACE_TYPE]], null
// CHECK: br i1 [[IFACE_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[IFACE_METHOD_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK: [[IFACE_METHOD_SLOT:%.*]] = getelementptr inbounds { %"{{.*}}iface" }, ptr [[IFACE_METHOD_ENV]], i32 0, i32 0
// CHECK: store %"{{.*}}iface" [[IFACE]], ptr [[IFACE_METHOD_SLOT]]
// CHECK: [[IFACE_METHOD:%.*]] = insertvalue { ptr, ptr } { ptr @"main.interface{Add(int) int}.Add$bound", ptr undef }, ptr [[IFACE_METHOD_ENV]], 1
// CHECK: [[IFACE_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[IFACE_METHOD]], 1
// CHECK: [[IFACE_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[IFACE_METHOD]], 0
// CHECK: [[IFACE_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[IFACE_CODE_RAW]])
// CHECK: call i64 [[IFACE_CODE]](ptr {{(nest|swiftself)}} [[IFACE_CALL_ENV]], i64 9)
// CHECK: call double @sqrt(double 4.000000e+00)
// CHECK: call i32 @main.callCInt({ ptr, ptr } { ptr @abs, ptr null }, i32 -3)
// CHECK: call i32 @main.callCallback(ptr @"main.main$1", i32 7)

// CHECK-LABEL: define i32 @"main.main$1"(i32 %0){{.*}} {
// CHECK: [[CALLBACK_RESULT:%[0-9]+]] = add i32 %0, 1
// CHECK: ret i32 [[CALLBACK_RESULT]]

// CHECK-LABEL: define %main.Fn @main.makeNoFree(){{.*}} {
// CHECK:   ret %main.Fn { ptr @"main.makeNoFree$1", ptr null }

// CHECK-LABEL: define i64 @"main.makeNoFree$1"(i64 %0){{.*}} {
// CHECK: [[NO_FREE_RESULT:%[0-9]+]] = add i64 %0, 1
// CHECK: ret i64 [[NO_FREE_RESULT]]

// CHECK-LABEL: define %main.Fn @main.makeWithFree(i64 %0){{.*}} {
// CHECK: [[BASE_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: store i64 %0, ptr [[BASE_ADDR]]
// CHECK-NEXT: [[WITH_FREE_ENV_OUT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: [[WITH_FREE_ENV_SLOT:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[WITH_FREE_ENV_OUT]], i32 0, i32 0
// CHECK-NEXT: store ptr [[BASE_ADDR]], ptr [[WITH_FREE_ENV_SLOT]]
// CHECK-NEXT: [[WITH_FREE_FN:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.makeWithFree$1", ptr undef }, ptr [[WITH_FREE_ENV_OUT]], 1
// CHECK-NEXT: [[WITH_FREE_OUT:%[0-9]+]] = alloca %main.Fn
// CHECK-NEXT: store { ptr, ptr } [[WITH_FREE_FN]], ptr [[WITH_FREE_OUT]]
// CHECK-NEXT: [[WITH_FREE_RET:%[0-9]+]] = load %main.Fn, ptr [[WITH_FREE_OUT]]
// CHECK: ret %main.Fn [[WITH_FREE_RET]]

// CHECK-LABEL: define i64 @"main.makeWithFree$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
// CHECK: [[FREE_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[FREE_ADDR:%[0-9]+]] = extractvalue { ptr } [[FREE_ENV]], 0
// CHECK-NEXT: [[FREE_VALUE:%[0-9]+]] = load i64, ptr [[FREE_ADDR]]
// CHECK-NEXT: [[FREE_RESULT:%[0-9]+]] = add i64 %1, [[FREE_VALUE]]
// CHECK: ret i64 [[FREE_RESULT]]

// CHECK-LABEL: define i64 @"main.(*S).Add$bound"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
// CHECK: [[BOUND_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[BOUND_RECEIVER:%[0-9]+]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK-NEXT: [[BOUND_RESULT:%[0-9]+]] = call i64 @"main.(*S).Add"(ptr [[BOUND_RECEIVER]], i64 %1)
// CHECK: ret i64 [[BOUND_RESULT]]

// CHECK-LABEL: define i64 @"main.(*S).Add$thunk"(ptr %0, i64 %1){{.*}} {
// CHECK: [[THUNK_RESULT:%[0-9]+]] = call i64 @"main.(*S).Add"(ptr %0, i64 %1)
// CHECK: ret i64 [[THUNK_RESULT]]

// CHECK-LABEL: define i64 @"main.interface{Add(int) int}.Add$bound"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
// CHECK: [[BOUND_IFACE_ENV:%[0-9]+]] = load { %"{{.*}}iface" }, ptr %0
// CHECK: [[BOUND_IFACE:%[0-9]+]] = extractvalue { %"{{.*}}iface" } [[BOUND_IFACE_ENV]], 0
// CHECK: [[BOUND_DATA:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}iface" [[BOUND_IFACE]])
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
