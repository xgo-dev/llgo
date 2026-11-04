// LITTEST darwin/arm64 linux/amd64
package main

// Each source section below exercises a different equality lowering. Keep the
// checks function-scoped and follow the comparison result through to assert so
// a coincidental helper call elsewhere cannot satisfy the test.
// CHECK-LABEL: define void @main.assert(i1 %0){{.*}} {
// CHECK: br i1 %0, label %{{.*}}, label %{{.*}}

// Function values: the generated function descriptor pointer is non-nil and that
// predicate is what reaches assert.
// CHECK-LABEL: define void @"main.init#1"(){{.*}} {
// CHECK: [[EQUAL_CAPTURE:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[EQUAL_ALLOCATION:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK-NEXT: %[[CLOSURE:[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[EQUAL_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[EQUAL_ENV:%[0-9]+]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[EQUAL_ALLOCATION]], i32 0, i32 1
// CHECK: store ptr [[EQUAL_CAPTURE]], ptr %{{[0-9]+}}
// CHECK: [[EQUAL_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr %[[CLOSURE]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.init#1$2", ptr [[EQUAL_CODE_SLOT]]
// CHECK: %[[FUNC_NONNULL:[0-9]+]] = icmp ne ptr %[[CLOSURE]], null
// CHECK: call void @main.assert(i1 %[[FUNC_NONNULL]])

// CHECK-LABEL: define i64 @"main.init#1$1"(i64 %0, i64 %1){{.*}} {
// CHECK: [[FUNC_SUM:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[FUNC_SUM]]

// ARM64-LABEL: define void @"main.init#1$2"(ptr swiftself %0){{.*}} {
// AMD64-LABEL: define void @"main.init#1$2"(ptr nest %0){{.*}} {
// CHECK: load { ptr }, ptr %0

// Arrays: all three elements participate in equality, and inequality negates
// the aggregate result rather than changing element semantics.
// CHECK-LABEL: define void @"main.init#2"(){{.*}} {
// CHECK: extractvalue [3 x i64] %[[ARRAY_L:[0-9]+]], 0
// CHECK: extractvalue [3 x i64] %[[ARRAY_R:[0-9]+]], 0
// CHECK: extractvalue [3 x i64] %[[ARRAY_L]], 1
// CHECK: extractvalue [3 x i64] %[[ARRAY_R]], 1
// CHECK: extractvalue [3 x i64] %[[ARRAY_L]], 2
// CHECK: extractvalue [3 x i64] %[[ARRAY_R]], 2
// CHECK: %[[ARRAY_EQ:[0-9]+]] = and i1 %{{[0-9]+}}, %{{[0-9]+}}
// CHECK: call void @main.assert(i1 %[[ARRAY_EQ]])
// CHECK: %[[ARRAY_NE:[0-9]+]] = xor i1 %{{[0-9]+}}, true
// CHECK: call void @main.assert(i1 %[[ARRAY_NE]])

// Semantic structs retain field-wise comparison, including the interface
// equality calls whose order and panic behavior are observable. Large regular
// structs use the bytewise zero helper without materializing the aggregate.
// CHECK-LABEL: define void @"main.init#3"(){{.*}} {
// CHECK: %[[ZERO_STRING:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"({{.*}}zeroinitializer, {{.*}}zeroinitializer)
// CHECK: %[[ZERO_IFACE:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"({{.*}}zeroinitializer, {{.*}}zeroinitializer)
// CHECK: call void @main.assert(i1 %{{[0-9]+}})
// CHECK: extractvalue %main.T %{{[0-9]+}}, 0
// CHECK: %[[STRUCT_STRING:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"
// CHECK: %[[STRUCT_IFACE:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call void @main.assert(i1 %{{[0-9]+}})
// CHECK: %[[STRUCT_RAW_NE:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[STRUCT_NE:[0-9]+]] = xor i1 %{{[0-9]+}}, true
// CHECK: call void @main.assert(i1 %[[STRUCT_NE]])
// CHECK: %[[ZERO_EQ:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.memequalzero"
// CHECK: call void @main.assert(i1 %[[ZERO_EQ]])
// CHECK: %[[ZERO_RAW_NE:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.memequalzero"
// CHECK: %[[ZERO_NE:[0-9]+]] = xor i1 %[[ZERO_RAW_NE]], true
// CHECK: call void @main.assert(i1 %[[ZERO_NE]])
// CHECK-NOT: runtime.structequal

// Slices compare with nil through their data pointer. Check the non-empty and
// zero-length/non-zero-capacity make paths separately.
// CHECK-LABEL: define void @"main.init#4"(){{.*}} {
// CHECK: %[[SLICE_LEN:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %{{[0-9]+}}, i64 8, i64 2, i64 0, i64 2,{{.*}})
// CHECK: %[[SLICE_CAP:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %{{[0-9]+}}, i64 8, i64 2, i64 0, i64 0,{{.*}})
// CHECK: %[[SLICE_PTR:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE_CAP]], 0
// CHECK: %[[SLICE_NONNULL:[0-9]+]] = icmp ne ptr %[[SLICE_PTR]], null
// CHECK: call void @main.assert(i1 %[[SLICE_NONNULL]])

// Interface equality must feed the assertion directly, including a negated
// result for unequal dynamic types.
// CHECK-LABEL: define void @"main.init#5"(){{.*}} {
// CHECK: %[[IFACE_EQ:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call void @main.assert(i1 %[[IFACE_EQ]])
// CHECK: %[[IFACE_EQ2:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call void @main.assert(i1 %[[IFACE_EQ2]])
// CHECK: %[[IFACE_RAW_NE:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[IFACE_NE:[0-9]+]] = xor i1 %[[IFACE_RAW_NE]], true
// CHECK: call void @main.assert(i1 %[[IFACE_NE]])

// CHECK-LABEL: define void @"main.init#6"(){{.*}} {
// CHECK: %[[CHAN_A:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK: %[[CHAN_B:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK: %[[CHAN_NE:[0-9]+]] = icmp ne ptr %[[CHAN_A]], %[[CHAN_B]]
// CHECK: call void @main.assert(i1 %[[CHAN_NE]])

// CHECK-LABEL: define void @"main.init#7"(){{.*}} {
// CHECK: %[[MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK: %[[MAP_NONNULL:[0-9]+]] = icmp ne ptr %[[MAP]], null
// CHECK: call void @main.assert(i1 %[[MAP_NONNULL]])

func test() {}

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
	var large struct{ data [2049]byte }
	assert(large == struct{ data [2049]byte }{})
	large.data[1024] = 1
	assert(large != struct{ data [2049]byte }{})
	large.data[1024] = 0
	large.data[2048] = 1
	assert(large != struct{ data [2049]byte }{})
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

// map
func init() {
	m1 := make(map[int]string)
	var m2 map[int]string
	assert(m1 != nil)
	assert(m2 == nil)
}

func main() {
}
