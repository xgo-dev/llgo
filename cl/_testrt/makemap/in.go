// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make1"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make2"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make3"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make4"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make5"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make6"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/makemap.make7"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	make1()
	make2()
	make3()
	make4()
	make5()
	make6()
	make7()
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %1)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %3)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 3, ptr %5, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %5)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 4 }, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %7, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %7)
// CHECK-NEXT:   %9 = load %"{{.*}}/runtime/internal/runtime.String", ptr %8, align 8
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_string", ptr %0, ptr %10)
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   %13 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_int]_llgo_string", ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %15 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %14)
// CHECK-NEXT:   %16 = extractvalue { i1, ptr, ptr } %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_13
// CHECK-NEXT:   %17 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %68, 1
// CHECK-NEXT:   %18 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %68, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 1 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_13
// CHECK-NEXT:   %19 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %0)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_int", i64 %19)
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_int]_llgo_string", ptr %0)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_3
// CHECK-NEXT:   %22 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %21)
// CHECK-NEXT:   %23 = extractvalue { i1, ptr, ptr } %22, 0
// CHECK-NEXT:   br i1 %23, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_16
// CHECK-NEXT:   %24 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %76, 1
// CHECK-NEXT:   %25 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %76, 2
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %25, ptr %26, align 8
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_int", ptr %20, ptr %26)
// CHECK-NEXT:   store i64 %24, ptr %27, align 8
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_16
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 4 }, ptr %28, align 8
// CHECK-NEXT:   %29 = call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"(ptr @"map[_llgo_string]_llgo_int", ptr %20, ptr %28)
// CHECK-NEXT:   %30 = extractvalue { ptr, i1 } %29, 0
// CHECK-NEXT:   %31 = load i64, ptr %30, align 8
// CHECK-NEXT:   %32 = extractvalue { ptr, i1 } %29, 1
// CHECK-NEXT:   %33 = insertvalue { i64, i1 } undef, i64 %31, 0
// CHECK-NEXT:   %34 = insertvalue { i64, i1 } %33, i1 %32, 1
// CHECK-NEXT:   %35 = extractvalue { i64, i1 } %34, 0
// CHECK-NEXT:   %36 = extractvalue { i64, i1 } %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %36)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 2 }, ptr %37, align 8
// CHECK-NEXT:   %38 = call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"(ptr @"map[_llgo_string]_llgo_int", ptr %20, ptr %37)
// CHECK-NEXT:   %39 = extractvalue { ptr, i1 } %38, 0
// CHECK-NEXT:   %40 = load i64, ptr %39, align 8
// CHECK-NEXT:   %41 = extractvalue { ptr, i1 } %38, 1
// CHECK-NEXT:   %42 = insertvalue { i64, i1 } undef, i64 %40, 0
// CHECK-NEXT:   %43 = insertvalue { i64, i1 } %42, i1 %41, 1
// CHECK-NEXT:   %44 = extractvalue { i64, i1 } %43, 0
// CHECK-NEXT:   %45 = extractvalue { i64, i1 } %43, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %44)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 4 }, ptr %46, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.MapDelete"(ptr @"map[_llgo_string]_llgo_int", ptr %20, ptr %46)
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 4 }, ptr %47, align 8
// CHECK-NEXT:   %48 = call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"(ptr @"map[_llgo_string]_llgo_int", ptr %20, ptr %47)
// CHECK-NEXT:   %49 = extractvalue { ptr, i1 } %48, 0
// CHECK-NEXT:   %50 = load i64, ptr %49, align 8
// CHECK-NEXT:   %51 = extractvalue { ptr, i1 } %48, 1
// CHECK-NEXT:   %52 = insertvalue { i64, i1 } undef, i64 %50, 0
// CHECK-NEXT:   %53 = insertvalue { i64, i1 } %52, i1 %51, 1
// CHECK-NEXT:   %54 = extractvalue { i64, i1 } %53, 0
// CHECK-NEXT:   %55 = extractvalue { i64, i1 } %53, 1
// CHECK-NEXT:   br i1 %55, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 7 }, ptr %56, align 8
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %56, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %57)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %58 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %20)
// CHECK-NEXT:   %59 = icmp ne i64 %58, 2
// CHECK-NEXT:   br i1 %59, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 7 }, ptr %60, align 8
// CHECK-NEXT:   %61 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %60, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %61)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_1
// CHECK-NEXT:   %62 = extractvalue { i1, ptr, ptr } %15, 1
// CHECK-NEXT:   %63 = extractvalue { i1, ptr, ptr } %15, 2
// CHECK-NEXT:   %64 = load i64, ptr %62, align 8
// CHECK-NEXT:   %65 = load %"{{.*}}/runtime/internal/runtime.String", ptr %63, align 8
// CHECK-NEXT:   %66 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } { i1 true, i64 undef, %"{{.*}}/runtime/internal/runtime.String" undef }, i64 %64, 1
// CHECK-NEXT:   %67 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %66, %"{{.*}}/runtime/internal/runtime.String" %65, 2
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_12, %_llgo_11
// CHECK-NEXT:   %68 = phi { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } [ %67, %_llgo_11 ], [ zeroinitializer, %_llgo_12 ]
// CHECK-NEXT:   %69 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %68, 0
// CHECK-NEXT:   br i1 %69, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %70 = extractvalue { i1, ptr, ptr } %22, 1
// CHECK-NEXT:   %71 = extractvalue { i1, ptr, ptr } %22, 2
// CHECK-NEXT:   %72 = load i64, ptr %70, align 8
// CHECK-NEXT:   %73 = load %"{{.*}}/runtime/internal/runtime.String", ptr %71, align 8
// CHECK-NEXT:   %74 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } { i1 true, i64 undef, %"{{.*}}/runtime/internal/runtime.String" undef }, i64 %72, 1
// CHECK-NEXT:   %75 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %74, %"{{.*}}/runtime/internal/runtime.String" %73, 2
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_4
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_15, %_llgo_14
// CHECK-NEXT:   %76 = phi { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } [ %75, %_llgo_14 ], [ zeroinitializer, %_llgo_15 ]
// CHECK-NEXT:   %77 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %76, 0
// CHECK-NEXT:   br i1 %77, label %_llgo_5, label %_llgo_6
// CHECK-NEXT: }
func make1() {
	m := make(map[int]string)
	m[1] = "hello"
	m[2] = "world"
	m[3] = "llgo"
	println(m, m[1], m[2], len(m))
	for k, v := range m {
		println(k, ":", v)
	}

	s := make(map[string]int, len(m))
	for k, v := range m {
		s[v] = k
	}

	id, ok := s["llgo"]
	println("llgo", id, ok)

	none, ok := s["go"]
	println("go", none, ok)

	delete(s, "llgo")
	if _, ok := s["llgo"]; ok {
		panic("bad key")
	}
	if len(s) != 2 {
		panic("bad len")
	}
}

