// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testrt/intgen.genInts"(i64 %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.MakeSlice"(i64 %0, i64 %0, i64 4)
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %2, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %4 = phi i64 [ -1, %_llgo_0 ], [ %5, %_llgo_2 ]
// CHECK-NEXT:   %5 = add i64 %4, 1
// CHECK-NEXT:   %6 = icmp slt i64 %5, %3
// CHECK-NEXT:   br i1 %6, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   %9 = call i32 %8(ptr %7)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %2, 0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %2, 1
// CHECK-NEXT:   %12 = icmp slt i64 %5, 0
// CHECK-NEXT:   %13 = icmp uge i64 %5, %11
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i32, ptr %10, i64 %5
// CHECK-NEXT:   store i32 %9, ptr %15, align 4
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }
func genInts(n int, gen func() c.Int) []c.Int {
	a := make([]c.Int, n)
	for i := range a {
		a[i] = gen()
	}
	return a
}

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/intgen.(*generator).next"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/intgen.generator", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i32, ptr %1, align 4
// CHECK-NEXT:   %3 = add i32 %2, 1
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/intgen.generator", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i32 %3, ptr %4, align 4
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testrt/intgen.generator", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load i32, ptr %5, align 4
// CHECK-NEXT:   ret i32 %6
// CHECK-NEXT: }
func (g *generator) next() c.Int {
	g.val++
	return g.val
}

type generator struct {
	val c.Int
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/intgen.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testrt/intgen.genInts"(i64 5, { ptr, ptr } { ptr @__llgo_stub.rand, ptr null })
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %2 = phi i64 [ -1, %_llgo_0 ], [ %3, %_llgo_2 ]
// CHECK-NEXT:   %3 = add i64 %2, 1
// CHECK-NEXT:   %4 = icmp slt i64 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %7 = icmp slt i64 %3, 0
// CHECK-NEXT:   %8 = icmp uge i64 %3, %6
// CHECK-NEXT:   %9 = or i1 %8, %7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds i32, ptr %5, i64 %3
// CHECK-NEXT:   %11 = load i32, ptr %10, align 4
// CHECK-NEXT:   %12 = call i32 (ptr, ...) @printf(ptr @0, i32 %11)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   store i32 1, ptr %13, align 4
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %15 = getelementptr inbounds { ptr }, ptr %14, i32 0, i32 0
// CHECK-NEXT:   store ptr %13, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/intgen.main$1", ptr undef }, ptr %14, 1
// CHECK-NEXT:   %17 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testrt/intgen.genInts"(i64 5, { ptr, ptr } %16)
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 1
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_3
// CHECK-NEXT:   %19 = phi i64 [ -1, %_llgo_3 ], [ %20, %_llgo_5 ]
// CHECK-NEXT:   %20 = add i64 %19, 1
// CHECK-NEXT:   %21 = icmp slt i64 %20, %18
// CHECK-NEXT:   br i1 %21, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %22 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 0
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 1
// CHECK-NEXT:   %24 = icmp slt i64 %20, 0
// CHECK-NEXT:   %25 = icmp uge i64 %20, %23
// CHECK-NEXT:   %26 = or i1 %25, %24
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %26)
// CHECK-NEXT:   %27 = getelementptr inbounds i32, ptr %22, i64 %20
// CHECK-NEXT:   %28 = load i32, ptr %27, align 4
// CHECK-NEXT:   %29 = call i32 (ptr, ...) @printf(ptr @1, i32 %28)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testrt/intgen.generator", ptr %30, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %31, align 4
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %33 = getelementptr inbounds { ptr }, ptr %32, i32 0, i32 0
// CHECK-NEXT:   store ptr %30, ptr %33, align 8
// CHECK-NEXT:   %34 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/intgen.(*generator).next$bound", ptr undef }, ptr %32, 1
// CHECK-NEXT:   %35 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testrt/intgen.genInts"(i64 5, { ptr, ptr } %34)
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_8, %_llgo_6
// CHECK-NEXT:   %37 = phi i64 [ -1, %_llgo_6 ], [ %38, %_llgo_8 ]
// CHECK-NEXT:   %38 = add i64 %37, 1
// CHECK-NEXT:   %39 = icmp slt i64 %38, %36
// CHECK-NEXT:   br i1 %39, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %40 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 0
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   %42 = icmp slt i64 %38, 0
// CHECK-NEXT:   %43 = icmp uge i64 %38, %41
// CHECK-NEXT:   %44 = or i1 %43, %42
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %44)
// CHECK-NEXT:   %45 = getelementptr inbounds i32, ptr %40, i64 %38
// CHECK-NEXT:   %46 = load i32, ptr %45, align 4
// CHECK-NEXT:   %47 = call i32 (ptr, ...) @printf(ptr @2, i32 %46)
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	for _, v := range genInts(5, c.Rand) {

		c.Printf(c.Str("%d\n"), v)
	}

	initVal := c.Int(1)
	ints := genInts(5, func() c.Int {
		// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/intgen.main$1"(ptr %0){{.*}} {
		// CHECK-NEXT: _llgo_0:
		// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
		// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
		// CHECK-NEXT:   %3 = load i32, ptr %2, align 4
		// CHECK-NEXT:   %4 = mul i32 %3, 2
		// CHECK-NEXT:   %5 = extractvalue { ptr } %1, 0
		// CHECK-NEXT:   store i32 %4, ptr %5, align 4
		// CHECK-NEXT:   %6 = extractvalue { ptr } %1, 0
		// CHECK-NEXT:   %7 = load i32, ptr %6, align 4
		// CHECK-NEXT:   ret i32 %7
		// CHECK-NEXT: }
		initVal *= 2
		return initVal
	})
	for _, v := range ints {
		c.Printf(c.Str("%d\n"), v)
	}

	g := &generator{val: 1}
	for _, v := range genInts(5, g.next) {
		c.Printf(c.Str("%d\n"), v)
	}
	// CHECK-LABEL: define linkonce i32 @__llgo_stub.rand(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = tail call i32 @rand()
	// CHECK-NEXT:   ret i32 %1
	// CHECK-NEXT: }
}

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/intgen.(*generator).next$bound"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call i32 @"{{.*}}/cl/_testrt/intgen.(*generator).next"(ptr %2)
// CHECK-NEXT:   ret i32 %3
// CHECK-NEXT: }
