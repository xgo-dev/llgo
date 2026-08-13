// LITTEST
package main

// CHECK-LABEL: define i64 @main.fromInt32(i32 %0){{.*}} {
// CHECK: [[INT_HINT:%[0-9]+]] = sext i32 %0 to i64
// CHECK-NEXT: [[INT_MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMap"(ptr @"map[_llgo_string]_llgo_int", i64 [[INT_HINT]])
// CHECK-NEXT: [[INT_LEN:%[0-9]+]] = call i64 @"{{.*}}MapLen"(ptr [[INT_MAP]])
// CHECK-NEXT: ret i64 [[INT_LEN]]
func fromInt32(n int32) int {
	return len(make(map[string]int, n))
}

// CHECK-LABEL: define i64 @main.fromUint32(i32 %0){{.*}} {
// CHECK: [[UINT_HINT:%[0-9]+]] = zext i32 %0 to i64
// CHECK-NEXT: [[UINT_MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMap"(ptr @"map[_llgo_string]_llgo_int", i64 [[UINT_HINT]])
// CHECK-NEXT: [[UINT_LEN:%[0-9]+]] = call i64 @"{{.*}}MapLen"(ptr [[UINT_MAP]])
// CHECK-NEXT: ret i64 [[UINT_LEN]]
func fromUint32(n uint32) int {
	return len(make(map[string]int, n))
}

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[UINT_RESULT:%[0-9]+]] = call i64 @main.fromUint32(i32 2)
	// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[UINT_RESULT]])
	// CHECK: [[INT_RESULT:%[0-9]+]] = call i64 @main.fromInt32(i32 3)
	// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[INT_RESULT]])
	println(fromUint32(2))
	println(fromInt32(3))
}