type N1 [1]int

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr %0)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   %3 = icmp ne ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/runtime/internal/runtime.MapLen"(ptr null)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr null)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 false)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_any]_llgo_int", i64 0)
// CHECK-NEXT:   %6 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %6, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %6, i64 0
// CHECK-NEXT:   store i64 1, ptr %7, align 8
// CHECK-NEXT:   %8 = load [1 x i64], ptr %6, align 8
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store [1 x i64] %8, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/makemap.N1", ptr undef }, ptr %9, 1
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %10, ptr %11, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %5, ptr %11)
// CHECK-NEXT:   store i64 100, ptr %12, align 8
// CHECK-NEXT:   %13 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %13, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %14 = getelementptr inbounds i64, ptr %13, i64 0
// CHECK-NEXT:   store i64 2, ptr %14, align 8
// CHECK-NEXT:   %15 = load [1 x i64], ptr %13, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store [1 x i64] %15, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/makemap.N1", ptr undef }, ptr %16, 1
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %18, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %5, ptr %18)
// CHECK-NEXT:   store i64 200, ptr %19, align 8
// CHECK-NEXT:   %20 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %20, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %21 = getelementptr inbounds i64, ptr %20, i64 0
// CHECK-NEXT:   store i64 3, ptr %21, align 8
// CHECK-NEXT:   %22 = load [1 x i64], ptr %20, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store [1 x i64] %22, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/makemap.N1", ptr undef }, ptr %23, 1
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %24, ptr %25, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %5, ptr %25)
// CHECK-NEXT:   store i64 300, ptr %26, align 8
// CHECK-NEXT:   %27 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %27, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %28 = getelementptr inbounds i64, ptr %27, i64 0
// CHECK-NEXT:   store i64 2, ptr %28, align 8
// CHECK-NEXT:   %29 = load [1 x i64], ptr %27, align 8
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store [1 x i64] %29, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/makemap.N1", ptr undef }, ptr %30, 1
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %31, ptr %32, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %5, ptr %32)
// CHECK-NEXT:   store i64 -200, ptr %33, align 8
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_any]_llgo_int", ptr %5)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_7, %_llgo_0
// CHECK-NEXT:   %35 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %34)
// CHECK-NEXT:   %36 = extractvalue { i1, ptr, ptr } %35, 0
// CHECK-NEXT:   br i1 %36, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %37 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 1
// CHECK-NEXT:   %38 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 2
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 0
// CHECK-NEXT:   %40 = icmp eq ptr %39, @"_llgo_github.com/goplus/llgo/cl/_testrt/makemap.N1"
// CHECK-NEXT:   br i1 %40, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %41 = extractvalue { i1, ptr, ptr } %35, 1
// CHECK-NEXT:   %42 = extractvalue { i1, ptr, ptr } %35, 2
// CHECK-NEXT:   %43 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %41, align 8
// CHECK-NEXT:   %44 = load i64, ptr %42, align 8
// CHECK-NEXT:   %45 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } { i1 true, %"{{.*}}/runtime/internal/runtime.eface" undef, i64 undef }, %"{{.*}}/runtime/internal/runtime.eface" %43, 1
// CHECK-NEXT:   %46 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %45, i64 %44, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %47 = phi { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } [ %46, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %48 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 0
// CHECK-NEXT:   br i1 %48, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %49 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 1
// CHECK-NEXT:   %50 = load [1 x i64], ptr %49, align 8
// CHECK-NEXT:   %51 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %51, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store [1 x i64] %50, ptr %51, align 8
// CHECK-NEXT:   %52 = getelementptr inbounds i64, ptr %51, i64 0
// CHECK-NEXT:   %53 = load i64, ptr %52, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %53)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %54 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @32, i64 81 }, ptr %54, align 8
// CHECK-NEXT:   %55 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %54, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %55)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func make2() {
	m2 := make(map[int]string)
	println(m2, len(m2), m2 == nil, m2 != nil)
	var m3 map[int]string
	println(m3, len(m3), m3 == nil, m3 != nil)

	n := make(map[any]int)
	n[N1{1}] = 100
	n[N1{2}] = 200
	n[N1{3}] = 300
	n[N1{2}] = -200
	for k, v := range n {
		println(k.(N1)[0], v)
	}
}

