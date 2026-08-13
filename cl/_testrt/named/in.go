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
// The recursive named structure is rooted through a slot captured by check.
// CHECK: [[M_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[M:%.*]] = call ptr @"{{.*}}AllocZ"(i64 64)
// CHECK-NEXT: store ptr [[M]], ptr [[M_SLOT]]
// CHECK: [[M_FOR_VALUE:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[M_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_VALUE]], i32 0, i32 4
// CHECK-NEXT: store i64 100, ptr [[M_VALUE_PTR]]
// CHECK: [[M_FOR_NEXT:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK: [[NEXT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 64)
// CHECK: [[M_NEXT_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_NEXT]], i32 0, i32 0
// CHECK-NEXT: store ptr [[NEXT]], ptr [[M_NEXT_PTR]]
// CHECK: [[M_FOR_NEXT_VALUE:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[NEXT_FOR_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_NEXT_VALUE]], i32 0, i32 0
// CHECK-NEXT: [[NEXT_FOR_VALUE:%.*]] = load ptr, ptr [[NEXT_FOR_VALUE_PTR]]
// CHECK-NEXT: [[NEXT_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[NEXT_FOR_VALUE]], i32 0, i32 4
// CHECK-NEXT: store i64 200, ptr [[NEXT_VALUE_PTR]]
// CHECK: [[M_FOR_LIST:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK: [[LIST:%.*]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK: [[M_LIST_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_LIST]], i32 0, i32 2
// CHECK-NEXT: store ptr [[LIST]], ptr [[M_LIST_PTR]]
// CHECK: [[M_FOR_LAST:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK: [[M_LIST_FOR_LAST_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_LAST]], i32 0, i32 2
// CHECK-NEXT: [[LIST_FOR_LAST:%.*]] = load ptr, ptr [[M_LIST_FOR_LAST_PTR]]
// CHECK: [[LAST:%.*]] = call ptr @"{{.*}}AllocZ"(i64 64)
// CHECK: [[LIST_LAST_PTR:%.*]] = getelementptr inbounds %main.mSpanList, ptr [[LIST_FOR_LAST]], i32 0, i32 1
// CHECK-NEXT: store ptr [[LAST]], ptr [[LIST_LAST_PTR]]
// CHECK: [[M_FOR_LAST_VALUE:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[M_LIST_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_LAST_VALUE]], i32 0, i32 2
// CHECK-NEXT: [[LIST_FOR_VALUE:%.*]] = load ptr, ptr [[M_LIST_VALUE_PTR]]
// CHECK-NEXT: [[LIST_LAST_VALUE_PTR:%.*]] = getelementptr inbounds %main.mSpanList, ptr [[LIST_FOR_VALUE]], i32 0, i32 1
// CHECK-NEXT: [[LAST_FOR_VALUE:%.*]] = load ptr, ptr [[LIST_LAST_VALUE_PTR]]
// CHECK-NEXT: [[LAST_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[LAST_FOR_VALUE]], i32 0, i32 4
// CHECK-NEXT: store i64 300, ptr [[LAST_VALUE_PTR]]
// CHECK: [[M_FOR_INFO:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[M_INFO:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_INFO]], i32 0, i32 3
// CHECK-NEXT: [[INFO_VALUE_PTR:%.*]] = getelementptr inbounds %main.minfo, ptr [[M_INFO]], i32 0, i32 1
// CHECK-NEXT: store i64 10, ptr [[INFO_VALUE_PTR]]
// CHECK: [[M_FOR_INFO_SPAN:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[M_INFO_FOR_SPAN:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_INFO_SPAN]], i32 0, i32 3
// CHECK-NEXT: [[M_SELF:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK: [[INFO_SPAN_PTR:%.*]] = getelementptr inbounds %main.minfo, ptr [[M_INFO_FOR_SPAN]], i32 0, i32 0
// CHECK-NEXT: store ptr [[M_SELF]], ptr [[INFO_SPAN_PTR]]
// CHECK: [[M_FOR_CHECK:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK: [[CHECK_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[M_SLOT]], ptr {{%.*}}
// CHECK: [[CHECK_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[CHECK_ENV]], 1
// CHECK: [[CHECK_FIELD:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_CHECK]], i32 0, i32 5
// CHECK-NEXT: store { ptr, ptr } [[CHECK_CLOSURE]], ptr [[CHECK_FIELD]]
// The six printf values follow the source field paths, including two calls to
// the same stored closure reached directly and through info.span.
// CHECK: [[M_FOR_PRINT_NEXT:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[PRINT_NEXT_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_PRINT_NEXT]], i32 0, i32 0
// CHECK-NEXT: [[PRINT_NEXT:%.*]] = load ptr, ptr [[PRINT_NEXT_PTR]]
// CHECK-NEXT: [[PRINT_NEXT_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[PRINT_NEXT]], i32 0, i32 4
// CHECK-NEXT: [[PRINT_NEXT_VALUE:%.*]] = load i64, ptr [[PRINT_NEXT_VALUE_PTR]]
// CHECK: [[M_FOR_PRINT_LIST:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[PRINT_LIST_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_PRINT_LIST]], i32 0, i32 2
// CHECK-NEXT: [[PRINT_LIST:%.*]] = load ptr, ptr [[PRINT_LIST_PTR]]
// CHECK-NEXT: [[PRINT_LAST_PTR:%.*]] = getelementptr inbounds %main.mSpanList, ptr [[PRINT_LIST]], i32 0, i32 1
// CHECK-NEXT: [[PRINT_LAST:%.*]] = load ptr, ptr [[PRINT_LAST_PTR]]
// CHECK-NEXT: [[PRINT_LAST_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[PRINT_LAST]], i32 0, i32 4
// CHECK-NEXT: [[PRINT_LAST_VALUE:%.*]] = load i64, ptr [[PRINT_LAST_VALUE_PTR]]
// CHECK: [[M_FOR_PRINT_INFO:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[PRINT_INFO:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_PRINT_INFO]], i32 0, i32 3
// CHECK-NEXT: [[PRINT_INFO_VALUE_PTR:%.*]] = getelementptr inbounds %main.minfo, ptr [[PRINT_INFO]], i32 0, i32 1
// CHECK-NEXT: [[PRINT_INFO_VALUE:%.*]] = load i64, ptr [[PRINT_INFO_VALUE_PTR]]
// CHECK: [[M_FOR_PRINT_SPAN:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[PRINT_INFO2:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_PRINT_SPAN]], i32 0, i32 3
// CHECK-NEXT: [[PRINT_INFO_SPAN_PTR:%.*]] = getelementptr inbounds %main.minfo, ptr [[PRINT_INFO2]], i32 0, i32 0
// CHECK-NEXT: [[PRINT_INFO_SPAN:%.*]] = load ptr, ptr [[PRINT_INFO_SPAN_PTR]]
// CHECK-NEXT: [[PRINT_INFO_SPAN_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[PRINT_INFO_SPAN]], i32 0, i32 4
// CHECK-NEXT: [[PRINT_INFO_SPAN_VALUE:%.*]] = load i64, ptr [[PRINT_INFO_SPAN_VALUE_PTR]]
// CHECK: [[M_FOR_DIRECT_CHECK:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[DIRECT_CHECK_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_DIRECT_CHECK]], i32 0, i32 5
// CHECK-NEXT: [[DIRECT_CHECK:%.*]] = load { ptr, ptr }, ptr [[DIRECT_CHECK_PTR]]
// CHECK: [[DIRECT_ENV:%.*]] = extractvalue { ptr, ptr } [[DIRECT_CHECK]], 1
// CHECK: [[DIRECT_FN:%.*]] = extractvalue { ptr, ptr } [[DIRECT_CHECK]], 0
// CHECK: [[DIRECT_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[DIRECT_FN]])
// CHECK-NEXT: [[DIRECT_RESULT:%.*]] = call i64 [[DIRECT_CODE]](ptr {{(nest|swiftself)}} [[DIRECT_ENV]], i64 -2)
// CHECK: [[M_FOR_INDIRECT_CHECK:%.*]] = load ptr, ptr [[M_SLOT]]
// CHECK-NEXT: [[INDIRECT_INFO:%.*]] = getelementptr inbounds %main.mspan, ptr [[M_FOR_INDIRECT_CHECK]], i32 0, i32 3
// CHECK-NEXT: [[INDIRECT_SPAN_PTR:%.*]] = getelementptr inbounds %main.minfo, ptr [[INDIRECT_INFO]], i32 0, i32 0
// CHECK-NEXT: [[INDIRECT_SPAN:%.*]] = load ptr, ptr [[INDIRECT_SPAN_PTR]]
// CHECK-NEXT: [[INDIRECT_CHECK_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[INDIRECT_SPAN]], i32 0, i32 5
// CHECK-NEXT: [[INDIRECT_CHECK:%.*]] = load { ptr, ptr }, ptr [[INDIRECT_CHECK_PTR]]
// CHECK: [[INDIRECT_ENV:%.*]] = extractvalue { ptr, ptr } [[INDIRECT_CHECK]], 1
// CHECK: [[INDIRECT_FN:%.*]] = extractvalue { ptr, ptr } [[INDIRECT_CHECK]], 0
// CHECK: [[INDIRECT_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[INDIRECT_FN]])
// CHECK-NEXT: [[INDIRECT_RESULT:%.*]] = call i64 [[INDIRECT_CODE]](ptr {{(nest|swiftself)}} [[INDIRECT_ENV]], i64 -3)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[PRINT_NEXT_VALUE]], i64 [[PRINT_LAST_VALUE]], i64 [[PRINT_INFO_VALUE]], i64 [[PRINT_INFO_SPAN_VALUE]], i64 [[DIRECT_RESULT]], i64 [[INDIRECT_RESULT]])
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
	// CHECK-LABEL: define i64 @"main.main$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
	// CHECK: [[NAMED_ENV:%.*]] = load { ptr }, ptr %0
	// CHECK-NEXT: [[NAMED_M_PTR:%.*]] = extractvalue { ptr } [[NAMED_ENV]], 0
	// CHECK-NEXT: [[NAMED_M:%.*]] = load ptr, ptr [[NAMED_M_PTR]]
	// CHECK: [[NAMED_VALUE_PTR:%.*]] = getelementptr inbounds %main.mspan, ptr [[NAMED_M]], i32 0, i32 4
	// CHECK-NEXT: [[NAMED_VALUE:%.*]] = load i64, ptr [[NAMED_VALUE_PTR]]
	// CHECK-NEXT: [[NAMED_RESULT:%.*]] = mul i64 [[NAMED_VALUE]], %1
	// CHECK-NEXT: ret i64 [[NAMED_RESULT]]
	c.Printf(c.Str("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}
