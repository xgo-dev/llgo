// LITTEST
package main

import (
	"fmt"
	"reflect"
)

// CHECK-LINE: @1 = private unnamed_addr constant [7 x i8] c"(%v,%v)", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [49 x i8] c"{{.*}}/cl/_testgo/reflectmk.Point", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [6 x i8] c"String", align 1
// CHECK-LINE: @12 = private unnamed_addr constant [13 x i8] c"arrayOf error", align 1
// CHECK-LINE: @13 = private unnamed_addr constant [12 x i8] c"chanOf error", align 1
// CHECK-LINE: @14 = private unnamed_addr constant [12 x i8] c"funcOf error", align 1
// CHECK-LINE: @15 = private unnamed_addr constant [11 x i8] c"mapOf error", align 1
// CHECK-LINE: @16 = private unnamed_addr constant [15 x i8] c"pointerTo error", align 1
// CHECK-LINE: @17 = private unnamed_addr constant [13 x i8] c"sliceOf error", align 1
// CHECK-LINE: @18 = private unnamed_addr constant [1 x i8] c"T", align 1
// CHECK-LINE: @19 = private unnamed_addr constant [14 x i8] c"structOf error", align 1
// CHECK-LINE: @20 = private unnamed_addr constant [12 x i8] c"method error", align 1
// CHECK-LINE: @21 = private unnamed_addr constant [18 x i8] c"methodByName error", align 1
// CHECK-LINE: @22 = private unnamed_addr constant [5 x i8] c"(1,2)", align 1
// CHECK-LINE: @23 = private unnamed_addr constant [18 x i8] c"value.Method error", align 1
// CHECK-LINE: @24 = private unnamed_addr constant [24 x i8] c"value.MethodByName error", align 1

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
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/reflectmk.Point" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %8, i64 0
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %4, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %10, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %11, ptr %9, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %8, i64 1
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %7, ptr %13, align 8
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %13, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %14, ptr %12, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %8, 0
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %15, i64 2, 1
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %16, i64 2, 2
// CHECK-NEXT:   %18 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 7 }, %"{{.*}}/runtime/internal/runtime.Slice" %17)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %18
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.(*Point).Set"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %4, align 8
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i64 %2, ptr %6, align 8
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
// CHECK-NEXT:   %67 = getelementptr ptr, ptr %66, i64 18
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
// CHECK-NEXT:   %89 = getelementptr ptr, ptr %88, i64 20
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
// CHECK-NEXT:   %108 = getelementptr ptr, ptr %107, i64 30
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
// CHECK-NEXT:   %192 = icmp eq ptr %191, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %192)
// CHECK-NEXT:   %193 = getelementptr inbounds %reflect.StructField, ptr %191, i32 0, i32 0
// CHECK-NEXT:   %194 = icmp eq ptr %191, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %194)
// CHECK-NEXT:   %195 = getelementptr inbounds %reflect.StructField, ptr %191, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 1 }, ptr %193, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %9, ptr %195, align 8
// CHECK-NEXT:   %196 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %190, 0
// CHECK-NEXT:   %197 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %196, i64 1, 1
// CHECK-NEXT:   %198 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %197, i64 1, 2
// CHECK-NEXT:   %199 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.StructOf(%"{{.*}}/runtime/internal/runtime.Slice" %198)
// CHECK-NEXT:   %200 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %199)
// CHECK-NEXT:   %201 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %199, 0
// CHECK-NEXT:   %202 = getelementptr ptr, ptr %201, i64 12
// CHECK-NEXT:   %203 = load ptr, ptr %202, align 8
// CHECK-NEXT:   %204 = insertvalue { ptr, ptr } undef, ptr %203, 0
// CHECK-NEXT:   %205 = insertvalue { ptr, ptr } %204, ptr %200, 1
// CHECK-NEXT:   %206 = extractvalue { ptr, ptr } %205, 1
// CHECK-NEXT:   %207 = extractvalue { ptr, ptr } %205, 0
// CHECK-NEXT:   %208 = call %reflect.StructField %207(ptr %206, i64 0)
// CHECK-NEXT:   %209 = extractvalue %reflect.StructField %208, 2
// CHECK-NEXT:   %210 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %209)
// CHECK-NEXT:   %211 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %209, 1
// CHECK-NEXT:   %212 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %210, 0
// CHECK-NEXT:   %213 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %212, ptr %211, 1
// CHECK-NEXT:   %214 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %215 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 1
// CHECK-NEXT:   %216 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %214, 0
// CHECK-NEXT:   %217 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %216, ptr %215, 1
// CHECK-NEXT:   %218 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %213, %"{{.*}}/runtime/internal/runtime.eface" %217)
// CHECK-NEXT:   %219 = xor i1 %218, true
// CHECK-NEXT:   br i1 %219, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %220 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 14 }, ptr %220, align 8
// CHECK-NEXT:   %221 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %220, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %221)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %222 = alloca %reflect.Method, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %222, i8 0, i64 80, i1 false)
// CHECK-NEXT:   %223 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %224 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %225 = getelementptr ptr, ptr %224, i64 23
// CHECK-NEXT:   %226 = load ptr, ptr %225, align 8
// CHECK-NEXT:   %227 = insertvalue { ptr, ptr } undef, ptr %226, 0
// CHECK-NEXT:   %228 = insertvalue { ptr, ptr } %227, ptr %223, 1
// CHECK-NEXT:   %229 = extractvalue { ptr, ptr } %228, 1
// CHECK-NEXT:   %230 = extractvalue { ptr, ptr } %228, 0
// CHECK-NEXT:   %231 = call %reflect.Method %230(ptr %229, i64 0)
// CHECK-NEXT:   store %reflect.Method %231, ptr %222, align 8
// CHECK-NEXT:   %232 = icmp eq ptr %222, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %232)
// CHECK-NEXT:   %233 = getelementptr inbounds %reflect.Method, ptr %222, i32 0, i32 0
// CHECK-NEXT:   %234 = load %"{{.*}}/runtime/internal/runtime.String", ptr %233, align 8
// CHECK-NEXT:   %235 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %234, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %236 = xor i1 %235, true
// CHECK-NEXT:   br i1 %236, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %237 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 12 }, ptr %237, align 8
// CHECK-NEXT:   %238 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %237, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %238)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %239 = alloca %reflect.Method, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %239, i8 0, i64 80, i1 false)
// CHECK-NEXT:   %240 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %241 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %242 = getelementptr ptr, ptr %241, i64 24
// CHECK-NEXT:   %243 = load ptr, ptr %242, align 8
// CHECK-NEXT:   %244 = insertvalue { ptr, ptr } undef, ptr %243, 0
// CHECK-NEXT:   %245 = insertvalue { ptr, ptr } %244, ptr %240, 1
// CHECK-NEXT:   %246 = extractvalue { ptr, ptr } %245, 1
// CHECK-NEXT:   %247 = extractvalue { ptr, ptr } %245, 0
// CHECK-NEXT:   %248 = call { %reflect.Method, i1 } %247(ptr %246, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %249 = extractvalue { %reflect.Method, i1 } %248, 0
// CHECK-NEXT:   store %reflect.Method %249, ptr %239, align 8
// CHECK-NEXT:   %250 = extractvalue { %reflect.Method, i1 } %248, 1
// CHECK-NEXT:   br i1 %250, label %_llgo_21, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_21, %_llgo_18
// CHECK-NEXT:   %251 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 18 }, ptr %251, align 8
// CHECK-NEXT:   %252 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %251, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %252)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %253 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %254 = icmp eq ptr %253, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %254)
// CHECK-NEXT:   %255 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %253, i32 0, i32 0
// CHECK-NEXT:   %256 = icmp eq ptr %253, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %256)
// CHECK-NEXT:   %257 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %253, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %255, align 8
// CHECK-NEXT:   store i64 2, ptr %257, align 8
// CHECK-NEXT:   %258 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %253, 1
// CHECK-NEXT:   %259 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %258)
// CHECK-NEXT:   %260 = call %reflect.Value @reflect.Value.Method(%reflect.Value %259, i64 1)
// CHECK-NEXT:   %261 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %260, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %262 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %261, 0
// CHECK-NEXT:   %263 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %261, 1
// CHECK-NEXT:   %264 = icmp uge i64 0, %263
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %264, i64 0, i1 true, i64 %263)
// CHECK-NEXT:   %265 = getelementptr inbounds %reflect.Value, ptr %262, i64 0
// CHECK-NEXT:   %266 = load %reflect.Value, ptr %265, align 8
// CHECK-NEXT:   %267 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %266)
// CHECK-NEXT:   %268 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %267, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %269 = xor i1 %268, true
// CHECK-NEXT:   br i1 %269, label %_llgo_22, label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %270 = icmp eq ptr %239, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %270)
// CHECK-NEXT:   %271 = getelementptr inbounds %reflect.Method, ptr %239, i32 0, i32 0
// CHECK-NEXT:   %272 = load %"{{.*}}/runtime/internal/runtime.String", ptr %271, align 8
// CHECK-NEXT:   %273 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %272, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %274 = xor i1 %273, true
// CHECK-NEXT:   br i1 %274, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %275 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 18 }, ptr %275, align 8
// CHECK-NEXT:   %276 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %275, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %276)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %277 = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %259, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 6 })
// CHECK-NEXT:   %278 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %277, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %279 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %278, 0
// CHECK-NEXT:   %280 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %278, 1
// CHECK-NEXT:   %281 = icmp uge i64 0, %280
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %281, i64 0, i1 true, i64 %280)
// CHECK-NEXT:   %282 = getelementptr inbounds %reflect.Value, ptr %279, i64 0
// CHECK-NEXT:   %283 = load %reflect.Value, ptr %282, align 8
// CHECK-NEXT:   %284 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %283)
// CHECK-NEXT:   %285 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %284, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %286 = xor i1 %285, true
// CHECK-NEXT:   br i1 %286, label %_llgo_24, label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %287 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 24 }, ptr %287, align 8
// CHECK-NEXT:   %288 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %287, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %288)
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
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %3, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   %8 = call %reflect.Value @reflect.Value.Method(%reflect.Value %7, i64 %0)
// CHECK-NEXT:   %9 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %8, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 1
// CHECK-NEXT:   %12 = icmp uge i64 0, %11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %12, i64 0, i1 true, i64 %11)
// CHECK-NEXT:   %13 = getelementptr inbounds %reflect.Value, ptr %10, i64 0
// CHECK-NEXT:   %14 = load %reflect.Value, ptr %13, align 8
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %14)
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %17 = xor i1 %16, true
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 18 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmk.methodByName"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflectmk.Point", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %3, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflectmk.Point", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   %8 = call %reflect.Value @reflect.Value.MethodByName(%reflect.Value %7, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %9 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %8, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 1
// CHECK-NEXT:   %12 = icmp uge i64 0, %11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %12, i64 0, i1 true, i64 %11)
// CHECK-NEXT:   %13 = getelementptr inbounds %reflect.Value, ptr %10, i64 0
// CHECK-NEXT:   %14 = load %reflect.Value, ptr %13, align 8
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %14)
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 5 })
// CHECK-NEXT:   %17 = xor i1 %16, true
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 24 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
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

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/reflectmk.Point.String"(ptr %0, %"{{.*}}/cl/_testgo/reflectmk.Point" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reflectmk.Point.String"(%"{{.*}}/cl/_testgo/reflectmk.Point" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/reflectmk.(*Point).Set"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/reflectmk.(*Point).Set"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/cl/_testgo/reflectmk.(*Point).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testgo/reflectmk.(*Point).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }
