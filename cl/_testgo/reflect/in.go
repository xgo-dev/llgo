// LITTEST
package main

import (
	"reflect"
	"unsafe"
)

func main() {
	callSlice()
	callFunc()
	callClosure()
	callMethod()
	callIMethod()
	mapDemo1()
	mapDemo2()
}

func demo(n1, n2, n3, n4, n5, n6, n7, n8, n9 int, a ...interface{}) (int, int) {
	var sum int
	for _, v := range a {
		sum += v.(int)
	}
	return n1 + n2 + n3 + n4 + n5 + n6 + n7 + n8 + n9, sum
}

func callSlice() {
	v := reflect.ValueOf(demo)
	n := reflect.ValueOf(1)
	r := v.Call([]reflect.Value{n, n, n, n, n, n, n, n, n,
		reflect.ValueOf(1), reflect.ValueOf(2), reflect.ValueOf(3)})
	println("call.slice", r[0].Int(), r[1].Int())
	r = v.CallSlice([]reflect.Value{n, n, n, n, n, n, n, n, n,
		reflect.ValueOf([]interface{}{1, 2, 3})})
	println("call.slice", r[0].Int(), r[1].Int())
}

func callFunc() {
	var f any = func(n int) int {
		println("call.func")
		return n + 1
	}
	fn := reflect.ValueOf(f)
	println("func", fn.Kind(), fn.Type().String())
	r := fn.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r[0].Int())
	ifn, ok := fn.Interface().(func(int) int)
	if !ok {
		panic("error")
	}
	ifn(100)
}

func callClosure() {
	m := 100
	var f any = func(n int) int {
		println("call.closure")
		return m + n + 1
	}
	fn := reflect.ValueOf(f)
	println("closure", fn.Kind(), fn.Type().String())
	r := fn.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r[0].Int())
	ifn, ok := fn.Interface().(func(int) int)
	if !ok {
		panic("error")
	}
	ifn(100)
}

type T struct {
	n int
}

func (t *T) Add(n int) int {
	println("call.method")
	t.n += n
	return t.n
}

type I interface {
	Add(n int) int
}

type abi struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

func callMethod() {
	t := &T{1}
	v := reflect.ValueOf(t)
	fn := v.Method(0)
	println("method", fn.Kind(), fn.Type().String())
	r := fn.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r[0].Int())
	ifn, ok := fn.Interface().(func(int) int)
	if !ok {
		panic("error")
	}
	ifn(1)
	v2 := reflect.ValueOf(fn.Interface())
	r2 := v2.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r2[0].Int())
}

func callIMethod() {
	var i I = &T{1}
	v := reflect.ValueOf(i)
	fn := v.Method(0)
	println("imethod", fn.Kind(), fn.Type().String())
	r := fn.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r[0].Int())
	ifn, ok := fn.Interface().(func(int) int)
	if !ok {
		panic("error")
	}
	ifn(1)
	v2 := reflect.ValueOf(fn.Interface())
	r2 := v2.Call([]reflect.Value{reflect.ValueOf(100)})
	println(r2[0].Int())
}

func mapDemo1() {
	m := map[int]string{
		1: "hello",
		2: "world",
	}
	v := reflect.ValueOf(m)
	if v.Len() != 2 || len(v.MapKeys()) != 2 {
		panic("error")
	}
	if v.MapIndex(reflect.ValueOf(2)).String() != "world" {
		panic("MapIndex error")
	}
	v.SetMapIndex(reflect.ValueOf(2), reflect.ValueOf("todo"))
	if v.MapIndex(reflect.ValueOf(2)).String() != "todo" {
		panic("MapIndex error")
	}
	if v.MapIndex(reflect.ValueOf(0)).IsValid() {
		println("must invalid")
	}
	key := reflect.New(v.Type().Key()).Elem()
	value := reflect.New(v.Type().Elem()).Elem()
	iter := v.MapRange()
	for iter.Next() {
		key.SetIterKey(iter)
		value.SetIterValue(iter)
		if key.Int() != iter.Key().Int() || value.String() != iter.Value().String() {
			panic("MapIter error")
		}
	}
}

func mapDemo2() {
	v := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(0), reflect.TypeOf("")))
	v.SetMapIndex(reflect.ValueOf(1), reflect.ValueOf("hello"))
	v.SetMapIndex(reflect.ValueOf(2), reflect.ValueOf("world"))
	if v.Len() != 2 || len(v.MapKeys()) != 2 {
		panic("error")
	}
	if v.MapIndex(reflect.ValueOf(2)).String() != "world" {
		panic("MapIndex error")
	}
	v.SetMapIndex(reflect.ValueOf(2), reflect.ValueOf("todo"))
	if v.MapIndex(reflect.ValueOf(2)).String() != "todo" {
		panic("MapIndex error")
	}
	if v.MapIndex(reflect.ValueOf(0)).IsValid() {
		println("must invalid")
	}
	key := reflect.New(v.Type().Key()).Elem()
	value := reflect.New(v.Type().Elem()).Elem()
	iter := v.MapRange()
	for iter.Next() {
		key.SetIterKey(iter)
		value.SetIterValue(iter)
		if key.Int() != iter.Key().Int() || value.String() != iter.Value().String() {
			panic("MapIter error")
		}
	}
}

