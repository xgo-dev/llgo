// LITTEST
package main

import (
	"fmt"
	"reflect"
)

// CHECK: {{^}}@1 = private unnamed_addr constant [7 x i8] c"(%v,%v)", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [49 x i8] c"{{.*}}/cl/_testgo/reflectmk.Point", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [6 x i8] c"String", align 1{{$}}
// CHECK: {{^}}@12 = private unnamed_addr constant [13 x i8] c"arrayOf error", align 1{{$}}
// CHECK: {{^}}@13 = private unnamed_addr constant [12 x i8] c"chanOf error", align 1{{$}}
// CHECK: {{^}}@14 = private unnamed_addr constant [12 x i8] c"funcOf error", align 1{{$}}
// CHECK: {{^}}@15 = private unnamed_addr constant [11 x i8] c"mapOf error", align 1{{$}}
// CHECK: {{^}}@16 = private unnamed_addr constant [15 x i8] c"pointerTo error", align 1{{$}}
// CHECK: {{^}}@17 = private unnamed_addr constant [13 x i8] c"sliceOf error", align 1{{$}}
// CHECK: {{^}}@18 = private unnamed_addr constant [1 x i8] c"T", align 1{{$}}
// CHECK: {{^}}@19 = private unnamed_addr constant [14 x i8] c"structOf error", align 1{{$}}
// CHECK: {{^}}@20 = private unnamed_addr constant [12 x i8] c"method error", align 1{{$}}
// CHECK: {{^}}@21 = private unnamed_addr constant [18 x i8] c"methodByName error", align 1{{$}}
// CHECK: {{^}}@22 = private unnamed_addr constant [5 x i8] c"(1,2)", align 1{{$}}
// CHECK: {{^}}@23 = private unnamed_addr constant [18 x i8] c"value.Method error", align 1{{$}}
// CHECK: {{^}}@24 = private unnamed_addr constant [24 x i8] c"value.MethodByName error", align 1{{$}}

type Point struct {
	x int
	y int
}

