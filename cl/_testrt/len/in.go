// LITTEST
package main

type data struct {
	s string
	c chan int
	m map[int]string
	a []int
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 56)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.String", ptr %1, align 8
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %2, 1
// CHECK-NEXT:   %4 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %5)
// CHECK-NEXT:   %7 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 2
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %8)
// CHECK-NEXT:   %10 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 3
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %10, align 8
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %11, 1
// CHECK-NEXT:   %13 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %14 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %15 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %14)
// CHECK-NEXT:   %16 = getelementptr inbounds %main.data, ptr %0, i32 0, i32 3
// CHECK-NEXT:   %17 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %16, align 8
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %17, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 56)
// CHECK-NEXT:   %20 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 0
// CHECK-NEXT:   %21 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 1
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 2)
// CHECK-NEXT:   %23 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 2
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 1)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %25, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %24, ptr %25)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 3
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %29 = getelementptr inbounds i64, ptr %28, i64 0
// CHECK-NEXT:   store i64 1, ptr %29, align 8
// CHECK-NEXT:   %30 = getelementptr inbounds i64, ptr %28, i64 1
// CHECK-NEXT:   store i64 2, ptr %30, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds i64, ptr %28, i64 2
// CHECK-NEXT:   store i64 3, ptr %31, align 8
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %28, 0
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, i64 3, 1
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 3, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %20, align 8
// CHECK-NEXT:   store ptr %22, ptr %21, align 8
// CHECK-NEXT:   store ptr %24, ptr %23, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %34, ptr %27, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 0
// CHECK-NEXT:   %36 = load %"{{.*}}/runtime/internal/runtime.String", ptr %35, align 8
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %36, 1
// CHECK-NEXT:   %38 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 1
// CHECK-NEXT:   %39 = load ptr, ptr %38, align 8
// CHECK-NEXT:   %40 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %39)
// CHECK-NEXT:   %41 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 2
// CHECK-NEXT:   %42 = load ptr, ptr %41, align 8
// CHECK-NEXT:   %43 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %42)
// CHECK-NEXT:   %44 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 3
// CHECK-NEXT:   %45 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %44, align 8
// CHECK-NEXT:   %46 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %45, 1
// CHECK-NEXT:   %47 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 1
// CHECK-NEXT:   %48 = load ptr, ptr %47, align 8
// CHECK-NEXT:   %49 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %48)
// CHECK-NEXT:   %50 = getelementptr inbounds %main.data, ptr %19, i32 0, i32 3
// CHECK-NEXT:   %51 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %50, align 8
// CHECK-NEXT:   %52 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %51, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %37)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %40)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %43)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %46)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %49)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %52)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 56, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 56, i1 false)
// ESCAPE-NEXT:   %0 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %1 = load %"{{.*}}/runtime/internal/runtime.String", ptr %0, align 8
// ESCAPE-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %1, 1
// ESCAPE-NEXT:   %3 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   %4 = load ptr, ptr %3, align 8
// ESCAPE-NEXT:   %5 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %4)
// ESCAPE-NEXT:   %6 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 2
// ESCAPE-NEXT:   %7 = load ptr, ptr %6, align 8
// ESCAPE-NEXT:   %8 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %7)
// ESCAPE-NEXT:   %9 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 3
// ESCAPE-NEXT:   %10 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %9, align 8
// ESCAPE-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, 1
// ESCAPE-NEXT:   %12 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 1
// ESCAPE-NEXT:   %13 = load ptr, ptr %12, align 8
// ESCAPE-NEXT:   %14 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %13)
// ESCAPE-NEXT:   %15 = getelementptr inbounds %main.data, ptr %.stack, i32 0, i32 3
// ESCAPE-NEXT:   %16 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %15, align 8
// ESCAPE-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %16, 2
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %14)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %.stack1 = alloca i8, i64 56, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack1, i8 0, i64 56, i1 false)
// ESCAPE-NEXT:   %18 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 0
// ESCAPE-NEXT:   %19 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 1
// ESCAPE-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 2)
// ESCAPE-NEXT:   %21 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 2
// ESCAPE-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 1)
// ESCAPE-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// ESCAPE-NEXT:   store i64 1, ptr %23, align 8
// ESCAPE-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %22, ptr %23)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %24, align 8
// ESCAPE-NEXT:   %25 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 3
// ESCAPE-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// ESCAPE-NEXT:   %27 = getelementptr inbounds i64, ptr %26, i64 0
// ESCAPE-NEXT:   store i64 1, ptr %27, align 8
// ESCAPE-NEXT:   %28 = getelementptr inbounds i64, ptr %26, i64 1
// ESCAPE-NEXT:   store i64 2, ptr %28, align 8
// ESCAPE-NEXT:   %29 = getelementptr inbounds i64, ptr %26, i64 2
// ESCAPE-NEXT:   store i64 3, ptr %29, align 8
// ESCAPE-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %26, 0
// ESCAPE-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, i64 3, 1
// ESCAPE-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, i64 3, 2
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %18, align 8
// ESCAPE-NEXT:   store ptr %20, ptr %19, align 8
// ESCAPE-NEXT:   store ptr %22, ptr %21, align 8
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %32, ptr %25, align 8
// ESCAPE-NEXT:   %33 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 0
// ESCAPE-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.String", ptr %33, align 8
// ESCAPE-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %34, 1
// ESCAPE-NEXT:   %36 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 1
// ESCAPE-NEXT:   %37 = load ptr, ptr %36, align 8
// ESCAPE-NEXT:   %38 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %37)
// ESCAPE-NEXT:   %39 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 2
// ESCAPE-NEXT:   %40 = load ptr, ptr %39, align 8
// ESCAPE-NEXT:   %41 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %40)
// ESCAPE-NEXT:   %42 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 3
// ESCAPE-NEXT:   %43 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %42, align 8
// ESCAPE-NEXT:   %44 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %43, 1
// ESCAPE-NEXT:   %45 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 1
// ESCAPE-NEXT:   %46 = load ptr, ptr %45, align 8
// ESCAPE-NEXT:   %47 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %46)
// ESCAPE-NEXT:   %48 = getelementptr inbounds %main.data, ptr %.stack1, i32 0, i32 3
// ESCAPE-NEXT:   %49 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %48, align 8
// ESCAPE-NEXT:   %50 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, 2
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %35)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %38)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %41)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %44)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %47)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %50)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }
func main() {
	d := &data{}
	println(len(d.s), len(d.c), len(d.m), len(d.a), cap(d.c), cap(d.a))
	v := &data{s: "hello", c: make(chan int, 2), m: map[int]string{1: "hello"}, a: []int{1, 2, 3}}
	println(len(v.s), len(v.c), len(v.m), len(v.a), cap(v.c), cap(v.a))
}
