// LITTEST
package main

import (
	"strings"
	"unicode"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/strings.main"(){{.*}} {
func main() {
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
