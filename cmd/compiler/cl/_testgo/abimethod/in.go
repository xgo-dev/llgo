// LITTEST
package main

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"unsafe"
)

type T struct {
	n int
}

func (t T) Demo1() int {
	return t.n
}

func (t *T) Demo2() int {
	return t.n
}

func (t *T) demo3() int {
	return t.n
}

func main() {
	testGeneric()
	testNamed1()
	testNamed2()
	testNamed3()
	testAnonymous1()
	testAnonymous2()
	testAnonymous3()
	testAnonymous4()
	testAnonymous5()
	testAnonymous6()
	testAnonymous7()
	testAnonymous8()
	testAnonymousBuffer()
}

func testAnonymous1() {
	var s I = &struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous1 error")
	}
}

func testAnonymous2() {
	var s I = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous2 error")
	}
}

func testAnonymous3() {
	var s I = struct {
		m int
		T
	}{10, T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous3 error")
	}
}

func testAnonymous4() {
	var s I = &struct {
		m int
		T
	}{10, T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous4 error")
	}
}

func testAnonymous5() {
	var s I2 = &struct {
		m int
		T
	}{10, T{100}}
	if s.Demo2() != 100 {
		panic("testAnonymous5 error")
	}
}

func testAnonymous6() {
	var s I2 = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo2() != 100 {
		panic("testAnonymous6 error")
	}
}

func testAnonymous7() {
	var s interface {
		Demo1() int
		Demo2() int
	} = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous7 error")
	}
	if s.Demo2() != 100 {
		panic("testAnonymous7 error")
	}
}

func testAnonymous8() {
	var s interface {
		Demo1() int
		Demo2() int
		demo3() int
	} = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous8 error")
	}
	if s.Demo2() != 100 {
		panic("testAnonymous8 error")
	}
	if s.demo3() != 100 {
		panic("testAnonymous8 error")
	}
}

func testAnonymousBuffer() {
	var s fmt.Stringer = &struct {
		m int
		*bytes.Buffer
	}{10, bytes.NewBufferString("hello")}
	if s.String() != "hello" {
		panic("testAnonymousBuffer error")
	}
}

func testGeneric() {
	var p IP = &Pointer[any]{}
	p.Store(func() *any {
		var a any = 100
		return &a
	}())
	if (*p.Load()).(int) != 100 {
		panic("testGeneric error")
	}
}

func testNamed1() {
	var a I = &T{100}
	if a.Demo1() != 100 {
		panic("testNamed1 error")
	}
}

func testNamed2() {
	var a I = T{100}
	if a.Demo1() != 100 {
		panic("testNamed2 error")
	}
}

func testNamed3() {
	var a I2 = &T{100}
	if a.Demo2() != 100 {
		panic("testNamed4 error")
	}
}

type Pointer[T any] struct {
	// Mention *T in a field to disallow conversion between Pointer types.
	// See go.dev/issue/56603 for more details.
	// Use *T, not T, to avoid spurious recursive type definition errors.
	_ [0]*T
	v unsafe.Pointer
}

// Load atomically loads and returns the value stored in x.
func (x *Pointer[T]) Load() *T { return (*T)(atomic.LoadPointer(&x.v)) }

// Store atomically stores val into x.
func (x *Pointer[T]) Store(val *T) { atomic.StorePointer(&x.v, unsafe.Pointer(val)) }

type IP interface {
	Store(*any)
	Load() *any
}

type I interface {
	Demo1() int
}

type I2 interface {
	Demo2() int
}

// The declared value method and its pointer wrapper must both return T.n.
// CHECK-LABEL: define i64 @main.T.Demo1(%main.T %0){{.*}} {
// CHECK: [[T_ADDR:%[0-9]+]] = alloca %main.T
// CHECK: store %main.T %0, ptr [[T_ADDR]]
// CHECK-NEXT: [[T_N_PTR:%[0-9]+]] = getelementptr inbounds nuw %main.T, ptr [[T_ADDR]], i32 0, i32 0
// CHECK: [[T_N:%[0-9]+]] = load i64, ptr [[T_N_PTR]]
// CHECK: ret i64 [[T_N]]

// CHECK-LABEL: define i64 @"main.(*T).Demo1"(ptr %0){{.*}} {
// CHECK: [[T_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[T_NIL]],
// CHECK: [[T_VALUE:%[0-9]+]] = load %main.T, ptr %0
// CHECK: [[T_VALUE_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[T_VALUE]])
// CHECK: ret i64 [[T_VALUE_RES]]

// CHECK-LABEL: define i64 @"main.(*T).Demo2"(ptr %0){{.*}} {
// CHECK: [[T2_N_PTR:%[0-9]+]] = getelementptr inbounds nuw %main.T, ptr %0, i32 0, i32 0
// CHECK: [[T2_N:%[0-9]+]] = load i64, ptr [[T2_N_PTR]]
// CHECK: ret i64 [[T2_N]]

// CHECK-LABEL: define i64 @"main.(*T).demo3"(ptr %0){{.*}} {
// CHECK: [[T3_N_PTR:%[0-9]+]] = getelementptr inbounds nuw %main.T, ptr %0, i32 0, i32 0
// CHECK: [[T3_N:%[0-9]+]] = load i64, ptr [[T3_N_PTR]]
// CHECK: ret i64 [[T3_N]]

// Pointer-to-anonymous-struct embedding *T: form I, devirtualize Demo1, and test its result.
// CHECK-LABEL: define void @main.testAnonymous1(){{.*}} {
// CHECK: [[A1_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[A1_OBJ:%[0-9]+]], 1
// CHECK-NEXT: [[A1_RES:%[0-9]+]] = call i64 @"main.*struct{m int; *main.T}.Demo1"(ptr [[A1_OBJ]])
// CHECK-NEXT: [[A1_BAD:%[0-9]+]] = icmp ne i64 [[A1_RES]], 100
// CHECK-NEXT: br i1 [[A1_BAD]],

// Value anonymous struct embedding *T uses the value descriptor, and devirtualizes the promoted method.
// CHECK-LABEL: define void @main.testAnonymous2(){{.*}} {
// CHECK: [[A2_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: [[A2_RES:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo1"({ i64, ptr } %{{[0-9]+}})
// CHECK-NEXT: [[A2_BAD:%[0-9]+]] = icmp ne i64 [[A2_RES]], 100
// CHECK-NEXT: br i1 [[A2_BAD]],

// Value and pointer anonymous structs embedding T select distinct concrete descriptors.
// CHECK-LABEL: define void @main.testAnonymous3(){{.*}} {
// CHECK: [[A3_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: [[A3_RES:%[0-9]+]] = call i64 @"main.struct{m int; main.T}.Demo1"({ i64, %main.T } %{{[0-9]+}})
// CHECK-NEXT: [[A3_BAD:%[0-9]+]] = icmp ne i64 [[A3_RES]], 100
// CHECK-NEXT: br i1 [[A3_BAD]],

// CHECK-LABEL: define void @main.testAnonymous4(){{.*}} {
// CHECK: [[A4_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[A4_OBJ:%[0-9]+]], 1
// CHECK-NEXT: [[A4_RES:%[0-9]+]] = call i64 @"main.*struct{m int; main.T}.Demo1"(ptr [[A4_OBJ]])
// CHECK-NEXT: [[A4_BAD:%[0-9]+]] = icmp ne i64 [[A4_RES]], 100
// CHECK-NEXT: br i1 [[A4_BAD]],

// Demo2 requires the pointer method set, for both embedding T and embedding *T.
// CHECK-LABEL: define void @main.testAnonymous5(){{.*}} {
// CHECK: [[A5_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[A5_OBJ:%[0-9]+]], 1
// CHECK-NEXT: [[A5_RES:%[0-9]+]] = call i64 @"main.*struct{m int; main.T}.Demo2"(ptr [[A5_OBJ]])
// CHECK-NEXT: [[A5_BAD:%[0-9]+]] = icmp ne i64 [[A5_RES]], 100
// CHECK-NEXT: br i1 [[A5_BAD]],

// CHECK-LABEL: define void @main.testAnonymous6(){{.*}} {
// CHECK: [[A6_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: [[A6_RES:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo2"({ i64, ptr } %{{[0-9]+}})
// CHECK-NEXT: [[A6_BAD:%[0-9]+]] = icmp ne i64 [[A6_RES]], 100
// CHECK-NEXT: br i1 [[A6_BAD]],

// A two-method anonymous interface devirtualizes both promoted methods on the same value.
// CHECK-LABEL: define void @main.testAnonymous7(){{.*}} {
// CHECK: [[A7_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: [[A7_RES1:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo1"({ i64, ptr } [[A7_VAL:%[0-9]+]])
// CHECK-NEXT: [[A7_BAD1:%[0-9]+]] = icmp ne i64 [[A7_RES1]], 100
// CHECK-NEXT: br i1 [[A7_BAD1]],
// CHECK: [[A7_RES2:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo2"({ i64, ptr } [[A7_VAL]])
// CHECK-NEXT: [[A7_BAD2:%[0-9]+]] = icmp ne i64 [[A7_RES2]], 100
// CHECK-NEXT: br i1 [[A7_BAD2]],

// The package-local interface adds the unexported demo3 method after Demo1 and Demo2.
// CHECK-LABEL: define void @main.testAnonymous8(){{.*}} {
// CHECK: [[A8_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: [[A8_RES1:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo1"({ i64, ptr } [[A8_VAL:%[0-9]+]])
// CHECK-NEXT: [[A8_BAD1:%[0-9]+]] = icmp ne i64 [[A8_RES1]], 100
// CHECK-NEXT: br i1 [[A8_BAD1]],
// CHECK: [[A8_RES2:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.Demo2"({ i64, ptr } [[A8_VAL]])
// CHECK-NEXT: [[A8_BAD2:%[0-9]+]] = icmp ne i64 [[A8_RES2]], 100
// CHECK-NEXT: br i1 [[A8_BAD2]],
// CHECK: [[A8_RES3:%[0-9]+]] = call i64 @"main.struct{m int; *main.T}.demo3"({ i64, ptr } [[A8_VAL]])
// CHECK-NEXT: [[A8_BAD3:%[0-9]+]] = icmp ne i64 [[A8_RES3]], 100
// CHECK-NEXT: br i1 [[A8_BAD3]],

// The promoted bytes.Buffer.String method remains the implementation behind fmt.Stringer.
// CHECK-LABEL: define void @main.testAnonymousBuffer(){{.*}} {
// CHECK: [[BUF_OWNER:%[0-9]+]] = alloca { i64, ptr }
// CHECK: [[BUF_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr [[BUF_OWNER]], i32 0, i32 1
// CHECK-NEXT: [[BUF:%[0-9]+]] = call ptr @bytes.NewBufferString(
// CHECK: store ptr [[BUF]], ptr [[BUF_FIELD]]
// CHECK-NEXT: [[BUF_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[BUF_OWNER]], 1
// CHECK-NEXT: [[BUF_STR:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"main.*struct{m int; *bytes.Buffer}.String"(ptr [[BUF_OWNER]])
// CHECK: [[BUF_EQ:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" [[BUF_STR]],
// CHECK: [[BUF_BAD:%[0-9]+]] = xor i1 [[BUF_EQ]], true
// CHECK: br i1 [[BUF_BAD]],

