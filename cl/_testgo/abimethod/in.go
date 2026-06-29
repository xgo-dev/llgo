// LITTEST
package main

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"unsafe"
)

// CHECK: {{^}}@0 = private unnamed_addr constant [45 x i8] c"{{.*}}/cl/_testgo/abimethod.T", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [5 x i8] c"Demo1", align 1{{$}}
// CHECK: {{^}}@5 = private unnamed_addr constant [3 x i8] c"int", align 1{{$}}
// CHECK: {{^}}@14 = private unnamed_addr constant [20 x i8] c"testAnonymous1 error", align 1{{$}}
// CHECK: {{^}}@16 = private unnamed_addr constant [20 x i8] c"testAnonymous2 error", align 1{{$}}
// CHECK: {{^}}@18 = private unnamed_addr constant [20 x i8] c"testAnonymous3 error", align 1{{$}}
// CHECK: {{^}}@19 = private unnamed_addr constant [20 x i8] c"testAnonymous4 error", align 1{{$}}
// CHECK: {{^}}@21 = private unnamed_addr constant [20 x i8] c"testAnonymous5 error", align 1{{$}}
// CHECK: {{^}}@22 = private unnamed_addr constant [20 x i8] c"testAnonymous6 error", align 1{{$}}
// CHECK: {{^}}@24 = private unnamed_addr constant [20 x i8] c"testAnonymous7 error", align 1{{$}}
// CHECK: {{^}}@26 = private unnamed_addr constant [20 x i8] c"testAnonymous8 error", align 1{{$}}
// CHECK: {{^}}@27 = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@[[ANONBUF_ERR:[0-9]+]] = private unnamed_addr constant [25 x i8] c"testAnonymousBuffer error", align 1{{$}}
// CHECK: {{^}}@[[GENERIC_ERR:[0-9]+]] = private unnamed_addr constant [17 x i8] c"testGeneric error", align 1{{$}}
// CHECK: {{^}}@[[NAMED1_ERR:[0-9]+]] = private unnamed_addr constant [16 x i8] c"testNamed1 error", align 1{{$}}
// CHECK: {{^}}@[[NAMED2_ERR:[0-9]+]] = private unnamed_addr constant [16 x i8] c"testNamed2 error", align 1{{$}}
// CHECK: {{^}}@[[NAMED4_ERR:[0-9]+]] = private unnamed_addr constant [16 x i8] c"testNamed4 error", align 1{{$}}

