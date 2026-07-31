// LITTEST
package main

type My[T any] struct {
	fn   func(n T)
	next *My[T]
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK:  %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
	// CHECK-NEXT:  %1 = getelementptr inbounds %"main.My[int]", ptr %0, i32 0, i32 1
	// CHECK-NEXT:  %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
	// CHECK-NEXT:  %3 = getelementptr inbounds %"main.My[int]", ptr %2, i32 0, i32 0
	// CHECK-NEXT:  store { ptr, ptr } { ptr @"main.main$1", ptr null }, ptr %3, align 8
	// CHECK-NEXT:  store ptr %2, ptr %1, align 8
	// CHECK-NEXT:  %4 = getelementptr inbounds %"main.My[int]", ptr %0, i32 0, i32 1
	// CHECK-NEXT:  %5 = load ptr, ptr %4, align 8
	// CHECK-NEXT:  %6 = getelementptr inbounds %"main.My[int]", ptr %5, i32 0, i32 0
	// CHECK-NEXT:  %7 = load { ptr, ptr }, ptr %6, align 8
	// CHECK-NEXT:  %8 = extractvalue { ptr, ptr } %7, 1
	// CHECK-NEXT:  %9 = extractvalue { ptr, ptr } %7, 0
	// CHECK-NEXT:  %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %9)
	// CHECK-NEXT:  call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %8, i64 100)
	// CHECK-NEXT:  ret void
	// CHECK-NEXT:}
	m := &My[int]{next: &My[int]{fn: func(n int) { println(n) }}}
	m.next.fn(100)
}

// CHECK-LABEL: define void @"main.main$1"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
// CHECK-NEXT:   ret void