// IP.Store and IP.Load must operate on the same Pointer[any] receiver.
// CHECK-LABEL: define void @main.testGeneric(){{.*}} {
// CHECK: [[P_OBJECT:%[0-9]+]] = alloca %"main.Pointer[interface{}]"
// CHECK: [[P_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[P_OBJECT]], 1
// CHECK: [[P_VALUE:%[0-9]+]] = call ptr @"main.testGeneric$1"()
// CHECK-NEXT: call void @"main.(*Pointer[interface{}]).Store"(ptr [[P_OBJECT]], ptr [[P_VALUE]])
// CHECK-NEXT: [[P_LOADED:%[0-9]+]] = call ptr @"main.(*Pointer[interface{}]).Load"(ptr [[P_OBJECT]])
// CHECK: [[P_ANY:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.eface", ptr [[P_LOADED]]
// CHECK: [[P_TYPE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[P_ANY]], 0
// CHECK: [[P_IS_INT:%[0-9]+]] = icmp eq ptr [[P_TYPE]], @_llgo_int
// CHECK: br i1 [[P_IS_INT]],
// CHECK: [[P_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[P_ANY]], 1
// CHECK: [[P_INT:%[0-9]+]] = load i64, ptr [[P_DATA]]
// CHECK: [[P_BAD:%[0-9]+]] = icmp ne i64 [[P_INT]], 100

// CHECK-LABEL: define ptr @"main.testGeneric$1"(){{.*}} {
// CHECK: store %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr @"_llgo_ifacebox${{.*}}" }, ptr [[P_RESULT:%[0-9]+]]
// CHECK: ret ptr [[P_RESULT]]

// Named T and *T select the proper descriptor and devirtualize the requested interface method.
// CHECK-LABEL: define void @main.testNamed1(){{.*}} {
// CHECK: [[N1_OBJECT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocAtomicU"(i64 8)
// CHECK-NEXT: call void @llvm.memset.p0.i64(ptr [[N1_OBJECT]], i8 0, i64 8, i1 false)
// CHECK: [[N1_FIELD:%[0-9]+]] = getelementptr inbounds nuw %main.T, ptr [[N1_OBJECT]], i32 0, i32 0
// CHECK-NEXT: store i64 100, ptr [[N1_FIELD]]
// CHECK-NEXT: [[N1_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[N1_OBJECT]], 1
// CHECK-NEXT: [[N1_RES:%[0-9]+]] = call i64 @"main.(*T).Demo1"(ptr [[N1_OBJECT]])
// CHECK-NEXT: [[N1_BAD:%[0-9]+]] = icmp ne i64 [[N1_RES]], 100
// CHECK-NEXT: br i1 [[N1_BAD]],

// CHECK-LABEL: define void @main.testNamed2(){{.*}} {
// CHECK: [[N2_VALUE:%[0-9]+]] = load %main.T, ptr %{{[0-9]+}}
// CHECK: store %main.T [[N2_VALUE]], ptr [[N2_OBJECT:%[0-9]+]]
// CHECK-NEXT: [[N2_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[N2_OBJECT]], 1
// CHECK-NEXT: [[N2_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[N2_VALUE]])
// CHECK-NEXT: [[N2_BAD:%[0-9]+]] = icmp ne i64 [[N2_RES]], 100
// CHECK-NEXT: br i1 [[N2_BAD]],

// CHECK-LABEL: define void @main.testNamed3(){{.*}} {
// CHECK: [[N3_OBJECT:%[0-9]+]] = alloca %main.T
// CHECK: [[N3_FIELD:%[0-9]+]] = getelementptr inbounds nuw %main.T, ptr [[N3_OBJECT]], i32 0, i32 0
// CHECK-NEXT: store i64 100, ptr [[N3_FIELD]]
// CHECK-NEXT: [[N3_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[N3_OBJECT]], 1
// CHECK-NEXT: [[N3_RES:%[0-9]+]] = call i64 @"main.(*T).Demo2"(ptr [[N3_OBJECT]])
// CHECK-NEXT: [[N3_BAD:%[0-9]+]] = icmp ne i64 [[N3_RES]], 100
// CHECK-NEXT: br i1 [[N3_BAD]],

// Promoted methods must address the embedded field, call the underlying method, and return it.
// CHECK-LABEL: define i64 @"main.*struct{m int; *main.T}.Demo1"(ptr %0){{.*}} {
// CHECK: [[SPP1_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK: [[SPP1_T:%[0-9]+]] = load ptr, ptr [[SPP1_FIELD]]
// CHECK: [[SPP1_NIL:%[0-9]+]] = icmp eq ptr [[SPP1_T]], null
// CHECK: br i1 [[SPP1_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[SPP1_VALUE:%[0-9]+]] = load %main.T, ptr [[SPP1_T]]
// CHECK: [[SPP1_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[SPP1_VALUE]])
// CHECK: ret i64 [[SPP1_RES]]

// CHECK-LABEL: define i64 @"main.*struct{m int; *main.T}.Demo2"(ptr %0){{.*}} {
// CHECK: [[SPP2_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK: [[SPP2_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[SPP2_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[SPP2_T:%[0-9]+]] = load ptr, ptr [[SPP2_FIELD]]
// CHECK: [[SPP2_RES:%[0-9]+]] = call i64 @"main.(*T).Demo2"(ptr [[SPP2_T]])
// CHECK: ret i64 [[SPP2_RES]]

// CHECK-LABEL: define i64 @"main.*struct{m int; *main.T}.demo3"(ptr %0){{.*}} {
// CHECK: [[SPP3_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK: [[SPP3_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[SPP3_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[SPP3_T:%[0-9]+]] = load ptr, ptr [[SPP3_FIELD]]
// CHECK: [[SPP3_RES:%[0-9]+]] = call i64 @"main.(*T).demo3"(ptr [[SPP3_T]])
// CHECK: ret i64 [[SPP3_RES]]

// CHECK-LABEL: define i64 @"main.struct{m int; *main.T}.Demo1"({ i64, ptr } %0){{.*}} {
// CHECK: [[SVP1_ADDR:%[0-9]+]] = alloca { i64, ptr }
// CHECK: store { i64, ptr } %0, ptr [[SVP1_ADDR]]
// CHECK-NEXT: [[SVP1_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr [[SVP1_ADDR]], i32 0, i32 1
// CHECK: [[SVP1_T:%[0-9]+]] = load ptr, ptr [[SVP1_FIELD]]
// CHECK: [[SVP1_NIL:%[0-9]+]] = icmp eq ptr [[SVP1_T]], null
// CHECK: br i1 [[SVP1_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[SVP1_VALUE:%[0-9]+]] = load %main.T, ptr [[SVP1_T]]
// CHECK: [[SVP1_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[SVP1_VALUE]])
// CHECK: ret i64 [[SVP1_RES]]

// CHECK-LABEL: define i64 @"main.struct{m int; *main.T}.Demo2"({ i64, ptr } %0){{.*}} {
// CHECK: [[SVP2_ADDR:%[0-9]+]] = alloca { i64, ptr }
// CHECK: store { i64, ptr } %0, ptr [[SVP2_ADDR]]
// CHECK-NEXT: [[SVP2_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr [[SVP2_ADDR]], i32 0, i32 1
// CHECK: [[SVP2_T:%[0-9]+]] = load ptr, ptr [[SVP2_FIELD]]
// CHECK: [[SVP2_RES:%[0-9]+]] = call i64 @"main.(*T).Demo2"(ptr [[SVP2_T]])
// CHECK: ret i64 [[SVP2_RES]]

// CHECK-LABEL: define i64 @"main.struct{m int; *main.T}.demo3"({ i64, ptr } %0){{.*}} {
// CHECK: [[SVP3_ADDR:%[0-9]+]] = alloca { i64, ptr }
// CHECK: store { i64, ptr } %0, ptr [[SVP3_ADDR]]
// CHECK-NEXT: [[SVP3_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr [[SVP3_ADDR]], i32 0, i32 1
// CHECK: [[SVP3_T:%[0-9]+]] = load ptr, ptr [[SVP3_FIELD]]
// CHECK: [[SVP3_RES:%[0-9]+]] = call i64 @"main.(*T).demo3"(ptr [[SVP3_T]])
// CHECK: ret i64 [[SVP3_RES]]

