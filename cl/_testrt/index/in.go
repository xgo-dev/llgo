// LITTEST
package main

// CHECK-LINE: %"{{.*}}/cl/_testrt/index.point" = type { i64, i64 }
// CHECK-LINE: %"{{.*}}/runtime/internal/runtime.String" = type { ptr, i64 }
// CHECK-LINE: %"{{.*}}/runtime/internal/runtime.Slice" = type { ptr, i64, i64 }
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
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 1
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 2
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %8, i32 0, i32 0
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %3, align 8
// CHECK-NEXT:   store i64 2, ptr %4, align 8
// CHECK-NEXT:   store i64 3, ptr %6, align 8
// CHECK-NEXT:   store i64 4, ptr %7, align 8
// CHECK-NEXT:   store i64 5, ptr %9, align 8
// CHECK-NEXT:   store i64 6, ptr %10, align 8
// CHECK-NEXT:   %11 = load [3 x %"{{.*}}/cl/_testrt/index.point"], ptr %1, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %1, i64 2
// CHECK-NEXT:   %13 = load %"{{.*}}/cl/_testrt/index.point", ptr %12, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/index.point" %13, ptr %0, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %15 = load i64, ptr %14, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/cl/_testrt/index.point", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %17 = load i64, ptr %16, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %18, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %19 = alloca [2 x [2 x i64]], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %19, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %20 = getelementptr inbounds [2 x i64], ptr %19, i64 0
// CHECK-NEXT:   %21 = getelementptr inbounds i64, ptr %20, i64 0
// CHECK-NEXT:   %22 = getelementptr inbounds i64, ptr %20, i64 1
// CHECK-NEXT:   %23 = getelementptr inbounds [2 x i64], ptr %19, i64 1
// CHECK-NEXT:   %24 = getelementptr inbounds i64, ptr %23, i64 0
// CHECK-NEXT:   %25 = getelementptr inbounds i64, ptr %23, i64 1
// CHECK-NEXT:   store i64 1, ptr %21, align 8
// CHECK-NEXT:   store i64 2, ptr %22, align 8
// CHECK-NEXT:   store i64 3, ptr %24, align 8
// CHECK-NEXT:   store i64 4, ptr %25, align 8
// CHECK-NEXT:   %26 = load [2 x [2 x i64]], ptr %19, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds [2 x i64], ptr %19, i64 1
// CHECK-NEXT:   %28 = load [2 x i64], ptr %27, align 8
// CHECK-NEXT:   store [2 x i64] %28, ptr %18, align 8
// CHECK-NEXT:   %29 = getelementptr inbounds i64, ptr %18, i64 0
// CHECK-NEXT:   %30 = load i64, ptr %29, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds i64, ptr %18, i64 1
// CHECK-NEXT:   %32 = load i64, ptr %31, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %30)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %33 = alloca [5 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %33, i8 0, i64 40, i1 false)
// CHECK-NEXT:   %34 = getelementptr inbounds i64, ptr %33, i64 0
// CHECK-NEXT:   %35 = getelementptr inbounds i64, ptr %33, i64 1
// CHECK-NEXT:   %36 = getelementptr inbounds i64, ptr %33, i64 2
// CHECK-NEXT:   %37 = getelementptr inbounds i64, ptr %33, i64 3
// CHECK-NEXT:   %38 = getelementptr inbounds i64, ptr %33, i64 4
// CHECK-NEXT:   store i64 1, ptr %34, align 8
// CHECK-NEXT:   store i64 2, ptr %35, align 8
// CHECK-NEXT:   store i64 3, ptr %36, align 8
// CHECK-NEXT:   store i64 4, ptr %37, align 8
// CHECK-NEXT:   store i64 5, ptr %38, align 8
// CHECK-NEXT:   %39 = load [5 x i64], ptr %33, align 8
// CHECK-NEXT:   %40 = getelementptr inbounds i64, ptr %33, i64 2
// CHECK-NEXT:   %41 = load i64, ptr %40, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %42 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 2), align 1
// CHECK-NEXT:   %43 = zext i8 %42 to i64
// CHECK-NEXT:   %44 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %43)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %44)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %45 = load i8, ptr getelementptr inbounds (i8, ptr @0, i64 1), align 1
// CHECK-NEXT:   %46 = zext i8 %45 to i64
// CHECK-NEXT:   %47 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %46)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %49 = getelementptr inbounds i64, ptr %48, i64 0
// CHECK-NEXT:   %50 = getelementptr inbounds i64, ptr %48, i64 1
// CHECK-NEXT:   store i64 1, ptr %49, align 8
// CHECK-NEXT:   store i64 2, ptr %50, align 8
// CHECK-NEXT:   %51 = getelementptr inbounds i64, ptr %48, i64 1
// CHECK-NEXT:   %52 = load i64, ptr %51, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %52)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %53 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %54 = getelementptr inbounds i64, ptr %53, i64 0
// CHECK-NEXT:   store i64 1, ptr %54, align 8
// CHECK-NEXT:   %55 = getelementptr inbounds i64, ptr %53, i64 1
// CHECK-NEXT:   store i64 2, ptr %55, align 8
// CHECK-NEXT:   %56 = getelementptr inbounds i64, ptr %53, i64 2
// CHECK-NEXT:   store i64 3, ptr %56, align 8
// CHECK-NEXT:   %57 = getelementptr inbounds i64, ptr %53, i64 3
// CHECK-NEXT:   store i64 4, ptr %57, align 8
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %53, 0
// CHECK-NEXT:   %59 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %58, i64 4, 1
// CHECK-NEXT:   %60 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %59, i64 4, 2
// CHECK-NEXT:   %61 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %60, 0
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %60, 1
// CHECK-NEXT:   %63 = icmp uge i64 1, %62
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %63)
// CHECK-NEXT:   %64 = getelementptr inbounds i64, ptr %61, i64 1
// CHECK-NEXT:   %65 = load i64, ptr %64, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %65)
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
