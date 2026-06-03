// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tptypes.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tptypes.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tptypes.init$guard", align 1
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tptypes.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[int]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[int]" %2, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %4, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %4, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %5, align 8
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %4, align 8
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[string]" %6, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[int]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %8, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %9, align 8
// CHECK-NEXT:   %10 = load %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %8, align 8
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[int]" %10, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %12, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %12, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %13, align 8
// CHECK-NEXT:   %14 = load %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %12, align 8
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[string]" %14, 0
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
// CHECK-NEXT:   %22 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append"(ptr %16, %"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %24, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %24, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %26, i64 1, 1
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %27, i64 1, 2
// CHECK-NEXT:   %29 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]string,string]).Append"(ptr %23, %"{{.*}}/runtime/internal/runtime.Slice" %28)
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
// CHECK-NEXT:   %39 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append"(ptr %30, %"{{.*}}/runtime/internal/runtime.Slice" %38)
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
// CHECK-NEXT:   %48 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append2"(ptr %30, %"{{.*}}/runtime/internal/runtime.Slice" %47)
// CHECK-NEXT:   %49 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %16, i32 0, i32 0
// CHECK-NEXT:   %50 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %49, align 8
// CHECK-NEXT:   %51 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %16, i32 0, i32 0
// CHECK-NEXT:   %52 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %51, align 8
// CHECK-NEXT:   %53 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, 0
// CHECK-NEXT:   %54 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %52, 1
// CHECK-NEXT:   %55 = icmp uge i64 0, %54
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %55)
// CHECK-NEXT:   %56 = getelementptr inbounds i64, ptr %53, i64 0
// CHECK-NEXT:   %57 = load i64, ptr %56, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %50)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %57)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %58 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %23, i32 0, i32 0
// CHECK-NEXT:   %59 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %58, align 8
// CHECK-NEXT:   %60 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %23, i32 0, i32 0
// CHECK-NEXT:   %61 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %60, align 8
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %61, 0
// CHECK-NEXT:   %63 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %61, 1
// CHECK-NEXT:   %64 = icmp uge i64 0, %63
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %64)
// CHECK-NEXT:   %65 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %62, i64 0
// CHECK-NEXT:   %66 = load %"{{.*}}/runtime/internal/runtime.String", ptr %65, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %59)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %66)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %67 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %30, i32 0, i32 0
// CHECK-NEXT:   %68 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %67, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %30, i32 0, i32 0
// CHECK-NEXT:   %70 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %69, align 8
// CHECK-NEXT:   %71 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %70, 0
// CHECK-NEXT:   %72 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %70, 1
// CHECK-NEXT:   %73 = icmp uge i64 0, %72
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %73)
// CHECK-NEXT:   %74 = getelementptr inbounds i64, ptr %71, i64 0
// CHECK-NEXT:   %75 = load i64, ptr %74, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %68)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %75)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]string,string]).Append"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 16)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append2"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, i64 %5, i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %8, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %9
// CHECK-NEXT: }
