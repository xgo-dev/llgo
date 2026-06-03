// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/result.add$1"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func add() func(int, int) int {
	return func(x, y int) int {
		return x + y
	}
}

// CHECK-LABEL: define { { ptr, ptr }, i64 } @"{{.*}}/cl/_testrt/result.add2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret { { ptr, ptr }, i64 } { { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/result.add2$1", ptr null }, i64 1 }
// CHECK-NEXT: }
func add2() (func(int, int) int, int) {
	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/result.add2$1"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = add i64 %0, %1
	// CHECK-NEXT:   ret i64 %2
	// CHECK-NEXT: }
	return func(x, y int) int {
		return x + y
	}, 1
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/result.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call { ptr, ptr } @"{{.*}}/cl/_testrt/result.main$1"()
// CHECK-NEXT:   %1 = extractvalue { ptr, ptr } %0, 1
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %0, 0
// CHECK-NEXT:   %3 = call i64 %2(ptr %1, i64 100, i64 200)
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @printf(ptr @0, i64 %3)
// CHECK-NEXT:   %5 = call { ptr, ptr } @"{{.*}}/cl/_testrt/result.add"()
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %5, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %5, 0
// CHECK-NEXT:   %8 = call i64 %7(ptr %6, i64 100, i64 200)
// CHECK-NEXT:   %9 = call i32 (ptr, ...) @printf(ptr @1, i64 %8)
// CHECK-NEXT:   %10 = call { { ptr, ptr }, i64 } @"{{.*}}/cl/_testrt/result.add2"()
// CHECK-NEXT:   %11 = extractvalue { { ptr, ptr }, i64 } %10, 0
// CHECK-NEXT:   %12 = extractvalue { { ptr, ptr }, i64 } %10, 1
// CHECK-NEXT:   %13 = call { ptr, ptr } @"{{.*}}/cl/_testrt/result.add"()
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %13, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   %16 = call i64 %15(ptr %14, i64 100, i64 200)
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @2, i64 %16, i64 %12)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	// CHECK-LABEL: define { ptr, ptr } @"{{.*}}/cl/_testrt/result.main$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/result.main$1$1", ptr null }
	// CHECK-NEXT: }
	fn := func() func(int, int) int {
		// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/result.main$1$1"(i64 %0, i64 %1){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %2 = add i64 %0, %1
		// CHECK-NEXT:   ret i64 %2
		// CHECK-NEXT: }
		return func(x, y int) int {
			return x + y
		}
	}()
	c.Printf(c.Str("%d\n"), fn(100, 200))
	c.Printf(c.Str("%d\n"), add()(100, 200))
	fn, n := add2()
	c.Printf(c.Str("%d %d\n"), add()(100, 200), n)
}

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/result.add$1"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testrt/result.add$1"(i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/result.add2$1"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testrt/result.add2$1"(i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/result.main$1$1"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testrt/result.main$1$1"(i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }
