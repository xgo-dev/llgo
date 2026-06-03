// LITTEST
package main

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
// func
func init() {
	fn1 := test
	fn2 := func(i, j int) int { return i + j }
	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/equal.init#1$1"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = add i64 %0, %1
	// CHECK-NEXT:   ret i64 %2
	// CHECK-NEXT: }
	var n int
	fn3 := func() { println(n) }
	// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#1$2"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr {{%[0-9]+}}, align 8
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call void @"{{.*}}/cl/_testgo/equal.assert"
// CHECK: ret void
// CHECK-NEXT: }
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
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice3"(ptr %7, i64 8, i64 2, i64 0, i64 2, i64 2)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice3"(ptr %9, i64 8, i64 2, i64 0, i64 0, i64 2)
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr {{%[0-9]+}}, align 8
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr {{%[0-9]+}}, align 8
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call void @"{{.*}}/cl/_testgo/equal.assert"
// CHECK: ret void
// CHECK-NEXT: }
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
// chan
func init() {
	a := make(chan int)
	b := make(chan int)
	assert(a == a)
	assert(a != b)
	assert(a != nil)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/equal.init#7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = icmp ne ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 %1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/equal.assert"(i1 true)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
// map
func init() {
	m1 := make(map[int]string)
	var m2 map[int]string
	assert(m1 != nil)
	assert(m2 == nil)
}

// CHECK-LABEL: define {{.*}} @"{{.*}}/equal.main"{{.*}}
// CHECK: ret void
func main() {
}
