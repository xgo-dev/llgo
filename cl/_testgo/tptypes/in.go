// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type Data[T any] struct {
	v T
}

func (p *Data[T]) Set(v T) {
	p.v = v
}

func (p *(Data[T1])) Set2(v T1) {
	p.v = v
}

type sliceOf[E any] interface {
	~[]E
}

type Slice[S sliceOf[T], T any] struct {
	Data S
}

func (p *Slice[S, T]) Append(t ...T) S {
	p.Data = append(p.Data, t...)
	return p.Data
}

func (p *Slice[S1, T1]) Append2(t ...T1) S1 {
	p.Data = append(p.Data, t...)
	return p.Data
}

type (
	DataInt     = Data[int]
	SliceInt    = Slice[[]int, int]
	DataString  = Data[string]
	SliceString = Slice[[]string, string]
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"main.Data[int]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"main.Data[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"main.Data[int]", ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue %"main.Data[int]" %2, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = alloca %"main.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %4, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %5 = getelementptr inbounds %"main.Data[string]", ptr %4, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %5, align 8
// CHECK-NEXT:   %6 = load %"main.Data[string]", ptr %4, align 8
// CHECK-NEXT:   %7 = extractvalue %"main.Data[string]" %6, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = alloca %"main.Data[int]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %8, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %9 = getelementptr inbounds %"main.Data[int]", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %9, align 8
// CHECK-NEXT:   %10 = load %"main.Data[int]", ptr %8, align 8
// CHECK-NEXT:   %11 = extractvalue %"main.Data[int]" %10, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = alloca %"main.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %12, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %13 = getelementptr inbounds %"main.Data[string]", ptr %12, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %13, align 8
// CHECK-NEXT:   %14 = load %"main.Data[string]", ptr %12, align 8
// CHECK-NEXT:   %15 = extractvalue %"main.Data[string]" %14, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %18 = getelementptr inbounds i64, ptr %17, i64 0
// CHECK-NEXT:   store i64 100, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %17, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 1, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 1, 2
// CHECK-NEXT:   %22 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %16, %"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %24, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %24, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %26, i64 1, 1
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %27, i64 1, 2
// CHECK-NEXT:   %29 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]string,string\]}}).Append"(ptr %23, %"{{.*}}/runtime/internal/runtime.Slice" %28)
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %32 = getelementptr inbounds i64, ptr %31, i64 0
// CHECK-NEXT:   store i64 1, ptr %32, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds i64, ptr %31, i64 1
// CHECK-NEXT:   store i64 2, ptr %33, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds i64, ptr %31, i64 2
// CHECK-NEXT:   store i64 3, ptr %34, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds i64, ptr %31, i64 3
// CHECK-NEXT:   store i64 4, ptr %35, align 8
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %31, 0
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %36, i64 4, 1
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %37, i64 4, 2
// CHECK-NEXT:   %39 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %30, %"{{.*}}/runtime/internal/runtime.Slice" %38)
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %41 = getelementptr inbounds i64, ptr %40, i64 0
// CHECK-NEXT:   store i64 1, ptr %41, align 8
// CHECK-NEXT:   %42 = getelementptr inbounds i64, ptr %40, i64 1
// CHECK-NEXT:   store i64 2, ptr %42, align 8
// CHECK-NEXT:   %43 = getelementptr inbounds i64, ptr %40, i64 2
// CHECK-NEXT:   store i64 3, ptr %43, align 8
// CHECK-NEXT:   %44 = getelementptr inbounds i64, ptr %40, i64 3
// CHECK-NEXT:   store i64 4, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %40, 0
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %45, i64 4, 1
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %46, i64 4, 2
// CHECK-NEXT:   %48 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append2"(ptr %30, %"{{.*}}/runtime/internal/runtime.Slice" %47)
// CHECK-NEXT:   %49 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %16, i32 0, i32 0
// CHECK-NEXT:   %50 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %49, align 8
// CHECK-NEXT:   %51 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %16, i32 0, i32 0
// CHECK-NEXT:   %52 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %51, align 8
// CHECK-NEXT:   %53 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, 0
// CHECK-NEXT:   %54 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, 1
// CHECK-NEXT:   %55 = icmp uge i64 0, %54
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %55, {{.*}})
// CHECK-NEXT:   %56 = getelementptr inbounds i64, ptr %53, i64 0
// CHECK-NEXT:   %57 = load i64, ptr %56, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %50)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %57)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %58 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %23, i32 0, i32 0
// CHECK-NEXT:   %59 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %58, align 8
// CHECK-NEXT:   %60 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %23, i32 0, i32 0
// CHECK-NEXT:   %61 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %60, align 8
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %61, 0
// CHECK-NEXT:   %63 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %61, 1
// CHECK-NEXT:   %64 = icmp uge i64 0, %63
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %64, {{.*}})
// CHECK-NEXT:   %65 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %62, i64 0
// CHECK-NEXT:   %66 = load %"{{.*}}/runtime/internal/runtime.String", ptr %65, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %59)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %66)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %67 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %30, i32 0, i32 0
// CHECK-NEXT:   %68 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %67, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %30, i32 0, i32 0
// CHECK-NEXT:   %70 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %69, align 8
// CHECK-NEXT:   %71 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %70, 0
// CHECK-NEXT:   %72 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %70, 1
// CHECK-NEXT:   %73 = icmp uge i64 0, %72
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %73, {{.*}})
// CHECK-NEXT:   %74 = getelementptr inbounds i64, ptr %71, i64 0
// CHECK-NEXT:   %75 = load i64, ptr %74, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %68)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %75)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %0 = alloca %"main.Data[int]", align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   %1 = getelementptr inbounds %"main.Data[int]", ptr %0, i32 0, i32 0
// ESCAPE-NEXT:   store i64 1, ptr %1, align 8
// ESCAPE-NEXT:   %2 = load %"main.Data[int]", ptr %0, align 8
// ESCAPE-NEXT:   %3 = extractvalue %"main.Data[int]" %2, 0
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %4 = alloca %"main.Data[string]", align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %4, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %5 = getelementptr inbounds %"main.Data[string]", ptr %4, i32 0, i32 0
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %5, align 8
// ESCAPE-NEXT:   %6 = load %"main.Data[string]", ptr %4, align 8
// ESCAPE-NEXT:   %7 = extractvalue %"main.Data[string]" %6, 0
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %7)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %8 = alloca %"main.Data[int]", align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %8, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   %9 = getelementptr inbounds %"main.Data[int]", ptr %8, i32 0, i32 0
// ESCAPE-NEXT:   store i64 100, ptr %9, align 8
// ESCAPE-NEXT:   %10 = load %"main.Data[int]", ptr %8, align 8
// ESCAPE-NEXT:   %11 = extractvalue %"main.Data[int]" %10, 0
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %12 = alloca %"main.Data[string]", align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %12, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %13 = getelementptr inbounds %"main.Data[string]", ptr %12, i32 0, i32 0
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %13, align 8
// ESCAPE-NEXT:   %14 = load %"main.Data[string]", ptr %12, align 8
// ESCAPE-NEXT:   %15 = extractvalue %"main.Data[string]" %14, 0
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %15)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %.stack = alloca i8, i64 24, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 24, i1 false)
// ESCAPE-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// ESCAPE-NEXT:   %17 = getelementptr inbounds i64, ptr %16, i64 0
// ESCAPE-NEXT:   store i64 100, ptr %17, align 8
// ESCAPE-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %16, 0
// ESCAPE-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %18, i64 1, 1
// ESCAPE-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 1, 2
// ESCAPE-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %.stack, %"{{.*}}/runtime/internal/runtime.Slice" %20)
// ESCAPE-NEXT:   %.stack1 = alloca i8, i64 24, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack1, i8 0, i64 24, i1 false)
// ESCAPE-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// ESCAPE-NEXT:   %23 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %22, i64 0
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %23, align 8
// ESCAPE-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %22, 0
// ESCAPE-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, i64 1, 1
// ESCAPE-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %25, i64 1, 2
// ESCAPE-NEXT:   %27 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]string,string\]}}).Append"(ptr %.stack1, %"{{.*}}/runtime/internal/runtime.Slice" %26)
// ESCAPE-NEXT:   %.stack2 = alloca i8, i64 24, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack2, i8 0, i64 24, i1 false)
// ESCAPE-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// ESCAPE-NEXT:   %29 = getelementptr inbounds i64, ptr %28, i64 0
// ESCAPE-NEXT:   store i64 1, ptr %29, align 8
// ESCAPE-NEXT:   %30 = getelementptr inbounds i64, ptr %28, i64 1
// ESCAPE-NEXT:   store i64 2, ptr %30, align 8
// ESCAPE-NEXT:   %31 = getelementptr inbounds i64, ptr %28, i64 2
// ESCAPE-NEXT:   store i64 3, ptr %31, align 8
// ESCAPE-NEXT:   %32 = getelementptr inbounds i64, ptr %28, i64 3
// ESCAPE-NEXT:   store i64 4, ptr %32, align 8
// ESCAPE-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %28, 0
// ESCAPE-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 4, 1
// ESCAPE-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, i64 4, 2
// ESCAPE-NEXT:   %36 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %.stack2, %"{{.*}}/runtime/internal/runtime.Slice" %35)
// ESCAPE-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// ESCAPE-NEXT:   %38 = getelementptr inbounds i64, ptr %37, i64 0
// ESCAPE-NEXT:   store i64 1, ptr %38, align 8
// ESCAPE-NEXT:   %39 = getelementptr inbounds i64, ptr %37, i64 1
// ESCAPE-NEXT:   store i64 2, ptr %39, align 8
// ESCAPE-NEXT:   %40 = getelementptr inbounds i64, ptr %37, i64 2
// ESCAPE-NEXT:   store i64 3, ptr %40, align 8
// ESCAPE-NEXT:   %41 = getelementptr inbounds i64, ptr %37, i64 3
// ESCAPE-NEXT:   store i64 4, ptr %41, align 8
// ESCAPE-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %37, 0
// ESCAPE-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, i64 4, 1
// ESCAPE-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %43, i64 4, 2
// ESCAPE-NEXT:   %45 = call %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append2"(ptr %.stack2, %"{{.*}}/runtime/internal/runtime.Slice" %44)
// ESCAPE-NEXT:   %46 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %47 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %46, align 8
// ESCAPE-NEXT:   %48 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %49 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %48, align 8
// ESCAPE-NEXT:   %50 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, 0
// ESCAPE-NEXT:   %51 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, 1
// ESCAPE-NEXT:   %52 = icmp uge i64 0, %51
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %52, i64 0, i1 true, i64 %51)
// ESCAPE-NEXT:   %53 = getelementptr inbounds i64, ptr %50, i64 0
// ESCAPE-NEXT:   %54 = load i64, ptr %53, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %47)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %54)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %55 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %.stack1, i32 0, i32 0
// ESCAPE-NEXT:   %56 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %55, align 8
// ESCAPE-NEXT:   %57 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %.stack1, i32 0, i32 0
// ESCAPE-NEXT:   %58 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %57, align 8
// ESCAPE-NEXT:   %59 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, 0
// ESCAPE-NEXT:   %60 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, 1
// ESCAPE-NEXT:   %61 = icmp uge i64 0, %60
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %61, i64 0, i1 true, i64 %60)
// ESCAPE-NEXT:   %62 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %59, i64 0
// ESCAPE-NEXT:   %63 = load %"{{.*}}/runtime/internal/runtime.String", ptr %62, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %56)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %63)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   %64 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %.stack2, i32 0, i32 0
// ESCAPE-NEXT:   %65 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %64, align 8
// ESCAPE-NEXT:   %66 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %.stack2, i32 0, i32 0
// ESCAPE-NEXT:   %67 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %66, align 8
// ESCAPE-NEXT:   %68 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, 0
// ESCAPE-NEXT:   %69 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, 1
// ESCAPE-NEXT:   %70 = icmp uge i64 0, %69
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %70, i64 0, i1 true, i64 %69)
// ESCAPE-NEXT:   %71 = getelementptr inbounds i64, ptr %68, i64 0
// ESCAPE-NEXT:   %72 = load i64, ptr %71, align 8
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %65)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %72)
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

func main() {
	println(DataInt{1}.v)
	println(DataString{"hello"}.v)
	println(Data[int]{100}.v)
	println(Data[string]{"hello"}.v)

	// TODO
	println(Data[struct {
		X int
		Y int
	}]{}.v.X)

	v1 := SliceInt{}
	v1.Append(100)
	v2 := SliceString{}
	v2.Append("hello")
	v3 := Slice[[]int, int]{}
	v3.Append([]int{1, 2, 3, 4}...)
	v3.Append2([]int{1, 2, 3, 4}...)

	println(v1.Data, v1.Data[0])
	println(v2.Data, v2.Data[0])
	println(v3.Data, v3.Data[0])
}

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]string,string\]}}).Append"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 16)
// CHECK-NEXT:   %7 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"main.Slice{{\[\[\]string,string\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"main.(*Slice{{\[\[\]int,int\]}}).Append2"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"main.Slice{{\[\[\]int,int\]}}", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }
