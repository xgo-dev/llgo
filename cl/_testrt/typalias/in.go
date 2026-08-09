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
// CHECK-NEXT:   %2 = load i1, ptr %1, align 1
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load i32, ptr %3, align 4
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @main.format, i32 %4)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
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
// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 8, align 4
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   %0 = getelementptr inbounds { i32, i1 }, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %1 = getelementptr inbounds { i32, i1 }, ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   store i32 100, ptr %0, align 4
// ESCAPE-NEXT:   store i1 true, ptr %1, align 1
// ESCAPE-NEXT:   call void @main.Print(ptr %.stack)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }
func main() {
	foo := &Foo{100, true}
	Print(foo)
}