type N struct {
	n1 int8
	n2 int8
}
type K [1]N
type K2 [1]*N

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca [1 x %"{{.*}}/cl/_testrt/makemap.N"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %0, i64 0
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %2, align 1
// CHECK-NEXT:   store i8 2, ptr %3, align 1
// CHECK-NEXT:   %4 = load [1 x %"{{.*}}/cl/_testrt/makemap.N"], ptr %0, align 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [1 x %"{{.*}}/cl/_testrt/makemap.N"] %4, ptr %5, align 1
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K", ptr undef }, ptr %5, 1
// CHECK-NEXT:   %7 = alloca [1 x %"{{.*}}/cl/_testrt/makemap.N"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %7, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %7, i64 0
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %8, i32 0, i32 0
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %9, align 1
// CHECK-NEXT:   store i8 2, ptr %10, align 1
// CHECK-NEXT:   %11 = load [1 x %"{{.*}}/cl/_testrt/makemap.N"], ptr %7, align 1
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [1 x %"{{.*}}/cl/_testrt/makemap.N"] %11, ptr %12, align 1
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K", ptr undef }, ptr %12, 1
// CHECK-NEXT:   %14 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %6, %"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_any]_llgo_int", i64 0)
// CHECK-NEXT:   %16 = alloca [1 x %"{{.*}}/cl/_testrt/makemap.N"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %16, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %16, i64 0
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %17, i32 0, i32 0
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %17, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %18, align 1
// CHECK-NEXT:   store i8 2, ptr %19, align 1
// CHECK-NEXT:   %20 = load [1 x %"{{.*}}/cl/_testrt/makemap.N"], ptr %16, align 1
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [1 x %"{{.*}}/cl/_testrt/makemap.N"] %20, ptr %21, align 1
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K", ptr undef }, ptr %21, 1
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %22, ptr %23, align 8
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %15, ptr %23)
// CHECK-NEXT:   store i64 100, ptr %24, align 8
// CHECK-NEXT:   %25 = alloca [1 x %"{{.*}}/cl/_testrt/makemap.N"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %25, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %25, i64 0
// CHECK-NEXT:   %27 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %26, i32 0, i32 0
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %26, i32 0, i32 1
// CHECK-NEXT:   store i8 3, ptr %27, align 1
// CHECK-NEXT:   store i8 4, ptr %28, align 1
// CHECK-NEXT:   %29 = load [1 x %"{{.*}}/cl/_testrt/makemap.N"], ptr %25, align 1
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [1 x %"{{.*}}/cl/_testrt/makemap.N"] %29, ptr %30, align 1
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K", ptr undef }, ptr %30, 1
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %31, ptr %32, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %15, ptr %32)
// CHECK-NEXT:   store i64 200, ptr %33, align 8
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_any]_llgo_int", ptr %15)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_7, %_llgo_0
// CHECK-NEXT:   %35 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %34)
// CHECK-NEXT:   %36 = extractvalue { i1, ptr, ptr } %35, 0
// CHECK-NEXT:   br i1 %36, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %37 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 1
// CHECK-NEXT:   %38 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 2
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 0
// CHECK-NEXT:   %40 = icmp eq ptr %39, @"_llgo_{{.*}}/cl/_testrt/makemap.K"
// CHECK-NEXT:   br i1 %40, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %41 = extractvalue { i1, ptr, ptr } %35, 1
// CHECK-NEXT:   %42 = extractvalue { i1, ptr, ptr } %35, 2
// CHECK-NEXT:   %43 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %41, align 8
// CHECK-NEXT:   %44 = load i64, ptr %42, align 8
// CHECK-NEXT:   %45 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } { i1 true, %"{{.*}}/runtime/internal/runtime.eface" undef, i64 undef }, %"{{.*}}/runtime/internal/runtime.eface" %43, 1
// CHECK-NEXT:   %46 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %45, i64 %44, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %47 = phi { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } [ %46, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %48 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %47, 0
// CHECK-NEXT:   br i1 %48, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %49 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %37, 1
// CHECK-NEXT:   %50 = load [1 x %"{{.*}}/cl/_testrt/makemap.N"], ptr %49, align 1
// CHECK-NEXT:   %51 = alloca [1 x %"{{.*}}/cl/_testrt/makemap.N"], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %51, i8 0, i64 2, i1 false)
// CHECK-NEXT:   store [1 x %"{{.*}}/cl/_testrt/makemap.N"] %50, ptr %51, align 1
// CHECK-NEXT:   %52 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %51, i64 0
// CHECK-NEXT:   %53 = load %"{{.*}}/cl/_testrt/makemap.N", ptr %52, align 1
// CHECK-NEXT:   %54 = extractvalue %"{{.*}}/cl/_testrt/makemap.N" %53, 0
// CHECK-NEXT:   %55 = sext i8 %54 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 80 }, ptr %56, align 8
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %56, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %57)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func make3() {
	var a any = K{N{1, 2}}
	var b any = K{N{1, 2}}
	println(a == b)

	m := make(map[any]int)
	m[K{N{1, 2}}] = 100
	m[K{N{3, 4}}] = 200
	for k, v := range m {
		println(k.(K)[0].n1, v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca [1 x ptr], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds ptr, ptr %0, i64 0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %3, align 1
// CHECK-NEXT:   store i8 2, ptr %4, align 1
// CHECK-NEXT:   store ptr %2, ptr %1, align 8
// CHECK-NEXT:   %5 = load [1 x ptr], ptr %0, align 8
// CHECK-NEXT:   %6 = extractvalue [1 x ptr] %5, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K2", ptr undef }, ptr %6, 1
// CHECK-NEXT:   %8 = alloca [1 x ptr], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %8, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %9 = getelementptr inbounds ptr, ptr %8, i64 0
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %10, i32 0, i32 0
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %10, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %11, align 1
// CHECK-NEXT:   store i8 2, ptr %12, align 1
// CHECK-NEXT:   store ptr %10, ptr %9, align 8
// CHECK-NEXT:   %13 = load [1 x ptr], ptr %8, align 8
// CHECK-NEXT:   %14 = extractvalue [1 x ptr] %13, 0
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K2", ptr undef }, ptr %14, 1
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %7, %"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_any]_llgo_int", i64 0)
// CHECK-NEXT:   %18 = alloca [1 x ptr], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %18, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %19 = getelementptr inbounds ptr, ptr %18, i64 0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %20, i32 0, i32 0
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %20, i32 0, i32 1
// CHECK-NEXT:   store i8 1, ptr %21, align 1
// CHECK-NEXT:   store i8 2, ptr %22, align 1
// CHECK-NEXT:   store ptr %20, ptr %19, align 8
// CHECK-NEXT:   %23 = load [1 x ptr], ptr %18, align 8
// CHECK-NEXT:   %24 = extractvalue [1 x ptr] %23, 0
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K2", ptr undef }, ptr %24, 1
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %25, ptr %26, align 8
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %17, ptr %26)
// CHECK-NEXT:   store i64 100, ptr %27, align 8
// CHECK-NEXT:   %28 = alloca [1 x ptr], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %28, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %29 = getelementptr inbounds ptr, ptr %28, i64 0
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %30, i32 0, i32 0
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %30, i32 0, i32 1
// CHECK-NEXT:   store i8 3, ptr %31, align 1
// CHECK-NEXT:   store i8 4, ptr %32, align 1
// CHECK-NEXT:   store ptr %30, ptr %29, align 8
// CHECK-NEXT:   %33 = load [1 x ptr], ptr %28, align 8
// CHECK-NEXT:   %34 = extractvalue [1 x ptr] %33, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/makemap.K2", ptr undef }, ptr %34, 1
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %35, ptr %36, align 8
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_any]_llgo_int", ptr %17, ptr %36)
// CHECK-NEXT:   store i64 200, ptr %37, align 8
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_any]_llgo_int", ptr %17)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_7, %_llgo_0
// CHECK-NEXT:   %39 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %38)
// CHECK-NEXT:   %40 = extractvalue { i1, ptr, ptr } %39, 0
// CHECK-NEXT:   br i1 %40, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %41 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %51, 1
// CHECK-NEXT:   %42 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %51, 2
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %41, 0
// CHECK-NEXT:   %44 = icmp eq ptr %43, @"_llgo_{{.*}}/cl/_testrt/makemap.K2"
// CHECK-NEXT:   br i1 %44, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %45 = extractvalue { i1, ptr, ptr } %39, 1
// CHECK-NEXT:   %46 = extractvalue { i1, ptr, ptr } %39, 2
// CHECK-NEXT:   %47 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %45, align 8
// CHECK-NEXT:   %48 = load i64, ptr %46, align 8
// CHECK-NEXT:   %49 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } { i1 true, %"{{.*}}/runtime/internal/runtime.eface" undef, i64 undef }, %"{{.*}}/runtime/internal/runtime.eface" %47, 1
// CHECK-NEXT:   %50 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %49, i64 %48, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %51 = phi { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } [ %50, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %52 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.eface", i64 } %51, 0
// CHECK-NEXT:   br i1 %52, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %53 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %41, 1
// CHECK-NEXT:   %54 = insertvalue [1 x ptr] undef, ptr %53, 0
// CHECK-NEXT:   %55 = alloca [1 x ptr], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %55, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store [1 x ptr] %54, ptr %55, align 8
// CHECK-NEXT:   %56 = getelementptr inbounds ptr, ptr %55, i64 0
// CHECK-NEXT:   %57 = load ptr, ptr %56, align 8
// CHECK-NEXT:   %58 = getelementptr inbounds %"{{.*}}/cl/_testrt/makemap.N", ptr %57, i32 0, i32 0
// CHECK-NEXT:   %59 = load i8, ptr %58, align 1
// CHECK-NEXT:   %60 = sext i8 %59 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %60)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %61 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 81 }, ptr %61, align 8
// CHECK-NEXT:   %62 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %61, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %62)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func make4() {
	var a any = K2{&N{1, 2}}
	var b any = K2{&N{1, 2}}
	println(a == b)

	m := make(map[any]int)
	m[K2{&N{1, 2}}] = 100
	m[K2{&N{3, 4}}] = 200
	for k, v := range m {
		println(k.(K2)[0].n1, v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %1, %"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[chan _llgo_int]_llgo_int", i64 0)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[chan _llgo_int]_llgo_int", ptr %4, ptr %5)
// CHECK-NEXT:   store i64 100, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store ptr %0, ptr %7, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[chan _llgo_int]_llgo_int", ptr %4, ptr %7)
// CHECK-NEXT:   store i64 200, ptr %8, align 8
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[chan _llgo_int]_llgo_int", ptr %4)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %10 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %9)
// CHECK-NEXT:   %11 = extractvalue { i1, ptr, ptr } %10, 0
// CHECK-NEXT:   br i1 %11, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %12 = extractvalue { i1, ptr, i64 } %20, 1
// CHECK-NEXT:   %13 = extractvalue { i1, ptr, i64 } %20, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %14 = extractvalue { i1, ptr, ptr } %10, 1
// CHECK-NEXT:   %15 = extractvalue { i1, ptr, ptr } %10, 2
// CHECK-NEXT:   %16 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %17 = load i64, ptr %15, align 8
// CHECK-NEXT:   %18 = insertvalue { i1, ptr, i64 } { i1 true, ptr undef, i64 undef }, ptr %16, 1
// CHECK-NEXT:   %19 = insertvalue { i1, ptr, i64 } %18, i64 %17, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %20 = phi { i1, ptr, i64 } [ %19, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %21 = extractvalue { i1, ptr, i64 } %20, 0
// CHECK-NEXT:   br i1 %21, label %_llgo_2, label %_llgo_3
// CHECK-NEXT: }

func make5() {
	ch := make(chan int)
	var a any = ch
	var b any = ch
	println(a == b)
	m := make(map[chan int]int)
	m[ch] = 100
	m[ch] = 200
	for k, v := range m {
		println(k, v)
	}
}

type M map[int]string

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 0)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"_llgo_{{.*}}/cl/_testrt/makemap.M", ptr %0, ptr %1)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"_llgo_{{.*}}/cl/_testrt/makemap.M", ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %4 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %3)
// CHECK-NEXT:   %5 = extractvalue { i1, ptr, ptr } %4, 0
// CHECK-NEXT:   br i1 %5, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %6 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %14, 1
// CHECK-NEXT:   %7 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %14, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %8 = extractvalue { i1, ptr, ptr } %4, 1
// CHECK-NEXT:   %9 = extractvalue { i1, ptr, ptr } %4, 2
// CHECK-NEXT:   %10 = load i64, ptr %8, align 8
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.String", ptr %9, align 8
// CHECK-NEXT:   %12 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } { i1 true, i64 undef, %"{{.*}}/runtime/internal/runtime.String" undef }, i64 %10, 1
// CHECK-NEXT:   %13 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %12, %"{{.*}}/runtime/internal/runtime.String" %11, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %14 = phi { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } [ %13, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %15 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %14, 0
// CHECK-NEXT:   br i1 %15, label %_llgo_2, label %_llgo_3
// CHECK-NEXT: }

