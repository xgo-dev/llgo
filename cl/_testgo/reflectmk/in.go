// LITTEST
package main

import (
	"fmt"
	"reflect"
)

type Point struct {
	x int
	y int
}

func (p *Point) Set(x int, y int) {
	p.x = x
	p.y = y
}

func (p Point) String() string {
	return fmt.Sprintf("(%v,%v)", p.x, p.y)
}

func main() {
	rt := reflect.TypeOf((*Point)(nil)).Elem()
	if t := reflect.ArrayOf(1, rt); t.Elem() != rt {
		panic("arrayOf error")
	}
	if t := reflect.ChanOf(reflect.SendDir, rt); t.Elem() != rt {
		panic("chanOf error")
	}
	if t := reflect.FuncOf([]reflect.Type{rt}, []reflect.Type{rt}, false); t.In(0) != rt || t.Out(0) != rt {
		panic("funcOf error")
	}
	if t := reflect.MapOf(rt, rt); t.Key() != rt || t.Elem() != rt {
		panic("mapOf error")
	}
	if t := reflect.PointerTo(rt); t.Elem() != rt {
		panic("pointerTo error")
	}
	if t := reflect.SliceOf(rt); t.Elem() != rt {
		panic("sliceOf error")
	}
	if t := reflect.StructOf([]reflect.StructField{
		{Name: "T", Type: rt},
	}); t.Field(0).Type != rt {
		panic("structOf error")
	}
	if t := rt.Method(0); t.Name != "String" {
		panic("method error")
	}
	if t, ok := rt.MethodByName("String"); !ok || t.Name != "String" {
		panic("methodByName error")
	}
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.Method(1).Call(nil); r[0].String() != "(1,2)" {
		panic("value.Method error")
	}
	if r := v.MethodByName("String").Call(nil); r[0].String() != "(1,2)" {
		panic("value.MethodByName error")
	}
	method(1)
	methodByName("String")
}

func method(n int) {
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.Method(n).Call(nil); r[0].String() != "(1,2)" {
		panic("value.Method error")
	}
}

func methodByName(name string) {
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.MethodByName(name).Call(nil); r[0].String() != "(1,2)" {
		panic("value.MethodByName error")
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// Recover the element Type from *Point and reuse that exact interface for all
// constructor inputs.
// CHECK: %[[PTR_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.Point", ptr null })
// CHECK: %[[PTR_DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[PTR_TYPE]])
// CHECK: %[[RT:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}(ptr %{{[0-9]+}})

// Each constructed type must be queried and compared back to RT. This proves
// more than the presence of seven reflect helper calls.
// CHECK: %[[ARRAY_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.ArrayOf(i64 1, %"{{.*}}/runtime/internal/runtime.iface" %[[RT]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[ARRAY_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[CHAN_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.ChanOf(i64 2, %"{{.*}}/runtime/internal/runtime.iface" %[[RT]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[CHAN_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %[[RT]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %[[RT]], ptr %{{[0-9]+}}
// CHECK: %[[FUNC_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.FuncOf(%"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, i1 false)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[FUNC_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[MAP_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.MapOf(%"{{.*}}/runtime/internal/runtime.iface" %[[RT]], %"{{.*}}/runtime/internal/runtime.iface" %[[RT]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[MAP_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[PTR_TO_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.PointerTo(%"{{.*}}/runtime/internal/runtime.iface" %[[RT]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[PTR_TO_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: %[[SLICE_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.SliceOf(%"{{.*}}/runtime/internal/runtime.iface" %[[RT]])
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[SLICE_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"
// CHECK: store %"{{.*}}/runtime/internal/runtime.iface" %[[RT]], ptr %{{[0-9]+}}
// CHECK: %[[STRUCT_TYPE:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.StructOf(%"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}})
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[STRUCT_TYPE]])
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"

// Type.Method's returned Method is stored and its Name field is what reaches
// StringEqual. MethodByName additionally carries the tuple's ok bit to a branch.
// CHECK: %[[METHOD_SLOT:[0-9]+]] = alloca %reflect.Method
// CHECK: %[[METHOD:[0-9]+]] = call %reflect.Method %{{[0-9]+}}(ptr %{{[0-9]+}}, i64 0)
// CHECK: store %reflect.Method %[[METHOD]], ptr %[[METHOD_SLOT]]
// CHECK: %[[METHOD_NAME_PTR:[0-9]+]] = getelementptr inbounds %reflect.Method, ptr %[[METHOD_SLOT]], i32 0, i32 0
// CHECK: %[[METHOD_NAME:[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr %[[METHOD_NAME_PTR]]
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %[[METHOD_NAME]],{{.*}})
// CHECK: %[[NAMED_METHOD:[0-9]+]] = call { %reflect.Method, i1 } %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK: extractvalue { %reflect.Method, i1 } %[[NAMED_METHOD]], 0
// CHECK: %[[METHOD_OK:[0-9]+]] = extractvalue { %reflect.Method, i1 } %[[NAMED_METHOD]], 1
// CHECK: br i1 %[[METHOD_OK]]

// Value.Method and MethodByName must invoke the Value they return, range-check
// the returned slice, and compare the first result's string.
// CHECK: %[[VALUE:[0-9]+]] = call %reflect.Value @reflect.ValueOf
// CHECK: %[[INDEX_METHOD:[0-9]+]] = call %reflect.Value @reflect.Value.Method(%reflect.Value %[[VALUE]], i64 1)
// CHECK: %[[INDEX_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[INDEX_METHOD]], %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK: %[[INDEX_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[INDEX_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 0, i1 true, i64 %[[INDEX_LEN]])
// CHECK: %[[INDEX_STRING:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %[[INDEX_STRING]],{{.*}})
// CHECK: %[[NAME_METHOD:[0-9]+]] = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %[[VALUE]], %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK: %[[NAME_RESULTS:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[NAME_METHOD]], %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK: %[[NAME_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[NAME_RESULTS]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 0, i1 true, i64 %[[NAME_LEN]])
// CHECK: %[[NAME_STRING:[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %[[NAME_STRING]],{{.*}})
// CHECK:   call void @main.method(i64 1)
// CHECK:   call void @main.methodByName(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })

// CHECK-LABEL: define void @main.method(i64 %0){{.*}} {
// CHECK: %[[DYNAMIC_INDEX:[0-9]+]] = call %reflect.Value @reflect.Value.Method(%reflect.Value %{{[0-9]+}}, i64 %0)
// CHECK: call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[DYNAMIC_INDEX]],

// CHECK-LABEL: define void @main.methodByName(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK: %[[DYNAMIC_NAME:[0-9]+]] = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK: call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %[[DYNAMIC_NAME]],
