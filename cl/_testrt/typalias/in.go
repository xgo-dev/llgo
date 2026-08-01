// LITTEST
package main

import "C"
import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

type Foo = struct {
	A  C.int
	ok bool
}

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @main.Print(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %5, label %6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %6
// CHECK-NEXT:   %3 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %4, label %8, label %9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %9, %6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: 5:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 6:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %7 = load i1, ptr %1, align 1
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 8:                                                ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %_llgo_1
// CHECK-NEXT:   %10 = load i32, ptr %3, align 4
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @main.format, i32 %10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-NEXT: }

func Print(p *Foo) {
	if p.ok {
		printf(&format[0], p.A)
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i32 100, ptr %1, align 4
// CHECK-NEXT:   store i1 true, ptr %2, align 1
// CHECK-NEXT:   call void @main.Print(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	foo := &Foo{100, true}
	Print(foo)
}
