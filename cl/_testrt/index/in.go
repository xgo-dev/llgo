// LITTEST
package main

// CHECK: @0 = private unnamed_addr constant [6 x i8] c"123456", align 1

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

type point struct {
	x int
	y int
}

type N [2]int
type T *N
type S []int

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %main.point, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = alloca [3 x %main.point], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %2 = getelementptr inbounds %main.point, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds %main.point, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %main.point, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %5 = getelementptr inbounds %main.point, ptr %1, i64 1
// CHECK-NEXT:   %6 = getelementptr inbounds %main.point, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = getelementptr inbounds %main.point, ptr %5, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %main.point, ptr %1, i64 2
// CHECK-NEXT:   %9 = getelementptr inbounds %main.point, ptr %8, i32 0, i32 0
// CHECK-NEXT:   %10 = getelementptr inbounds %main.point, ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %3, align 8
// CHECK-NEXT:   store i64 2, ptr %4, align 8
// CHECK-NEXT:   store i64 3, ptr %6, align 8
// CHECK-NEXT:   store i64 4, ptr %7, align 8
// CHECK-NEXT:   store i64 5, ptr %9, align 8
// CHECK-NEXT:   store i64 6, ptr %10, align 8
// CHECK-NEXT:   %11 = load [3 x %main.point], ptr %1, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %main.point, ptr %1, i64 2
// CHECK-NEXT:   %13 = load %main.point, ptr %12, align 8
// CHECK-NEXT:   store %main.point %13, ptr %0, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %main.point, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %15 = load i64, ptr %14, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %main.point, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %17 = load i64, ptr %16, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %18, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %19 = alloca [2 x [2 x i64]], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %19, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %20 = getelementptr inbounds [2 x i64], ptr %19, i64 0
// CHECK-NEXT:   %21 = icmp eq ptr %20, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = getelementptr inbounds i64, ptr %20, i64 0
// CHECK-NEXT:   %23 = icmp eq ptr %20, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %23)
// CHECK-NEXT:   %24 = getelementptr inbounds i64, ptr %20, i64 1
// CHECK-NEXT:   %25 = getelementptr inbounds [2 x i64], ptr %19, i64 1
// CHECK-NEXT:   %26 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %26)
// CHECK-NEXT:   %27 = getelementptr inbounds i64, ptr %25, i64 0
// CHECK-NEXT:   %28 = icmp eq ptr %25, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %28)
// CHECK-NEXT:   %29 = getelementptr inbounds i64, ptr %25, i64 1
// CHECK-NEXT:   store i64 1, ptr %22, align 8
// CHECK-NEXT:   store i64 2, ptr %24, align 8
// CHECK-NEXT:   store i64 3, ptr %27, align 8
// CHECK-NEXT:   store i64 4, ptr %29, align 8
// CHECK-NEXT:   %30 = load [2 x [2 x i64]], ptr %19, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds [2 x i64], ptr %19, i64 1
// CHECK-NEXT:   %32 = load [2 x i64], ptr %31, align 8
// CHECK-NEXT:   store [2 x i64] %32, ptr %18, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds i64, ptr %18, i64 0
// CHECK-NEXT:   %34 = load i64, ptr %33, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds i64, ptr %18, i64 1
// CHECK-NEXT:   %36 = load i64, ptr %35, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %34)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %36)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %37 = alloca [5 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %37, i8 0, i64 40, i1 false)
// CHECK-NEXT:   %38 = getelementptr inbounds i64, ptr %37, i64 0
// CHECK-NEXT:   %39 = getelementptr inbounds i64, ptr %37, i64 1
// CHECK-NEXT:   %40 = getelementptr inbounds i64, ptr %37, i64 2
// CHECK-NEXT:   %41 = getelementptr inbounds i64, ptr %37, i64 3
// CHECK-NEXT:   %42 = getelementptr inbounds i64, ptr %37, i64 4
// CHECK-NEXT:   store i64 1, ptr %38, align 8
// CHECK-NEXT:   store i64 2, ptr %39, align 8
// CHECK-NEXT:   store i64 3, ptr %40, align 8
// CHECK-NEXT:   store i64 4, ptr %41, align 8
// CHECK-NEXT:   store i64 5, ptr %42, align 8
// CHECK-NEXT:   %43 = load [5 x i64], ptr %37, align 8
// CHECK-NEXT:   %44 = getelementptr inbounds i64, ptr %37, i64 2
// CHECK-NEXT:   %45 = load i64, ptr %44, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %46 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 2), align 1
// CHECK-NEXT:   %47 = zext i8 %46 to i64
// CHECK-NEXT:   %48 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %48)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %49 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 1), align 1
// CHECK-NEXT:   %50 = zext i8 %49 to i64
// CHECK-NEXT:   %51 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %50)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %51)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %53 = getelementptr inbounds i64, ptr %52, i64 0
// CHECK-NEXT:   %54 = getelementptr inbounds i64, ptr %52, i64 1
// CHECK-NEXT:   store i64 1, ptr %53, align 8
// CHECK-NEXT:   store i64 2, ptr %54, align 8
// CHECK-NEXT:   %55 = getelementptr inbounds i64, ptr %52, i64 1
// CHECK-NEXT:   %56 = icmp eq ptr %52, null
// CHECK-NEXT:   br i1 %56, label %57, label %58
// CHECK-EMPTY:
// CHECK-NEXT: 57:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 58:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %59 = load i64, ptr %55, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %59)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %61 = getelementptr inbounds i64, ptr %60, i64 0
// CHECK-NEXT:   store i64 1, ptr %61, align 8
// CHECK-NEXT:   %62 = getelementptr inbounds i64, ptr %60, i64 1
// CHECK-NEXT:   store i64 2, ptr %62, align 8
// CHECK-NEXT:   %63 = getelementptr inbounds i64, ptr %60, i64 2
// CHECK-NEXT:   store i64 3, ptr %63, align 8
// CHECK-NEXT:   %64 = getelementptr inbounds i64, ptr %60, i64 3
// CHECK-NEXT:   store i64 4, ptr %64, align 8
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %60, 0
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %65, i64 4, 1
// CHECK-NEXT:   %67 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, i64 4, 2
// CHECK-NEXT:   %68 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, 0
// CHECK-NEXT:   %69 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, 1
// CHECK-NEXT:   %70 = icmp uge i64 1, %69
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %70, i64 1, i1 true, i64 %69)
// CHECK-NEXT:   %71 = getelementptr inbounds i64, ptr %68, i64 1
// CHECK-NEXT:   %72 = load i64, ptr %71, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %72)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	a := [...]point{{1, 2}, {3, 4}, {5, 6}}[2]
	println(a.x, a.y)

	b := [...][2]int{[2]int{1, 2}, [2]int{3, 4}}[1]
	println(b[0], b[1])

	var i int = 2
	println([...]int{1, 2, 3, 4, 5}[i])

	s := "123456"
	println(string(s[i]))
	println(string("123456"[1]))

	var n = N{1, 2}
	var t T = &n
	println(t[1])
	var s1 = S{1, 2, 3, 4}
	println(s1[1])

	println([2]int{}[0])
}
