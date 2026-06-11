// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"failed", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [2 x i8] c"ok", align 1

func test() {}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.assert"(i1 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 }, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %1, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func assert(cond bool) {
	if !cond {
		panic("failed")
	}
}

// func
func init() {
	fn1 := test
	fn2 := func(i, j int) int { return i + j }
	var n int
	fn3 := func() { println(n) }
	var fn4 func() int
	assert(test != nil)
	assert(nil != test)
	assert(fn1 != nil)
	assert(nil != fn1)
	assert(fn2 != nil)
	assert(nil != fn2)
	assert(fn3 != nil)
	assert(nil != fn3)
	assert(fn4 == nil)
	assert(nil == fn4)
}

// array
func init() {
	assert([0]float64{} == [0]float64{})
	ar1 := [...]int{1, 2, 3}
	ar2 := [...]int{1, 2, 3}
	assert(ar1 == ar2)
	ar2[1] = 1
	assert(ar1 != ar2)
}

type T struct {
	X int
	Y int
	Z string
	V any
}

type N struct{}

// struct
func init() {
	var n1, n2 N
	var t1, t2 T
	x := T{10, 20, "hello", 1}
	y := T{10, 20, "hello", 1}
	z := T{10, 20, "hello", "ok"}
	assert(n1 == n2)
	assert(t1 == t2)
	assert(x == y)
	assert(x != z)
	assert(y != z)
}

// slice
func init() {
	var a []int
	var b = []int{1, 2, 3}
	c := make([]int, 2)
	d := make([]int, 0, 2)
	assert(a == nil)
	assert(b != nil)
	assert(c != nil)
	assert(d != nil)
	b = nil
	assert(b == nil)
}

// iface
func init() {
	var a any = 100
	var b any = struct{}{}
	var c any = T{10, 20, "hello", 1}
	x := T{10, 20, "hello", 1}
	y := T{10, 20, "hello", "ok"}
	assert(a == 100)
	assert(b == struct{}{})
	assert(b != N{})
	assert(c == x)
	assert(c != y)
}

