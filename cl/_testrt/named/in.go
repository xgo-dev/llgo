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
// CHECK-NEXT:   %4 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %2, i32 0, i32 4
// CHECK-NEXT:   store i64 100, ptr %5, align 8
// CHECK-NEXT:   %6 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %8 = icmp eq ptr %6, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = icmp eq ptr %6, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %6, i32 0, i32 0
// CHECK-NEXT:   store ptr %7, ptr %10, align 8
// CHECK-NEXT:   %11 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %12 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %11, i32 0, i32 0
// CHECK-NEXT:   %15 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %16 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %17)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %15, i32 0, i32 4
// CHECK-NEXT:   store i64 200, ptr %18, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %21 = icmp eq ptr %19, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = icmp eq ptr %19, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %19, i32 0, i32 2
// CHECK-NEXT:   store ptr %20, ptr %23, align 8
// CHECK-NEXT:   %24 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %25 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %25)
// CHECK-NEXT:   %26 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %26)
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %24, i32 0, i32 2
// CHECK-NEXT:   %28 = load ptr, ptr %27, align 8
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %30 = icmp eq ptr %28, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %30)
// CHECK-NEXT:   %31 = icmp eq ptr %28, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %31)
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %28, i32 0, i32 1
// CHECK-NEXT:   store ptr %29, ptr %32, align 8
// CHECK-NEXT:   %33 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %34 = icmp eq ptr %33, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = icmp eq ptr %33, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %35)
// CHECK-NEXT:   %36 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %33, i32 0, i32 2
// CHECK-NEXT:   %37 = load ptr, ptr %36, align 8
// CHECK-NEXT:   %38 = icmp eq ptr %37, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %38)
// CHECK-NEXT:   %39 = icmp eq ptr %37, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %39)
// CHECK-NEXT:   %40 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %37, i32 0, i32 1
// CHECK-NEXT:   %41 = load ptr, ptr %40, align 8
// CHECK-NEXT:   %42 = icmp eq ptr %41, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %42)
// CHECK-NEXT:   %43 = icmp eq ptr %41, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %43)
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %41, i32 0, i32 4
// CHECK-NEXT:   store i64 300, ptr %44, align 8
// CHECK-NEXT:   %45 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %46 = icmp eq ptr %45, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %46)
// CHECK-NEXT:   %47 = icmp eq ptr %45, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %47)
// CHECK-NEXT:   %48 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %45, i32 0, i32 3
// CHECK-NEXT:   %49 = icmp eq ptr %48, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %49)
// CHECK-NEXT:   %50 = icmp eq ptr %48, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %50)
// CHECK-NEXT:   %51 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %48, i32 0, i32 1
// CHECK-NEXT:   store i64 10, ptr %51, align 8
// CHECK-NEXT:   %52 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %53 = icmp eq ptr %52, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = icmp eq ptr %52, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %54)
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %52, i32 0, i32 3
// CHECK-NEXT:   %56 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %57 = icmp eq ptr %55, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %57)
// CHECK-NEXT:   %58 = icmp eq ptr %55, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %58)
// CHECK-NEXT:   %59 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %55, i32 0, i32 0
// CHECK-NEXT:   store ptr %56, ptr %59, align 8
// CHECK-NEXT:   %60 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %61 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %62 = getelementptr inbounds { ptr }, ptr %61, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %62, align 8
// CHECK-NEXT:   %63 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/named.main$1", ptr undef }, ptr %61, 1
// CHECK-NEXT:   %64 = icmp eq ptr %60, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %64)
// CHECK-NEXT:   %65 = icmp eq ptr %60, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %65)
// CHECK-NEXT:   %66 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %60, i32 0, i32 5
// CHECK-NEXT:   store { ptr, ptr } %63, ptr %66, align 8
// CHECK-NEXT:   %67 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %68 = icmp eq ptr %67, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %68)
// CHECK-NEXT:   %69 = icmp eq ptr %67, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %69)
// CHECK-NEXT:   %70 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %67, i32 0, i32 0
// CHECK-NEXT:   %71 = load ptr, ptr %70, align 8
// CHECK-NEXT:   %72 = icmp eq ptr %71, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %72)
// CHECK-NEXT:   %73 = icmp eq ptr %71, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %73)
// CHECK-NEXT:   %74 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %71, i32 0, i32 4
// CHECK-NEXT:   %75 = load i64, ptr %74, align 8
// CHECK-NEXT:   %76 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %77 = icmp eq ptr %76, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %77)
// CHECK-NEXT:   %78 = icmp eq ptr %76, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %78)
// CHECK-NEXT:   %79 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %76, i32 0, i32 2
// CHECK-NEXT:   %80 = load ptr, ptr %79, align 8
// CHECK-NEXT:   %81 = icmp eq ptr %80, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %81)
// CHECK-NEXT:   %82 = icmp eq ptr %80, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %82)
// CHECK-NEXT:   %83 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %80, i32 0, i32 1
// CHECK-NEXT:   %84 = load ptr, ptr %83, align 8
// CHECK-NEXT:   %85 = icmp eq ptr %84, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %85)
// CHECK-NEXT:   %86 = icmp eq ptr %84, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %86)
// CHECK-NEXT:   %87 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %84, i32 0, i32 4
// CHECK-NEXT:   %88 = load i64, ptr %87, align 8
// CHECK-NEXT:   %89 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %90 = icmp eq ptr %89, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %90)
// CHECK-NEXT:   %91 = icmp eq ptr %89, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %91)
// CHECK-NEXT:   %92 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %89, i32 0, i32 3
// CHECK-NEXT:   %93 = icmp eq ptr %92, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %93)
// CHECK-NEXT:   %94 = icmp eq ptr %92, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %94)
// CHECK-NEXT:   %95 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %92, i32 0, i32 1
// CHECK-NEXT:   %96 = load i64, ptr %95, align 8
// CHECK-NEXT:   %97 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %98 = icmp eq ptr %97, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %98)
// CHECK-NEXT:   %99 = icmp eq ptr %97, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %99)
// CHECK-NEXT:   %100 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %97, i32 0, i32 3
// CHECK-NEXT:   %101 = icmp eq ptr %100, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %101)
// CHECK-NEXT:   %102 = icmp eq ptr %100, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %102)
// CHECK-NEXT:   %103 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %100, i32 0, i32 0
// CHECK-NEXT:   %104 = load ptr, ptr %103, align 8
// CHECK-NEXT:   %105 = icmp eq ptr %104, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %105)
// CHECK-NEXT:   %106 = icmp eq ptr %104, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %106)
// CHECK-NEXT:   %107 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %104, i32 0, i32 4
// CHECK-NEXT:   %108 = load i64, ptr %107, align 8
// CHECK-NEXT:   %109 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %110 = icmp eq ptr %109, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %110)
// CHECK-NEXT:   %111 = icmp eq ptr %109, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %111)
// CHECK-NEXT:   %112 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %109, i32 0, i32 5
// CHECK-NEXT:   %113 = load { ptr, ptr }, ptr %112, align 8
// CHECK-NEXT:   %114 = extractvalue { ptr, ptr } %113, 1
// CHECK-NEXT:   %115 = extractvalue { ptr, ptr } %113, 0
// CHECK-NEXT:   %116 = call i64 %115(ptr %114, i64 -2)
// CHECK-NEXT:   %117 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %118 = icmp eq ptr %117, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %118)
// CHECK-NEXT:   %119 = icmp eq ptr %117, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %119)
// CHECK-NEXT:   %120 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %117, i32 0, i32 3
// CHECK-NEXT:   %121 = icmp eq ptr %120, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %121)
// CHECK-NEXT:   %122 = icmp eq ptr %120, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %122)
// CHECK-NEXT:   %123 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %120, i32 0, i32 0
// CHECK-NEXT:   %124 = load ptr, ptr %123, align 8
// CHECK-NEXT:   %125 = icmp eq ptr %124, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %125)
// CHECK-NEXT:   %126 = icmp eq ptr %124, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %126)
// CHECK-NEXT:   %127 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %124, i32 0, i32 5
// CHECK-NEXT:   %128 = load { ptr, ptr }, ptr %127, align 8
// CHECK-NEXT:   %129 = extractvalue { ptr, ptr } %128, 1
// CHECK-NEXT:   %130 = extractvalue { ptr, ptr } %128, 0
// CHECK-NEXT:   %131 = call i64 %130(ptr %129, i64 -3)
// CHECK-NEXT:   %132 = call i32 (ptr, ...) @printf(ptr @0, i64 %75, i64 %88, i64 %96, i64 %108, i64 %116, i64 %131)
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
	// CHECK-NEXT:   %6 = icmp eq ptr %4, null
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
	// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %4, i32 0, i32 4
	// CHECK-NEXT:   %8 = load i64, ptr %7, align 8
	// CHECK-NEXT:   %9 = mul i64 %8, %1
	// CHECK-NEXT:   ret i64 %9
	// CHECK-NEXT: }

	m.check = func(n int) int {
		return m.value * n
	}
	c.Printf(c.Str("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}
