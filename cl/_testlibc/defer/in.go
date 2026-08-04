// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [4 x i8] c"%s\0A\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [7 x i8] c"world\0A\00", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"bye\0A\00", align 1{{$}}

func f(s string) bool {
	return len(s) > 2
}

func main() {
	c.GoDeferData()
	if s := "hello"; f(s) {
		defer c.Printf(c.Str("%s\n"), c.AllocaCStr(s))
	} else {
		defer c.Printf(c.Str("world\n"))
	}
	defer c.Printf(c.Str("bye\n"))
}

// CHECK-LABEL: define i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %0, 1
// CHECK-NEXT:   %2 = icmp sgt i64 %1, 2
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = call i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %3 = alloca i8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %3, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 2
// CHECK-NEXT:   store ptr %2, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_6), ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %4)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 1
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 3
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 4
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %12, align 8
// CHECK-NEXT:   %13 = call i32 @{{(__)?}}sigsetjmp(ptr %3, i32 0)
// CHECK-NEXT:   %14 = icmp eq i32 %13, 0
// CHECK-NEXT:   br i1 %14, label %_llgo_5, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %15 = alloca i8, i64 6, align 1
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.CStrCopy"(ptr %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT:   %17 = load i64, ptr %9, align 8
// CHECK-NEXT:   %18 = or i64 %17, 1
// CHECK-NEXT:   store i64 %18, ptr %9, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %20, i32 0, i32 0
// CHECK-NEXT:   store ptr %19, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %20, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %20, i32 0, i32 2
// CHECK-NEXT:   store ptr @{{[0-9]+}}, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %20, i32 0, i32 3
// CHECK-NEXT:   store ptr %16, ptr %24, align 8
// CHECK-NEXT:   store ptr %20, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_1
// CHECK-NEXT:   %25 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, ptr }, ptr %26, i32 0, i32 0
// CHECK-NEXT:   store ptr %25, ptr %27, align 8
// CHECK-NEXT:   %28 = getelementptr inbounds { ptr, i64, ptr }, ptr %26, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %28, align 8
// CHECK-NEXT:   %29 = getelementptr inbounds { ptr, i64, ptr }, ptr %26, i32 0, i32 2
// CHECK-NEXT:   store ptr @{{[0-9]+}}, ptr %29, align 8
// CHECK-NEXT:   store ptr %26, ptr %12, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_9), ptr %11, align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %30 = load i64, ptr %9, align 8
// CHECK-NEXT:   %31 = or i64 %30, 2
// CHECK-NEXT:   store i64 %31, ptr %9, align 8
// CHECK-NEXT:   %32 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   %34 = getelementptr inbounds { ptr, i64, ptr }, ptr %33, i32 0, i32 0
// CHECK-NEXT:   store ptr %32, ptr %34, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds { ptr, i64, ptr }, ptr %33, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %35, align 8
// CHECK-NEXT:   %36 = getelementptr inbounds { ptr, i64, ptr }, ptr %33, i32 0, i32 2
// CHECK-NEXT:   store ptr @{{[0-9]+}}, ptr %36, align 8
// CHECK-NEXT:   store ptr %33, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_11), ptr %10, align 8
// CHECK-NEXT:   %37 = load i64, ptr %9, align 8
// CHECK-NEXT:   %38 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %39 = icmp ne ptr %38, null
// CHECK-NEXT:   br i1 %39, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_8, %_llgo_19
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %2)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_7), ptr %11, align 8
// CHECK-NEXT:   %40 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %40, [label %_llgo_7, label %_llgo_10, label %_llgo_11, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_19
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8, %_llgo_15
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_7), ptr %10, align 8
// CHECK-NEXT:   %41 = load i64, ptr %9, align 8
// CHECK-NEXT:   %42 = and i64 %41, 1
// CHECK-NEXT:   %43 = icmp ne i64 %42, 0
// CHECK-NEXT:   br i1 %43, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_8, %_llgo_13
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %10, align 8
// CHECK-NEXT:   %44 = load i64, ptr %9, align 8
// CHECK-NEXT:   %45 = and i64 %44, 2
// CHECK-NEXT:   %46 = icmp ne i64 %45, 0
// CHECK-NEXT:   br i1 %46, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %47 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %48 = load { ptr, i64, ptr }, ptr %47, align 8
// CHECK-NEXT:   %49 = extractvalue { ptr, i64, ptr } %48, 0
// CHECK-NEXT:   store ptr %49, ptr %12, align 8
// CHECK-NEXT:   %50 = extractvalue { ptr, i64, ptr } %48, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %47)
// CHECK-NEXT:   %51 = call i32 (ptr, ...) @printf(ptr %50)
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_12, %_llgo_6
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %52 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %53 = icmp ne ptr %52, null
// CHECK-NEXT:   br i1 %53, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_17, %_llgo_11
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %54 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %55 = load { ptr, i64, ptr }, ptr %54, align 8
// CHECK-NEXT:   %56 = extractvalue { ptr, i64, ptr } %55, 0
// CHECK-NEXT:   store ptr %56, ptr %12, align 8
// CHECK-NEXT:   %57 = extractvalue { ptr, i64, ptr } %55, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %54)
// CHECK-NEXT:   %58 = call i32 (ptr, ...) @printf(ptr %57)
// CHECK-NEXT:   br label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_16, %_llgo_14
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %59 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %60 = icmp ne ptr %59, null
// CHECK-NEXT:   br i1 %60, label %_llgo_20, label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_21, %_llgo_10
// CHECK-NEXT:   %61 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %4, align 8
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %61, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %62)
// CHECK-NEXT:   %63 = load ptr, ptr %11, align 8
// CHECK-NEXT:   indirectbr ptr %63, [label %_llgo_7, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %64 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %65 = load { ptr, i64, ptr, ptr }, ptr %64, align 8
// CHECK-NEXT:   %66 = extractvalue { ptr, i64, ptr, ptr } %65, 0
// CHECK-NEXT:   store ptr %66, ptr %12, align 8
// CHECK-NEXT:   %67 = extractvalue { ptr, i64, ptr, ptr } %65, 2
// CHECK-NEXT:   %68 = extractvalue { ptr, i64, ptr, ptr } %65, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %64)
// CHECK-NEXT:   %69 = call i32 (ptr, ...) @printf(ptr %67, ptr %68)
// CHECK-NEXT:   br label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_20, %_llgo_18
// CHECK-NEXT:   br label %_llgo_19
// CHECK-NEXT: }
