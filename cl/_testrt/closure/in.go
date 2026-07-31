// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"main.main$1"(i64 100, i64 200)
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"main.main$2", ptr null }, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"main.main$3", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %3, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %3, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	// CHECK-LABEL: define void @"main.main$1"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = call i32 (ptr, ...) @printf(ptr @0, i64 %0, i64 %1)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	func(n1, n2 int) {
		c.Printf(c.Str("%d %d\n"), n1, n2)
	}(100, 200)

	// CHECK-LABEL: define void @"main.main$2"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = call i32 (ptr, ...) @printf(ptr @1, i64 %0, i64 %1)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	fn1 := func(n1, n2 int) {
		c.Printf(c.Str("%d %d\n"), n1, n2)
	}

	// CHECK-LABEL: define void @"main.main$3"(ptr {{(nest|swiftself)}} %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   %3 = load { ptr, ptr }, ptr %2, align 8
	// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %3, 1
	// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %3, 0
	// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
	// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, i64 100, i64 200)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	fn2 := func() {
		fn1(100, 200)
	}
	fn2()
}
