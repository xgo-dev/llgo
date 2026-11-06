// LITTEST
// Scope: common
package main

// Map capacity hints must honor the source integer's signedness before the
// common runtime size. These were historically separate package compiles.
// CHECK-LABEL: define i64 @main.fromInt32(i32 %0){{.*}} {
// CHECK: [[INT_HINT:%[0-9]+]] = sext i32 %0 to i64
// CHECK: [[INT_MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMapBuffer"(ptr @"map[_llgo_string]_llgo_int", i64 [[INT_HINT]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK-NEXT: [[INT_LEN:%[0-9]+]] = call i64 @"{{.*}}MapLen"(ptr [[INT_MAP]])
// CHECK-NEXT: ret i64 [[INT_LEN]]

// CHECK-LABEL: define i64 @main.fromUint32(i32 %0){{.*}} {
// CHECK: [[UINT_HINT:%[0-9]+]] = zext i32 %0 to i64
// CHECK: [[UINT_MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMapBuffer"(ptr @"map[_llgo_string]_llgo_int", i64 [[UINT_HINT]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK-NEXT: [[UINT_LEN:%[0-9]+]] = call i64 @"{{.*}}MapLen"(ptr [[UINT_MAP]])
// CHECK-NEXT: ret i64 [[UINT_LEN]]

// make1 covers the full lifecycle. Tie every operation to the map returned by
// MakeMap, while leaving scalar extraction and control flow to focused tests.
// CHECK-LABEL: define void @main.make1(){{.*}} {
// CHECK: %[[MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[_llgo_int]_llgo_string", i64 0, ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: %[[ASSIGN:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64"(ptr @"map[_llgo_int]_llgo_string", ptr %[[MAP]], i64 1)
// CHECK: %[[VALUE:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"(ptr @"map[_llgo_int]_llgo_string", ptr %[[MAP]], i64 1)
// CHECK: %[[ITER:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_int]_llgo_string", ptr %[[MAP]])
// CHECK: call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %[[ITER]])
// CHECK: %[[MAP_LEN:[0-9]+]] = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %[[MAP]])
// CHECK: %[[REVERSE:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[_llgo_string]_llgo_int", i64 %[[MAP_LEN]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFastStr"(ptr @"map[_llgo_string]_llgo_int", ptr %[[REVERSE]], %"{{.*}}/runtime/internal/runtime.String" %{{[0-9]+}})
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFastStr"(ptr @"map[_llgo_string]_llgo_int", ptr %[[REVERSE]], %"{{.*}}/runtime/internal/runtime.String" {{.*}})
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2FastStr"(ptr @"map[_llgo_string]_llgo_int", ptr %[[REVERSE]], %"{{.*}}/runtime/internal/runtime.String" {{.*}})

// Interface keys retain their dynamic type while assignment and iteration use
// the same any-key map.
// CHECK-LABEL: define void @main.make2(){{.*}} {
// A nil map lookup must use the ordinary access lowering and materialize the
// element zero value returned by the runtime.
// CHECK: %[[NIL_VALUE_PTR:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"(ptr @"map[_llgo_int]_llgo_string", ptr null, i64 42)
// CHECK: %[[NIL_VALUE:[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr %[[NIL_VALUE_PTR]]
// CHECK: %[[ANY_MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[_llgo_any]_llgo_int", i64 0, ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.N1, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %[[ANY_MAP]], ptr %{{[0-9]+}})
// CHECK: %[[ANY_ITER:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_any]_llgo_int", ptr %[[ANY_MAP]])
// CHECK: call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %[[ANY_ITER]])

// Array-of-value and array-of-pointer interface keys both go through dynamic
// equality, but remain separate source scenarios.
// CHECK-LABEL: define void @main.make3(){{.*}} {
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int"

// CHECK-LABEL: define void @main.make4(){{.*}} {
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int"

