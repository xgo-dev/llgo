// LITTEST
package main

type My[T any] struct {
	fn   func(n T)
	next *My[T]
}

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[OUTER:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 24)
	// CHECK-NEXT: [[NEXT_FIELD:%[0-9]+]] = getelementptr inbounds %"main.My[int]", ptr [[OUTER]], i32 0, i32 1
	// CHECK-NEXT: [[INNER:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 24)
	// CHECK-NEXT: [[FN_FIELD:%[0-9]+]] = getelementptr inbounds %"main.My[int]", ptr [[INNER]], i32 0, i32 0
	// CHECK-NEXT: store { ptr, ptr } { ptr @"main.main$1", ptr null }, ptr [[FN_FIELD]]
	// CHECK-NEXT: store ptr [[INNER]], ptr [[NEXT_FIELD]]
	// CHECK-NEXT: [[NEXT_FIELD_USE:%[0-9]+]] = getelementptr inbounds %"main.My[int]", ptr [[OUTER]], i32 0, i32 1
	// CHECK-NEXT: [[NEXT:%[0-9]+]] = load ptr, ptr [[NEXT_FIELD_USE]]
	// CHECK-NEXT: [[FN_FIELD_USE:%[0-9]+]] = getelementptr inbounds %"main.My[int]", ptr [[NEXT]], i32 0, i32 0
	// CHECK-NEXT: [[FN:%[0-9]+]] = load { ptr, ptr }, ptr [[FN_FIELD_USE]]
	// CHECK-NEXT: [[FN_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[FN]], 1
	// CHECK-NEXT: [[FN_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[FN]], 0
	// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[FN_RAW_CODE]])
	// CHECK-NEXT: call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[FN_ENV]], i64 100)
	m := &My[int]{next: &My[int]{fn: func(n int) { println(n) }}}
	m.next.fn(100)
}

// CHECK-LABEL: define void @"main.main$1"(i64 %0){{.*}} {
// CHECK: call void @"{{.*}}PrintInt"(i64 %0)
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT: ret void
