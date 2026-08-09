// LITTEST
package main

type My[T any] struct {
	fn   func(n T)
	next *My[T]
}

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 24, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 24, i1 false)
// ESCAPE-NEXT:   %0 = getelementptr inbounds %"main.My[int]", ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// ESCAPE-NEXT:   %2 = getelementptr inbounds %"main.My[int]", ptr %1, i32 0, i32 0
// ESCAPE-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.main.main$1", ptr null }, ptr %2, align 8
// ESCAPE-NEXT:   store ptr %1, ptr %0, align 8
// ESCAPE-NEXT:   %3 = getelementptr inbounds %"main.My[int]", ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   %4 = load ptr, ptr %3, align 8
// ESCAPE-NEXT:   %5 = getelementptr inbounds %"main.My[int]", ptr %4, i32 0, i32 0
// ESCAPE-NEXT:   %6 = load { ptr, ptr }, ptr %5, align 8
// ESCAPE-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// ESCAPE-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// ESCAPE-NEXT:   call void %8(ptr %7, i64 100)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

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
