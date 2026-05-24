// LITTEST
package main

// CHECK-LINE: @16 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/len.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/len.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/len.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type data struct {
	s string
	c chan int
	m map[int]string
	a []int
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/len.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 56)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.String", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %3, 1
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %7)
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %11)
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %15 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %14, align 8
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %15, 1
// CHECK-NEXT:   %17 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %17)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %19 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %20 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %19)
// CHECK-NEXT:   %21 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %23 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %22, align 8
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %23, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %20)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 56)
// CHECK-NEXT:   %26 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %26)
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 0
// CHECK-NEXT:   %28 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %28)
// CHECK-NEXT:   %29 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 1
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 2)
// CHECK-NEXT:   %31 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %31)
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 2
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 1)
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %34, align 8
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %33, ptr %34)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %35, align 8
// CHECK-NEXT:   %36 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %36)
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 3
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %39 = getelementptr inbounds i64, ptr %38, i64 0
// CHECK-NEXT:   store i64 1, ptr %39, align 8
// CHECK-NEXT:   %40 = getelementptr inbounds i64, ptr %38, i64 1
// CHECK-NEXT:   store i64 2, ptr %40, align 8
// CHECK-NEXT:   %41 = getelementptr inbounds i64, ptr %38, i64 2
// CHECK-NEXT:   store i64 3, ptr %41, align 8
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %38, 0
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, i64 3, 1
// CHECK-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %43, i64 3, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %27, align 8
// CHECK-NEXT:   store ptr %30, ptr %29, align 8
// CHECK-NEXT:   store ptr %33, ptr %32, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %44, ptr %37, align 8
// CHECK-NEXT:   %45 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %45)
// CHECK-NEXT:   %46 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 0
// CHECK-NEXT:   %47 = load %"{{.*}}/runtime/internal/runtime.String", ptr %46, align 8
// CHECK-NEXT:   %48 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %47, 1
// CHECK-NEXT:   %49 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %49)
// CHECK-NEXT:   %50 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 1
// CHECK-NEXT:   %51 = load ptr, ptr %50, align 8
// CHECK-NEXT:   %52 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %51)
// CHECK-NEXT:   %53 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 2
// CHECK-NEXT:   %55 = load ptr, ptr %54, align 8
// CHECK-NEXT:   %56 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %55)
// CHECK-NEXT:   %57 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %57)
// CHECK-NEXT:   %58 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 3
// CHECK-NEXT:   %59 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %58, align 8
// CHECK-NEXT:   %60 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %59, 1
// CHECK-NEXT:   %61 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %61)
// CHECK-NEXT:   %62 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 1
// CHECK-NEXT:   %63 = load ptr, ptr %62, align 8
// CHECK-NEXT:   %64 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %63)
// CHECK-NEXT:   %65 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %65)
// CHECK-NEXT:   %66 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %25, i32 0, i32 3
// CHECK-NEXT:   %67 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %66, align 8
// CHECK-NEXT:   %68 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %48)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %52)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %56)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %60)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %64)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %68)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	d := &data{}
	println(len(d.s), len(d.c), len(d.m), len(d.a), cap(d.c), cap(d.a))
	v := &data{s: "hello", c: make(chan int, 2), m: map[int]string{1: "hello"}, a: []int{1, 2, 3}}
	println(len(v.s), len(v.c), len(v.m), len(v.a), cap(v.c), cap(v.a))
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
