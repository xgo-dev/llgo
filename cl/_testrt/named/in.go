// LITTEST darwin/arm64 linux/amd64
// Scope: arch (arm64, amd64)
package main

import _ "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

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

// The recursive named structure is rooted through a slot captured by check.
// The six printf values follow the source field paths, including two calls to
// the same stored closure reached directly and through info.span.
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
	printf(cstr("%d %d %d %d %d %d\n"), m.next.value, m.list.last.value, m.info.info,
		m.info.span.value, m.check(-2), m.info.span.check(-3))
}

// The closure captures the root slot in its environment, is stored as a
// {code, env} pair, and both direct and info.span paths recover that pair.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[ROOTSLOT:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: %[[ROOT:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK: store ptr %[[ROOT]], ptr %[[ROOTSLOT]], align 8
// CHECK: %[[ENV:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: %[[ENVSLOT:[0-9]+]] = getelementptr inbounds nuw { ptr }, ptr %[[ENV]], i32 0, i32 0
// CHECK: store ptr %[[ROOTSLOT]], ptr %[[ENVSLOT]], align 8
// CHECK: %[[CLOSURE:[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %[[ENV]], 1
// CHECK: %[[CHECKFIELD:[0-9]+]] = getelementptr inbounds nuw %main.mspan, ptr %{{[0-9]+}}, i32 0, i32 5
// CHECK: store { ptr, ptr } %[[CLOSURE]], ptr %[[CHECKFIELD]], align 8
// CHECK: %[[PAIR1:[0-9]+]] = load { ptr, ptr }, ptr %{{[0-9]+}}, align 8
// CHECK: %[[ENV1:[0-9]+]] = extractvalue { ptr, ptr } %[[PAIR1]], 1
// CHECK: %[[CODE1:[0-9]+]] = extractvalue { ptr, ptr } %[[PAIR1]], 0
// CHECK: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %[[CODE1]])
// ARM64: %[[FIRST:[0-9]+]] = call i64 %__llgo_funcval_code(ptr swiftself %[[ENV1]], i64 -2)
// AMD64: %[[FIRST:[0-9]+]] = call i64 %__llgo_funcval_code(ptr nest %[[ENV1]], i64 -2)
// CHECK: getelementptr inbounds nuw %main.minfo, ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK: getelementptr inbounds nuw %main.mspan, ptr %{{[0-9]+}}, i32 0, i32 5
// CHECK: %[[PAIR2:[0-9]+]] = load { ptr, ptr }, ptr %{{[0-9]+}}, align 8
// CHECK: %[[ENV2:[0-9]+]] = extractvalue { ptr, ptr } %[[PAIR2]], 1
// CHECK: %[[CODE2:[0-9]+]] = extractvalue { ptr, ptr } %[[PAIR2]], 0
// CHECK: %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %[[CODE2]])
// ARM64: %[[SECOND:[0-9]+]] = call i64 %__llgo_funcval_code1(ptr swiftself %[[ENV2]], i64 -3)
// AMD64: %[[SECOND:[0-9]+]] = call i64 %__llgo_funcval_code1(ptr nest %[[ENV2]], i64 -3)
// CHECK: call i32 (ptr, ...) @printf({{.*}}i64 %[[FIRST]], i64 %[[SECOND]])

// CHECK-LABEL: define i64 @"main.main$1"(
// ARM64-SAME: ptr swiftself %[[CENV:[0-9]+]], i64 %[[ARG:[0-9]+]]){{.*}} {
// AMD64-SAME: ptr nest %[[CENV:[0-9]+]], i64 %[[ARG:[0-9]+]]){{.*}} {
// CHECK: %[[CAPTURE:[0-9]+]] = load { ptr }, ptr %[[CENV]], align 8
// CHECK: %[[CAPTUREDSLOT:[0-9]+]] = extractvalue { ptr } %[[CAPTURE]], 0
// CHECK: %[[CAPTUREDROOT:[0-9]+]] = load ptr, ptr %[[CAPTUREDSLOT]], align 8
// CHECK: %[[VALUEFIELD:[0-9]+]] = getelementptr inbounds nuw %main.mspan, ptr %[[CAPTUREDROOT]], i32 0, i32 4
// CHECK: %[[VALUE:[0-9]+]] = load i64, ptr %[[VALUEFIELD]], align 8
// CHECK: %[[PRODUCT:[0-9]+]] = mul i64 %[[VALUE]], %[[ARG]]
// CHECK: ret i64 %[[PRODUCT]]