type T struct {
	n int
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/abimethod.T", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/abimethod.T" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 45 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 5 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/abimethod.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/abimethod.init$guard", align 1
// CHECK-NEXT:   call void @bytes.init()
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @"sync/atomic.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testGeneric"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testNamed1"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testNamed2"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testNamed3"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous1"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous2"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous3"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous4"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous5"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous6"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous7"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymous8"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/abimethod.testAnonymousBuffer"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*{{.*}}/cl/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %6, ptr %0, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %7, 0
// CHECK-NEXT:   %10 = getelementptr ptr, ptr %9, i64 3
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } undef, ptr %11, 0
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } %12, ptr %8, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %13, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   %16 = call i64 %15(ptr %14)
// CHECK-NEXT:   %17 = icmp ne i64 %16, 100
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 20 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"{{.*}}/cl/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = load { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, %"{{.*}}/cl/_testgo/abimethod.T" } %4, ptr %5, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"{{.*}}/cl/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %7, ptr %5, 1
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %8)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %8, 0
// CHECK-NEXT:   %11 = getelementptr ptr, ptr %10, i64 3
// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } undef, ptr %12, 0
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } %13, ptr %9, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %14, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %14, 0
// CHECK-NEXT:   %17 = call i64 %16(ptr %15)
// CHECK-NEXT:   %18 = icmp ne i64 %17, 100
// CHECK-NEXT:   br i1 %18, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 20 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*{{.*}}/cl/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 20 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"*{{.*}}/cl/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 20 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"{{.*}}/cl/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw", ptr @"{{.*}}/cl/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 4
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call i64 %29(ptr %28)
// CHECK-NEXT:   %31 = icmp ne i64 %30, 100
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 20 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymous8"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/cl/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA", ptr @"{{.*}}/cl/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 4
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call i64 %29(ptr %28)
// CHECK-NEXT:   %31 = icmp ne i64 %30, 100
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 20 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %36 = getelementptr ptr, ptr %35, i64 5
// CHECK-NEXT:   %37 = load ptr, ptr %36, align 8
// CHECK-NEXT:   %38 = insertvalue { ptr, ptr } undef, ptr %37, 0
// CHECK-NEXT:   %39 = insertvalue { ptr, ptr } %38, ptr %34, 1
// CHECK-NEXT:   %40 = extractvalue { ptr, ptr } %39, 1
// CHECK-NEXT:   %41 = extractvalue { ptr, ptr } %39, 0
// CHECK-NEXT:   %42 = call i64 %41(ptr %40)
// CHECK-NEXT:   %43 = icmp ne i64 %42, 100
// CHECK-NEXT:   br i1 %43, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 20 }, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %44, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testAnonymousBuffer"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @bytes.NewBufferString(%"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 5 })
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*{{.*}}/cl/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" %14(ptr %13)
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 5 })
// CHECK-NEXT:   %17 = xor i1 %16, true
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @[[ANONBUF_ERR]], i64 25 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testGeneric"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds", ptr @"*_llgo_{{.*}}/cl/_testgo/abimethod.Pointer[any]")
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %1, 0
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %2, ptr %0, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/cl/_testgo/abimethod.testGeneric$1"()
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 4
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   call void %12(ptr %11, ptr %4)
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %15 = getelementptr ptr, ptr %14, i64 3
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } undef, ptr %16, 0
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } %17, ptr %13, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   %21 = call ptr %20(ptr %19)
// CHECK-NEXT:   %22 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %21, align 8
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 0
// CHECK-NEXT:   %24 = icmp eq ptr %23, @_llgo_int
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @[[GENERIC_ERR]], i64 17 }, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %25, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %26)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 1
// CHECK-NEXT:   %28 = load i64, ptr %27, align 8
// CHECK-NEXT:   %29 = icmp ne i64 %28, 100
// CHECK-NEXT:   br i1 %29, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %23, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/abimethod.testGeneric$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %0, align 8
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testNamed1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*_llgo_{{.*}}/cl/_testgo/abimethod.T")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   %14 = icmp ne i64 %13, 100
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @[[NAMED1_ERR]], i64 16 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testNamed2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testgo/abimethod.T", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/abimethod.T" %2, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"_llgo_{{.*}}/cl/_testgo/abimethod.T")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %3, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @[[NAMED2_ERR]], i64 16 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.testNamed3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"*_llgo_{{.*}}/cl/_testgo/abimethod.T")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   %14 = icmp ne i64 %13, 100
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @[[NAMED4_ERR]], i64 16 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %2)
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %5, align 8
// CHECK-NEXT:   %7 = call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %6)
// CHECK-NEXT:   ret i64 %7
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %3)
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %4, align 8
// CHECK-NEXT:   %6 = call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %5)
// CHECK-NEXT:   ret i64 %6
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.T.Demo1"(ptr %0, %"{{.*}}/cl/_testgo/abimethod.T" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.(*T).Demo1"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo1"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *{{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"({ i64, %"{{.*}}/cl/_testgo/abimethod.T" } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, %"{{.*}}/cl/_testgo/abimethod.T" } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %1)
// CHECK-NEXT:   %4 = load %"{{.*}}/cl/_testgo/abimethod.T", ptr %3, align 8
// CHECK-NEXT:   %5 = call i64 @"{{.*}}/cl/_testgo/abimethod.T.Demo1"(%"{{.*}}/cl/_testgo/abimethod.T" %4)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).Demo2"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/cl/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testgo/abimethod.(*T).demo3"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo2"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; {{.*}}/cl/_testgo/abimethod.T}.demo3"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"(ptr %0, { i64, %"{{.*}}/cl/_testgo/abimethod.T" } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; {{.*}}/cl/_testgo/abimethod.T}.Demo1"({ i64, %"{{.*}}/cl/_testgo/abimethod.T" } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Available"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).AvailableBuffer"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Cap"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Grow"(ptr %3, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Len"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Next"(ptr %3, i64 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Read"(ptr %3, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadByte"(ptr %2)
// CHECK-NEXT:   %4 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %3, 1
// CHECK-NEXT:   %6 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %4, 0
// CHECK-NEXT:   %7 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %6, %"{{.*}}/runtime/internal/runtime.iface" %5, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %7
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadBytes"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadFrom"(ptr %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadRune"(ptr %2)
// CHECK-NEXT:   %4 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 1
// CHECK-NEXT:   %6 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 2
// CHECK-NEXT:   %7 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %4, 0
// CHECK-NEXT:   %8 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, i64 %5, 1
// CHECK-NEXT:   %9 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %6, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadString"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.String" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Reset"(ptr %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Truncate"(ptr %3, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadByte"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadRune"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %3, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).WriteByte"(ptr %3, i8 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune"(ptr %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteRune"(ptr %3, i32 %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %3, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteTo"(ptr %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i1 @"bytes.(*Buffer).empty"(ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).grow"(ptr %3, i64 %1)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).readSlice"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, i1 } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, i1 } @"bytes.(*Buffer).tryGrowByReslice"(ptr %3, i64 %1)
// CHECK-NEXT:   %5 = extractvalue { i64, i1 } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, i1 } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, i1 } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, i1 } %7, i1 %6, 1
// CHECK-NEXT:   ret { i64, i1 } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Available"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Available"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.AvailableBuffer"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).AvailableBuffer"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Bytes"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Cap"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Cap"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Grow"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Grow"(ptr %4, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Len"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Len"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Next"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Next"(ptr %4, i64 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %5
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Read"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Read"(ptr %4, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadByte"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadByte"(ptr %3)
// CHECK-NEXT:   %5 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadBytes"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadBytes"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadFrom"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadFrom"(ptr %4, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadRune"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadRune"(ptr %3)
// CHECK-NEXT:   %5 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 2
// CHECK-NEXT:   %8 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %5, 0
// CHECK-NEXT:   %9 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, i64 %6, 1
// CHECK-NEXT:   %10 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %9, %"{{.*}}/runtime/internal/runtime.iface" %7, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %10
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadString"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadString"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.String" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Reset"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Reset"(ptr %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.String"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Truncate"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Truncate"(ptr %4, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadByte"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadByte"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadRune"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadRune"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Write"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %4, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteByte"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).WriteByte"(ptr %4, i8 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteRune"({ i64, ptr } %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteRune"(ptr %4, i32 %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteString"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %4, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteTo"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteTo"(ptr %4, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.empty"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i1 @"bytes.(*Buffer).empty"(ptr %3)
// CHECK-NEXT:   ret i1 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.grow"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call i64 @"bytes.(*Buffer).grow"(ptr %4, i64 %1)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.readSlice"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).readSlice"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, i1 } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.tryGrowByReslice"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, i1 } @"bytes.(*Buffer).tryGrowByReslice"(ptr %4, i64 %1)
// CHECK-NEXT:   %6 = extractvalue { i64, i1 } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, i1 } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, i1 } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, i1 } %8, i1 %7, 1
// CHECK-NEXT:   ret { i64, i1 } %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.bytes.(*Buffer).Available"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"bytes.(*Buffer).Available"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.bytes.(*Buffer).AvailableBuffer"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).AvailableBuffer"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.bytes.(*Buffer).Bytes"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.bytes.(*Buffer).Cap"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"bytes.(*Buffer).Cap"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.bytes.(*Buffer).Grow"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"bytes.(*Buffer).Grow"(ptr %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.bytes.(*Buffer).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"bytes.(*Buffer).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.bytes.(*Buffer).Next"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Next"(ptr %1, i64 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).ReadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadByte"(ptr %1)
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).ReadBytes"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadBytes"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).ReadFrom"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadFrom"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).ReadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadRune"(ptr %1)
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).ReadString"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadString"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.bytes.(*Buffer).Reset"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"bytes.(*Buffer).Reset"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.bytes.(*Buffer).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.bytes.(*Buffer).Truncate"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"bytes.(*Buffer).Truncate"(ptr %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.bytes.(*Buffer).UnreadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadByte"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.bytes.(*Buffer).UnreadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadRune"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).Write"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.bytes.(*Buffer).WriteByte"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).WriteByte"(ptr %1, i8 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).WriteRune"(ptr %0, ptr %1, i32 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteRune"(ptr %1, i32 %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).WriteString"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.bytes.(*Buffer).empty"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"bytes.(*Buffer).empty"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.bytes.(*Buffer).grow"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"bytes.(*Buffer).grow"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.bytes.(*Buffer).readSlice"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).readSlice"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, i1 } @"__llgo_stub.bytes.(*Buffer).tryGrowByReslice"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, i1 } @"bytes.(*Buffer).tryGrowByReslice"(ptr %1, i64 %2)
// CHECK-NEXT:   ret { i64, i1 } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Available"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Available"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.AvailableBuffer"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.AvailableBuffer"({ i64, ptr } %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Bytes"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Bytes"({ i64, ptr } %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Cap"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Cap"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Grow"(ptr %0, { i64, ptr } %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Grow"({ i64, ptr } %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Len"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Len"({ i64, ptr } %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Next"(ptr %0, { i64, ptr } %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Next"({ i64, ptr } %1, i64 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Read"(ptr %0, { i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Read"({ i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadByte"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadByte"({ i64, ptr } %1)
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadBytes"(ptr %0, { i64, ptr } %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadBytes"({ i64, ptr } %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadFrom"(ptr %0, { i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadFrom"({ i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadRune"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadRune"({ i64, ptr } %1)
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadString"(ptr %0, { i64, ptr } %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadString"({ i64, ptr } %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Reset"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Reset"({ i64, ptr } %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.String"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.String"({ i64, ptr } %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Truncate"(ptr %0, { i64, ptr } %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Truncate"({ i64, ptr } %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadByte"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadByte"({ i64, ptr } %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadRune"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadRune"({ i64, ptr } %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Write"(ptr %0, { i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.Write"({ i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteByte"(ptr %0, { i64, ptr } %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteByte"({ i64, ptr } %1, i8 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteRune"(ptr %0, { i64, ptr } %1, i32 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteRune"({ i64, ptr } %1, i32 %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteString"(ptr %0, { i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteString"({ i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteTo"(ptr %0, { i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteTo"({ i64, ptr } %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.empty"(ptr %0, { i64, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.empty"({ i64, ptr } %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.grow"(ptr %0, { i64, ptr } %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.grow"({ i64, ptr } %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.readSlice"(ptr %0, { i64, ptr } %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.readSlice"({ i64, ptr } %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, i1 } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.tryGrowByReslice"(ptr %0, { i64, ptr } %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, i1 } @"{{.*}}/cl/_testgo/abimethod.struct{m int; *bytes.Buffer}.tryGrowByReslice"({ i64, ptr } %1, i64 %2)
// CHECK-NEXT:   ret { i64, i1 } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow"(ptr %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next"(ptr %1, i64 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte"(ptr %1)
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune"(ptr %1)
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate"(ptr %1, i64 %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write"(ptr %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.iface" @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte"(ptr %1, i8 %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune"(ptr %0, ptr %1, i32 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune"(ptr %1, i32 %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo"(ptr %0, ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo"(ptr %1, %"{{.*}}/runtime/internal/runtime.iface" %2)
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice"(ptr %0, ptr %1, i8 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice"(ptr %1, i8 %2)
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, i1 } @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call { i64, i1 } @"{{.*}}/cl/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice"(ptr %1, i64 %2)
// CHECK-NEXT:   ret { i64, i1 } %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Load"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.Pointer[any]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load atomic ptr, ptr %2 seq_cst, align 8
// CHECK-NEXT:   ret ptr %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Store"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/abimethod.Pointer[any]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store atomic ptr %1, ptr %3 seq_cst, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Load"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Load"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Store"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/abimethod.(*Pointer[any]).Store"(ptr %1, ptr %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
