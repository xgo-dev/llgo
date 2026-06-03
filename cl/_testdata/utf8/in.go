// LITTEST
package main

import (
	"unicode/utf8"
)

// CHECK-LABEL: define i8 @"{{.*}}.index"(i8 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = sext i8 %0 to i64
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   %3 = icmp uge i64 %1, 8
// CHECK-NEXT:   %4 = or i1 %3, %2
// CHECK-NEXT:   call void @"{{.*}}.AssertIndexRange"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds i8, ptr @"{{.*}}.array", i64 %1
// CHECK-NEXT:   %6 = load i8, ptr %5, align 1
// CHECK-NEXT:   ret i8 %6
// CHECK-NEXT: }
func index(n int8) uint8 {
	return array[n]
}

// CHECK-LABEL: define void @"{{.*}}.init"(){{.*}} {
// CHECK:   call void @"unicode/utf8.init"()
// CHECK:   store i8 1, ptr @"{{.*}}.array", align 1
var array = [...]uint8{
	1, 2, 3, 4, 5, 6, 7, 8,
}

// CHECK-LABEL: define void @"{{.*}}cl/_testdata/utf8.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %0 = phi i64 [ 0, %_llgo_0 ], [ %6, %_llgo_2 ]
// CHECK-NEXT:   %1 = icmp slt i64 %0, 7
// CHECK-NEXT:   br i1 %1, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %2 = call %"{{.*}}runtime/internal/runtime.String" @"{{.*}}runtime/internal/runtime.StringSlice"(%"{{.*}}runtime/internal/runtime.String" { ptr @0, i64 7 }, i64 %0, i64 7)
// CHECK-NEXT:   %3 = call { i32, i64 } @"unicode/utf8.DecodeRuneInString"(%"{{.*}}runtime/internal/runtime.String" %2)
// CHECK-NEXT:   %4 = extractvalue { i32, i64 } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i32, i64 } %3, 1
// CHECK-NEXT:   %6 = add i64 %0, %5
// CHECK-NEXT:   %7 = sext i32 %4 to i64
// CHECK-NEXT:   call void @"{{.*}}runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %8 = call i8 @"{{.*}}cl/_testdata/utf8.index"(i8 2)
// CHECK-NEXT:   %9 = icmp eq i8 %8, 3
// CHECK-NEXT:   call void @"{{.*}}runtime/internal/runtime.PrintBool"(i1 %9)
// CHECK-NEXT:   call void @"{{.*}}runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	var str = "中abcd"
	for i := 0; i < len(str); {
		r, n := utf8.DecodeRuneInString(str[i:])
		i += n
		println(r)
	}
	println(index(2) == 3)
}