func (p *Point) Set(x int, y int) {
	p.x = x
	p.y = y
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reflectmk.Point.String"(%"{{.*}}/cl/_testgo/reflectmk.Point" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/reflectmk.Point", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reflectmk.Point" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %6, i64 0
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %3, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %8, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %9, ptr %7, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %6, i64 1
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %5, ptr %11, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %11, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %12, ptr %10, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %6, 0
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %13, i64 2, 1
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %14, i64 2, 2
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 7 }, %"{{.*}}/runtime/internal/runtime.Slice" %15)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %16
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.(*Point).Set"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %2, ptr %4, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reflectmk.(*Point).String"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 49 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %0, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reflectmk.Point.String"(%"{{.*}}/cl/_testgo/reflectmk.Point" %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/reflectmk.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/reflectmk.init$guard", align 1
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @reflect.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr null })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 11
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   %9 = call %"{{.*}}/runtime/internal/runtime.iface" %8(ptr %7)
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.ArrayOf(i64 1, %"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %10)
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %10, 0
// CHECK-NEXT:   %13 = getelementptr ptr, ptr %12, i64 11
// CHECK-NEXT:   %14 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue { ptr, ptr } %15, ptr %11, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %16, 1
// CHECK-NEXT:   %18 = extractvalue { ptr, ptr } %16, 0
// CHECK-NEXT:   %19 = call %"{{.*}}/runtime/internal/runtime.iface" %18(ptr %17)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %19)
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %19, 1
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %20, 0
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %22, ptr %21, 1
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %24, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %26, ptr %25, 1
// CHECK-NEXT:   %28 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %23, %"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   %29 = xor i1 %28, true
// CHECK-NEXT:   br i1 %29, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @12, i64 13 }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %32 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.ChanOf(i64 2, %"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %32)
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %32, 0
// CHECK-NEXT:   %35 = getelementptr ptr, ptr %34, i64 11
// CHECK-NEXT:   %36 = load ptr, ptr %35, align 8
// CHECK-NEXT:   %37 = insertvalue { ptr, ptr } undef, ptr %36, 0
// CHECK-NEXT:   %38 = insertvalue { ptr, ptr } %37, ptr %33, 1
// CHECK-NEXT:   %39 = extractvalue { ptr, ptr } %38, 1
// CHECK-NEXT:   %40 = extractvalue { ptr, ptr } %38, 0
// CHECK-NEXT:   %41 = call %"{{.*}}/runtime/internal/runtime.iface" %40(ptr %39)
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %41)
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %41, 1
// CHECK-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %42, 0
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %44, ptr %43, 1
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %46, 0
// CHECK-NEXT:   %49 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %48, ptr %47, 1
// CHECK-NEXT:   %50 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %45, %"{{.*}}/runtime/internal/runtime.eface" %49)
// CHECK-NEXT:   %51 = xor i1 %50, true
// CHECK-NEXT:   br i1 %51, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 12 }, ptr %52, align 8
// CHECK-NEXT:   %53 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %52, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %53)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %54 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %54, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %9, ptr %55, align 8
// CHECK-NEXT:   %56 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %54, 0
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %56, i64 1, 1
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %57, i64 1, 2
// CHECK-NEXT:   %59 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %60 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %59, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %9, ptr %60, align 8
// CHECK-NEXT:   %61 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %59, 0
// CHECK-NEXT:   %62 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %61, i64 1, 1
// CHECK-NEXT:   %63 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %62, i64 1, 2
// CHECK-NEXT:   %64 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.FuncOf(%"{{.*}}/runtime/internal/runtime.Slice" %58, %"{{.*}}/runtime/internal/runtime.Slice" %63, i1 false)
// CHECK-NEXT:   %65 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %64)
// CHECK-NEXT:   %66 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %64, 0
// CHECK-NEXT:   %67 = getelementptr ptr, ptr %66, i64 {{(18|19)}}
// CHECK-NEXT:   %68 = load ptr, ptr %67, align 8
// CHECK-NEXT:   %69 = insertvalue { ptr, ptr } undef, ptr %68, 0
// CHECK-NEXT:   %70 = insertvalue { ptr, ptr } %69, ptr %65, 1
// CHECK-NEXT:   %71 = extractvalue { ptr, ptr } %70, 1
// CHECK-NEXT:   %72 = extractvalue { ptr, ptr } %70, 0
// CHECK-NEXT:   %73 = call %"{{.*}}/runtime/internal/runtime.iface" %72(ptr %71, i64 0)
// CHECK-NEXT:   %74 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %73)
// CHECK-NEXT:   %75 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %73, 1
// CHECK-NEXT:   %76 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %74, 0
// CHECK-NEXT:   %77 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %76, ptr %75, 1
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %79 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %80 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %78, 0
// CHECK-NEXT:   %81 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %80, ptr %79, 1
// CHECK-NEXT:   %82 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %77, %"{{.*}}/runtime/internal/runtime.eface" %81)
// CHECK-NEXT:   %83 = xor i1 %82, true
// CHECK-NEXT:   br i1 %83, label %_llgo_5, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_4
// CHECK-NEXT:   %84 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 12 }, ptr %84, align 8
// CHECK-NEXT:   %85 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %84, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %85)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %86 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.MapOf(%"{{.*}}/runtime/internal/runtime.iface" %9, %"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %87 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %86)
// CHECK-NEXT:   %88 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %86, 0
// CHECK-NEXT:   %89 = getelementptr ptr, ptr %88, i64 {{(20|22)}}
// CHECK-NEXT:   %90 = load ptr, ptr %89, align 8
// CHECK-NEXT:   %91 = insertvalue { ptr, ptr } undef, ptr %90, 0
// CHECK-NEXT:   %92 = insertvalue { ptr, ptr } %91, ptr %87, 1
// CHECK-NEXT:   %93 = extractvalue { ptr, ptr } %92, 1
// CHECK-NEXT:   %94 = extractvalue { ptr, ptr } %92, 0
// CHECK-NEXT:   %95 = call %"{{.*}}/runtime/internal/runtime.iface" %94(ptr %93)
// CHECK-NEXT:   %96 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %95)
// CHECK-NEXT:   %97 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %95, 1
// CHECK-NEXT:   %98 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %96, 0
// CHECK-NEXT:   %99 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %98, ptr %97, 1
// CHECK-NEXT:   %100 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %101 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %102 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %100, 0
// CHECK-NEXT:   %103 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %102, ptr %101, 1
// CHECK-NEXT:   %104 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %99, %"{{.*}}/runtime/internal/runtime.eface" %103)
// CHECK-NEXT:   %105 = xor i1 %104, true
// CHECK-NEXT:   br i1 %105, label %_llgo_8, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %106 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %64)
// CHECK-NEXT:   %107 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %64, 0
// CHECK-NEXT:   %108 = getelementptr ptr, ptr %107, i64 {{(30|33)}}
// CHECK-NEXT:   %109 = load ptr, ptr %108, align 8
// CHECK-NEXT:   %110 = insertvalue { ptr, ptr } undef, ptr %109, 0
// CHECK-NEXT:   %111 = insertvalue { ptr, ptr } %110, ptr %106, 1
// CHECK-NEXT:   %112 = extractvalue { ptr, ptr } %111, 1
// CHECK-NEXT:   %113 = extractvalue { ptr, ptr } %111, 0
// CHECK-NEXT:   %114 = call %"{{.*}}/runtime/internal/runtime.iface" %113(ptr %112, i64 0)
// CHECK-NEXT:   %115 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %114)
// CHECK-NEXT:   %116 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %114, 1
// CHECK-NEXT:   %117 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %115, 0
// CHECK-NEXT:   %118 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %117, ptr %116, 1
// CHECK-NEXT:   %119 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %120 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %121 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %119, 0
// CHECK-NEXT:   %122 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %121, ptr %120, 1
// CHECK-NEXT:   %123 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %118, %"{{.*}}/runtime/internal/runtime.eface" %122)
// CHECK-NEXT:   %124 = xor i1 %123, true
// CHECK-NEXT:   br i1 %124, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10, %_llgo_6
// CHECK-NEXT:   %125 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @15, i64 11 }, ptr %125, align 8
// CHECK-NEXT:   %126 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %125, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %126)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_10
// CHECK-NEXT:   %127 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.PointerTo(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %128 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %127)
// CHECK-NEXT:   %129 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %127, 0
// CHECK-NEXT:   %130 = getelementptr ptr, ptr %129, i64 11
// CHECK-NEXT:   %131 = load ptr, ptr %130, align 8
// CHECK-NEXT:   %132 = insertvalue { ptr, ptr } undef, ptr %131, 0
// CHECK-NEXT:   %133 = insertvalue { ptr, ptr } %132, ptr %128, 1
// CHECK-NEXT:   %134 = extractvalue { ptr, ptr } %133, 1
// CHECK-NEXT:   %135 = extractvalue { ptr, ptr } %133, 0
// CHECK-NEXT:   %136 = call %"{{.*}}/runtime/internal/runtime.iface" %135(ptr %134)
// CHECK-NEXT:   %137 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %136)
// CHECK-NEXT:   %138 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %136, 1
// CHECK-NEXT:   %139 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %137, 0
// CHECK-NEXT:   %140 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %139, ptr %138, 1
// CHECK-NEXT:   %141 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %142 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %143 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %141, 0
// CHECK-NEXT:   %144 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %143, ptr %142, 1
// CHECK-NEXT:   %145 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %140, %"{{.*}}/runtime/internal/runtime.eface" %144)
// CHECK-NEXT:   %146 = xor i1 %145, true
// CHECK-NEXT:   br i1 %146, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %147 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %86)
// CHECK-NEXT:   %148 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %86, 0
// CHECK-NEXT:   %149 = getelementptr ptr, ptr %148, i64 11
// CHECK-NEXT:   %150 = load ptr, ptr %149, align 8
// CHECK-NEXT:   %151 = insertvalue { ptr, ptr } undef, ptr %150, 0
// CHECK-NEXT:   %152 = insertvalue { ptr, ptr } %151, ptr %147, 1
// CHECK-NEXT:   %153 = extractvalue { ptr, ptr } %152, 1
// CHECK-NEXT:   %154 = extractvalue { ptr, ptr } %152, 0
// CHECK-NEXT:   %155 = call %"{{.*}}/runtime/internal/runtime.iface" %154(ptr %153)
// CHECK-NEXT:   %156 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %155)
// CHECK-NEXT:   %157 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %155, 1
// CHECK-NEXT:   %158 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %156, 0
// CHECK-NEXT:   %159 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %158, ptr %157, 1
// CHECK-NEXT:   %160 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %161 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %162 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %160, 0
// CHECK-NEXT:   %163 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %162, ptr %161, 1
// CHECK-NEXT:   %164 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %159, %"{{.*}}/runtime/internal/runtime.eface" %163)
// CHECK-NEXT:   %165 = xor i1 %164, true
// CHECK-NEXT:   br i1 %165, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %166 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 15 }, ptr %166, align 8
// CHECK-NEXT:   %167 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %166, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %167)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %168 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.SliceOf(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %169 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %168)
// CHECK-NEXT:   %170 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %168, 0
// CHECK-NEXT:   %171 = getelementptr ptr, ptr %170, i64 11
// CHECK-NEXT:   %172 = load ptr, ptr %171, align 8
// CHECK-NEXT:   %173 = insertvalue { ptr, ptr } undef, ptr %172, 0
// CHECK-NEXT:   %174 = insertvalue { ptr, ptr } %173, ptr %169, 1
// CHECK-NEXT:   %175 = extractvalue { ptr, ptr } %174, 1
// CHECK-NEXT:   %176 = extractvalue { ptr, ptr } %174, 0
// CHECK-NEXT:   %177 = call %"{{.*}}/runtime/internal/runtime.iface" %176(ptr %175)
// CHECK-NEXT:   %178 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %177)
// CHECK-NEXT:   %179 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %177, 1
// CHECK-NEXT:   %180 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %178, 0
// CHECK-NEXT:   %181 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %180, ptr %179, 1
// CHECK-NEXT:   %182 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %183 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %184 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %182, 0
// CHECK-NEXT:   %185 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %184, ptr %183, 1
// CHECK-NEXT:   %186 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %181, %"{{.*}}/runtime/internal/runtime.eface" %185)
// CHECK-NEXT:   %187 = xor i1 %186, true
// CHECK-NEXT:   br i1 %187, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %188 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 13 }, ptr %188, align 8
// CHECK-NEXT:   %189 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %188, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %189)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %190 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 104)
// CHECK-NEXT:   %191 = getelementptr inbounds %reflect.StructField, ptr %190, i64 0
// CHECK-NEXT:   %192 = getelementptr inbounds %reflect.StructField, ptr %191, i32 0, i32 0
// CHECK-NEXT:   %193 = getelementptr inbounds %reflect.StructField, ptr %191, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 1 }, ptr %192, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %9, ptr %193, align 8
// CHECK-NEXT:   %194 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %190, 0
// CHECK-NEXT:   %195 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %194, i64 1, 1
// CHECK-NEXT:   %196 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %195, i64 1, 2
// CHECK-NEXT:   %197 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.StructOf(%"{{.*}}/runtime/internal/runtime.Slice" %196)
// CHECK-NEXT:   %198 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %197)
// CHECK-NEXT:   %199 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %197, 0
// CHECK-NEXT:   %200 = getelementptr ptr, ptr %199, i64 12
// CHECK-NEXT:   %201 = load ptr, ptr %200, align 8
// CHECK-NEXT:   %202 = insertvalue { ptr, ptr } undef, ptr %201, 0
// CHECK-NEXT:   %203 = insertvalue { ptr, ptr } %202, ptr %198, 1
// CHECK-NEXT:   %204 = extractvalue { ptr, ptr } %203, 1
// CHECK-NEXT:   %205 = extractvalue { ptr, ptr } %203, 0
// CHECK-NEXT:   %206 = call %reflect.StructField %205(ptr %204, i64 0)
// CHECK-NEXT:   %207 = extractvalue %reflect.StructField %206, 2
// CHECK-NEXT:   %208 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %207)
// CHECK-NEXT:   %209 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %207, 1
// CHECK-NEXT:   %210 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %208, 0
// CHECK-NEXT:   %211 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %210, ptr %209, 1
// CHECK-NEXT:   %212 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %213 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %214 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %212, 0
// CHECK-NEXT:   %215 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %214, ptr %213, 1
// CHECK-NEXT:   %216 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %211, %"{{.*}}/runtime/internal/runtime.eface" %215)
// CHECK-NEXT:   %217 = xor i1 %216, true
// CHECK-NEXT:   br i1 %217, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %218 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 14 }, ptr %218, align 8
// CHECK-NEXT:   %219 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %218, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %219)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %220 = alloca %reflect.Method, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %220, i8 0, i64 80, i1 false)
// CHECK-NEXT:   %221 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %222 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %223 = getelementptr ptr, ptr %222, i64 {{(23|25)}}
// CHECK-NEXT:   %224 = load ptr, ptr %223, align 8
// CHECK-NEXT:   %225 = insertvalue { ptr, ptr } undef, ptr %224, 0
// CHECK-NEXT:   %226 = insertvalue { ptr, ptr } %225, ptr %221, 1
// CHECK-NEXT:   %227 = extractvalue { ptr, ptr } %226, 1
// CHECK-NEXT:   %228 = extractvalue { ptr, ptr } %226, 0
// CHECK-NEXT:   %229 = call %reflect.Method %228(ptr %227, i64 0)
// CHECK-NEXT:   store %reflect.Method %229, ptr %220, align 8
// CHECK-NEXT:   %230 = getelementptr inbounds %reflect.Method, ptr %220, i32 0, i32 0
// CHECK-NEXT:   %231 = load %"{{.*}}/runtime/internal/runtime.String", ptr %230, align 8
// CHECK-NEXT:   %232 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %231, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %233 = xor i1 %232, true
// CHECK-NEXT:   br i1 %233, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %234 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 12 }, ptr %234, align 8
// CHECK-NEXT:   %235 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %234, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %235)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %236 = alloca %reflect.Method, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %236, i8 0, i64 80, i1 false)
// CHECK-NEXT:   %237 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %238 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %239 = getelementptr ptr, ptr %238, i64 {{(24|26)}}
// CHECK-NEXT:   %240 = load ptr, ptr %239, align 8
// CHECK-NEXT:   %241 = insertvalue { ptr, ptr } undef, ptr %240, 0
// CHECK-NEXT:   %242 = insertvalue { ptr, ptr } %241, ptr %237, 1
// CHECK-NEXT:   %243 = extractvalue { ptr, ptr } %242, 1
// CHECK-NEXT:   %244 = extractvalue { ptr, ptr } %242, 0
// CHECK-NEXT:   %245 = call { %reflect.Method, i1 } %244(ptr %243, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %246 = extractvalue { %reflect.Method, i1 } %245, 0
// CHECK-NEXT:   store %reflect.Method %246, ptr %236, align 8
// CHECK-NEXT:   %247 = extractvalue { %reflect.Method, i1 } %245, 1
// CHECK-NEXT:   br i1 %247, label %_llgo_21, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_21, %_llgo_18
// CHECK-NEXT:   %248 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 18 }, ptr %248, align 8
// CHECK-NEXT:   %249 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %248, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %249)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %250 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %251 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %250, i32 0, i32 0
// CHECK-NEXT:   %252 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %250, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %251, align 8
// CHECK-NEXT:   store i64 2, ptr %252, align 8
// CHECK-NEXT:   %253 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %250, 1
// CHECK-NEXT:   %254 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %253)
// CHECK-NEXT:   %255 = call %reflect.Value @reflect.Value.Method(%reflect.Value %254, i64 1)
// CHECK-NEXT:   %256 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %255, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %257 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %256, 0
// CHECK-NEXT:   %258 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %256, 1
// CHECK-NEXT:   %259 = icmp uge i64 0, %258
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %259, i64 0, i1 true, i64 %258)
// CHECK-NEXT:   %260 = getelementptr inbounds %reflect.Value, ptr %257, i64 0
// CHECK-NEXT:   %261 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %260)
// CHECK-NEXT:   %262 = load %reflect.Value, ptr %261, align 8
// CHECK-NEXT:   %263 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %262)
// CHECK-NEXT:   %264 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %263, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %265 = xor i1 %264, true
// CHECK-NEXT:   br i1 %265, label %_llgo_22, label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %266 = getelementptr inbounds %reflect.Method, ptr %236, i32 0, i32 0
// CHECK-NEXT:   %267 = load %"{{.*}}/runtime/internal/runtime.String", ptr %266, align 8
// CHECK-NEXT:   %268 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %267, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %269 = xor i1 %268, true
// CHECK-NEXT:   br i1 %269, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %270 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 18 }, ptr %270, align 8
// CHECK-NEXT:   %271 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %270, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %271)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %272 = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %254, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %273 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %272, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %274 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %273, 0
// CHECK-NEXT:   %275 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %273, 1
// CHECK-NEXT:   %276 = icmp uge i64 0, %275
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %276, i64 0, i1 true, i64 %275)
// CHECK-NEXT:   %277 = getelementptr inbounds %reflect.Value, ptr %274, i64 0
// CHECK-NEXT:   %278 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %277)
// CHECK-NEXT:   %279 = load %reflect.Value, ptr %278, align 8
// CHECK-NEXT:   %280 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %279)
// CHECK-NEXT:   %281 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %280, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %282 = xor i1 %281, true
// CHECK-NEXT:   br i1 %282, label %_llgo_24, label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %283 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 24 }, ptr %283, align 8
// CHECK-NEXT:   %284 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %283, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %284)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_23
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflectmk.method"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflectmk.methodByName"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.method"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   store i64 2, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %5 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = call %reflect.Value @reflect.Value.Method(%reflect.Value %5, i64 %0)
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %6, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %10 = icmp uge i64 0, %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %10, i64 0, i1 true, i64 %9)
// CHECK-NEXT:   %11 = getelementptr inbounds %reflect.Value, ptr %8, i64 0
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %11)
// CHECK-NEXT:   %13 = load %reflect.Value, ptr %12, align 8
// CHECK-NEXT:   %14 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %13)
// CHECK-NEXT:   %15 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %14, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %16 = xor i1 %15, true
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 18 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.methodByName"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   store i64 2, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %5 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %5, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %6, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %10 = icmp uge i64 0, %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %10, i64 0, i1 true, i64 %9)
// CHECK-NEXT:   %11 = getelementptr inbounds %reflect.Value, ptr %8, i64 0
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %11)
// CHECK-NEXT:   %13 = load %reflect.Value, ptr %12, align 8
// CHECK-NEXT:   %14 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %13)
// CHECK-NEXT:   %15 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %14, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %16 = xor i1 %15, true
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 24 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
