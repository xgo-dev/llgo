// LITTEST
package main

import (
	"fmt"
	"sync"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [1 x i8] c"1", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [7 x i8] c"%#v %v\0A", align 1{{$}}

func main() {
	var m sync.Map
	m.Store(1, "hello")
	m.Store("1", 100)
	v, ok := m.Load("1")
	fmt.Println(v, ok)
	m.Range(func(k, v interface{}) bool {
		fmt.Printf("%#v %v\n", k, v)
		return true
	})
}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @sync.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 {{(64|96)}})
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3, 1
// CHECK-NEXT:   call void @"sync.(*Map).Store"(ptr %0, %"{{.*}}/runtime/internal/runtime.eface" %2, %"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"sync.(*Map).Store"(ptr %0, %"{{.*}}/runtime/internal/runtime.eface" %6, %"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %9, 1
// CHECK-NEXT:   %11 = call { %"{{.*}}/runtime/internal/runtime.eface", i1 } @"sync.(*Map).Load"(ptr %0, %"{{.*}}/runtime/internal/runtime.eface" %10)
// CHECK-NEXT:   %12 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %11, 0
// CHECK-NEXT:   %13 = extractvalue { %"{{.*}}/runtime/internal/runtime.eface", i1 } %11, 1
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %14, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %12, ptr %15, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %14, i64 1
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 %13, ptr %17, align 1
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %17, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %18, ptr %16, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %14, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 2, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 2, 2
// CHECK-NEXT:   %22 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   call void @"sync.(*Map).Range"(ptr %0, { ptr, ptr } { ptr @"main.main$1", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"main.main$1"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %2, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %2, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %1, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %2, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 2, 1
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 2, 2
// CHECK-NEXT:   %8 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 7 }, %"{{.*}}/runtime/internal/runtime.Slice" %7)
// CHECK-NEXT:   ret i1 true
// CHECK-NEXT: }
