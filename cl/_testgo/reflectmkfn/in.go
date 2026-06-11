// LITTEST
package main

import (
	"reflect"
	"strings"
)

// CHECK-LINE: @6 = private unnamed_addr constant [57 x i8] c"struct{$f func(string, int) string; $data unsafe.Pointer}", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [3 x i8] c"abc", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [6 x i8] c"abcabc", align 1
// CHECK-LINE: @9 = private unnamed_addr constant [5 x i8] c"error", align 1

func main() {
	typ := reflect.FuncOf([]reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf("")}, false)
	fn := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		r := strings.Repeat(args[0].String(), int(args[1].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	r := fn.Interface().(func(string, int) string)("abc", 2)
	if r != "abcabc" {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmkfn.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/reflectmkfn.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/reflectmkfn.init$guard", align 1
// CHECK-NEXT:   call void @reflect.init()
// CHECK-NEXT:   call void @strings.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectmkfn.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %0, i64 0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2, 1
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %4, ptr %1, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %0, i64 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %6, 1
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %5, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, i64 2, 1
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, i64 2, 2
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %12, i64 0
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %14, 1
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %16, ptr %13, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %12, 0
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, i64 1, 1
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %18, i64 1, 2
// CHECK-NEXT:   %20 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.FuncOf(%"{{.*}}/runtime/internal/runtime.Slice" %11, %"{{.*}}/runtime/internal/runtime.Slice" %19, i1 false)
// CHECK-NEXT:   %21 = call %reflect.Value @reflect.MakeFunc(%"{{.*}}/runtime/internal/runtime.iface" %20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflectmkfn.main$1", ptr null })
// CHECK-NEXT:   %22 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %21)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 0
// CHECK-NEXT:   %24 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$XBbb2Vd9fa-WWUcWFPjreitD8Eex4qtMIsPbz__3VQU", ptr %23)
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %25, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %26)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 1
// CHECK-NEXT:   %28 = load { ptr, ptr }, ptr %27, align 8
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %28, 1
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %28, 0
// CHECK-NEXT:   %31 = call %"{{.*}}/runtime/internal/runtime.String" %30(ptr %29, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 3 }, i64 2)
// CHECK-NEXT:   %32 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %31, %"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 6 })
// CHECK-NEXT:   %33 = xor i1 %32, true
// CHECK-NEXT:   br i1 %33, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %23, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 57 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/reflectmkfn.main$1"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %3 = icmp uge i64 0, %2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %3, i64 0, i1 true, i64 %2)
// CHECK-NEXT:   %4 = getelementptr inbounds %reflect.Value, ptr %1, i64 0
// CHECK-NEXT:   %5 = load %reflect.Value, ptr %4, align 8
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value %5)
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %9 = icmp uge i64 1, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %9, i64 1, i1 true, i64 %8)
// CHECK-NEXT:   %10 = getelementptr inbounds %reflect.Value, ptr %7, i64 1
// CHECK-NEXT:   %11 = load %reflect.Value, ptr %10, align 8
// CHECK-NEXT:   %12 = call i64 @reflect.Value.Int(%reflect.Value %11)
// CHECK-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.String" @strings.Repeat(%"{{.*}}/runtime/internal/runtime.String" %6, i64 %12)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %15 = getelementptr inbounds %reflect.Value, ptr %14, i64 0
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %13, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %16, 1
// CHECK-NEXT:   %18 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   store %reflect.Value %18, ptr %15, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %14, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 1, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 1, 2
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %21
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/cl/_testgo/reflectmkfn.main$1"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/reflectmkfn.main$1"(%"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }
