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
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   %3 = load %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %0, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[int]" %3, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %5 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %6 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %5, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %7, align 8
// CHECK-NEXT:   %8 = load %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %5, align 8
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[string]" %8, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %10 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[int]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %10, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %11 = icmp eq ptr %10, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %10, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %12, align 8
// CHECK-NEXT:   %13 = load %"{{.*}}/cl/_testgo/tptypes.Data[int]", ptr %10, align 8
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[int]" %13, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %15 = alloca %"{{.*}}/cl/_testgo/tptypes.Data[string]", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %15, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %16 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %15, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %17, align 8
// CHECK-NEXT:   %18 = load %"{{.*}}/cl/_testgo/tptypes.Data[string]", ptr %15, align 8
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/cl/_testgo/tptypes.Data[string]" %18, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %22 = getelementptr inbounds i64, ptr %21, i64 0
// CHECK-NEXT:   store i64 100, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %21, 0
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %23, i64 1, 1
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, i64 1, 2
// CHECK-NEXT:   %26 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append"(ptr %20, %"{{.*}}/runtime/internal/runtime.Slice" %25)
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %29 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %28, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %29, align 8
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %28, 0
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, i64 1, 1
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, i64 1, 2
// CHECK-NEXT:   %33 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]string,string]).Append"(ptr %27, %"{{.*}}/runtime/internal/runtime.Slice" %32)
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %36 = getelementptr inbounds i64, ptr %35, i64 0
// CHECK-NEXT:   store i64 1, ptr %36, align 8
// CHECK-NEXT:   %37 = getelementptr inbounds i64, ptr %35, i64 1
// CHECK-NEXT:   store i64 2, ptr %37, align 8
// CHECK-NEXT:   %38 = getelementptr inbounds i64, ptr %35, i64 2
// CHECK-NEXT:   store i64 3, ptr %38, align 8
// CHECK-NEXT:   %39 = getelementptr inbounds i64, ptr %35, i64 3
// CHECK-NEXT:   store i64 4, ptr %39, align 8
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %35, 0
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %40, i64 4, 1
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %41, i64 4, 2
// CHECK-NEXT:   %43 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append"(ptr %34, %"{{.*}}/runtime/internal/runtime.Slice" %42)
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %45 = getelementptr inbounds i64, ptr %44, i64 0
// CHECK-NEXT:   store i64 1, ptr %45, align 8
// CHECK-NEXT:   %46 = getelementptr inbounds i64, ptr %44, i64 1
// CHECK-NEXT:   store i64 2, ptr %46, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds i64, ptr %44, i64 2
// CHECK-NEXT:   store i64 3, ptr %47, align 8
// CHECK-NEXT:   %48 = getelementptr inbounds i64, ptr %44, i64 3
// CHECK-NEXT:   store i64 4, ptr %48, align 8
// CHECK-NEXT:   %49 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %44, 0
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %49, i64 4, 1
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %50, i64 4, 2
// CHECK-NEXT:   %52 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append2"(ptr %34, %"{{.*}}/runtime/internal/runtime.Slice" %51)
// CHECK-NEXT:   %53 = icmp eq ptr %20, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %20, i32 0, i32 0
// CHECK-NEXT:   %55 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %54, align 8
// CHECK-NEXT:   %56 = icmp eq ptr %20, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %56)
// CHECK-NEXT:   %57 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %20, i32 0, i32 0
// CHECK-NEXT:   %58 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %57, align 8
// CHECK-NEXT:   %59 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, 0
// CHECK-NEXT:   %60 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, 1
// CHECK-NEXT:   %61 = icmp uge i64 0, %60
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %61, i64 0, i1 true, i64 %60)
// CHECK-NEXT:   %62 = getelementptr inbounds i64, ptr %59, i64 0
// CHECK-NEXT:   %63 = load i64, ptr %62, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %63)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %64 = icmp eq ptr %27, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %64)
// CHECK-NEXT:   %65 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %27, i32 0, i32 0
// CHECK-NEXT:   %66 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %65, align 8
// CHECK-NEXT:   %67 = icmp eq ptr %27, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %67)
// CHECK-NEXT:   %68 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %27, i32 0, i32 0
// CHECK-NEXT:   %69 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %68, align 8
// CHECK-NEXT:   %70 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %69, 0
// CHECK-NEXT:   %71 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %69, 1
// CHECK-NEXT:   %72 = icmp uge i64 0, %71
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %72, i64 0, i1 true, i64 %71)
// CHECK-NEXT:   %73 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %70, i64 0
// CHECK-NEXT:   %74 = load %"{{.*}}/runtime/internal/runtime.String", ptr %73, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %66)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %74)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %75 = icmp eq ptr %34, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %75)
// CHECK-NEXT:   %76 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %34, i32 0, i32 0
// CHECK-NEXT:   %77 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %76, align 8
// CHECK-NEXT:   %78 = icmp eq ptr %34, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %78)
// CHECK-NEXT:   %79 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %34, i32 0, i32 0
// CHECK-NEXT:   %80 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %79, align 8
// CHECK-NEXT:   %81 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %80, 0
// CHECK-NEXT:   %82 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %80, 1
// CHECK-NEXT:   %83 = icmp uge i64 0, %82
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %83, i64 0, i1 true, i64 %82)
// CHECK-NEXT:   %84 = getelementptr inbounds i64, ptr %81, i64 0
// CHECK-NEXT:   %85 = load i64, ptr %84, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %77)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %85)
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
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %4, ptr %5, i64 %6, i64 8)
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %7, ptr %9, align 8
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %11, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %12
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]string,string]).Append"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %4, ptr %5, i64 %6, i64 16)
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %7, ptr %9, align 8
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]string,string]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %11, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %12
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/tptypes.(*Slice[[]int,int]).Append2"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %4, ptr %5, i64 %6, i64 8)
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %7, ptr %9, align 8
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/tptypes.Slice[[]int,int]", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %11, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %12
// CHECK-NEXT: }
