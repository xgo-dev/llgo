// LITTEST
package main

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/makemaphint.fromInt32"(i32 %0){{.*}} {
// CHECK: %1 = sext i32 %0 to i64
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_int", i64 %1)
func fromInt32(n int32) int {
	return len(make(map[string]int, n))
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/makemaphint.fromUint32"(i32 %0){{.*}} {
// CHECK: %1 = zext i32 %0 to i64
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_int", i64 %1)
func fromUint32(n uint32) int {
	return len(make(map[string]int, n))
}

func main() {
	println(fromUint32(2))
	println(fromInt32(3))
}
