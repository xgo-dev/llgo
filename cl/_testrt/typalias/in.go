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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typalias.Print"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %4 = load i1, ptr %3, align 1
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %8 = load i32, ptr %7, align 4
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}/cl/_testrt/typalias.format", i32 %8)
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

func main() {
	foo := &Foo{100, true}
	Print(foo)
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/typalias._Cgo_ptr"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typalias.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/typalias.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/typalias.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   store i8 72, ptr @"{{.*}}/cl/_testrt/typalias.format", align 1
// CHECK-NEXT:   store i8 101, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 1), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 2), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 3), align 1
// CHECK-NEXT:   store i8 111, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 4), align 1
// CHECK-NEXT:   store i8 32, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 5), align 1
// CHECK-NEXT:   store i8 37, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 6), align 1
// CHECK-NEXT:   store i8 100, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 7), align 1
// CHECK-NEXT:   store i8 10, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 8), align 1
// CHECK-NEXT:   store i8 0, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/typalias.format", i64 9), align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typalias.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   store i32 100, ptr %2, align 4
// CHECK-NEXT:   store i1 true, ptr %4, align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/typalias.Print"(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
