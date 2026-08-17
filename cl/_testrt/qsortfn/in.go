// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	q "github.com/xgo-dev/llgo/cl/_testrt/qsortfn/qsort"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.sort1a()
// CHECK:   call void @main.sort1b()
// CHECK:   call void @main.sort2a()
// CHECK:   call void @main.sort2b()
// CHECK:   call void @main.sort3a()
// CHECK:   call void @main.sort3b()
// CHECK:   call void @main.sort4a()
// CHECK:   call void @main.sort4b()
// CHECK:   call void @main.sort5a()
// CHECK:   call void @main.sort5b()
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

// CHECK-LABEL: define void @main.sort1a(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort1a$1")
func sort1a() {
	c.Printf(c.Str("Comp => Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort1a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S1A_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S1A_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S1A_DIFF:%[0-9]+]] = sub i64 [[S1A_A]], [[S1A_B]]
	// CHECK-NEXT: [[S1A_RESULT:%[0-9]+]] = trunc i64 [[S1A_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S1A_RESULT]]
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort1b(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort1b$1")
func sort1b() {
	c.Printf(c.Str("fn => Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort1b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S1B_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S1B_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S1B_DIFF:%[0-9]+]] = sub i64 [[S1B_A]], [[S1B_B]]
	// CHECK-NEXT: [[S1B_RESULT:%[0-9]+]] = trunc i64 [[S1B_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S1B_RESULT]]
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort2a(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort2a$1")
func sort2a() {
	c.Printf(c.Str("Comp => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort2a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S2A_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S2A_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S2A_DIFF:%[0-9]+]] = sub i64 [[S2A_A]], [[S2A_B]]
	// CHECK-NEXT: [[S2A_RESULT:%[0-9]+]] = trunc i64 [[S2A_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S2A_RESULT]]
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort2b(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort2b$1")
func sort2b() {
	c.Printf(c.Str("fn => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort2b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S2B_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S2B_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S2B_DIFF:%[0-9]+]] = sub i64 [[S2B_A]], [[S2B_B]]
	// CHECK-NEXT: [[S2B_RESULT:%[0-9]+]] = trunc i64 [[S2B_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S2B_RESULT]]
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort3a(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort3a$1")
func sort3a() {
	c.Printf(c.Str("qsort.Comp => qsort.Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort3a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S3A_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S3A_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S3A_DIFF:%[0-9]+]] = sub i64 [[S3A_A]], [[S3A_B]]
	// CHECK-NEXT: [[S3A_RESULT:%[0-9]+]] = trunc i64 [[S3A_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S3A_RESULT]]
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort3b(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort3b$1")
func sort3b() {
	c.Printf(c.Str("fn => qsort.Comp\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort3b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S3B_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S3B_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S3B_DIFF:%[0-9]+]] = sub i64 [[S3B_A]], [[S3B_B]]
	// CHECK-NEXT: [[S3B_RESULT:%[0-9]+]] = trunc i64 [[S3B_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S3B_RESULT]]
	var fn = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort4a(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort4a$1")
func sort4a() {
	c.Printf(c.Str("qsort.Comp => fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort4a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S4A_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S4A_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S4A_DIFF:%[0-9]+]] = sub i64 [[S4A_A]], [[S4A_B]]
	// CHECK-NEXT: [[S4A_RESULT:%[0-9]+]] = trunc i64 [[S4A_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S4A_RESULT]]
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort4b(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort4b$1")
func sort4b() {
	c.Printf(c.Str("Comp => qsort.fn\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort4b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S4B_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S4B_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S4B_DIFF:%[0-9]+]] = sub i64 [[S4B_A]], [[S4B_B]]
	// CHECK-NEXT: [[S4B_RESULT:%[0-9]+]] = trunc i64 [[S4B_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S4B_RESULT]]
	var fn Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	q.Qsort2(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), fn)
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort5a(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort5a$1")
func sort5a() {
	c.Printf(c.Str("qsort.Comp => Comp()\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort5a$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S5A_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S5A_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S5A_DIFF:%[0-9]+]] = sub i64 [[S5A_A]], [[S5A_B]]
	// CHECK-NEXT: [[S5A_RESULT:%[0-9]+]] = trunc i64 [[S5A_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S5A_RESULT]]
	var fn q.Comp = func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), Comp(fn))
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}

// CHECK-LABEL: define void @main.sort5b(){{.*}} {
// CHECK:   call void @qsort(ptr %{{[0-9]+}}, i64 5, i64 8, ptr @"main.sort5b$1")
func sort5b() {
	c.Printf(c.Str("Comp => qsort.Comp()\n"))
	a := [...]int{100, 8, 23, 2, 7}
	// CHECK-LABEL: define i32 @"main.sort5b$1"(ptr %0, ptr %1){{.*}} {
	// CHECK: [[S5B_A:%[0-9]+]] = load i64, ptr %0
	// CHECK-NEXT: [[S5B_B:%[0-9]+]] = load i64, ptr %1
	// CHECK-NEXT: [[S5B_DIFF:%[0-9]+]] = sub i64 [[S5B_A]], [[S5B_B]]
	// CHECK-NEXT: [[S5B_RESULT:%[0-9]+]] = trunc i64 [[S5B_DIFF]] to i32
	// CHECK-NEXT: ret i32 [[S5B_RESULT]]
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
