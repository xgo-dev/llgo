// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/sync/atomic"
)

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 8, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   store atomic i64 100, ptr %.stack seq_cst, align 8
// ESCAPE-NEXT:   %0 = load atomic i64, ptr %.stack seq_cst, align 8
// ESCAPE-NEXT:   %1 = call i32 (ptr, ...) @printf(ptr @0, i64 %0)
// ESCAPE-NEXT:   %2 = atomicrmw add ptr %.stack, i64 1 seq_cst, align 8
// ESCAPE-NEXT:   %3 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   %4 = call i32 (ptr, ...) @printf(ptr @1, i64 %2, i64 %3)
// ESCAPE-NEXT:   %5 = cmpxchg ptr %.stack, i64 100, i64 102 seq_cst seq_cst, align 8
// ESCAPE-NEXT:   %6 = extractvalue { i64, i1 } %5, 0
// ESCAPE-NEXT:   %7 = extractvalue { i64, i1 } %5, 1
// ESCAPE-NEXT:   %8 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   %9 = call i32 (ptr, ...) @printf(ptr @2, i64 %6, i64 %8)
// ESCAPE-NEXT:   %10 = cmpxchg ptr %.stack, i64 101, i64 102 seq_cst seq_cst, align 8
// ESCAPE-NEXT:   %11 = extractvalue { i64, i1 } %10, 0
// ESCAPE-NEXT:   %12 = extractvalue { i64, i1 } %10, 1
// ESCAPE-NEXT:   %13 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   %14 = call i32 (ptr, ...) @printf(ptr @3, i64 %11, i64 %13)
// ESCAPE-NEXT:   %15 = atomicrmw sub ptr %.stack, i64 1 seq_cst, align 8
// ESCAPE-NEXT:   %16 = load i64, ptr %.stack, align 8
// ESCAPE-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @4, i64 %15, i64 %16)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	var v int64

	// CHECK: store atomic i64 100, ptr %0 seq_cst, align 8
	atomic.Store(&v, 100)
	// CHECK: %1 = load atomic i64, ptr %0 seq_cst, align 8
	c.Printf(c.Str("store: %ld\n"), atomic.Load(&v))
	// CHECK: %3 = atomicrmw add ptr %0, i64 1 seq_cst, align 8
	ret := atomic.Add(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)

	// CHECK: %6 = cmpxchg ptr %0, i64 100, i64 102 seq_cst seq_cst, align 8
	ret, _ = atomic.CompareAndExchange(&v, 100, 102)
	c.Printf(c.Str("ret: %ld vs 100, v: %ld\n"), ret, v)

	// CHECK: %11 = cmpxchg ptr %0, i64 101, i64 102 seq_cst seq_cst, align 8
	ret, _ = atomic.CompareAndExchange(&v, 101, 102)
	c.Printf(c.Str("ret: %ld vs 101, v: %ld\n"), ret, v)

	// CHECK: %16 = atomicrmw sub ptr %0, i64 1 seq_cst, align 8
	ret = atomic.Sub(&v, 1)
	c.Printf(c.Str("ret: %ld, v: %ld\n"), ret, v)
}
