// LITTEST
package main

import (
	"bytes"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/bytes.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   %2 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %3 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 5 })
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %0)
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 2 })
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 2 })
// CHECK-NEXT:   %8 = call i1 @bytes.EqualFold(%"{{.*}}/runtime/internal/runtime.Slice" %6, %"{{.*}}/runtime/internal/runtime.Slice" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	var b bytes.Buffer // A Buffer needs no initialization.
	b.Write([]byte("Hello "))
	b.WriteString("World")

	println("buf", b.Bytes(), b.String())

	println(bytes.EqualFold([]byte("Go"), []byte("go")))
}
