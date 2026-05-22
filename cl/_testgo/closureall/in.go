// LITTEST
package main

import _ "unsafe" // for go:linkname

import "github.com/goplus/lib/c"

//go:linkname cSqrt C.sqrt
// CHECK-LINE: @0 = private unnamed_addr constant [46 x i8] c"{{.*}}/cl/_testgo/closureall.S", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"Inc", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [3 x i8] c"Add", align 1
// CHECK-LINE: @9 = private unnamed_addr constant [23 x i8] c"interface{Add(int) int}", align 1

func cSqrt(x c.Double) c.Double

// llgo:link cAbs C.abs
func cAbs(x c.Int) c.Int { return 0 }

// llgo:type C
type CCallback func(c.Int) c.Int

type Fn func(int) int

type S struct {
	v int
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.S.Inc"(%"{{.*}}/cl/_testgo/closureall.S" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/cl/_testgo/closureall.S", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/closureall.S" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/closureall.S", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = add i64 %4, %1
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

func (s S) Inc(x int) int {
	return s.v + x
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/closureall.S", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = add i64 %3, %1
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

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
	ca := cAbs
	_ = ca(-3)

	cb := CCallback(func(x c.Int) c.Int { return x + 1 })
	_ = callCallback(cb, 7)
}

func makeNoFree() Fn {
	return func(x int) int { return x + 1 }
}

func makeWithFree(base int) Fn {
	return func(x int) int { return x + base }
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.(*S).Inc"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 46 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 3 })
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/closureall.S", ptr %0, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/closureall.S.Inc"(%"{{.*}}/cl/_testgo/closureall.S" %3, i64 %1)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/closureall.callCallback"(ptr %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call i32 %0(i32 %1)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.globalAdd"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closureall.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/closureall.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/closureall.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closureall.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/cl/_testgo/closureall.Fn" @"{{.*}}/cl/_testgo/closureall.makeNoFree"()
// CHECK-NEXT:   %1 = call %"{{.*}}/cl/_testgo/closureall.Fn" @"{{.*}}/cl/_testgo/closureall.makeWithFree"(i64 3)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testgo/closureall.Fn" %0, 1
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/cl/_testgo/closureall.Fn" %0, 0
// CHECK-NEXT:   %4 = call i64 %3(ptr %2, i64 1)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/cl/_testgo/closureall.Fn" %1, 1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/cl/_testgo/closureall.Fn" %1, 0
// CHECK-NEXT:   %7 = call i64 %6(ptr %5, i64 2)
// CHECK-NEXT:   %8 = call i64 @"{{.*}}/cl/_testgo/closureall.globalAdd"(i64 1, i64 2)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/closureall.S", ptr %9, i32 0, i32 0
// CHECK-NEXT:   store i64 5, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %12 = getelementptr inbounds { ptr }, ptr %11, i32 0, i32 0
// CHECK-NEXT:   store ptr %9, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closureall.(*S).Add$bound", ptr undef }, ptr %11, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %13, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   %16 = call i64 %15(ptr %14, i64 7)
// CHECK-NEXT:   %17 = call i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add$thunk"(ptr %9, i64 8)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$VdBKYV8-gcMjZtZfcf-u2oKoj9Lu3VXwuG8TGCW2S4A", ptr @"*_llgo_{{.*}}/cl/_testgo/closureall.S")
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %18, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %19, ptr %9, 1
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %20)
// CHECK-NEXT:   %22 = icmp ne ptr %21, null
// CHECK-NEXT:   br i1 %22, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %24 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %20, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closureall.interface{Add(int) int}.Add$bound", ptr undef }, ptr %23, 1
// CHECK-NEXT:   %26 = extractvalue { ptr, ptr } %25, 1
// CHECK-NEXT:   %27 = extractvalue { ptr, ptr } %25, 0
// CHECK-NEXT:   %28 = call i64 %27(ptr %26, i64 9)
// CHECK-NEXT:   %29 = call double @sqrt(double 4.000000e+00)
// CHECK-NEXT:   %30 = call i32 @abs(i32 -3)
// CHECK-NEXT:   %31 = call i32 @"{{.*}}/cl/_testgo/closureall.callCallback"(ptr @"{{.*}}/cl/_testgo/closureall.main$1", i32 7)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %21, %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 23 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 3 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/closureall.main$1"(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = add i32 %0, 1
// CHECK-NEXT:   ret i32 %1
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/cl/_testgo/closureall.Fn" @"{{.*}}/cl/_testgo/closureall.makeNoFree"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/closureall.Fn" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/closureall.makeNoFree$1", ptr null }
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.makeNoFree$1"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = add i64 %0, 1
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/cl/_testgo/closureall.Fn" @"{{.*}}/cl/_testgo/closureall.makeWithFree"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 %0, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closureall.makeWithFree$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = alloca %"{{.*}}/cl/_testgo/closureall.Fn", align 8
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %5, align 8
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testgo/closureall.Fn", ptr %5, align 8
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/closureall.Fn" %6
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.makeWithFree$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = add i64 %1, %4
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add$bound"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add"(ptr %3, i64 %1)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add$thunk"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add"(ptr %0, i64 %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/closureall.S.Inc"(ptr %0, %"{{.*}}/cl/_testgo/closureall.S" %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testgo/closureall.S.Inc"(%"{{.*}}/cl/_testgo/closureall.S" %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/closureall.(*S).Add"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testgo/closureall.(*S).Add"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/closureall.(*S).Inc"(ptr %0, ptr %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testgo/closureall.(*S).Inc"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/closureall.interface{Add(int) int}.Add$bound"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { %"{{.*}}/runtime/internal/runtime.iface" }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface" } %2, 0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %6 = getelementptr ptr, ptr %5, i64 3
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } %8, ptr %4, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %9, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %9, 0
// CHECK-NEXT:   %12 = call i64 %11(ptr %10, i64 %1)
// CHECK-NEXT:   ret i64 %12
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/closureall.makeNoFree$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/closureall.makeNoFree$1"(i64 %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
