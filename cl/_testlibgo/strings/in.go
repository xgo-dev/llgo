// LITTEST
package main

import (
	"strings"
	"unicode"
)

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"Hello ", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [5 x i8] c"World", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [4 x i8] c"len:", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [4 x i8] c"cap:", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [7 x i8] c"string:", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [13 x i8] c"Hello, \E4\B8\96\E7\95\8C", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [12 x i8] c"Hello, world", align 1

func main() {
	var b strings.Builder
	b.Write([]byte("Hello "))
	b.WriteString("World")

	println("len:", b.Len(), "cap:", b.Cap(), "string:", b.String())

	f := func(c rune) bool {
		return unicode.Is(unicode.Han, c)
	}
	println(strings.IndexFunc("Hello, 世界", f))
	println(strings.IndexFunc("Hello, world", f))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/strings.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testlibgo/strings.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testlibgo/strings.init$guard", align 1
// CHECK-NEXT:   call void @strings.init()
// CHECK-NEXT:   call void @unicode.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/strings.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   %2 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"strings.(*Builder).Write"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %3 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"strings.(*Builder).WriteString"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 5 })
// CHECK-NEXT:   %4 = call i64 @"strings.(*Builder).Len"(ptr %0)
// CHECK-NEXT:   %5 = call i64 @"strings.(*Builder).Cap"(ptr %0)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @"strings.(*Builder).String"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %7 = call i64 @strings.IndexFunc(%"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 13 }, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testlibgo/strings.main$1", ptr null })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = call i64 @strings.IndexFunc(%"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 12 }, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testlibgo/strings.main$1", ptr null })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"{{.*}}/cl/_testlibgo/strings.main$1"(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load ptr, ptr @unicode.Han, align 8
// CHECK-NEXT:   %2 = call i1 @unicode.Is(ptr %1, i32 %0)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/cl/_testlibgo/strings.main$1"(ptr %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/cl/_testlibgo/strings.main$1"(i32 %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }
