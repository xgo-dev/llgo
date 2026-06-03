// LITTEST
package main

import (
	"unsafe"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/funcdecl.check"({ ptr, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/funcdecl.demo", ptr null }, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } %0, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 0
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr %5)
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 1
// CHECK-NEXT:   %8 = load { ptr, ptr }, ptr %7, align 8
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %4, 0
// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr %9)
// CHECK-NEXT:   br i1 %10, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 68 }, ptr %11, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %11, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %12)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %4, 1
// CHECK-NEXT:   %14 = load { ptr, ptr }, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintEface"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintEface"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %0, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %14, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr @"{{.*}}/cl/_testrt/funcdecl.demo")
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/cl/_testrt/funcdecl.closurePtr"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/cl/_testrt/funcdecl.closurePtr"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %20 = icmp eq ptr %18, %19
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %20)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 68 }, ptr %21, align 8
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %21, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %22)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
func check(fn func()) {
	var a any = demo
	var b any = fn
	fn1 := a.(func())
	fn2 := b.(func())
	println(a, b, fn, fn1, fn2, demo)
	println(closurePtr(a) == closurePtr(b))
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/funcdecl.closurePtr"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/funcdecl.rtype", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr, ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   ret ptr %5
// CHECK-NEXT: }

func closurePtr(a any) unsafe.Pointer {
	return (*rtype)(unsafe.Pointer(&a)).ptr.fn
}

type rtype struct {
	typ unsafe.Pointer
	ptr *struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/funcdecl.demo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func demo() {
	println("demo")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/funcdecl.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 5 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/funcdecl.check"({ ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/funcdecl.demo", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	println("hello")
	check(demo)
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/funcdecl.demo"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/funcdecl.demo"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
