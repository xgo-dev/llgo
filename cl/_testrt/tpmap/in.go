// LITTEST
package main

// CHECK-LINE: @29 = private unnamed_addr constant [5 x i8] c"world", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpmap.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/tpmap.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/tpmap.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T1 int

type T2 struct {
	v int
}

type T3[T any] struct {
	v T
}

type cacheKey struct {
	t1 T1
	t2 T2
	t3 T3[any]
	t4 *int
	t5 uintptr
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpmap.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_{{.*}}/cl/_testrt/tpmap.cacheKey]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testrt/tpmap.cacheKey", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, i32 0, i32 1
// CHECK-NEXT:   %6 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.T2", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %8 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, i32 0, i32 2
// CHECK-NEXT:   %10 = icmp eq ptr %9, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.T3[any]", ptr %9, i32 0, i32 0
// CHECK-NEXT:   %12 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, i32 0, i32 3
// CHECK-NEXT:   %14 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, i32 0, i32 4
// CHECK-NEXT:   store i64 0, ptr %3, align 8
// CHECK-NEXT:   store i64 0, ptr %7, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %16, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %11, align 8
// CHECK-NEXT:   store ptr null, ptr %13, align 8
// CHECK-NEXT:   store i64 0, ptr %15, align 8
// CHECK-NEXT:   %18 = load %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %1, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpmap.cacheKey" %18, ptr %19, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_{{.*}}/cl/_testrt/tpmap.cacheKey]_llgo_string", ptr %0, ptr %19)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 5 }, ptr %20, align 8
// CHECK-NEXT:   %21 = alloca %"{{.*}}/cl/_testrt/tpmap.cacheKey", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %21, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %22 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, i32 0, i32 0
// CHECK-NEXT:   %24 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %24)
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, i32 0, i32 1
// CHECK-NEXT:   %26 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %26)
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.T2", ptr %25, i32 0, i32 0
// CHECK-NEXT:   %28 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %28)
// CHECK-NEXT:   %29 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, i32 0, i32 2
// CHECK-NEXT:   %30 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %30)
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.T3[any]", ptr %29, i32 0, i32 0
// CHECK-NEXT:   %32 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %32)
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, i32 0, i32 3
// CHECK-NEXT:   %34 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, i32 0, i32 4
// CHECK-NEXT:   store i64 0, ptr %23, align 8
// CHECK-NEXT:   store i64 0, ptr %27, align 8
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %36, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %37, ptr %31, align 8
// CHECK-NEXT:   store ptr null, ptr %33, align 8
// CHECK-NEXT:   store i64 0, ptr %35, align 8
// CHECK-NEXT:   %38 = load %"{{.*}}/cl/_testrt/tpmap.cacheKey", ptr %21, align 8
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/tpmap.cacheKey" %38, ptr %39, align 8
// CHECK-NEXT:   %40 = call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"(ptr @"map[_llgo_{{.*}}/cl/_testrt/tpmap.cacheKey]_llgo_string", ptr %0, ptr %39)
// CHECK-NEXT:   %41 = extractvalue { ptr, i1 } %40, 0
// CHECK-NEXT:   %42 = load %"{{.*}}/runtime/internal/runtime.String", ptr %41, align 8
// CHECK-NEXT:   %43 = extractvalue { ptr, i1 } %40, 1
// CHECK-NEXT:   %44 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" %42, 0
// CHECK-NEXT:   %45 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %44, i1 %43, 1
// CHECK-NEXT:   %46 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %45, 0
// CHECK-NEXT:   %47 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %45, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %46)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	m := map[cacheKey]string{}
	m[cacheKey{0, T2{0}, T3[any]{0}, nil, 0}] = "world"
	v, ok := m[cacheKey{0, T2{0}, T3[any]{0}, nil, 0}]
	println(v, ok)
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
