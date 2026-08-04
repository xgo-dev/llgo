// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/pthread/sync"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [9 x i8] c"Do once\0A\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [16 x i8] c"sync.Once demo\0A\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [16 x i8] c"sync.Once demo\0A\00", align 1{{$}}

var once sync.Once = sync.OnceInit

func f() {
	once.Do(func() {
		c.Printf(c.Str("Do once\n"))
	})
}

func main() {
	println(c.GoString(c.Str("sync.Once demo\n"), 9))
	println(c.GoString(c.Str("sync.Once demo\n")))
	f()
	f()
}

// CHECK-LABEL: define void @main.f(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @pthread_once(ptr @main.once, ptr @"main.f$1")
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.f$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @{{[0-9]+}})
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   %1 = load {{.*}}, ptr @llgoSyncOnceInitVal
// CHECK-NEXT:   store {{.*}} %1, ptr @main.once
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFrom"(ptr @{{[0-9]+}}, i64 9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromCStr"(ptr @{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @main.f()
// CHECK-NEXT:   call void @main.f()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