// CHECK-LABEL: define i64 @"main.struct{m int; main.T}.Demo1"({ i64, %main.T } %0){{.*}} {
// CHECK: [[SVV_ADDR:%[0-9]+]] = alloca { i64, %main.T }
// CHECK: store { i64, %main.T } %0, ptr [[SVV_ADDR]]
// CHECK-NEXT: [[SVV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, %main.T }, ptr [[SVV_ADDR]], i32 0, i32 1
// CHECK: [[SVV_T:%[0-9]+]] = load %main.T, ptr [[SVV_FIELD]]
// CHECK: [[SVV_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[SVV_T]])
// CHECK: ret i64 [[SVV_RES]]

// CHECK-LABEL: define i64 @"main.*struct{m int; main.T}.Demo1"(ptr %0){{.*}} {
// CHECK: [[SPV1_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, %main.T }, ptr %0, i32 0, i32 1
// CHECK: [[SPV1_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[SPV1_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[SPV1_T:%[0-9]+]] = load %main.T, ptr [[SPV1_FIELD]]
// CHECK: [[SPV1_RES:%[0-9]+]] = call i64 @main.T.Demo1(%main.T [[SPV1_T]])
// CHECK: ret i64 [[SPV1_RES]]

// CHECK-LABEL: define i64 @"main.*struct{m int; main.T}.Demo2"(ptr %0){{.*}} {
// CHECK: [[SPV2_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, %main.T }, ptr %0, i32 0, i32 1
// CHECK: [[SPV2_RES:%[0-9]+]] = call i64 @"main.(*T).Demo2"(ptr [[SPV2_FIELD]])
// CHECK: ret i64 [[SPV2_RES]]

// CHECK-LABEL: define i64 @"main.*struct{m int; main.T}.demo3"(ptr %0){{.*}} {
// CHECK: [[SPV3_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, %main.T }, ptr %0, i32 0, i32 1
// CHECK: [[SPV3_RES:%[0-9]+]] = call i64 @"main.(*T).demo3"(ptr [[SPV3_FIELD]])
// CHECK: ret i64 [[SPV3_RES]]

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"main.*struct{m int; *bytes.Buffer}.String"(ptr %0){{.*}} {
// CHECK: [[BSP_FIELD:%[0-9]+]] = getelementptr inbounds nuw { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK: [[BSP_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[BSP_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[BSP_BUFFER:%[0-9]+]] = load ptr, ptr [[BSP_FIELD]]
// CHECK: [[BSP_STRING:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr [[BSP_BUFFER]])
// CHECK: ret %"{{.*}}/runtime/internal/runtime.String" [[BSP_STRING]]

// Atomic Load and Store must address Pointer[any].v and preserve seq_cst ordering.
// CHECK-LABEL: define linkonce ptr @"main.(*Pointer{{\[.*\]}}).Load"(ptr %0){{.*}} {
// CHECK: [[LOAD_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[LOAD_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[LOAD_V:%[0-9]+]] = getelementptr inbounds nuw %"main.Pointer{{\[.*\]}}", ptr %0, i32 0, i32 1
// CHECK: [[LOAD_VALUE:%[0-9]+]] = load atomic ptr, ptr [[LOAD_V]] seq_cst
// CHECK: ret ptr [[LOAD_VALUE]]

// CHECK-LABEL: define linkonce void @"main.(*Pointer{{\[.*\]}}).Store"(ptr %0, ptr %1){{.*}} {
// CHECK: [[STORE_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[STORE_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[STORE_V:%[0-9]+]] = getelementptr inbounds nuw %"main.Pointer{{\[.*\]}}", ptr %0, i32 0, i32 1
// CHECK: store atomic ptr %1, ptr [[STORE_V]] seq_cst
