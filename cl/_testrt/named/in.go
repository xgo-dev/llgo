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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.mspan, ptr %2, i32 0, i32 4
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %6 = getelementptr inbounds %main.mspan, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %6, align 8
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %main.mspan, ptr %7, i32 0, i32 0
// CHECK-NEXT:   %9 = icmp eq ptr %7, null
// CHECK-NEXT:   br i1 %9, label %10, label %11
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %12 = load ptr, ptr %8, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %main.mspan, ptr %12, i32 0, i32 4
// CHECK-NEXT:   store i64 200, ptr %13, align 8
// CHECK-NEXT:   %14 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %16 = getelementptr inbounds %main.mspan, ptr %14, i32 0, i32 2
// CHECK-NEXT:   store ptr %15, ptr %16, align 8
// CHECK-NEXT:   %17 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %main.mspan, ptr %17, i32 0, i32 2
// CHECK-NEXT:   %19 = icmp eq ptr %17, null
// CHECK-NEXT:   br i1 %19, label %20, label %21
// CHECK-EMPTY:
// CHECK-NEXT: 20:                                               ; preds = %11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %11
// CHECK-NEXT:   %22 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %24 = getelementptr inbounds %main.mSpanList, ptr %22, i32 0, i32 1
// CHECK-NEXT:   store ptr %23, ptr %24, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds %main.mspan, ptr %25, i32 0, i32 2
// CHECK-NEXT:   %27 = icmp eq ptr %25, null
// CHECK-NEXT:   br i1 %27, label %28, label %29
// CHECK-EMPTY:
// CHECK-NEXT: 28:                                               ; preds = %21
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 29:                                               ; preds = %21
// CHECK-NEXT:   %30 = load ptr, ptr %26, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds %main.mSpanList, ptr %30, i32 0, i32 1
// CHECK-NEXT:   %32 = icmp eq ptr %25, null
// CHECK-NEXT:   br i1 %32, label %33, label %34
// CHECK-EMPTY:
// CHECK-NEXT: 33:                                               ; preds = %29
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 34:                                               ; preds = %29
// CHECK-NEXT:   %35 = icmp eq ptr %26, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %35)
// CHECK-NEXT:   %36 = icmp eq ptr %30, null
// CHECK-NEXT:   br i1 %36, label %37, label %38
// CHECK-EMPTY:
// CHECK-NEXT: 37:                                               ; preds = %34
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 38:                                               ; preds = %34
// CHECK-NEXT:   %39 = load ptr, ptr %31, align 8
// CHECK-NEXT:   %40 = getelementptr inbounds %main.mspan, ptr %39, i32 0, i32 4
// CHECK-NEXT:   store i64 300, ptr %40, align 8
// CHECK-NEXT:   %41 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %42 = getelementptr inbounds %main.mspan, ptr %41, i32 0, i32 3
// CHECK-NEXT:   %43 = getelementptr inbounds %main.minfo, ptr %42, i32 0, i32 1
// CHECK-NEXT:   store i64 10, ptr %43, align 8
// CHECK-NEXT:   %44 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %45 = getelementptr inbounds %main.mspan, ptr %44, i32 0, i32 3
// CHECK-NEXT:   %46 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds %main.minfo, ptr %45, i32 0, i32 0
// CHECK-NEXT:   store ptr %46, ptr %47, align 8
// CHECK-NEXT:   %48 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %50 = getelementptr inbounds { ptr }, ptr %49, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %50, align 8
// CHECK-NEXT:   %51 = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %49, 1
// CHECK-NEXT:   %52 = getelementptr inbounds %main.mspan, ptr %48, i32 0, i32 5
// CHECK-NEXT:   store { ptr, ptr } %51, ptr %52, align 8
// CHECK-NEXT:   %53 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %54 = getelementptr inbounds %main.mspan, ptr %53, i32 0, i32 0
// CHECK-NEXT:   %55 = icmp eq ptr %53, null
// CHECK-NEXT:   br i1 %55, label %56, label %57
// CHECK-EMPTY:
// CHECK-NEXT: 56:                                               ; preds = %38
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 57:                                               ; preds = %38
// CHECK-NEXT:   %58 = load ptr, ptr %54, align 8
// CHECK-NEXT:   %59 = getelementptr inbounds %main.mspan, ptr %58, i32 0, i32 4
// CHECK-NEXT:   %60 = icmp eq ptr %53, null
// CHECK-NEXT:   br i1 %60, label %61, label %62
// CHECK-EMPTY:
// CHECK-NEXT: 61:                                               ; preds = %57
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 62:                                               ; preds = %57
// CHECK-NEXT:   %63 = icmp eq ptr %54, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %63)
// CHECK-NEXT:   %64 = icmp eq ptr %58, null
// CHECK-NEXT:   br i1 %64, label %65, label %66
// CHECK-EMPTY:
// CHECK-NEXT: 65:                                               ; preds = %62
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 66:                                               ; preds = %62
// CHECK-NEXT:   %67 = load i64, ptr %59, align 8
// CHECK-NEXT:   %68 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds %main.mspan, ptr %68, i32 0, i32 2
// CHECK-NEXT:   %70 = icmp eq ptr %68, null
// CHECK-NEXT:   br i1 %70, label %71, label %72
// CHECK-EMPTY:
// CHECK-NEXT: 71:                                               ; preds = %66
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 72:                                               ; preds = %66
// CHECK-NEXT:   %73 = load ptr, ptr %69, align 8
// CHECK-NEXT:   %74 = getelementptr inbounds %main.mSpanList, ptr %73, i32 0, i32 1
// CHECK-NEXT:   %75 = icmp eq ptr %68, null
// CHECK-NEXT:   br i1 %75, label %76, label %77
// CHECK-EMPTY:
// CHECK-NEXT: 76:                                               ; preds = %72
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 77:                                               ; preds = %72
// CHECK-NEXT:   %78 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %78)
// CHECK-NEXT:   %79 = icmp eq ptr %73, null
// CHECK-NEXT:   br i1 %79, label %80, label %81
// CHECK-EMPTY:
// CHECK-NEXT: 80:                                               ; preds = %77
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 81:                                               ; preds = %77
// CHECK-NEXT:   %82 = load ptr, ptr %74, align 8
// CHECK-NEXT:   %83 = getelementptr inbounds %main.mspan, ptr %82, i32 0, i32 4
// CHECK-NEXT:   %84 = icmp eq ptr %68, null
// CHECK-NEXT:   br i1 %84, label %85, label %86
// CHECK-EMPTY:
// CHECK-NEXT: 85:                                               ; preds = %81
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 86:                                               ; preds = %81
// CHECK-NEXT:   %87 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %87)
// CHECK-NEXT:   %88 = icmp eq ptr %73, null
// CHECK-NEXT:   br i1 %88, label %89, label %90
// CHECK-EMPTY:
// CHECK-NEXT: 89:                                               ; preds = %86
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 90:                                               ; preds = %86
// CHECK-NEXT:   %91 = icmp eq ptr %74, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %91)
// CHECK-NEXT:   %92 = icmp eq ptr %82, null
// CHECK-NEXT:   br i1 %92, label %93, label %94
// CHECK-EMPTY:
// CHECK-NEXT: 93:                                               ; preds = %90
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 94:                                               ; preds = %90
// CHECK-NEXT:   %95 = load i64, ptr %83, align 8
// CHECK-NEXT:   %96 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %97 = getelementptr inbounds %main.mspan, ptr %96, i32 0, i32 3
// CHECK-NEXT:   %98 = getelementptr inbounds %main.minfo, ptr %97, i32 0, i32 1
// CHECK-NEXT:   %99 = icmp eq ptr %96, null
// CHECK-NEXT:   br i1 %99, label %100, label %101
// CHECK-EMPTY:
// CHECK-NEXT: 100:                                              ; preds = %94
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 101:                                              ; preds = %94
// CHECK-NEXT:   %102 = icmp eq ptr %97, null
// CHECK-NEXT:   br i1 %102, label %103, label %104
// CHECK-EMPTY:
// CHECK-NEXT: 103:                                              ; preds = %101
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 104:                                              ; preds = %101
// CHECK-NEXT:   %105 = load i64, ptr %98, align 8
// CHECK-NEXT:   %106 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %107 = getelementptr inbounds %main.mspan, ptr %106, i32 0, i32 3
// CHECK-NEXT:   %108 = getelementptr inbounds %main.minfo, ptr %107, i32 0, i32 0
// CHECK-NEXT:   %109 = icmp eq ptr %106, null
// CHECK-NEXT:   br i1 %109, label %110, label %111
// CHECK-EMPTY:
// CHECK-NEXT: 110:                                              ; preds = %104
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 111:                                              ; preds = %104
// CHECK-NEXT:   %112 = icmp eq ptr %107, null
// CHECK-NEXT:   br i1 %112, label %113, label %114
// CHECK-EMPTY:
// CHECK-NEXT: 113:                                              ; preds = %111
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 114:                                              ; preds = %111
// CHECK-NEXT:   %115 = load ptr, ptr %108, align 8
// CHECK-NEXT:   %116 = getelementptr inbounds %main.mspan, ptr %115, i32 0, i32 4
// CHECK-NEXT:   %117 = icmp eq ptr %106, null
// CHECK-NEXT:   br i1 %117, label %118, label %119
// CHECK-EMPTY:
// CHECK-NEXT: 118:                                              ; preds = %114
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 119:                                              ; preds = %114
// CHECK-NEXT:   %120 = icmp eq ptr %107, null
// CHECK-NEXT:   br i1 %120, label %121, label %122
// CHECK-EMPTY:
// CHECK-NEXT: 121:                                              ; preds = %119
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 122:                                              ; preds = %119
// CHECK-NEXT:   %123 = icmp eq ptr %108, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %123)
// CHECK-NEXT:   %124 = icmp eq ptr %115, null
// CHECK-NEXT:   br i1 %124, label %125, label %126
// CHECK-EMPTY:
// CHECK-NEXT: 125:                                              ; preds = %122
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 126:                                              ; preds = %122
// CHECK-NEXT:   %127 = load i64, ptr %116, align 8
// CHECK-NEXT:   %128 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %129 = getelementptr inbounds %main.mspan, ptr %128, i32 0, i32 5
// CHECK-NEXT:   %130 = icmp eq ptr %128, null
// CHECK-NEXT:   br i1 %130, label %131, label %132
// CHECK-EMPTY:
// CHECK-NEXT: 131:                                              ; preds = %126
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 132:                                              ; preds = %126
// CHECK-NEXT:   %133 = load { ptr, ptr }, ptr %129, align 8
// CHECK-NEXT:   %134 = extractvalue { ptr, ptr } %133, 1
// CHECK-NEXT:   %135 = extractvalue { ptr, ptr } %133, 0
// CHECK-NEXT:   %136 = call i64 %135(ptr %134, i64 -2)
// CHECK-NEXT:   %137 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %138 = getelementptr inbounds %main.mspan, ptr %137, i32 0, i32 3
// CHECK-NEXT:   %139 = getelementptr inbounds %main.minfo, ptr %138, i32 0, i32 0
// CHECK-NEXT:   %140 = icmp eq ptr %137, null
// CHECK-NEXT:   br i1 %140, label %141, label %142
// CHECK-EMPTY:
// CHECK-NEXT: 141:                                              ; preds = %132
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 142:                                              ; preds = %132
// CHECK-NEXT:   %143 = icmp eq ptr %138, null
// CHECK-NEXT:   br i1 %143, label %144, label %145
// CHECK-EMPTY:
// CHECK-NEXT: 144:                                              ; preds = %142
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 145:                                              ; preds = %142
// CHECK-NEXT:   %146 = load ptr, ptr %139, align 8
// CHECK-NEXT:   %147 = getelementptr inbounds %main.mspan, ptr %146, i32 0, i32 5
// CHECK-NEXT:   %148 = icmp eq ptr %137, null
// CHECK-NEXT:   br i1 %148, label %149, label %150
// CHECK-EMPTY:
// CHECK-NEXT: 149:                                              ; preds = %145
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 150:                                              ; preds = %145
// CHECK-NEXT:   %151 = icmp eq ptr %138, null
// CHECK-NEXT:   br i1 %151, label %152, label %153
// CHECK-EMPTY:
// CHECK-NEXT: 152:                                              ; preds = %150
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 153:                                              ; preds = %150
// CHECK-NEXT:   %154 = icmp eq ptr %139, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %154)
// CHECK-NEXT:   %155 = icmp eq ptr %146, null
// CHECK-NEXT:   br i1 %155, label %156, label %157
// CHECK-EMPTY:
// CHECK-NEXT: 156:                                              ; preds = %153
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 157:                                              ; preds = %153
// CHECK-NEXT:   %158 = load { ptr, ptr }, ptr %147, align 8
// CHECK-NEXT:   %159 = extractvalue { ptr, ptr } %158, 1
// CHECK-NEXT:   %160 = extractvalue { ptr, ptr } %158, 0
// CHECK-NEXT:   %161 = call i64 %160(ptr %159, i64 -3)
// CHECK-NEXT:   %162 = call i32 (ptr, ...) @printf(ptr @0, i64 %67, i64 %95, i64 %105, i64 %127, i64 %136, i64 %161)
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
// CHECK-LABEL: define i64 @"main.main$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %main.mspan, ptr %4, i32 0, i32 4
// CHECK-NEXT:   %6 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %7 = icmp eq ptr %6, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = icmp eq ptr %4, null
// CHECK-NEXT:   br i1 %8, label %9, label %10
// CHECK-EMPTY:
// CHECK-NEXT: 9:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %11 = load i64, ptr %5, align 8
// CHECK-NEXT:   %12 = mul i64 %11, %1
// CHECK-NEXT:   ret i64 %12
// CHECK-NEXT: }
	c.Printf(c.Str("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}