func make6() {
	var m M
	m = make(map[int]string)
	m[1] = "hello"
	for k, v := range m {
		println(k, v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/makemap.make7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_{{.*}}/cl/_testrt/makemap.N.7.0]_llgo_string", i64 2)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_{{.*}}/cl/_testrt/makemap.N.7.0]_llgo_string", ptr %0, ptr %1)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_{{.*}}/cl/_testrt/makemap.N.7.0]_llgo_string", ptr %0, ptr %3)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_{{.*}}/cl/_testrt/makemap.N.7.0]_llgo_string", ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %6 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %5)
// CHECK-NEXT:   %7 = extractvalue { i1, ptr, ptr } %6, 0
// CHECK-NEXT:   br i1 %7, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %8 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %19, 1
// CHECK-NEXT:   %9 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %19, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_{{.*}}/cl/_testrt/makemap.N.7.0]_llgo_string", ptr %0, ptr %10)
// CHECK-NEXT:   %12 = load %"{{.*}}/runtime/internal/runtime.String", ptr %11, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %13 = extractvalue { i1, ptr, ptr } %6, 1
// CHECK-NEXT:   %14 = extractvalue { i1, ptr, ptr } %6, 2
// CHECK-NEXT:   %15 = load i64, ptr %13, align 8
// CHECK-NEXT:   %16 = load %"{{.*}}/runtime/internal/runtime.String", ptr %14, align 8
// CHECK-NEXT:   %17 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } { i1 true, i64 undef, %"{{.*}}/runtime/internal/runtime.String" undef }, i64 %15, 1
// CHECK-NEXT:   %18 = insertvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %17, %"{{.*}}/runtime/internal/runtime.String" %16, 2
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %19 = phi { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } [ %18, %_llgo_4 ], [ zeroinitializer, %_llgo_5 ]
// CHECK-NEXT:   %20 = extractvalue { i1, i64, %"{{.*}}/runtime/internal/runtime.String" } %19, 0
// CHECK-NEXT:   br i1 %20, label %_llgo_2, label %_llgo_3
// CHECK-NEXT: }

func make7() {
	type N int
	m := map[N]string{
		1: "hello",
		2: "world",
	}
	for k, v := range m {
		println(k, v)
	}
	println(m[1])
}
