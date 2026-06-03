// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	q "github.com/goplus/llgo/cl/_testrt/qsortfn/qsort"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort1a"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort1b"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort2a"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort2b"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort3a"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort3b"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort4a"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort4b"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort5a"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/qsortfn.sort5b"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	sort1a()
	sort1b()
	sort2a()
	sort2b()
	sort3a()
	sort3b()
	sort4a()
	sort4b()
	sort5a()
	sort5b()
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort1a"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @0)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort1a$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @1, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort1a() {
	c.Printf(c.Str("Comp => Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort1a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort1b"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @2)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort1b$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @3, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort1b() {
	c.Printf(c.Str("fn => Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort1b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort2a"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @4)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort2a$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @5, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort2a() {
	c.Printf(c.Str("Comp => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort2a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort2b"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @6)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort2b$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @7, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort2b() {
	c.Printf(c.Str("fn => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort2b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort3a"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort3a$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @9, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort3a() {
	c.Printf(c.Str("qsort.Comp => qsort.Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort3a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort3b"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @10)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort3b$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @11, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort3b() {
	c.Printf(c.Str("fn => qsort.Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort3b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort4a"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @12)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort4a$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @13, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort4a() {
	c.Printf(c.Str("qsort.Comp => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort4a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort4b"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @14)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort4b$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @15, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort4b() {
	c.Printf(c.Str("Comp => qsort.fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort4b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort5a"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort5a$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @17, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort5a() {
	c.Printf(c.Str("qsort.Comp => Comp()\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort5a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), Comp(fn))
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsortfn.sort5b"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @18)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %1, i64 1
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %1, i64 2
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %1, i64 3
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %1, i64 4
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   store i64 8, ptr %3, align 8
// CHECK-NEXT:   store i64 23, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   store i64 7, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   call void @qsort(ptr %7, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsortfn.sort5b$1")
// CHECK-NEXT:   %8 = load [5 x i64], ptr %1, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %9 = phi i64 [ -1, %_llgo_0 ], [ %10, %_llgo_2 ]
// CHECK-NEXT:   %10 = add i64 %9, 1
// CHECK-NEXT:   %11 = icmp slt i64 %10, 5
// CHECK-NEXT:   br i1 %11, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %12 = icmp slt i64 %10, 0
// CHECK-NEXT:   %13 = icmp uge i64 %10, 5
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i64, ptr %1, i64 %10
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = call i32 (ptr, ...) @printf(ptr @19, i64 %16)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func sort5b() {
	c.Printf(c.Str("Comp => qsort.Comp()\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsortfn.sort5b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	//
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), q.Comp(fn))
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

//llgo:type C
type Comp func(a, b c.Pointer) c.Int

//go:linkname qsort C.qsort
func qsort(base c.Pointer, count, elem uintptr, compar Comp)

//go:linkname qsort2 C.qsort
func qsort2(base c.Pointer, count, elem uintptr, compar func(a, b c.Pointer) c.Int)
