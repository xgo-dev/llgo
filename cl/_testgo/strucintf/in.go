// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/foo"

// CHECK-LINE: @4 = private unnamed_addr constant [11 x i8] c"Foo: not ok", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [11 x i8] c"Bar: not ok", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [9 x i8] c"F: not ok", align 1

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testgo/strucintf.Foo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds { i64 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   %3 = load { i64 }, ptr %0, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/strucintf.struct$MYpsoM99ZwFY087IpUOkIw1zjBA_sgFXVodmn1m-G88", ptr undef }, ptr %4, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.eface" %5
// CHECK-NEXT: }

func Foo() any {
	return struct{ v int }{1}
}

func main() {
	v := Foo()

	if x, ok := v.(struct{ v int }); ok {
		println(x.v)
	} else {
		println("Foo: not ok")
	}

	bar := foo.Bar()

	if x, ok := bar.(struct{ V int }); ok {
		println(x.V)
	} else {
		println("Bar: not ok")
	}

	if x, ok := foo.F().(struct{ v int }); ok {
		println(x.v)
	} else {
		println("F: not ok")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/strucintf.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/strucintf.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/strucintf.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/foo.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/strucintf.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testgo/strucintf.Foo"()
// CHECK-NEXT:   %1 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %3 = icmp eq ptr %2, @"{{.*}}/cl/_testgo/strucintf.struct$MYpsoM99ZwFY087IpUOkIw1zjBA_sgFXVodmn1m-G88"
// CHECK-NEXT:   br i1 %3, label %_llgo_10, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_12
// CHECK-NEXT:   %4 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds { i64 }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_1
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.Bar"()
// CHECK-NEXT:   %8 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %8, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 0
// CHECK-NEXT:   %10 = icmp eq ptr %9, @"_llgo_struct$K-dZ9QotZfVPz2a0YdRa9vmZUuDXPTqZOlMShKEDJtk"
// CHECK-NEXT:   br i1 %10, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 11 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_15
// CHECK-NEXT:   %11 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds { i64 }, ptr %8, i32 0, i32 0
// CHECK-NEXT:   %13 = load i64, ptr %12, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_6, %_llgo_4
// CHECK-NEXT:   %14 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %14, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.F"()
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %15, 0
// CHECK-NEXT:   %17 = icmp eq ptr %16, @"{{.*}}/cl/_testgo/strucintf.struct$MYpsoM99ZwFY087IpUOkIw1zjBA_sgFXVodmn1m-G88"
// CHECK-NEXT:   br i1 %17, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_15
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 11 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_18
// CHECK-NEXT:   %18 = icmp eq ptr %14, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %18)
// CHECK-NEXT:   %19 = getelementptr inbounds { i64 }, ptr %14, i32 0, i32 0
// CHECK-NEXT:   %20 = load i64, ptr %19, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %20)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_9, %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_18
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_0
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %22 = load { i64 }, ptr %21, align 8
// CHECK-NEXT:   %23 = insertvalue { { i64 }, i1 } undef, { i64 } %22, 0
// CHECK-NEXT:   %24 = insertvalue { { i64 }, i1 } %23, i1 true, 1
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_11, %_llgo_10
// CHECK-NEXT:   %25 = phi { { i64 }, i1 } [ %24, %_llgo_10 ], [ zeroinitializer, %_llgo_11 ]
// CHECK-NEXT:   %26 = extractvalue { { i64 }, i1 } %25, 0
// CHECK-NEXT:   store { i64 } %26, ptr %1, align 8
// CHECK-NEXT:   %27 = extractvalue { { i64 }, i1 } %25, 1
// CHECK-NEXT:   br i1 %27, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_2
// CHECK-NEXT:   %28 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 1
// CHECK-NEXT:   %29 = load { i64 }, ptr %28, align 8
// CHECK-NEXT:   %30 = insertvalue { { i64 }, i1 } undef, { i64 } %29, 0
// CHECK-NEXT:   %31 = insertvalue { { i64 }, i1 } %30, i1 true, 1
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14, %_llgo_13
// CHECK-NEXT:   %32 = phi { { i64 }, i1 } [ %31, %_llgo_13 ], [ zeroinitializer, %_llgo_14 ]
// CHECK-NEXT:   %33 = extractvalue { { i64 }, i1 } %32, 0
// CHECK-NEXT:   store { i64 } %33, ptr %8, align 8
// CHECK-NEXT:   %34 = extractvalue { { i64 }, i1 } %32, 1
// CHECK-NEXT:   br i1 %34, label %_llgo_4, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_5
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %15, 1
// CHECK-NEXT:   %36 = load { i64 }, ptr %35, align 8
// CHECK-NEXT:   %37 = insertvalue { { i64 }, i1 } undef, { i64 } %36, 0
// CHECK-NEXT:   %38 = insertvalue { { i64 }, i1 } %37, i1 true, 1
// CHECK-NEXT:   br label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_17, %_llgo_16
// CHECK-NEXT:   %39 = phi { { i64 }, i1 } [ %38, %_llgo_16 ], [ zeroinitializer, %_llgo_17 ]
// CHECK-NEXT:   %40 = extractvalue { { i64 }, i1 } %39, 0
// CHECK-NEXT:   store { i64 } %40, ptr %14, align 8
// CHECK-NEXT:   %41 = extractvalue { { i64 }, i1 } %39, 1
// CHECK-NEXT:   br i1 %41, label %_llgo_7, label %_llgo_9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
