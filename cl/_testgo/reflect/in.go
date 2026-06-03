// LITTEST
package main

import (
	"reflect"
	"unsafe"
)

// CHECK-LINE: @0 = private unnamed_addr constant [11 x i8] c"call.method", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [7 x i8] c"closure", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [5 x i8] c"error", align 1
// CHECK-LINE: @9 = private unnamed_addr constant [12 x i8] c"call.closure", align 1
// CHECK-LINE: @10 = private unnamed_addr constant [4 x i8] c"func", align 1
// CHECK-LINE: @11 = private unnamed_addr constant [9 x i8] c"call.func", align 1
// CHECK-LINE: @17 = private unnamed_addr constant [7 x i8] c"imethod", align 1
// CHECK-LINE: @18 = private unnamed_addr constant [6 x i8] c"method", align 1
// CHECK-LINE: @22 = private unnamed_addr constant [10 x i8] c"call.slice", align 1
// CHECK-LINE: @23 = private unnamed_addr constant [40 x i8] c"type assertion interface{} -> int failed", align 1
// CHECK-LINE: @37 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @38 = private unnamed_addr constant [5 x i8] c"world", align 1
// CHECK-LINE: @39 = private unnamed_addr constant [14 x i8] c"MapIndex error", align 1
// CHECK-LINE: @40 = private unnamed_addr constant [4 x i8] c"todo", align 1
// CHECK-LINE: @41 = private unnamed_addr constant [12 x i8] c"must invalid", align 1
// CHECK-LINE: @42 = private unnamed_addr constant [13 x i8] c"MapIter error", align 1

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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/reflect.(*T).Add"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 11 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflect.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = add i64 %3, %1
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflect.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %4, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflect.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   ret i64 %7
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.callClosure"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/reflect.callClosure$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %6 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   %7 = call i64 @reflect.Value.Kind(%reflect.Value %6)
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %6)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %8)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %8, 0
// CHECK-NEXT:   %11 = getelementptr ptr, ptr %10, i64 37
// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } undef, ptr %12, 0
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } %13, ptr %9, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %14, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %14, 0
// CHECK-NEXT:   %17 = call %"{{.*}}/runtime/internal/runtime.String" %16(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %19 = getelementptr inbounds %reflect.Value, ptr %18, i64 0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %20, 1
// CHECK-NEXT:   %22 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   store %reflect.Value %22, ptr %19, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %18, 0
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %23, i64 1, 1
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, i64 1, 2
// CHECK-NEXT:   %26 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %6, %"{{.*}}/runtime/internal/runtime.Slice" %25)
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %26, 0
// CHECK-NEXT:   %28 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %26, 1
// CHECK-NEXT:   %29 = icmp uge i64 0, %28
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %29)
// CHECK-NEXT:   %30 = getelementptr inbounds %reflect.Value, ptr %27, i64 0
// CHECK-NEXT:   %31 = load %reflect.Value, ptr %30, align 8
// CHECK-NEXT:   %32 = call i64 @reflect.Value.Int(%reflect.Value %31)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %33 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %6)
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %33, 0
// CHECK-NEXT:   %35 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %34)
// CHECK-NEXT:   br i1 %35, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %38 = extractvalue { ptr, ptr } %46, 1
// CHECK-NEXT:   %39 = extractvalue { ptr, ptr } %46, 0
// CHECK-NEXT:   %40 = call i64 %39(ptr %38, i64 100)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %33, 1
// CHECK-NEXT:   %42 = load { ptr, ptr }, ptr %41, align 8
// CHECK-NEXT:   %43 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %42, 0
// CHECK-NEXT:   %44 = insertvalue { { ptr, ptr }, i1 } %43, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %45 = phi { { ptr, ptr }, i1 } [ %44, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %46 = extractvalue { { ptr, ptr }, i1 } %45, 0
// CHECK-NEXT:   %47 = extractvalue { { ptr, ptr }, i1 } %45, 1
// CHECK-NEXT:   br i1 %47, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/reflect.callClosure$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 12 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = add i64 %4, %1
// CHECK-NEXT:   %6 = add i64 %5, 1
// CHECK-NEXT:   ret i64 %6
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.callFunc"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflect.callFunc$1", ptr null }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %3 = call i64 @reflect.Value.Kind(%reflect.Value %2)
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %2)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 37
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.String" %12(ptr %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %15 = getelementptr inbounds %reflect.Value, ptr %14, i64 0
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %16, 1
// CHECK-NEXT:   %18 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   store %reflect.Value %18, ptr %15, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %14, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 1, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 1, 2
// CHECK-NEXT:   %22 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %2, %"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %22, 0
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %22, 1
// CHECK-NEXT:   %25 = icmp uge i64 0, %24
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %25)
// CHECK-NEXT:   %26 = getelementptr inbounds %reflect.Value, ptr %23, i64 0
// CHECK-NEXT:   %27 = load %reflect.Value, ptr %26, align 8
// CHECK-NEXT:   %28 = call i64 @reflect.Value.Int(%reflect.Value %27)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %28)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %29 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %2)
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %29, 0
// CHECK-NEXT:   %31 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %30)
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %34 = extractvalue { ptr, ptr } %42, 1
// CHECK-NEXT:   %35 = extractvalue { ptr, ptr } %42, 0
// CHECK-NEXT:   %36 = call i64 %35(ptr %34, i64 100)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %29, 1
// CHECK-NEXT:   %38 = load { ptr, ptr }, ptr %37, align 8
// CHECK-NEXT:   %39 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %38, 0
// CHECK-NEXT:   %40 = insertvalue { { ptr, ptr }, i1 } %39, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %41 = phi { { ptr, ptr }, i1 } [ %40, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %42 = extractvalue { { ptr, ptr }, i1 } %41, 0
// CHECK-NEXT:   %43 = extractvalue { { ptr, ptr }, i1 } %41, 1
// CHECK-NEXT:   br i1 %43, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/reflect.callFunc$1"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %1 = add i64 %0, 1
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.callIMethod"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflect.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$VdBKYV8-gcMjZtZfcf-u2oKoj9Lu3VXwuG8TGCW2S4A", ptr @"*_llgo_{{.*}}/cl/_testgo/reflect.T")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 1
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %5, 0
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %7, ptr %6, 1
// CHECK-NEXT:   %9 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   %10 = call %reflect.Value @reflect.Value.Method(%reflect.Value %9, i64 0)
// CHECK-NEXT:   %11 = call i64 @reflect.Value.Kind(%reflect.Value %10)
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %10)
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %12)
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %12, 0
// CHECK-NEXT:   %15 = getelementptr ptr, ptr %14, i64 37
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } undef, ptr %16, 0
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } %17, ptr %13, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.String" %20(ptr %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %23 = getelementptr inbounds %reflect.Value, ptr %22, i64 0
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %24, 1
// CHECK-NEXT:   %26 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %25)
// CHECK-NEXT:   store %reflect.Value %26, ptr %23, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %22, 0
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %27, i64 1, 1
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %28, i64 1, 2
// CHECK-NEXT:   %30 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %10, %"{{.*}}/runtime/internal/runtime.Slice" %29)
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, 0
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, 1
// CHECK-NEXT:   %33 = icmp uge i64 0, %32
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %33)
// CHECK-NEXT:   %34 = getelementptr inbounds %reflect.Value, ptr %31, i64 0
// CHECK-NEXT:   %35 = load %reflect.Value, ptr %34, align 8
// CHECK-NEXT:   %36 = call i64 @reflect.Value.Int(%reflect.Value %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %36)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %37 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %10)
// CHECK-NEXT:   %38 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 0
// CHECK-NEXT:   %39 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %38)
// CHECK-NEXT:   br i1 %39, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %40, align 8
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %40, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %41)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %42 = extractvalue { ptr, ptr } %67, 1
// CHECK-NEXT:   %43 = extractvalue { ptr, ptr } %67, 0
// CHECK-NEXT:   %44 = call i64 %43(ptr %42, i64 1)
// CHECK-NEXT:   %45 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %10)
// CHECK-NEXT:   %46 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %48 = getelementptr inbounds %reflect.Value, ptr %47, i64 0
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %49, align 8
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %49, 1
// CHECK-NEXT:   %51 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %50)
// CHECK-NEXT:   store %reflect.Value %51, ptr %48, align 8
// CHECK-NEXT:   %52 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %47, 0
// CHECK-NEXT:   %53 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, i64 1, 1
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %53, i64 1, 2
// CHECK-NEXT:   %55 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %46, %"{{.*}}/runtime/internal/runtime.Slice" %54)
// CHECK-NEXT:   %56 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %55, 0
// CHECK-NEXT:   %57 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %55, 1
// CHECK-NEXT:   %58 = icmp uge i64 0, %57
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %58)
// CHECK-NEXT:   %59 = getelementptr inbounds %reflect.Value, ptr %56, i64 0
// CHECK-NEXT:   %60 = load %reflect.Value, ptr %59, align 8
// CHECK-NEXT:   %61 = call i64 @reflect.Value.Int(%reflect.Value %60)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %61)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 1
// CHECK-NEXT:   %63 = load { ptr, ptr }, ptr %62, align 8
// CHECK-NEXT:   %64 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %63, 0
// CHECK-NEXT:   %65 = insertvalue { { ptr, ptr }, i1 } %64, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %66 = phi { { ptr, ptr }, i1 } [ %65, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %67 = extractvalue { { ptr, ptr }, i1 } %66, 0
// CHECK-NEXT:   %68 = extractvalue { { ptr, ptr }, i1 } %66, 1
// CHECK-NEXT:   br i1 %68, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.callMethod"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/reflect.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/reflect.T", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %4 = call %reflect.Value @reflect.Value.Method(%reflect.Value %3, i64 0)
// CHECK-NEXT:   %5 = call i64 @reflect.Value.Kind(%reflect.Value %4)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %4)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 37
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" %14(ptr %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %17 = getelementptr inbounds %reflect.Value, ptr %16, i64 0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %18, 1
// CHECK-NEXT:   %20 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   store %reflect.Value %20, ptr %17, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %16, 0
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %21, i64 1, 1
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %22, i64 1, 2
// CHECK-NEXT:   %24 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %4, %"{{.*}}/runtime/internal/runtime.Slice" %23)
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, 0
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, 1
// CHECK-NEXT:   %27 = icmp uge i64 0, %26
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %reflect.Value, ptr %25, i64 0
// CHECK-NEXT:   %29 = load %reflect.Value, ptr %28, align 8
// CHECK-NEXT:   %30 = call i64 @reflect.Value.Int(%reflect.Value %29)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %30)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %31 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %4)
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %31, 0
// CHECK-NEXT:   %33 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %32)
// CHECK-NEXT:   br i1 %33, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %36 = extractvalue { ptr, ptr } %61, 1
// CHECK-NEXT:   %37 = extractvalue { ptr, ptr } %61, 0
// CHECK-NEXT:   %38 = call i64 %37(ptr %36, i64 1)
// CHECK-NEXT:   %39 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %4)
// CHECK-NEXT:   %40 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %39)
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %42 = getelementptr inbounds %reflect.Value, ptr %41, i64 0
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %43, align 8
// CHECK-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %43, 1
// CHECK-NEXT:   %45 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %44)
// CHECK-NEXT:   store %reflect.Value %45, ptr %42, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %41, 0
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %46, i64 1, 1
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %47, i64 1, 2
// CHECK-NEXT:   %49 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %40, %"{{.*}}/runtime/internal/runtime.Slice" %48)
// CHECK-NEXT:   %50 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, 0
// CHECK-NEXT:   %51 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, 1
// CHECK-NEXT:   %52 = icmp uge i64 0, %51
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %52)
// CHECK-NEXT:   %53 = getelementptr inbounds %reflect.Value, ptr %50, i64 0
// CHECK-NEXT:   %54 = load %reflect.Value, ptr %53, align 8
// CHECK-NEXT:   %55 = call i64 @reflect.Value.Int(%reflect.Value %54)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %56 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %31, 1
// CHECK-NEXT:   %57 = load { ptr, ptr }, ptr %56, align 8
// CHECK-NEXT:   %58 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %57, 0
// CHECK-NEXT:   %59 = insertvalue { { ptr, ptr }, i1 } %58, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %60 = phi { { ptr, ptr }, i1 } [ %59, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %61 = extractvalue { { ptr, ptr }, i1 } %60, 0
// CHECK-NEXT:   %62 = extractvalue { { ptr, ptr }, i1 } %60, 1
// CHECK-NEXT:   br i1 %62, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.callSlice"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflect.demo", ptr null }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$FjMjjQr3-2iTiWyZP1IIQFOz0hUCa0OS6pEm5uVV6Pk", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 288)
// CHECK-NEXT:   %7 = getelementptr inbounds %reflect.Value, ptr %6, i64 0
// CHECK-NEXT:   store %reflect.Value %5, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %reflect.Value, ptr %6, i64 1
// CHECK-NEXT:   store %reflect.Value %5, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %reflect.Value, ptr %6, i64 2
// CHECK-NEXT:   store %reflect.Value %5, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %reflect.Value, ptr %6, i64 3
// CHECK-NEXT:   store %reflect.Value %5, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %reflect.Value, ptr %6, i64 4
// CHECK-NEXT:   store %reflect.Value %5, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %reflect.Value, ptr %6, i64 5
// CHECK-NEXT:   store %reflect.Value %5, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %reflect.Value, ptr %6, i64 6
// CHECK-NEXT:   store %reflect.Value %5, ptr %13, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %reflect.Value, ptr %6, i64 7
// CHECK-NEXT:   store %reflect.Value %5, ptr %14, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %reflect.Value, ptr %6, i64 8
// CHECK-NEXT:   store %reflect.Value %5, ptr %15, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %reflect.Value, ptr %6, i64 9
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %17, 1
// CHECK-NEXT:   %19 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   store %reflect.Value %19, ptr %16, align 8
// CHECK-NEXT:   %20 = getelementptr inbounds %reflect.Value, ptr %6, i64 10
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %21, align 8
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %21, 1
// CHECK-NEXT:   %23 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %22)
// CHECK-NEXT:   store %reflect.Value %23, ptr %20, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds %reflect.Value, ptr %6, i64 11
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 3, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %25, 1
// CHECK-NEXT:   %27 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %26)
// CHECK-NEXT:   store %reflect.Value %27, ptr %24, align 8
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %6, 0
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %28, i64 12, 1
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %29, i64 12, 2
// CHECK-NEXT:   %31 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.Call(%reflect.Value %2, %"{{.*}}/runtime/internal/runtime.Slice" %30)
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 0
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 1
// CHECK-NEXT:   %34 = icmp uge i64 0, %33
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds %reflect.Value, ptr %32, i64 0
// CHECK-NEXT:   %36 = load %reflect.Value, ptr %35, align 8
// CHECK-NEXT:   %37 = call i64 @reflect.Value.Int(%reflect.Value %36)
// CHECK-NEXT:   %38 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 0
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 1
// CHECK-NEXT:   %40 = icmp uge i64 1, %39
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %40)
// CHECK-NEXT:   %41 = getelementptr inbounds %reflect.Value, ptr %38, i64 1
// CHECK-NEXT:   %42 = load %reflect.Value, ptr %41, align 8
// CHECK-NEXT:   %43 = call i64 @reflect.Value.Int(%reflect.Value %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %37)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %43)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 240)
// CHECK-NEXT:   %45 = getelementptr inbounds %reflect.Value, ptr %44, i64 0
// CHECK-NEXT:   store %reflect.Value %5, ptr %45, align 8
// CHECK-NEXT:   %46 = getelementptr inbounds %reflect.Value, ptr %44, i64 1
// CHECK-NEXT:   store %reflect.Value %5, ptr %46, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds %reflect.Value, ptr %44, i64 2
// CHECK-NEXT:   store %reflect.Value %5, ptr %47, align 8
// CHECK-NEXT:   %48 = getelementptr inbounds %reflect.Value, ptr %44, i64 3
// CHECK-NEXT:   store %reflect.Value %5, ptr %48, align 8
// CHECK-NEXT:   %49 = getelementptr inbounds %reflect.Value, ptr %44, i64 4
// CHECK-NEXT:   store %reflect.Value %5, ptr %49, align 8
// CHECK-NEXT:   %50 = getelementptr inbounds %reflect.Value, ptr %44, i64 5
// CHECK-NEXT:   store %reflect.Value %5, ptr %50, align 8
// CHECK-NEXT:   %51 = getelementptr inbounds %reflect.Value, ptr %44, i64 6
// CHECK-NEXT:   store %reflect.Value %5, ptr %51, align 8
// CHECK-NEXT:   %52 = getelementptr inbounds %reflect.Value, ptr %44, i64 7
// CHECK-NEXT:   store %reflect.Value %5, ptr %52, align 8
// CHECK-NEXT:   %53 = getelementptr inbounds %reflect.Value, ptr %44, i64 8
// CHECK-NEXT:   store %reflect.Value %5, ptr %53, align 8
// CHECK-NEXT:   %54 = getelementptr inbounds %reflect.Value, ptr %44, i64 9
// CHECK-NEXT:   %55 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 48)
// CHECK-NEXT:   %56 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 0
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %57, align 8
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %57, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %58, ptr %56, align 8
// CHECK-NEXT:   %59 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 1
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %60, align 8
// CHECK-NEXT:   %61 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %60, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %61, ptr %59, align 8
// CHECK-NEXT:   %62 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 2
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 3, ptr %63, align 8
// CHECK-NEXT:   %64 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %63, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %64, ptr %62, align 8
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %55, 0
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %65, i64 3, 1
// CHECK-NEXT:   %67 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, i64 3, 2
// CHECK-NEXT:   %68 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %67, ptr %68, align 8
// CHECK-NEXT:   %69 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_any", ptr undef }, ptr %68, 1
// CHECK-NEXT:   %70 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %69)
// CHECK-NEXT:   store %reflect.Value %70, ptr %54, align 8
// CHECK-NEXT:   %71 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %44, 0
// CHECK-NEXT:   %72 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %71, i64 10, 1
// CHECK-NEXT:   %73 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %72, i64 10, 2
// CHECK-NEXT:   %74 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.CallSlice(%reflect.Value %2, %"{{.*}}/runtime/internal/runtime.Slice" %73)
// CHECK-NEXT:   %75 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %74, 0
// CHECK-NEXT:   %76 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %74, 1
// CHECK-NEXT:   %77 = icmp uge i64 0, %76
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %77)
// CHECK-NEXT:   %78 = getelementptr inbounds %reflect.Value, ptr %75, i64 0
// CHECK-NEXT:   %79 = load %reflect.Value, ptr %78, align 8
// CHECK-NEXT:   %80 = call i64 @reflect.Value.Int(%reflect.Value %79)
// CHECK-NEXT:   %81 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %74, 0
// CHECK-NEXT:   %82 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %74, 1
// CHECK-NEXT:   %83 = icmp uge i64 1, %82
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %83)
// CHECK-NEXT:   %84 = getelementptr inbounds %reflect.Value, ptr %81, i64 1
// CHECK-NEXT:   %85 = load %reflect.Value, ptr %84, align 8
// CHECK-NEXT:   %86 = call i64 @reflect.Value.Int(%reflect.Value %85)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %80)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %86)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, i64 } @"{{.*}}/cl/_testgo/reflect.demo"(i64 %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, %"{{.*}}/runtime/internal/runtime.Slice" %9){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %11 = phi i64 [ 0, %_llgo_0 ], [ %36, %_llgo_4 ]
// CHECK-NEXT:   %12 = phi i64 [ -1, %_llgo_0 ], [ %13, %_llgo_4 ]
// CHECK-NEXT:   %13 = add i64 %12, 1
// CHECK-NEXT:   %14 = icmp slt i64 %13, %10
// CHECK-NEXT:   br i1 %14, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 0
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 1
// CHECK-NEXT:   %17 = icmp slt i64 %13, 0
// CHECK-NEXT:   %18 = icmp uge i64 %13, %16
// CHECK-NEXT:   %19 = or i1 %18, %17
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %19)
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %15, i64 %13
// CHECK-NEXT:   %21 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %20, align 8
// CHECK-NEXT:   %22 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %21, 0
// CHECK-NEXT:   %23 = icmp eq ptr %22, @_llgo_int
// CHECK-NEXT:   br i1 %23, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %24 = add i64 %0, %1
// CHECK-NEXT:   %25 = add i64 %24, %2
// CHECK-NEXT:   %26 = add i64 %25, %3
// CHECK-NEXT:   %27 = add i64 %26, %4
// CHECK-NEXT:   %28 = add i64 %27, %5
// CHECK-NEXT:   %29 = add i64 %28, %6
// CHECK-NEXT:   %30 = add i64 %29, %7
// CHECK-NEXT:   %31 = add i64 %30, %8
// CHECK-NEXT:   %32 = insertvalue { i64, i64 } undef, i64 %31, 0
// CHECK-NEXT:   %33 = insertvalue { i64, i64 } %32, i64 %11, 1
// CHECK-NEXT:   ret { i64, i64 } %33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %21, 1
// CHECK-NEXT:   %35 = load i64, ptr %34, align 8
// CHECK-NEXT:   %36 = add i64 %11, %35
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 40 }, ptr %37, align 8
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %37, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %38)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/reflect.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/reflect.init$guard", align 1
// CHECK-NEXT:   call void @reflect.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.callSlice"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.callFunc"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.callClosure"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.callMethod"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.callIMethod"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.mapDemo1"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/reflect.mapDemo2"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.mapDemo1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 2)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %1)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @37, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %3)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_string", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %6 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   %7 = call i64 @reflect.Value.Len(%reflect.Value %6)
// CHECK-NEXT:   %8 = icmp ne i64 %7, 2
// CHECK-NEXT:   br i1 %8, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %9, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %10)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %11, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %11, 1
// CHECK-NEXT:   %13 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %12)
// CHECK-NEXT:   %14 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %6, %reflect.Value %13)
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %14)
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 5 })
// CHECK-NEXT:   %17 = xor i1 %16, true
// CHECK-NEXT:   br i1 %17, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.MapKeys(%reflect.Value %6)
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %18, 1
// CHECK-NEXT:   %20 = icmp ne i64 %19, 2
// CHECK-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr %21, align 8
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %21, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %22)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %23, 1
// CHECK-NEXT:   %25 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %24)
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 4 }, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   %28 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   call void @reflect.Value.SetMapIndex(%reflect.Value %6, %reflect.Value %25, %reflect.Value %28)
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %29, align 8
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %29, 1
// CHECK-NEXT:   %31 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %30)
// CHECK-NEXT:   %32 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %6, %reflect.Value %31)
// CHECK-NEXT:   %33 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %32)
// CHECK-NEXT:   %34 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %33, %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 4 })
// CHECK-NEXT:   %35 = xor i1 %34, true
// CHECK-NEXT:   br i1 %35, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %38, 1
// CHECK-NEXT:   %40 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %39)
// CHECK-NEXT:   %41 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %6, %reflect.Value %40)
// CHECK-NEXT:   %42 = call i1 @reflect.Value.IsValid(%reflect.Value %41)
// CHECK-NEXT:   br i1 %42, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 12 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8, %_llgo_7
// CHECK-NEXT:   %43 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %6)
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %43)
// CHECK-NEXT:   %45 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %43, 0
// CHECK-NEXT:   %46 = getelementptr ptr, ptr %45, i64 20
// CHECK-NEXT:   %47 = load ptr, ptr %46, align 8
// CHECK-NEXT:   %48 = insertvalue { ptr, ptr } undef, ptr %47, 0
// CHECK-NEXT:   %49 = insertvalue { ptr, ptr } %48, ptr %44, 1
// CHECK-NEXT:   %50 = extractvalue { ptr, ptr } %49, 1
// CHECK-NEXT:   %51 = extractvalue { ptr, ptr } %49, 0
// CHECK-NEXT:   %52 = call %"{{.*}}/runtime/internal/runtime.iface" %51(ptr %50)
// CHECK-NEXT:   %53 = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" %52)
// CHECK-NEXT:   %54 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %53)
// CHECK-NEXT:   %55 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %6)
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %55)
// CHECK-NEXT:   %57 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %55, 0
// CHECK-NEXT:   %58 = getelementptr ptr, ptr %57, i64 11
// CHECK-NEXT:   %59 = load ptr, ptr %58, align 8
// CHECK-NEXT:   %60 = insertvalue { ptr, ptr } undef, ptr %59, 0
// CHECK-NEXT:   %61 = insertvalue { ptr, ptr } %60, ptr %56, 1
// CHECK-NEXT:   %62 = extractvalue { ptr, ptr } %61, 1
// CHECK-NEXT:   %63 = extractvalue { ptr, ptr } %61, 0
// CHECK-NEXT:   %64 = call %"{{.*}}/runtime/internal/runtime.iface" %63(ptr %62)
// CHECK-NEXT:   %65 = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" %64)
// CHECK-NEXT:   %66 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %65)
// CHECK-NEXT:   %67 = call ptr @reflect.Value.MapRange(%reflect.Value %6)
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_12
// CHECK-NEXT:   call void @reflect.Value.SetIterKey(%reflect.Value %54, ptr %67)
// CHECK-NEXT:   call void @reflect.Value.SetIterValue(%reflect.Value %66, ptr %67)
// CHECK-NEXT:   %68 = call i64 @reflect.Value.Int(%reflect.Value %54)
// CHECK-NEXT:   %69 = call %reflect.Value @"reflect.(*MapIter).Key"(ptr %67)
// CHECK-NEXT:   %70 = call i64 @reflect.Value.Int(%reflect.Value %69)
// CHECK-NEXT:   %71 = icmp ne i64 %68, %70
// CHECK-NEXT:   br i1 %71, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_12
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_14, %_llgo_9
// CHECK-NEXT:   %72 = call i1 @"reflect.(*MapIter).Next"(ptr %67)
// CHECK-NEXT:   br i1 %72, label %_llgo_10, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_14, %_llgo_10
// CHECK-NEXT:   %73 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 13 }, ptr %73, align 8
// CHECK-NEXT:   %74 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %73, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %74)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %75 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %66)
// CHECK-NEXT:   %76 = call %reflect.Value @"reflect.(*MapIter).Value"(ptr %67)
// CHECK-NEXT:   %77 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %76)
// CHECK-NEXT:   %78 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %75, %"{{.*}}/runtime/internal/runtime.String" %77)
// CHECK-NEXT:   %79 = xor i1 %78, true
// CHECK-NEXT:   br i1 %79, label %_llgo_13, label %_llgo_12
// CHECK-NEXT: }

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflect.mapDemo2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.MapOf(%"{{.*}}/runtime/internal/runtime.iface" %2, %"{{.*}}/runtime/internal/runtime.iface" %5)
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.MakeMap(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %8, 1
// CHECK-NEXT:   %10 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @37, i64 5 }, ptr %11, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %11, 1
// CHECK-NEXT:   %13 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %12)
// CHECK-NEXT:   call void @reflect.Value.SetMapIndex(%reflect.Value %7, %reflect.Value %10, %reflect.Value %13)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %14, 1
// CHECK-NEXT:   %16 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 5 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   %19 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   call void @reflect.Value.SetMapIndex(%reflect.Value %7, %reflect.Value %16, %reflect.Value %19)
// CHECK-NEXT:   %20 = call i64 @reflect.Value.Len(%reflect.Value %7)
// CHECK-NEXT:   %21 = icmp ne i64 %20, 2
// CHECK-NEXT:   br i1 %21, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %22, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %24, 1
// CHECK-NEXT:   %26 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %25)
// CHECK-NEXT:   %27 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %7, %reflect.Value %26)
// CHECK-NEXT:   %28 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %27)
// CHECK-NEXT:   %29 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %28, %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 5 })
// CHECK-NEXT:   %30 = xor i1 %29, true
// CHECK-NEXT:   br i1 %30, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %31 = call %"{{.*}}/runtime/internal/runtime.Slice" @reflect.Value.MapKeys(%reflect.Value %7)
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 1
// CHECK-NEXT:   %33 = icmp ne i64 %32, 2
// CHECK-NEXT:   br i1 %33, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %36, 1
// CHECK-NEXT:   %38 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 4 }, ptr %39, align 8
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %39, 1
// CHECK-NEXT:   %41 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %40)
// CHECK-NEXT:   call void @reflect.Value.SetMapIndex(%reflect.Value %7, %reflect.Value %38, %reflect.Value %41)
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %42, align 8
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %42, 1
// CHECK-NEXT:   %44 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %43)
// CHECK-NEXT:   %45 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %7, %reflect.Value %44)
// CHECK-NEXT:   %46 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %45)
// CHECK-NEXT:   %47 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %46, %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 4 })
// CHECK-NEXT:   %48 = xor i1 %47, true
// CHECK-NEXT:   br i1 %48, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr %49, align 8
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %49, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %50)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %51 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %51, align 8
// CHECK-NEXT:   %52 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %51, 1
// CHECK-NEXT:   %53 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %52)
// CHECK-NEXT:   %54 = call %reflect.Value @reflect.Value.MapIndex(%reflect.Value %7, %reflect.Value %53)
// CHECK-NEXT:   %55 = call i1 @reflect.Value.IsValid(%reflect.Value %54)
// CHECK-NEXT:   br i1 %55, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 12 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8, %_llgo_7
// CHECK-NEXT:   %56 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %7)
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %56)
// CHECK-NEXT:   %58 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %56, 0
// CHECK-NEXT:   %59 = getelementptr ptr, ptr %58, i64 20
// CHECK-NEXT:   %60 = load ptr, ptr %59, align 8
// CHECK-NEXT:   %61 = insertvalue { ptr, ptr } undef, ptr %60, 0
// CHECK-NEXT:   %62 = insertvalue { ptr, ptr } %61, ptr %57, 1
// CHECK-NEXT:   %63 = extractvalue { ptr, ptr } %62, 1
// CHECK-NEXT:   %64 = extractvalue { ptr, ptr } %62, 0
// CHECK-NEXT:   %65 = call %"{{.*}}/runtime/internal/runtime.iface" %64(ptr %63)
// CHECK-NEXT:   %66 = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" %65)
// CHECK-NEXT:   %67 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %66)
// CHECK-NEXT:   %68 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %7)
// CHECK-NEXT:   %69 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %68)
// CHECK-NEXT:   %70 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %68, 0
// CHECK-NEXT:   %71 = getelementptr ptr, ptr %70, i64 11
// CHECK-NEXT:   %72 = load ptr, ptr %71, align 8
// CHECK-NEXT:   %73 = insertvalue { ptr, ptr } undef, ptr %72, 0
// CHECK-NEXT:   %74 = insertvalue { ptr, ptr } %73, ptr %69, 1
// CHECK-NEXT:   %75 = extractvalue { ptr, ptr } %74, 1
// CHECK-NEXT:   %76 = extractvalue { ptr, ptr } %74, 0
// CHECK-NEXT:   %77 = call %"{{.*}}/runtime/internal/runtime.iface" %76(ptr %75)
// CHECK-NEXT:   %78 = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" %77)
// CHECK-NEXT:   %79 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %78)
// CHECK-NEXT:   %80 = call ptr @reflect.Value.MapRange(%reflect.Value %7)
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_12
// CHECK-NEXT:   call void @reflect.Value.SetIterKey(%reflect.Value %67, ptr %80)
// CHECK-NEXT:   call void @reflect.Value.SetIterValue(%reflect.Value %79, ptr %80)
// CHECK-NEXT:   %81 = call i64 @reflect.Value.Int(%reflect.Value %67)
// CHECK-NEXT:   %82 = call %reflect.Value @"reflect.(*MapIter).Key"(ptr %80)
// CHECK-NEXT:   %83 = call i64 @reflect.Value.Int(%reflect.Value %82)
// CHECK-NEXT:   %84 = icmp ne i64 %81, %83
// CHECK-NEXT:   br i1 %84, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_12
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_14, %_llgo_9
// CHECK-NEXT:   %85 = call i1 @"reflect.(*MapIter).Next"(ptr %80)
// CHECK-NEXT:   br i1 %85, label %_llgo_10, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_14, %_llgo_10
// CHECK-NEXT:   %86 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 13 }, ptr %86, align 8
// CHECK-NEXT:   %87 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %86, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %87)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %88 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %79)
// CHECK-NEXT:   %89 = call %reflect.Value @"reflect.(*MapIter).Value"(ptr %80)
// CHECK-NEXT:   %90 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %89)
// CHECK-NEXT:   %91 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %88, %"{{.*}}/runtime/internal/runtime.String" %90)
// CHECK-NEXT:   %92 = xor i1 %91, true
// CHECK-NEXT:   br i1 %92, label %_llgo_13, label %_llgo_12
// CHECK-NEXT: }

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

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/reflect.callFunc$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/reflect.callFunc$1"(i64 %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce { i64, i64 } @"__llgo_stub.{{.*}}/cl/_testgo/reflect.demo"(ptr %0, i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9, %"{{.*}}/runtime/internal/runtime.Slice" %10){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %11 = tail call { i64, i64 } @"{{.*}}/cl/_testgo/reflect.demo"(i64 %1, i64 %2, i64 %3, i64 %4, i64 %5, i64 %6, i64 %7, i64 %8, i64 %9, %"{{.*}}/runtime/internal/runtime.Slice" %10)
// CHECK-NEXT:   ret { i64, i64 } %11
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
