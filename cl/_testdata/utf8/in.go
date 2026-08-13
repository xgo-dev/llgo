// LITTEST
package main

import (
	"unicode/utf8"
)

// CHECK-LABEL: define i8 @main.index(i8 %0){{.*}} {
// CHECK: [[INDEX:%[0-9]+]] = sext i8 %0 to i64
// CHECK-NEXT: [[INDEX_NEG:%[0-9]+]] = icmp slt i64 [[INDEX]], 0
// CHECK-NEXT: [[INDEX_UPPER:%[0-9]+]] = icmp uge i64 [[INDEX]], 8
// CHECK-NEXT: [[INDEX_OOB:%[0-9]+]] = or i1 [[INDEX_UPPER]], [[INDEX_NEG]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 [[INDEX_OOB]], i64 [[INDEX]], i1 true, i64 8)
// CHECK-NEXT: [[INDEX_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr @main.array, i64 [[INDEX]]
// CHECK-NEXT: [[INDEX_VALUE:%[0-9]+]] = load i8, ptr [[INDEX_PTR]]
// CHECK-NEXT: ret i8 [[INDEX_VALUE]]
func index(n int8) uint8 {
	return array[n]
}

var array = [...]uint8{
	1, 2, 3, 4, 5, 6, 7, 8,
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[OFFSET:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_OFFSET:%[0-9]+]], %{{.*}} ]
// CHECK: [[SUFFIX:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}StringSlice2"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 7 }, i64 [[OFFSET]], i64 7, i1 true, i1 true)
// CHECK-NEXT: [[DECODE:%[0-9]+]] = call { i32, i64 } @"unicode/utf8.DecodeRuneInString"(%"{{.*}}String" [[SUFFIX]])
// CHECK-NEXT: [[RUNE:%[0-9]+]] = extractvalue { i32, i64 } [[DECODE]], 0
// CHECK-NEXT: [[WIDTH:%[0-9]+]] = extractvalue { i32, i64 } [[DECODE]], 1
// CHECK-NEXT: [[NEXT_OFFSET]] = add i64 [[OFFSET]], [[WIDTH]]
// CHECK-NEXT: [[PRINT_RUNE:%[0-9]+]] = sext i32 [[RUNE]] to i64
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[PRINT_RUNE]])
// CHECK: [[INDEXED:%[0-9]+]] = call i8 @main.index(i8 2)
// CHECK-NEXT: [[IS_THREE:%[0-9]+]] = icmp eq i8 [[INDEXED]], 3
// CHECK-NEXT: call void @"{{.*}}PrintBool"(i1 [[IS_THREE]])
func main() {
	var str = "中abcd"
	for i := 0; i < len(str); {
		r, n := utf8.DecodeRuneInString(str[i:])
		i += n
		println(r)
	}
	println(index(2) == 3)
}
