// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LINE: @0 = private unnamed_addr constant [19 x i8] c"%d %d %d %d %d %d\0A\00", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/named.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/named.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/named.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type mSpanList struct {
	first *mspan
	last  *mspan
}

type minfo struct {
	span *mspan
	info int
}

type mspan struct {
	next  *mspan
	prev  *mspan
	list  *mSpanList
	info  minfo
	value int
	check func(int) int
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/named.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %3 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %2, i32 0, i32 4
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   %5 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %7 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %6, ptr %8, align 8
// CHECK-NEXT:   %9 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %10 = icmp eq ptr %9, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %9, i32 0, i32 0
// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %13 = icmp eq ptr %12, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %12, i32 0, i32 4
// CHECK-NEXT:   store i64 200, ptr %14, align 8
// CHECK-NEXT:   %15 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %17 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %17)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %15, i32 0, i32 2
// CHECK-NEXT:   store ptr %16, ptr %18, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %20 = icmp eq ptr %19, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %20)
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %19, i32 0, i32 2
// CHECK-NEXT:   %22 = load ptr, ptr %21, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %24 = icmp eq ptr %22, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %24)
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %22, i32 0, i32 1
// CHECK-NEXT:   store ptr %23, ptr %25, align 8
// CHECK-NEXT:   %26 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %27 = icmp eq ptr %26, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %26, i32 0, i32 2
// CHECK-NEXT:   %29 = load ptr, ptr %28, align 8
// CHECK-NEXT:   %30 = icmp eq ptr %29, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %30)
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %29, i32 0, i32 1
// CHECK-NEXT:   %32 = load ptr, ptr %31, align 8
// CHECK-NEXT:   %33 = icmp eq ptr %32, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %33)
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %32, i32 0, i32 4
// CHECK-NEXT:   store i64 300, ptr %34, align 8
// CHECK-NEXT:   %35 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %36 = icmp eq ptr %35, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %36)
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %35, i32 0, i32 3
// CHECK-NEXT:   %38 = icmp eq ptr %37, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %38)
// CHECK-NEXT:   %39 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %37, i32 0, i32 1
// CHECK-NEXT:   store i64 10, ptr %39, align 8
// CHECK-NEXT:   %40 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %41 = icmp eq ptr %40, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %41)
// CHECK-NEXT:   %42 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %40, i32 0, i32 3
// CHECK-NEXT:   %43 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %44 = icmp eq ptr %42, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %44)
// CHECK-NEXT:   %45 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %42, i32 0, i32 0
// CHECK-NEXT:   store ptr %43, ptr %45, align 8
// CHECK-NEXT:   %46 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %48 = getelementptr inbounds { ptr }, ptr %47, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %48, align 8
// CHECK-NEXT:   %49 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/named.main$1", ptr undef }, ptr %47, 1
// CHECK-NEXT:   %50 = icmp eq ptr %46, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %50)
// CHECK-NEXT:   %51 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %46, i32 0, i32 5
// CHECK-NEXT:   store { ptr, ptr } %49, ptr %51, align 8
// CHECK-NEXT:   %52 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %53 = icmp eq ptr %52, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %52, i32 0, i32 0
// CHECK-NEXT:   %55 = load ptr, ptr %54, align 8
// CHECK-NEXT:   %56 = icmp eq ptr %55, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %56)
// CHECK-NEXT:   %57 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %55, i32 0, i32 4
// CHECK-NEXT:   %58 = load i64, ptr %57, align 8
// CHECK-NEXT:   %59 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %60 = icmp eq ptr %59, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %60)
// CHECK-NEXT:   %61 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %59, i32 0, i32 2
// CHECK-NEXT:   %62 = load ptr, ptr %61, align 8
// CHECK-NEXT:   %63 = icmp eq ptr %62, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %63)
// CHECK-NEXT:   %64 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %62, i32 0, i32 1
// CHECK-NEXT:   %65 = load ptr, ptr %64, align 8
// CHECK-NEXT:   %66 = icmp eq ptr %65, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %66)
// CHECK-NEXT:   %67 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %65, i32 0, i32 4
// CHECK-NEXT:   %68 = load i64, ptr %67, align 8
// CHECK-NEXT:   %69 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %70 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %70)
// CHECK-NEXT:   %71 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %69, i32 0, i32 3
// CHECK-NEXT:   %72 = icmp eq ptr %71, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %72)
// CHECK-NEXT:   %73 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %71, i32 0, i32 1
// CHECK-NEXT:   %74 = load i64, ptr %73, align 8
// CHECK-NEXT:   %75 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %76 = icmp eq ptr %75, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %76)
// CHECK-NEXT:   %77 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %75, i32 0, i32 3
// CHECK-NEXT:   %78 = icmp eq ptr %77, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %78)
// CHECK-NEXT:   %79 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %77, i32 0, i32 0
// CHECK-NEXT:   %80 = load ptr, ptr %79, align 8
// CHECK-NEXT:   %81 = icmp eq ptr %80, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %81)
// CHECK-NEXT:   %82 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %80, i32 0, i32 4
// CHECK-NEXT:   %83 = load i64, ptr %82, align 8
// CHECK-NEXT:   %84 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %85 = icmp eq ptr %84, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %85)
// CHECK-NEXT:   %86 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %84, i32 0, i32 5
// CHECK-NEXT:   %87 = load { ptr, ptr }, ptr %86, align 8
// CHECK-NEXT:   %88 = extractvalue { ptr, ptr } %87, 1
// CHECK-NEXT:   %89 = extractvalue { ptr, ptr } %87, 0
// CHECK-NEXT:   %90 = call i64 %89(ptr %88, i64 -2)
// CHECK-NEXT:   %91 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %92 = icmp eq ptr %91, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %92)
// CHECK-NEXT:   %93 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %91, i32 0, i32 3
// CHECK-NEXT:   %94 = icmp eq ptr %93, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %94)
// CHECK-NEXT:   %95 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %93, i32 0, i32 0
// CHECK-NEXT:   %96 = load ptr, ptr %95, align 8
// CHECK-NEXT:   %97 = icmp eq ptr %96, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %97)
// CHECK-NEXT:   %98 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %96, i32 0, i32 5
// CHECK-NEXT:   %99 = load { ptr, ptr }, ptr %98, align 8
// CHECK-NEXT:   %100 = extractvalue { ptr, ptr } %99, 1
// CHECK-NEXT:   %101 = extractvalue { ptr, ptr } %99, 0
// CHECK-NEXT:   %102 = call i64 %101(ptr %100, i64 -3)
// CHECK-NEXT:   %103 = call i32 (ptr, ...) @printf(ptr @0, i64 %58, i64 %68, i64 %74, i64 %83, i64 %90, i64 %102)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	m := &mspan{}
	m.value = 100
	m.next = &mspan{}
	m.next.value = 200
	m.list = &mSpanList{}
	m.list.last = &mspan{}
	m.list.last.value = 300
	m.info.info = 10
	m.info.span = m

	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/named.main$1"(ptr %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
	// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
	// CHECK-NEXT:   %5 = icmp eq ptr %4, null
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
	// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %4, i32 0, i32 4
	// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
	// CHECK-NEXT:   %8 = mul i64 %7, %1
	// CHECK-NEXT:   ret i64 %8
	// CHECK-NEXT: }

	m.check = func(n int) int {
		return m.value * n
	}
	c.Printf(c.Str("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}
