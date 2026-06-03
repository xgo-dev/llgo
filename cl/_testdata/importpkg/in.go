// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/importpkg/stdio"

// CHECK-LABEL: define void @"{{.*}}.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:{{.*}}
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/stdio.init"()
// CHECK-NEXT:   store i8 72, ptr @"{{.*}}.hello", align 1
// CHECK-NEXT:   store i8 101, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 1), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 2), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 3), align 1
// CHECK-NEXT:   store i8 111, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 4), align 1
// CHECK-NEXT:   store i8 10, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 5), align 1
// CHECK-NEXT:   store i8 0, ptr getelementptr inbounds (i8, ptr @"{{.*}}importpkg.hello", i64 6), align 1
var hello = [...]int8{'H', 'e', 'l', 'l', 'o', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/stdio.Max"(i64 2, i64 100)
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}.hello")
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	_ = stdio.Max(2, 100)
	stdio.Printf(&hello[0])
}