// CHECK-LABEL: define void @main.callClosure(){{.*}} {
// CHECK: %[[CLOSURE_VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[CLOSURE_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[CLOSURE_VALUE]], %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}})
// CHECK: %[[CLOSURE_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[CLOSURE_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 0, i1 true, i64 %[[CLOSURE_LEN]])
// CHECK: %[[CLOSURE_IFACE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %[[CLOSURE_VALUE]])
// CHECK: %[[CLOSURE_TYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %[[CLOSURE_IFACE]], 0
// CHECK: %[[CLOSURE_MATCH:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %[[CLOSURE_TYPE]])
// CHECK: br i1 %[[CLOSURE_MATCH]]
// CHECK: call i64 %__llgo_funcval_code(ptr {{(nest|swiftself)}} %{{[0-9]+}}, i64 100)

// CHECK-LABEL: define i64 @"main.callClosure$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
// CHECK: [[CC_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[CC_BASE_PTR:%[0-9]+]] = extractvalue { ptr } [[CC_ENV]], 0
// CHECK-NEXT: [[CC_BASE:%[0-9]+]] = load i64, ptr [[CC_BASE_PTR]]
// CHECK-NEXT: [[CC_SUM:%[0-9]+]] = add i64 [[CC_BASE]], %1
// CHECK-NEXT: [[CC_RESULT:%[0-9]+]] = add i64 [[CC_SUM]], 1
// CHECK-NEXT: ret i64 [[CC_RESULT]]

// CHECK-LABEL: define void @main.callFunc(){{.*}} {
// CHECK: %[[FUNC_VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[FUNC_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[FUNC_VALUE]], %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}})
// CHECK: %[[FUNC_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[FUNC_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 0, i1 true, i64 %[[FUNC_LEN]])
// CHECK: call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %[[FUNC_VALUE]])

// CHECK-LABEL: define void @main.callIMethod(){{.*}} {
// CHECK: %[[IFACE_VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[IFACE_METHOD:[0-9]+]] = call %reflect.Value @reflect.Value.Method(%reflect.Value %[[IFACE_VALUE]], i64 0)
// CHECK: %[[IFACE_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[IFACE_METHOD]],{{.*}})
// CHECK: call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %[[IFACE_METHOD]])
// CHECK: call i64 %__llgo_funcval_code(ptr {{(nest|swiftself)}} %{{[0-9]+}}, i64 1)

// CHECK-LABEL: define void @main.callMethod(){{.*}} {
// CHECK: %[[RECV_VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[METHOD_VALUE:[0-9]+]] = call %reflect.Value @reflect.Value.Method(%reflect.Value %[[RECV_VALUE]], i64 0)
// CHECK: %[[METHOD_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[METHOD_VALUE]],{{.*}})
// CHECK: call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %[[METHOD_VALUE]])

// CHECK-LABEL: define void @main.callSlice(){{.*}} {
// CHECK: %[[SLICE_FUNC:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[SLICE_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.CallSlice(%reflect.Value %[[SLICE_FUNC]], %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}})
// CHECK: %[[SLICE_RESULT_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 0, i1 true, i64 %[[SLICE_RESULT_LEN]])

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.callSlice()
// CHECK: call void @main.callFunc()
// CHECK: call void @main.callClosure()
// CHECK: call void @main.callMethod()
// CHECK: call void @main.callIMethod()
// CHECK: call void @main.mapDemo1()
// CHECK: call void @main.mapDemo2()

// CHECK-LABEL: define void @main.mapDemo1(){{.*}} {
// CHECK: %[[MAP_VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[MAP_ENTRY:[0-9]+]] = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %[[MAP_VALUE]], %reflect.Value %{{[0-9]+}})
// CHECK: %[[MAP_STRING:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %[[MAP_ENTRY]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %[[MAP_STRING]],{{.*}})
// CHECK: call void @reflect.Value.SetMapIndex(%reflect.Value %[[MAP_VALUE]], %reflect.Value %{{[0-9]+}}, %reflect.Value %{{[0-9]+}})
// CHECK: %[[MAP_ITER:[0-9]+]] = call ptr @reflect.Value.MapRange(%reflect.Value %[[MAP_VALUE]])
// CHECK: call void @reflect.Value.SetIterKey(%reflect.Value %{{[0-9]+}}, ptr %[[MAP_ITER]])
// CHECK: call void @reflect.Value.SetIterValue(%reflect.Value %{{[0-9]+}}, ptr %[[MAP_ITER]])
// CHECK: call i1 @"reflect.(*MapIter).Next"(ptr %[[MAP_ITER]])

// CHECK-LABEL: define void @main.mapDemo2(){{.*}} {
// CHECK: %[[MAP_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.MapOf(%"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}})
// CHECK: %[[MADE_MAP:[0-9]+]] = call %reflect.Value @reflect.MakeMap(%"{{.*}}/runtime/internal/runtime.iface" %[[MAP_TYPE]])
// CHECK: call void @reflect.Value.SetMapIndex(%reflect.Value %[[MADE_MAP]], %reflect.Value %{{[0-9]+}}, %reflect.Value %{{[0-9]+}})
// CHECK: call i64 @reflect.Value.Len(%reflect.Value %[[MADE_MAP]])
