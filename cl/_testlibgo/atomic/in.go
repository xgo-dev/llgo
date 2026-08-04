// LITTEST
package main

import (
	"sync/atomic"
)

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 8, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   store atomic i64 100, ptr %.stack seq_cst, align 8
// ESCAPE-NEXT:   %0 = load atomic i64, ptr %.stack seq_cst, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %1 = atomicrmw add ptr %.stack, i64 1 seq_cst, align 8
// ESCAPE-NEXT:   %2 = add i64 %1, 1
// ESCAPE-NEXT:   %3 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 2 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %4 = cmpxchg ptr %.stack, i64 100, i64 102 seq_cst seq_cst, align 8
// ESCAPE-NEXT:   %5 = extractvalue { i64, i1 } %4, 1
// ESCAPE-NEXT:   %6 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 4 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %5)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 2 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %6)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %7 = cmpxchg ptr %.stack, i64 101, i64 102 seq_cst seq_cst, align 8
// ESCAPE-NEXT:   %8 = extractvalue { i64, i1 } %7, 1
// ESCAPE-NEXT:   %9 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 4 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %8)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 2 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %10 = atomicrmw add ptr %.stack, i64 -1 seq_cst, align 8
// ESCAPE-NEXT:   %11 = add i64 %10, -1
// ESCAPE-NEXT:   %12 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 2 })
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	var v int64

	// CHECK: store atomic i64 100, ptr %0 seq_cst, align 8
	// CHECK: %1 = load atomic i64, ptr %0 seq_cst, align 8
	atomic.StoreInt64(&v, 100)
	println("store:", atomic.LoadInt64(&v))

	// CHECK: %2 = atomicrmw add ptr %0, i64 1 seq_cst, align 8
	// CHECK: %3 = add i64 %2, 1
	// CHECK: %4 = load i64, ptr %0, align 8
	ret := atomic.AddInt64(&v, 1)
	println("ret:", ret, "v:", v)

	// CHECK: %5 = cmpxchg ptr %0, i64 100, i64 102 seq_cst seq_cst, align 8
	// CHECK: %6 = extractvalue { i64, i1 } %5, 1
	// CHECK: %7 = load i64, ptr %0, align 8
	swp := atomic.CompareAndSwapInt64(&v, 100, 102)
	println("swp:", swp, "v:", v)

	// CHECK: %8 = cmpxchg ptr %0, i64 101, i64 102 seq_cst seq_cst, align 8
	// CHECK: %9 = extractvalue { i64, i1 } %8, 1
	// CHECK: %10 = load i64, ptr %0, align 8
	swp = atomic.CompareAndSwapInt64(&v, 101, 102)
	println("swp:", swp, "v:", v)

	// CHECK: %11 = atomicrmw add ptr %0, i64 -1 seq_cst, align 8
	// CHECK: %12 = add i64 %11, -1
	// CHECK: %13 = load i64, ptr %0, align 8
	ret = atomic.AddInt64(&v, -1)
	println("ret:", ret, "v:", v)
}
