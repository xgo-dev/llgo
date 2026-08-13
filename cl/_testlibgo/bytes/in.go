// LITTEST
package main

import (
	"bytes"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[BUFFER:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 40)
// CHECK-NEXT: [[HELLO:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}StringToBytes"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT: call { i64, %"{{.*}}iface" } @"bytes.(*Buffer).Write"(ptr [[BUFFER]], %"{{.*}}Slice" [[HELLO]])
// CHECK-NEXT: call { i64, %"{{.*}}iface" } @"bytes.(*Buffer).WriteString"(ptr [[BUFFER]], %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[BYTES:%[0-9]+]] = call %"{{.*}}Slice" @"bytes.(*Buffer).Bytes"(ptr [[BUFFER]])
// CHECK-NEXT: [[STRING:%[0-9]+]] = call %"{{.*}}String" @"bytes.(*Buffer).String"(ptr [[BUFFER]])
// CHECK: call void @"{{.*}}PrintSlice"(%"{{.*}}Slice" [[BYTES]])
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" [[STRING]])
// CHECK: [[UPPER:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}StringToBytes"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT: [[LOWER:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}StringToBytes"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT: [[EQUAL:%[0-9]+]] = call i1 @bytes.EqualFold(%"{{.*}}Slice" [[UPPER]], %"{{.*}}Slice" [[LOWER]])
// CHECK-NEXT: call void @"{{.*}}PrintBool"(i1 [[EQUAL]])
func main() {
	var b bytes.Buffer // A Buffer needs no initialization.
	b.Write([]byte("Hello "))
	b.WriteString("World")

	println("buf", b.Bytes(), b.String())

	println(bytes.EqualFold([]byte("Go"), []byte("go")))
}
