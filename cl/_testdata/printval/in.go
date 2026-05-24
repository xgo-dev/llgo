// LITTEST
package main

import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/printval.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/printval.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/printval.init$guard", align 1
// CHECK-NEXT:   store i8 72, ptr @"{{.*}}/cl/_testdata/printval.format", align 1
// CHECK-NEXT:   store i8 101, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 1), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 2), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 3), align 1
// CHECK-NEXT:   store i8 111, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 4), align 1
// CHECK-NEXT:   store i8 32, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 5), align 1
// CHECK-NEXT:   store i8 37, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 6), align 1
// CHECK-NEXT:   store i8 100, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 7), align 1
// CHECK-NEXT:   store i8 10, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 8), align 1
// CHECK-NEXT:   store i8 0, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/printval.format", i64 9), align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/printval.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}/cl/_testdata/printval.format", i64 100)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	printf(&format[0], 100)
}
