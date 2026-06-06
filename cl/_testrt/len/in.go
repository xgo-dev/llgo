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
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %4, 1
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %9 = load ptr, ptr %8, align 8
// CHECK-NEXT:   %10 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %9)
// CHECK-NEXT:   %11 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %14 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %15 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %14)
// CHECK-NEXT:   %16 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %17)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %19 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %18, align 8
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, 1
// CHECK-NEXT:   %21 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %24 = load ptr, ptr %23, align 8
// CHECK-NEXT:   %25 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %24)
// CHECK-NEXT:   %26 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %26)
// CHECK-NEXT:   %27 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %28, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %20)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %30)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 56)
// CHECK-NEXT:   %32 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %32)
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 0
// CHECK-NEXT:   %34 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 1
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 2)
// CHECK-NEXT:   %37 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %37)
// CHECK-NEXT:   %38 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 2
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 1)
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %40, align 8
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %39, ptr %40)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %41, align 8
// CHECK-NEXT:   %42 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %42)
// CHECK-NEXT:   %43 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 3
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %45 = getelementptr inbounds i64, ptr %44, i64 0
// CHECK-NEXT:   store i64 1, ptr %45, align 8
// CHECK-NEXT:   %46 = getelementptr inbounds i64, ptr %44, i64 1
// CHECK-NEXT:   store i64 2, ptr %46, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds i64, ptr %44, i64 2
// CHECK-NEXT:   store i64 3, ptr %47, align 8
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %44, 0
// CHECK-NEXT:   %49 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %48, i64 3, 1
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, i64 3, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %33, align 8
// CHECK-NEXT:   store ptr %36, ptr %35, align 8
// CHECK-NEXT:   store ptr %39, ptr %38, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %50, ptr %43, align 8
// CHECK-NEXT:   %51 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %51)
// CHECK-NEXT:   %52 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %52)
// CHECK-NEXT:   %53 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 0
// CHECK-NEXT:   %54 = load %"{{.*}}/runtime/internal/runtime.String", ptr %53, align 8
// CHECK-NEXT:   %55 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %54, 1
// CHECK-NEXT:   %56 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %56)
// CHECK-NEXT:   %57 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %57)
// CHECK-NEXT:   %58 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 1
// CHECK-NEXT:   %59 = load ptr, ptr %58, align 8
// CHECK-NEXT:   %60 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %59)
// CHECK-NEXT:   %61 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %61)
// CHECK-NEXT:   %62 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %62)
// CHECK-NEXT:   %63 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 2
// CHECK-NEXT:   %64 = load ptr, ptr %63, align 8
// CHECK-NEXT:   %65 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %64)
// CHECK-NEXT:   %66 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %66)
// CHECK-NEXT:   %67 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %67)
// CHECK-NEXT:   %68 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 3
// CHECK-NEXT:   %69 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %68, align 8
// CHECK-NEXT:   %70 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %69, 1
// CHECK-NEXT:   %71 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %71)
// CHECK-NEXT:   %72 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %72)
// CHECK-NEXT:   %73 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 1
// CHECK-NEXT:   %74 = load ptr, ptr %73, align 8
// CHECK-NEXT:   %75 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %74)
// CHECK-NEXT:   %76 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %76)
// CHECK-NEXT:   %77 = icmp eq ptr %31, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %77)
// CHECK-NEXT:   %78 = getelementptr inbounds %"{{.*}}/cl/_testrt/len.data", ptr %31, i32 0, i32 3
// CHECK-NEXT:   %79 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %78, align 8
// CHECK-NEXT:   %80 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %79, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %60)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %65)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %70)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %75)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %80)
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
