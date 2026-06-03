// LITTEST
package main

import "github.com/goplus/lib/c"

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
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %2, i32 0, i32 4
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %6, align 8
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %7, i32 0, i32 0
// CHECK-NEXT:   %9 = load ptr, ptr %8, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %9, i32 0, i32 4
// CHECK-NEXT:   store i64 200, ptr %10, align 8
// CHECK-NEXT:   %11 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %11, i32 0, i32 2
// CHECK-NEXT:   store ptr %12, ptr %13, align 8
// CHECK-NEXT:   %14 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %14, i32 0, i32 2
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %16, i32 0, i32 1
// CHECK-NEXT:   store ptr %17, ptr %18, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %19, i32 0, i32 2
// CHECK-NEXT:   %21 = load ptr, ptr %20, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %21, i32 0, i32 1
// CHECK-NEXT:   %23 = load ptr, ptr %22, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %23, i32 0, i32 4
// CHECK-NEXT:   store i64 300, ptr %24, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %25, i32 0, i32 3
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %26, i32 0, i32 1
// CHECK-NEXT:   store i64 10, ptr %27, align 8
// CHECK-NEXT:   %28 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %29 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %28, i32 0, i32 3
// CHECK-NEXT:   %30 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %29, i32 0, i32 0
// CHECK-NEXT:   store ptr %30, ptr %31, align 8
// CHECK-NEXT:   %32 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %34 = getelementptr inbounds { ptr }, ptr %33, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/named.main$1", ptr undef }, ptr %33, 1
// CHECK-NEXT:   %36 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %32, i32 0, i32 5
// CHECK-NEXT:   store { ptr, ptr } %35, ptr %36, align 8
// CHECK-NEXT:   %37 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %38 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %37, i32 0, i32 0
// CHECK-NEXT:   %39 = load ptr, ptr %38, align 8
// CHECK-NEXT:   %40 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %39, i32 0, i32 4
// CHECK-NEXT:   %41 = load i64, ptr %40, align 8
// CHECK-NEXT:   %42 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %43 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %42, i32 0, i32 2
// CHECK-NEXT:   %44 = load ptr, ptr %43, align 8
// CHECK-NEXT:   %45 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mSpanList", ptr %44, i32 0, i32 1
// CHECK-NEXT:   %46 = load ptr, ptr %45, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %46, i32 0, i32 4
// CHECK-NEXT:   %48 = load i64, ptr %47, align 8
// CHECK-NEXT:   %49 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %50 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %49, i32 0, i32 3
// CHECK-NEXT:   %51 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %50, i32 0, i32 1
// CHECK-NEXT:   %52 = load i64, ptr %51, align 8
// CHECK-NEXT:   %53 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %53, i32 0, i32 3
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %54, i32 0, i32 0
// CHECK-NEXT:   %56 = load ptr, ptr %55, align 8
// CHECK-NEXT:   %57 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %56, i32 0, i32 4
// CHECK-NEXT:   %58 = load i64, ptr %57, align 8
// CHECK-NEXT:   %59 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %60 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %59, i32 0, i32 5
// CHECK-NEXT:   %61 = load { ptr, ptr }, ptr %60, align 8
// CHECK-NEXT:   %62 = extractvalue { ptr, ptr } %61, 1
// CHECK-NEXT:   %63 = extractvalue { ptr, ptr } %61, 0
// CHECK-NEXT:   %64 = call i64 %63(ptr %62, i64 -2)
// CHECK-NEXT:   %65 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %66 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %65, i32 0, i32 3
// CHECK-NEXT:   %67 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.minfo", ptr %66, i32 0, i32 0
// CHECK-NEXT:   %68 = load ptr, ptr %67, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %68, i32 0, i32 5
// CHECK-NEXT:   %70 = load { ptr, ptr }, ptr %69, align 8
// CHECK-NEXT:   %71 = extractvalue { ptr, ptr } %70, 1
// CHECK-NEXT:   %72 = extractvalue { ptr, ptr } %70, 0
// CHECK-NEXT:   %73 = call i64 %72(ptr %71, i64 -3)
// CHECK-NEXT:   %74 = call i32 (ptr, ...) @printf(ptr @0, i64 %41, i64 %48, i64 %52, i64 %58, i64 %64, i64 %73)
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
	m.check = func(n int) int {
		return m.value * n
	}
	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/named.main$1"(ptr %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
	// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
	// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testrt/named.mspan", ptr %4, i32 0, i32 4
	// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
	// CHECK-NEXT:   %7 = mul i64 %6, %1
	// CHECK-NEXT:   ret i64 %7
	// CHECK-NEXT: }
	c.Printf(c.Str("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}