// chan
func init() {
	a := make(chan int)
	b := make(chan int)
	assert(a == a)
	assert(a != b)
	assert(a != nil)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/equal.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/equal.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#1"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#2"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#3"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#4"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#5"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#6"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.init#7"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// map
func init() {
	m1 := make(map[int]string)
	var m2 map[int]string
	assert(m1 != nil)
	assert(m2 == nil)
}

func main() {
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/equal.init#1$2", ptr undef }, ptr %1, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %3, 0
// CHECK-NEXT:   %5 = icmp ne ptr %4, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %5)
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %3, 0
// CHECK-NEXT:   %7 = icmp ne ptr null, %6
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %7)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/equal.init#1$1"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#1$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   %0 = alloca [3 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 24, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   store i64 2, ptr %2, align 8
// CHECK-NEXT:   store i64 3, ptr %3, align 8
// CHECK-NEXT:   %4 = alloca [3 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %4, i8 0, i64 24, i1 false)
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %4, i64 0
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %4, i64 1
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %4, i64 2
// CHECK-NEXT:   store i64 1, ptr %5, align 8
// CHECK-NEXT:   store i64 2, ptr %6, align 8
// CHECK-NEXT:   store i64 3, ptr %7, align 8
// CHECK-NEXT:   %8 = load [3 x i64], ptr %0, align 8
// CHECK-NEXT:   %9 = load [3 x i64], ptr %4, align 8
// CHECK-NEXT:   %10 = extractvalue [3 x i64] %8, 0
// CHECK-NEXT:   %11 = extractvalue [3 x i64] %9, 0
// CHECK-NEXT:   %12 = icmp eq i64 %10, %11
// CHECK-NEXT:   %13 = and i1 true, %12
// CHECK-NEXT:   %14 = extractvalue [3 x i64] %8, 1
// CHECK-NEXT:   %15 = extractvalue [3 x i64] %9, 1
// CHECK-NEXT:   %16 = icmp eq i64 %14, %15
// CHECK-NEXT:   %17 = and i1 %13, %16
// CHECK-NEXT:   %18 = extractvalue [3 x i64] %8, 2
// CHECK-NEXT:   %19 = extractvalue [3 x i64] %9, 2
// CHECK-NEXT:   %20 = icmp eq i64 %18, %19
// CHECK-NEXT:   %21 = and i1 %17, %20
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %21)
// CHECK-NEXT:   %22 = getelementptr inbounds i64, ptr %4, i64 1
// CHECK-NEXT:   store i64 1, ptr %22, align 8
// CHECK-NEXT:   %23 = load [3 x i64], ptr %0, align 8
// CHECK-NEXT:   %24 = load [3 x i64], ptr %4, align 8
// CHECK-NEXT:   %25 = extractvalue [3 x i64] %23, 0
// CHECK-NEXT:   %26 = extractvalue [3 x i64] %24, 0
// CHECK-NEXT:   %27 = icmp eq i64 %25, %26
// CHECK-NEXT:   %28 = and i1 true, %27
// CHECK-NEXT:   %29 = extractvalue [3 x i64] %23, 1
// CHECK-NEXT:   %30 = extractvalue [3 x i64] %24, 1
// CHECK-NEXT:   %31 = icmp eq i64 %29, %30
// CHECK-NEXT:   %32 = and i1 %28, %31
// CHECK-NEXT:   %33 = extractvalue [3 x i64] %23, 2
// CHECK-NEXT:   %34 = extractvalue [3 x i64] %24, 2
// CHECK-NEXT:   %35 = icmp eq i64 %33, %34
// CHECK-NEXT:   %36 = and i1 %32, %35
// CHECK-NEXT:   %37 = xor i1 %36, true
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %37)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %0, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %2, align 8
// CHECK-NEXT:   store i64 20, ptr %4, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %6, align 8
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %9, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %10, ptr %8, align 8
// CHECK-NEXT:   %11 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %11, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %12 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %11, i32 0, i32 0
// CHECK-NEXT:   %14 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %11, i32 0, i32 1
// CHECK-NEXT:   %16 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %11, i32 0, i32 2
// CHECK-NEXT:   %18 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %18)
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %11, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %13, align 8
// CHECK-NEXT:   store i64 20, ptr %15, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %17, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %20, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %21, ptr %19, align 8
// CHECK-NEXT:   %22 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %22, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %23 = icmp eq ptr %22, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %23)
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %22, i32 0, i32 0
// CHECK-NEXT:   %25 = icmp eq ptr %22, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %25)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %22, i32 0, i32 1
// CHECK-NEXT:   %27 = icmp eq ptr %22, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %22, i32 0, i32 2
// CHECK-NEXT:   %29 = icmp eq ptr %22, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %29)
// CHECK-NEXT:   %30 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %22, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %24, align 8
// CHECK-NEXT:   store i64 20, ptr %26, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %28, align 8
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 2 }, ptr %31, align 8
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %31, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %32, ptr %30, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   %33 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" zeroinitializer, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   %34 = and i1 true, %33
// CHECK-NEXT:   %35 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" zeroinitializer, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %36 = and i1 %34, %35
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %36)
// CHECK-NEXT:   %37 = load %"{{.*}}/cl/_testgo/equal.T", ptr %0, align 8
// CHECK-NEXT:   %38 = load %"{{.*}}/cl/_testgo/equal.T", ptr %11, align 8
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %37, 0
// CHECK-NEXT:   %40 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %38, 0
// CHECK-NEXT:   %41 = icmp eq i64 %39, %40
// CHECK-NEXT:   %42 = and i1 true, %41
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %37, 1
// CHECK-NEXT:   %44 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %38, 1
// CHECK-NEXT:   %45 = icmp eq i64 %43, %44
// CHECK-NEXT:   %46 = and i1 %42, %45
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %37, 2
// CHECK-NEXT:   %48 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %38, 2
// CHECK-NEXT:   %49 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %47, %"{{.*}}/runtime/internal/runtime.String" %48)
// CHECK-NEXT:   %50 = and i1 %46, %49
// CHECK-NEXT:   %51 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %37, 3
// CHECK-NEXT:   %52 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %38, 3
// CHECK-NEXT:   %53 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %51, %"{{.*}}/runtime/internal/runtime.eface" %52)
// CHECK-NEXT:   %54 = and i1 %50, %53
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %54)
// CHECK-NEXT:   %55 = load %"{{.*}}/cl/_testgo/equal.T", ptr %0, align 8
// CHECK-NEXT:   %56 = load %"{{.*}}/cl/_testgo/equal.T", ptr %22, align 8
// CHECK-NEXT:   %57 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %55, 0
// CHECK-NEXT:   %58 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %56, 0
// CHECK-NEXT:   %59 = icmp eq i64 %57, %58
// CHECK-NEXT:   %60 = and i1 true, %59
// CHECK-NEXT:   %61 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %55, 1
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %56, 1
// CHECK-NEXT:   %63 = icmp eq i64 %61, %62
// CHECK-NEXT:   %64 = and i1 %60, %63
// CHECK-NEXT:   %65 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %55, 2
// CHECK-NEXT:   %66 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %56, 2
// CHECK-NEXT:   %67 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %65, %"{{.*}}/runtime/internal/runtime.String" %66)
// CHECK-NEXT:   %68 = and i1 %64, %67
// CHECK-NEXT:   %69 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %55, 3
// CHECK-NEXT:   %70 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %56, 3
// CHECK-NEXT:   %71 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %69, %"{{.*}}/runtime/internal/runtime.eface" %70)
// CHECK-NEXT:   %72 = and i1 %68, %71
// CHECK-NEXT:   %73 = xor i1 %72, true
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %73)
// CHECK-NEXT:   %74 = load %"{{.*}}/cl/_testgo/equal.T", ptr %11, align 8
// CHECK-NEXT:   %75 = load %"{{.*}}/cl/_testgo/equal.T", ptr %22, align 8
// CHECK-NEXT:   %76 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %74, 0
// CHECK-NEXT:   %77 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %75, 0
// CHECK-NEXT:   %78 = icmp eq i64 %76, %77
// CHECK-NEXT:   %79 = and i1 true, %78
// CHECK-NEXT:   %80 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %74, 1
// CHECK-NEXT:   %81 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %75, 1
// CHECK-NEXT:   %82 = icmp eq i64 %80, %81
// CHECK-NEXT:   %83 = and i1 %79, %82
// CHECK-NEXT:   %84 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %74, 2
// CHECK-NEXT:   %85 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %75, 2
// CHECK-NEXT:   %86 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %84, %"{{.*}}/runtime/internal/runtime.String" %85)
// CHECK-NEXT:   %87 = and i1 %83, %86
// CHECK-NEXT:   %88 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %74, 3
// CHECK-NEXT:   %89 = extractvalue %"{{.*}}/cl/_testgo/equal.T" %75, 3
// CHECK-NEXT:   %90 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %88, %"{{.*}}/runtime/internal/runtime.eface" %89)
// CHECK-NEXT:   %91 = and i1 %87, %90
// CHECK-NEXT:   %92 = xor i1 %91, true
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %92)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   store i64 2, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   store i64 3, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4, i64 3, 1
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 3, 2
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %7, i64 8, i64 2, i64 0, i64 2, i1 true, i1 true, i1 true)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %9, i64 8, i64 2, i64 0, i64 0, i1 true, i1 true, i1 true)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %12 = icmp ne ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %12)
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, 0
// CHECK-NEXT:   %14 = icmp ne ptr %13, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %14)
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, 0
// CHECK-NEXT:   %16 = icmp ne ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %16)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store {} zeroinitializer, ptr %2, align 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %4 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %4, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %5 = icmp eq ptr %4, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %4, i32 0, i32 0
// CHECK-NEXT:   %7 = icmp eq ptr %4, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %4, i32 0, i32 1
// CHECK-NEXT:   %9 = icmp eq ptr %4, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %4, i32 0, i32 2
// CHECK-NEXT:   %11 = icmp eq ptr %4, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %4, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %6, align 8
// CHECK-NEXT:   store i64 20, ptr %8, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %10, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %13, align 8
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %13, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %14, ptr %12, align 8
// CHECK-NEXT:   %15 = load %"{{.*}}/cl/_testgo/equal.T", ptr %4, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/equal.T" %15, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/equal.T", ptr undef }, ptr %16, 1
// CHECK-NEXT:   %18 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %18, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %19 = icmp eq ptr %18, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %19)
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %18, i32 0, i32 0
// CHECK-NEXT:   %21 = icmp eq ptr %18, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %18, i32 0, i32 1
// CHECK-NEXT:   %23 = icmp eq ptr %18, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %23)
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %18, i32 0, i32 2
// CHECK-NEXT:   %25 = icmp eq ptr %18, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %25)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %18, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %20, align 8
// CHECK-NEXT:   store i64 20, ptr %22, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %24, align 8
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %27, align 8
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %27, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %28, ptr %26, align 8
// CHECK-NEXT:   %29 = alloca %"{{.*}}/cl/_testgo/equal.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %29, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %30 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %30)
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %29, i32 0, i32 0
// CHECK-NEXT:   %32 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %32)
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %29, i32 0, i32 1
// CHECK-NEXT:   %34 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %29, i32 0, i32 2
// CHECK-NEXT:   %36 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %36)
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/cl/_testgo/equal.T", ptr %29, i32 0, i32 3
// CHECK-NEXT:   store i64 10, ptr %31, align 8
// CHECK-NEXT:   store i64 20, ptr %33, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 5 }, ptr %35, align 8
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 2 }, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %38, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %39, ptr %37, align 8
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %40, align 8
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %40, 1
// CHECK-NEXT:   %42 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %1, %"{{.*}}/runtime/internal/runtime.eface" %41)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %42)
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store {} zeroinitializer, ptr %43, align 1
// CHECK-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr undef }, ptr %43, 1
// CHECK-NEXT:   %45 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %3, %"{{.*}}/runtime/internal/runtime.eface" %44)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %45)
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/equal.N" zeroinitializer, ptr %46, align 1
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/equal.N", ptr undef }, ptr %46, 1
// CHECK-NEXT:   %48 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %3, %"{{.*}}/runtime/internal/runtime.eface" %47)
// CHECK-NEXT:   %49 = xor i1 %48, true
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %49)
// CHECK-NEXT:   %50 = load %"{{.*}}/cl/_testgo/equal.T", ptr %18, align 8
// CHECK-NEXT:   %51 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/equal.T" %50, ptr %51, align 8
// CHECK-NEXT:   %52 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/equal.T", ptr undef }, ptr %51, 1
// CHECK-NEXT:   %53 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %17, %"{{.*}}/runtime/internal/runtime.eface" %52)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %53)
// CHECK-NEXT:   %54 = load %"{{.*}}/cl/_testgo/equal.T", ptr %29, align 8
// CHECK-NEXT:   %55 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/equal.T" %54, ptr %55, align 8
// CHECK-NEXT:   %56 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/equal.T", ptr undef }, ptr %55, 1
// CHECK-NEXT:   %57 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %17, %"{{.*}}/runtime/internal/runtime.eface" %56)
// CHECK-NEXT:   %58 = xor i1 %57, true
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %58)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   %2 = icmp eq ptr %0, %0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %2)
// CHECK-NEXT:   %3 = icmp ne ptr %0, %1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %3)
// CHECK-NEXT:   %4 = icmp ne ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = icmp ne ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.test"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal0"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal0"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
