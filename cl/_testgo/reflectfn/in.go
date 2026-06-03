// LITTEST
package main

import (
	"fmt"
	"reflect"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectfn.demo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func demo() {
	println("demo")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/reflectfn.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/reflectfn.main$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %4, i64 0
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %6, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %7, ptr %5, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, i64 1, 1
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, i64 1, 2
// CHECK-NEXT:   %11 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %10)
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %12, i64 0
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflectfn.demo", ptr null }, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %14, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %15, ptr %13, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %12, 0
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %16, i64 1, 1
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, i64 1, 2
// CHECK-NEXT:   %19 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %18)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %20, i64 0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflectfn.demo", ptr null }, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %22, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %23, ptr %21, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %20, 0
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, i64 1, 1
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %25, i64 1, 2
// CHECK-NEXT:   %27 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %26)
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %28, align 8
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %28, 1
// CHECK-NEXT:   %30 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %29)
// CHECK-NEXT:   %31 = call ptr @reflect.Value.UnsafePointer(%reflect.Value %30)
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %32, i64 0
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %31, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %34, ptr %33, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %32, 0
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, i64 1, 1
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %36, i64 1, 2
// CHECK-NEXT:   %38 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %37)
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflectfn.demo", ptr null }, ptr %39, align 8
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %39, 1
// CHECK-NEXT:   %41 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %40)
// CHECK-NEXT:   %42 = call ptr @reflect.Value.UnsafePointer(%reflect.Value %41)
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %43, i64 0
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %42, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %45, ptr %44, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %43, 0
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %46, i64 1, 1
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %47, i64 1, 2
// CHECK-NEXT:   %49 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %48)
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/reflectfn.demo", ptr null }, ptr %50, align 8
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %50, 1
// CHECK-NEXT:   %52 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %51)
// CHECK-NEXT:   %53 = call ptr @reflect.Value.UnsafePointer(%reflect.Value %52)
// CHECK-NEXT:   %54 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %54, i64 0
// CHECK-NEXT:   %56 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %53, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %56, ptr %55, align 8
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %54, 0
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %57, i64 1, 1
// CHECK-NEXT:   %59 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, i64 1, 2
// CHECK-NEXT:   %60 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %59)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	v := 100
	fn := func() {
		println(v)
	}
	fdemo := demo
	fmt.Println(fn)
	fmt.Println(demo)
	fmt.Println(fdemo)
	fmt.Println(reflect.ValueOf(fn).UnsafePointer())
	fmt.Println(reflect.ValueOf(demo).UnsafePointer())
	fmt.Println(reflect.ValueOf(fdemo).UnsafePointer())
}
