// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[PANIC_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 13 }, ptr [[PANIC_DATA]]
// CHECK-NEXT: [[PANIC_VALUE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[PANIC_DATA]], 1
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[PANIC_VALUE]])
// CHECK-NEXT: unreachable

func main() {
	panic("panic message")
}
