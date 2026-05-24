// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"123456", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/index.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/index.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/index.init$guard", align 1
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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/index.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/cl/_testrt/index.point", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = alloca [3 x %"{{.*}}/cl/_testrt/index.point"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 48, i1 false)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 0
// CHECK-NEXT:   %3 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %5 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 1
// CHECK-NEXT:   %8 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %7, i32 0, i32 0
// CHECK-NEXT:   %10 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %7, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 2
// CHECK-NEXT:   %13 = icmp eq ptr %12, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %12, i32 0, i32 0
// CHECK-NEXT:   %15 = icmp eq ptr %12, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %15)
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %12, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %4, align 8
// CHECK-NEXT:   store i64 2, ptr %6, align 8
// CHECK-NEXT:   store i64 3, ptr %9, align 8
// CHECK-NEXT:   store i64 4, ptr %11, align 8
// CHECK-NEXT:   store i64 5, ptr %14, align 8
// CHECK-NEXT:   store i64 6, ptr %16, align 8
// CHECK-NEXT:   %17 = load [3 x %"{{.*}}/cl/_testrt/index.point"], ptr %1, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 2
// CHECK-NEXT:   %19 = load %"{{.*}}/cl/_testrt/index.point", ptr %18, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/index.point" %19, ptr %0, align 8
// CHECK-NEXT:   %20 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %20)
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %22 = load i64, ptr %21, align 8
// CHECK-NEXT:   %23 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %23)
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %25 = load i64, ptr %24, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %22)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %26 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %26, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %27 = alloca [2 x [2 x i64]], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %27, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %28 = getelementptr inbounds [2 x i64], ptr %27, i64 0
// CHECK-NEXT:   %29 = icmp eq ptr %28, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %29)
// CHECK-NEXT:   %30 = getelementptr inbounds i64, ptr %28, i64 0
// CHECK-NEXT:   %31 = icmp eq ptr %28, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %31)
// CHECK-NEXT:   %32 = getelementptr inbounds i64, ptr %28, i64 1
// CHECK-NEXT:   %33 = getelementptr inbounds [2 x i64], ptr %27, i64 1
// CHECK-NEXT:   %34 = icmp eq ptr %33, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds i64, ptr %33, i64 0
// CHECK-NEXT:   %36 = icmp eq ptr %33, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %36)
// CHECK-NEXT:   %37 = getelementptr inbounds i64, ptr %33, i64 1
// CHECK-NEXT:   store i64 1, ptr %30, align 8
// CHECK-NEXT:   store i64 2, ptr %32, align 8
// CHECK-NEXT:   store i64 3, ptr %35, align 8
// CHECK-NEXT:   store i64 4, ptr %37, align 8
// CHECK-NEXT:   %38 = load [2 x [2 x i64]], ptr %27, align 8
// CHECK-NEXT:   %39 = getelementptr inbounds [2 x i64], ptr %27, i64 1
// CHECK-NEXT:   %40 = load [2 x i64], ptr %39, align 8
// CHECK-NEXT:   store [2 x i64] %40, ptr %26, align 8
// CHECK-NEXT:   %41 = getelementptr inbounds i64, ptr %26, i64 0
// CHECK-NEXT:   %42 = load i64, ptr %41, align 8
// CHECK-NEXT:   %43 = getelementptr inbounds i64, ptr %26, i64 1
// CHECK-NEXT:   %44 = load i64, ptr %43, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %44)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %45 = alloca [5 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %45, i8 0, i64 40, i1 false)
// CHECK-NEXT:   %46 = getelementptr inbounds i64, ptr %45, i64 0
// CHECK-NEXT:   %47 = getelementptr inbounds i64, ptr %45, i64 1
// CHECK-NEXT:   %48 = getelementptr inbounds i64, ptr %45, i64 2
// CHECK-NEXT:   %49 = getelementptr inbounds i64, ptr %45, i64 3
// CHECK-NEXT:   %50 = getelementptr inbounds i64, ptr %45, i64 4
// CHECK-NEXT:   store i64 1, ptr %46, align 8
// CHECK-NEXT:   store i64 2, ptr %47, align 8
// CHECK-NEXT:   store i64 3, ptr %48, align 8
// CHECK-NEXT:   store i64 4, ptr %49, align 8
// CHECK-NEXT:   store i64 5, ptr %50, align 8
// CHECK-NEXT:   %51 = load [5 x i64], ptr %45, align 8
// CHECK-NEXT:   %52 = getelementptr inbounds i64, ptr %45, i64 2
// CHECK-NEXT:   %53 = load i64, ptr %52, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %53)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %54 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 2), align 1
// CHECK-NEXT:   %55 = zext i8 %54 to i64
// CHECK-NEXT:   %56 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %56)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %57 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 1), align 1
// CHECK-NEXT:   %58 = zext i8 %57 to i64
// CHECK-NEXT:   %59 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %58)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %59)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %61 = getelementptr inbounds i64, ptr %60, i64 0
// CHECK-NEXT:   %62 = getelementptr inbounds i64, ptr %60, i64 1
// CHECK-NEXT:   store i64 1, ptr %61, align 8
// CHECK-NEXT:   store i64 2, ptr %62, align 8
// CHECK-NEXT:   %63 = getelementptr inbounds i64, ptr %60, i64 1
// CHECK-NEXT:   %64 = load i64, ptr %63, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %64)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %65 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %66 = getelementptr inbounds i64, ptr %65, i64 0
// CHECK-NEXT:   store i64 1, ptr %66, align 8
// CHECK-NEXT:   %67 = getelementptr inbounds i64, ptr %65, i64 1
// CHECK-NEXT:   store i64 2, ptr %67, align 8
// CHECK-NEXT:   %68 = getelementptr inbounds i64, ptr %65, i64 2
// CHECK-NEXT:   store i64 3, ptr %68, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds i64, ptr %65, i64 3
// CHECK-NEXT:   store i64 4, ptr %69, align 8
// CHECK-NEXT:   %70 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %65, 0
// CHECK-NEXT:   %71 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %70, i64 4, 1
// CHECK-NEXT:   %72 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %71, i64 4, 2
// CHECK-NEXT:   %73 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %72, 0
// CHECK-NEXT:   %74 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %72, 1
// CHECK-NEXT:   %75 = icmp uge i64 1, %74
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %75, i64 1, i1 true, i64 %74)
// CHECK-NEXT:   %76 = getelementptr inbounds i64, ptr %73, i64 1
// CHECK-NEXT:   %77 = load i64, ptr %76, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %77)
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