// Channel identity is used both through interface equality and as a direct map
// key; both operations must consume the channel created here.
// CHECK-LABEL: define void @main.make5(){{.*}} {
// CHECK: %[[CHAN:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr undef }, ptr %[[CHAN]], 1
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[CHAN_MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[chan _llgo_int]_llgo_int", i64 0, ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64Ptr"(ptr @"map[chan _llgo_int]_llgo_int", ptr %[[CHAN_MAP]], ptr %[[CHAN]])

// A named map uses its named descriptor for operations even though allocation
// uses the identical underlying map layout.
// CHECK-LABEL: define void @main.make6(){{.*}} {
// CHECK: %[[NAMED_MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[_llgo_int]_llgo_string", i64 0, ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64"(ptr @_llgo_main.M, ptr %[[NAMED_MAP]], i64 1)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @_llgo_main.M, ptr %[[NAMED_MAP]])

// A local named key has its own descriptor and the literal's two entries are
// reflected in the allocation hint.
// CHECK-LABEL: define void @main.make7(){{.*}} {
// CHECK: %[[LOCAL_MAP:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMapBuffer"(ptr @"map[_llgo_main.N.7.0]_llgo_string", i64 2, ptr %{{[0-9]+}}, ptr %{{[0-9]+}})
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64"(ptr @"map[_llgo_main.N.7.0]_llgo_string", ptr %[[LOCAL_MAP]], i64 1)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"(ptr @"map[_llgo_main.N.7.0]_llgo_string", ptr %[[LOCAL_MAP]], i64 1)

func main() {
	make1()
	make2()
	make3()
	make4()
	make5()
	make6()
	make7()
}

func make1() {
	m := make(map[int]string)
	m[1] = "hello"
	m[2] = "world"
	m[3] = "llgo"
	println(m, m[1], m[2], len(m))
	for k, v := range m {
		println(k, ":", v)
	}

	s := make(map[string]int, len(m))
	for k, v := range m {
		s[v] = k
	}

	id, ok := s["llgo"]
	println("llgo", id, ok)

	none, ok := s["go"]
	println("go", none, ok)

	delete(s, "llgo")
	if _, ok := s["llgo"]; ok {
		panic("bad key")
	}
	if len(s) != 2 {
		panic("bad len")
	}
}

type N1 [1]int

func make2() {
	m2 := make(map[int]string)
	println(m2, len(m2), m2 == nil, m2 != nil)
	var m3 map[int]string
	println(m3, len(m3), m3 == nil, m3 != nil)
	if got := m3[42]; got != "" {
		panic("nil map lookup returned a non-zero value")
	}

	n := make(map[any]int)
	n[N1{1}] = 100
	n[N1{2}] = 200
	n[N1{3}] = 300
	n[N1{2}] = -200
	for k, v := range n {
		println(k.(N1)[0], v)
	}
}

type N struct {
	n1 int8
	n2 int8
}
type K [1]N
type K2 [1]*N

func make3() {
	var a any = K{N{1, 2}}
	var b any = K{N{1, 2}}
	println(a == b)

	m := make(map[any]int)
	m[K{N{1, 2}}] = 100
	m[K{N{3, 4}}] = 200
	for k, v := range m {
		println(k.(K)[0].n1, v)
	}
}

func make4() {
	var a any = K2{&N{1, 2}}
	var b any = K2{&N{1, 2}}
	println(a == b)

	m := make(map[any]int)
	m[K2{&N{1, 2}}] = 100
	m[K2{&N{3, 4}}] = 200
	for k, v := range m {
		println(k.(K2)[0].n1, v)
	}
}

func make5() {
	ch := make(chan int)
	var a any = ch
	var b any = ch
	println(a == b)
	m := make(map[chan int]int)
	m[ch] = 100
	m[ch] = 200
	for k, v := range m {
		println(k, v)
	}
}

type M map[int]string

func make6() {
	var m M
	m = make(map[int]string)
	m[1] = "hello"
	for k, v := range m {
		println(k, v)
	}
}

func make7() {
	type N int
	m := map[N]string{
		1: "hello",
		2: "world",
	}
	for k, v := range m {
		println(k, v)
	}
	println(m[1])
}

func fromInt32(n int32) int {
	return len(make(map[string]int, n))
}

func fromUint32(n uint32) int {
	return len(make(map[string]int, n))
}
