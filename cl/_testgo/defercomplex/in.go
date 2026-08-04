// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"cleanup-final", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [19 x i8] c"cleanup-before-loop", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"exit-outer", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [9 x i8] c"post-loop", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [11 x i8] c"branch-even", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"branch-odd", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"nested", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [11 x i8] c"nested-tail", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [1 x i8] c"-", align 1{{$}}

func main() {
	for _, label := range complexOrder() {
		println(label)
	}
}

func complexOrder() (res []string) {
	record := func(label string) { res = append(res, label) }

	defer record(label1("cleanup-final", 0))
	defer record(label1("cleanup-before-loop", 0))

	for i := 0; i < 2; i++ {
		defer record(label1("exit-outer", i))
		for j := 0; j < 2; j++ {
			if j == 0 {
				defer record(label2("branch-even", i, j))
			} else {
				defer record(label2("branch-odd", i, j))
			}
			for k := 0; k < 2; k++ {
				nested := label3("nested", i, j, k)
				defer record(nested)
				if k == 1 {
					defer record(label3("nested-tail", i, j, k))
				}
			}
		}
	}

	defer record(label1("post-loop", 0))
	return
}

func label1(prefix string, a int) string {
	return prefix + "-" + digit(a)
}

func label2(prefix string, a, b int) string {
	return prefix + "-" + digit(a) + "-" + digit(b)
}

func label3(prefix string, a, b, c int) string {
	return prefix + "-" + digit(a) + "-" + digit(b) + "-" + digit(c)
}

func digit(n int) string {
	return string(rune('0' + n))
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @main.complexOrder(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"main.complexOrder$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @main.label1(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 }, i64 0)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %6 = alloca i8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 0
// CHECK-NEXT:   store ptr %6, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 2
// CHECK-NEXT:   store ptr %5, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %11, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %7)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 4
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %15, align 8
// CHECK-NEXT:   %16 = call i32 @{{(__)?}}sigsetjmp(ptr %6, i32 0)
// CHECK-NEXT:   %17 = icmp eq i32 %16, 0
// CHECK-NEXT:   br i1 %17, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_17
// CHECK-NEXT:   %18 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %0, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_7, %_llgo_18
// CHECK-NEXT:   %19 = phi i64 [ 0, %_llgo_18 ], [ %38, %_llgo_7 ]
// CHECK-NEXT:   %20 = icmp slt i64 %19, 2
// CHECK-NEXT:   br i1 %20, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.String" @main.label1(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 }, i64 %19)
// CHECK-NEXT:   %22 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %21, ptr %27, align 8
// CHECK-NEXT:   store ptr %23, ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %28 = call %"{{.*}}/runtime/internal/runtime.String" @main.label1(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 9 }, i64 0)
// CHECK-NEXT:   %29 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %31 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 0
// CHECK-NEXT:   store ptr %29, ptr %31, align 8
// CHECK-NEXT:   %32 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 1
// CHECK-NEXT:   store i64 3, ptr %32, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %33, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %28, ptr %34, align 8
// CHECK-NEXT:   store ptr %30, ptr %15, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_20), ptr %14, align 8
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_13, %_llgo_3
// CHECK-NEXT:   %35 = phi i64 [ 0, %_llgo_3 ], [ %63, %_llgo_13 ]
// CHECK-NEXT:   %36 = icmp slt i64 %35, 2
// CHECK-NEXT:   br i1 %36, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %37 = icmp eq i64 %35, 0
// CHECK-NEXT:   br i1 %37, label %_llgo_8, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %38 = add i64 %19, 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %39 = call %"{{.*}}/runtime/internal/runtime.String" @main.label2(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 11 }, i64 %19, i64 %35)
// CHECK-NEXT:   %40 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %42 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %41, i32 0, i32 0
// CHECK-NEXT:   store ptr %40, ptr %42, align 8
// CHECK-NEXT:   %43 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %41, i32 0, i32 1
// CHECK-NEXT:   store i64 4, ptr %43, align 8
// CHECK-NEXT:   %44 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %41, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %44, align 8
// CHECK-NEXT:   %45 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %41, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %39, ptr %45, align 8
// CHECK-NEXT:   store ptr %41, ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_10, %_llgo_8
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %46 = call %"{{.*}}/runtime/internal/runtime.String" @main.label2(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 }, i64 %19, i64 %35)
// CHECK-NEXT:   %47 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %49 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %48, i32 0, i32 0
// CHECK-NEXT:   store ptr %47, ptr %49, align 8
// CHECK-NEXT:   %50 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %48, i32 0, i32 1
// CHECK-NEXT:   store i64 5, ptr %50, align 8
// CHECK-NEXT:   %51 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %48, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %51, align 8
// CHECK-NEXT:   %52 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %48, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %46, ptr %52, align 8
// CHECK-NEXT:   store ptr %48, ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_15, %_llgo_9
// CHECK-NEXT:   %53 = phi i64 [ 0, %_llgo_9 ], [ %71, %_llgo_15 ]
// CHECK-NEXT:   %54 = icmp slt i64 %53, 2
// CHECK-NEXT:   br i1 %54, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %55 = call %"{{.*}}/runtime/internal/runtime.String" @main.label3(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, i64 %19, i64 %35, i64 %53)
// CHECK-NEXT:   %56 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %58 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %57, i32 0, i32 0
// CHECK-NEXT:   store ptr %56, ptr %58, align 8
// CHECK-NEXT:   %59 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %57, i32 0, i32 1
// CHECK-NEXT:   store i64 6, ptr %59, align 8
// CHECK-NEXT:   %60 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %57, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %60, align 8
// CHECK-NEXT:   %61 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %57, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %55, ptr %61, align 8
// CHECK-NEXT:   store ptr %57, ptr %15, align 8
// CHECK-NEXT:   %62 = icmp eq i64 %53, 1
// CHECK-NEXT:   br i1 %62, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %63 = add i64 %35, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %64 = call %"{{.*}}/runtime/internal/runtime.String" @main.label3(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 11 }, i64 %19, i64 %35, i64 %53)
// CHECK-NEXT:   %65 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %66 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %67 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %66, i32 0, i32 0
// CHECK-NEXT:   store ptr %65, ptr %67, align 8
// CHECK-NEXT:   %68 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %66, i32 0, i32 1
// CHECK-NEXT:   store i64 7, ptr %68, align 8
// CHECK-NEXT:   %69 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %66, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %69, align 8
// CHECK-NEXT:   %70 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %66, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %64, ptr %70, align 8
// CHECK-NEXT:   store ptr %66, ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14, %_llgo_12
// CHECK-NEXT:   %71 = add i64 %53, 1
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_19, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_27), ptr %13, align 8
// CHECK-NEXT:   %72 = load i64, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_19, %_llgo_79
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %5)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_0
// CHECK-NEXT:   %73 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %74 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %75 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %74, i32 0, i32 0
// CHECK-NEXT:   store ptr %73, ptr %75, align 8
// CHECK-NEXT:   %76 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %74, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %76, align 8
// CHECK-NEXT:   %77 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %74, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %77, align 8
// CHECK-NEXT:   %78 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %74, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %4, ptr %78, align 8
// CHECK-NEXT:   store ptr %74, ptr %15, align 8
// CHECK-NEXT:   %79 = call %"{{.*}}/runtime/internal/runtime.String" @main.label1(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 19 }, i64 0)
// CHECK-NEXT:   %80 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %81 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %82 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %81, i32 0, i32 0
// CHECK-NEXT:   store ptr %80, ptr %82, align 8
// CHECK-NEXT:   %83 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %81, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %83, align 8
// CHECK-NEXT:   %84 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %81, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %84, align 8
// CHECK-NEXT:   %85 = getelementptr inbounds { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %81, i32 0, i32 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %79, ptr %85, align 8
// CHECK-NEXT:   store ptr %81, ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_17), ptr %14, align 8
// CHECK-NEXT:   %86 = load ptr, ptr %13, align 8
// CHECK-NEXT:   indirectbr ptr %86, [label %_llgo_17, label %_llgo_21, label %_llgo_22, label %_llgo_23, label %_llgo_24, label %_llgo_25, label %_llgo_26, label %_llgo_27, label %_llgo_16]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_79
// CHECK-NEXT:   %87 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %0, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %87
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_19, %_llgo_77
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_17), ptr %13, align 8
// CHECK-NEXT:   %88 = load i64, ptr %12, align 8
// CHECK-NEXT:   %89 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %90 = icmp ne ptr %89, null
// CHECK-NEXT:   br i1 %90, label %_llgo_78, label %_llgo_79
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_19, %_llgo_54
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_21), ptr %13, align 8
// CHECK-NEXT:   %91 = load i64, ptr %12, align 8
// CHECK-NEXT:   %92 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %93 = icmp ne ptr %92, null
// CHECK-NEXT:   br i1 %93, label %_llgo_76, label %_llgo_77
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_19, %_llgo_52
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_22), ptr %13, align 8
// CHECK-NEXT:   %94 = load i64, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_19, %_llgo_25
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %95 = load i64, ptr %12, align 8
// CHECK-NEXT:   %96 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %97 = icmp ne ptr %96, null
// CHECK-NEXT:   br i1 %97, label %_llgo_51, label %_llgo_52
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_19, %_llgo_26
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_24), ptr %13, align 8
// CHECK-NEXT:   %98 = load i64, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_19, %_llgo_27
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_25), ptr %13, align 8
// CHECK-NEXT:   %99 = load i64, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_19, %_llgo_29
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_26), ptr %13, align 8
// CHECK-NEXT:   %100 = load i64, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_50, %_llgo_48, %_llgo_46, %_llgo_44, %_llgo_42, %_llgo_16
// CHECK-NEXT:   %101 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %102 = icmp ne ptr %101, null
// CHECK-NEXT:   br i1 %102, label %_llgo_30, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_39, %_llgo_28
// CHECK-NEXT:   br label %_llgo_27
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %103 = load { ptr, i64 }, ptr %101, align 8
// CHECK-NEXT:   %104 = extractvalue { ptr, i64 } %103, 1
// CHECK-NEXT:   br label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_30
// CHECK-NEXT:   %105 = icmp eq i64 %104, 2
// CHECK-NEXT:   br i1 %105, label %_llgo_32, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_31
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %13, align 8
// CHECK-NEXT:   %106 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %107 = icmp ne ptr %106, null
// CHECK-NEXT:   br i1 %107, label %_llgo_41, label %_llgo_42
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_31
// CHECK-NEXT:   %108 = icmp eq i64 %104, 4
// CHECK-NEXT:   br i1 %108, label %_llgo_34, label %_llgo_35
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_34:                                         ; preds = %_llgo_33
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %13, align 8
// CHECK-NEXT:   %109 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %110 = icmp ne ptr %109, null
// CHECK-NEXT:   br i1 %110, label %_llgo_43, label %_llgo_44
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_35:                                         ; preds = %_llgo_33
// CHECK-NEXT:   %111 = icmp eq i64 %104, 5
// CHECK-NEXT:   br i1 %111, label %_llgo_36, label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_36:                                         ; preds = %_llgo_35
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %13, align 8
// CHECK-NEXT:   %112 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %113 = icmp ne ptr %112, null
// CHECK-NEXT:   br i1 %113, label %_llgo_45, label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_37:                                         ; preds = %_llgo_35
// CHECK-NEXT:   %114 = icmp eq i64 %104, 6
// CHECK-NEXT:   br i1 %114, label %_llgo_38, label %_llgo_39
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_38:                                         ; preds = %_llgo_37
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %13, align 8
// CHECK-NEXT:   %115 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %116 = icmp ne ptr %115, null
// CHECK-NEXT:   br i1 %116, label %_llgo_47, label %_llgo_48
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_39:                                         ; preds = %_llgo_37
// CHECK-NEXT:   %117 = icmp eq i64 %104, 7
// CHECK-NEXT:   br i1 %117, label %_llgo_40, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_40:                                         ; preds = %_llgo_39
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_16), ptr %13, align 8
// CHECK-NEXT:   %118 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %119 = icmp ne ptr %118, null
// CHECK-NEXT:   br i1 %119, label %_llgo_49, label %_llgo_50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_41:                                         ; preds = %_llgo_32
// CHECK-NEXT:   %120 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %121 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %120, align 8
// CHECK-NEXT:   %122 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %121, 0
// CHECK-NEXT:   store ptr %122, ptr %15, align 8
// CHECK-NEXT:   %123 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %121, 2
// CHECK-NEXT:   %124 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %121, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %120)
// CHECK-NEXT:   %125 = extractvalue { ptr, ptr } %123, 1
// CHECK-NEXT:   %126 = extractvalue { ptr, ptr } %123, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %126)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %125, %"{{.*}}/runtime/internal/runtime.String" %124)
// CHECK-NEXT:   br label %_llgo_42
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_42:                                         ; preds = %_llgo_41, %_llgo_32
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_43:                                         ; preds = %_llgo_34
// CHECK-NEXT:   %127 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %128 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %127, align 8
// CHECK-NEXT:   %129 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %128, 0
// CHECK-NEXT:   store ptr %129, ptr %15, align 8
// CHECK-NEXT:   %130 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %128, 2
// CHECK-NEXT:   %131 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %128, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %127)
// CHECK-NEXT:   %132 = extractvalue { ptr, ptr } %130, 1
// CHECK-NEXT:   %133 = extractvalue { ptr, ptr } %130, 0
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %133)
// CHECK-NEXT:   call void %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %132, %"{{.*}}/runtime/internal/runtime.String" %131)
// CHECK-NEXT:   br label %_llgo_44
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_44:                                         ; preds = %_llgo_43, %_llgo_34
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_45:                                         ; preds = %_llgo_36
// CHECK-NEXT:   %134 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %135 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %134, align 8
// CHECK-NEXT:   %136 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %135, 0
// CHECK-NEXT:   store ptr %136, ptr %15, align 8
// CHECK-NEXT:   %137 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %135, 2
// CHECK-NEXT:   %138 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %135, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %134)
// CHECK-NEXT:   %139 = extractvalue { ptr, ptr } %137, 1
// CHECK-NEXT:   %140 = extractvalue { ptr, ptr } %137, 0
// CHECK-NEXT:   %__llgo_funcval_code2 = call ptr asm "", "=r,0"(ptr %140)
// CHECK-NEXT:   call void %__llgo_funcval_code2(ptr {{(nest|swiftself)}} %139, %"{{.*}}/runtime/internal/runtime.String" %138)
// CHECK-NEXT:   br label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_46:                                         ; preds = %_llgo_45, %_llgo_36
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_47:                                         ; preds = %_llgo_38
// CHECK-NEXT:   %141 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %142 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %141, align 8
// CHECK-NEXT:   %143 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %142, 0
// CHECK-NEXT:   store ptr %143, ptr %15, align 8
// CHECK-NEXT:   %144 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %142, 2
// CHECK-NEXT:   %145 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %142, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %141)
// CHECK-NEXT:   %146 = extractvalue { ptr, ptr } %144, 1
// CHECK-NEXT:   %147 = extractvalue { ptr, ptr } %144, 0
// CHECK-NEXT:   %__llgo_funcval_code3 = call ptr asm "", "=r,0"(ptr %147)
// CHECK-NEXT:   call void %__llgo_funcval_code3(ptr {{(nest|swiftself)}} %146, %"{{.*}}/runtime/internal/runtime.String" %145)
// CHECK-NEXT:   br label %_llgo_48
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_48:                                         ; preds = %_llgo_47, %_llgo_38
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_49:                                         ; preds = %_llgo_40
// CHECK-NEXT:   %148 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %149 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %148, align 8
// CHECK-NEXT:   %150 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %149, 0
// CHECK-NEXT:   store ptr %150, ptr %15, align 8
// CHECK-NEXT:   %151 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %149, 2
// CHECK-NEXT:   %152 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %149, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %148)
// CHECK-NEXT:   %153 = extractvalue { ptr, ptr } %151, 1
// CHECK-NEXT:   %154 = extractvalue { ptr, ptr } %151, 0
// CHECK-NEXT:   %__llgo_funcval_code4 = call ptr asm "", "=r,0"(ptr %154)
// CHECK-NEXT:   call void %__llgo_funcval_code4(ptr {{(nest|swiftself)}} %153, %"{{.*}}/runtime/internal/runtime.String" %152)
// CHECK-NEXT:   br label %_llgo_50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_50:                                         ; preds = %_llgo_49, %_llgo_40
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_51:                                         ; preds = %_llgo_24
// CHECK-NEXT:   %155 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %156 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %155, align 8
// CHECK-NEXT:   %157 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %156, 0
// CHECK-NEXT:   store ptr %157, ptr %15, align 8
// CHECK-NEXT:   %158 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %156, 2
// CHECK-NEXT:   %159 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %156, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %155)
// CHECK-NEXT:   %160 = extractvalue { ptr, ptr } %158, 1
// CHECK-NEXT:   %161 = extractvalue { ptr, ptr } %158, 0
// CHECK-NEXT:   %__llgo_funcval_code5 = call ptr asm "", "=r,0"(ptr %161)
// CHECK-NEXT:   call void %__llgo_funcval_code5(ptr {{(nest|swiftself)}} %160, %"{{.*}}/runtime/internal/runtime.String" %159)
// CHECK-NEXT:   br label %_llgo_52
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_52:                                         ; preds = %_llgo_51, %_llgo_24
// CHECK-NEXT:   br label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_53:                                         ; preds = %_llgo_75, %_llgo_73, %_llgo_71, %_llgo_69, %_llgo_67, %_llgo_23
// CHECK-NEXT:   %162 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %163 = icmp ne ptr %162, null
// CHECK-NEXT:   br i1 %163, label %_llgo_55, label %_llgo_54
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_54:                                         ; preds = %_llgo_64, %_llgo_53
// CHECK-NEXT:   br label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_55:                                         ; preds = %_llgo_53
// CHECK-NEXT:   %164 = load { ptr, i64 }, ptr %162, align 8
// CHECK-NEXT:   %165 = extractvalue { ptr, i64 } %164, 1
// CHECK-NEXT:   br label %_llgo_56
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_56:                                         ; preds = %_llgo_55
// CHECK-NEXT:   %166 = icmp eq i64 %165, 2
// CHECK-NEXT:   br i1 %166, label %_llgo_57, label %_llgo_58
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_57:                                         ; preds = %_llgo_56
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %167 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %168 = icmp ne ptr %167, null
// CHECK-NEXT:   br i1 %168, label %_llgo_66, label %_llgo_67
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_58:                                         ; preds = %_llgo_56
// CHECK-NEXT:   %169 = icmp eq i64 %165, 4
// CHECK-NEXT:   br i1 %169, label %_llgo_59, label %_llgo_60
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_59:                                         ; preds = %_llgo_58
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %170 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %171 = icmp ne ptr %170, null
// CHECK-NEXT:   br i1 %171, label %_llgo_68, label %_llgo_69
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_60:                                         ; preds = %_llgo_58
// CHECK-NEXT:   %172 = icmp eq i64 %165, 5
// CHECK-NEXT:   br i1 %172, label %_llgo_61, label %_llgo_62
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_61:                                         ; preds = %_llgo_60
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %173 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %174 = icmp ne ptr %173, null
// CHECK-NEXT:   br i1 %174, label %_llgo_70, label %_llgo_71
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_62:                                         ; preds = %_llgo_60
// CHECK-NEXT:   %175 = icmp eq i64 %165, 6
// CHECK-NEXT:   br i1 %175, label %_llgo_63, label %_llgo_64
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_63:                                         ; preds = %_llgo_62
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %176 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %177 = icmp ne ptr %176, null
// CHECK-NEXT:   br i1 %177, label %_llgo_72, label %_llgo_73
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_64:                                         ; preds = %_llgo_62
// CHECK-NEXT:   %178 = icmp eq i64 %165, 7
// CHECK-NEXT:   br i1 %178, label %_llgo_65, label %_llgo_54
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_65:                                         ; preds = %_llgo_64
// CHECK-NEXT:   store ptr blockaddress(@main.complexOrder, %_llgo_23), ptr %13, align 8
// CHECK-NEXT:   %179 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %180 = icmp ne ptr %179, null
// CHECK-NEXT:   br i1 %180, label %_llgo_74, label %_llgo_75
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_66:                                         ; preds = %_llgo_57
// CHECK-NEXT:   %181 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %182 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %181, align 8
// CHECK-NEXT:   %183 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %182, 0
// CHECK-NEXT:   store ptr %183, ptr %15, align 8
// CHECK-NEXT:   %184 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %182, 2
// CHECK-NEXT:   %185 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %182, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %181)
// CHECK-NEXT:   %186 = extractvalue { ptr, ptr } %184, 1
// CHECK-NEXT:   %187 = extractvalue { ptr, ptr } %184, 0
// CHECK-NEXT:   %__llgo_funcval_code6 = call ptr asm "", "=r,0"(ptr %187)
// CHECK-NEXT:   call void %__llgo_funcval_code6(ptr {{(nest|swiftself)}} %186, %"{{.*}}/runtime/internal/runtime.String" %185)
// CHECK-NEXT:   br label %_llgo_67
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_67:                                         ; preds = %_llgo_66, %_llgo_57
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_68:                                         ; preds = %_llgo_59
// CHECK-NEXT:   %188 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %189 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %188, align 8
// CHECK-NEXT:   %190 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %189, 0
// CHECK-NEXT:   store ptr %190, ptr %15, align 8
// CHECK-NEXT:   %191 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %189, 2
// CHECK-NEXT:   %192 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %189, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %188)
// CHECK-NEXT:   %193 = extractvalue { ptr, ptr } %191, 1
// CHECK-NEXT:   %194 = extractvalue { ptr, ptr } %191, 0
// CHECK-NEXT:   %__llgo_funcval_code7 = call ptr asm "", "=r,0"(ptr %194)
// CHECK-NEXT:   call void %__llgo_funcval_code7(ptr {{(nest|swiftself)}} %193, %"{{.*}}/runtime/internal/runtime.String" %192)
// CHECK-NEXT:   br label %_llgo_69
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_69:                                         ; preds = %_llgo_68, %_llgo_59
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_70:                                         ; preds = %_llgo_61
// CHECK-NEXT:   %195 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %196 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %195, align 8
// CHECK-NEXT:   %197 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %196, 0
// CHECK-NEXT:   store ptr %197, ptr %15, align 8
// CHECK-NEXT:   %198 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %196, 2
// CHECK-NEXT:   %199 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %196, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %195)
// CHECK-NEXT:   %200 = extractvalue { ptr, ptr } %198, 1
// CHECK-NEXT:   %201 = extractvalue { ptr, ptr } %198, 0
// CHECK-NEXT:   %__llgo_funcval_code8 = call ptr asm "", "=r,0"(ptr %201)
// CHECK-NEXT:   call void %__llgo_funcval_code8(ptr {{(nest|swiftself)}} %200, %"{{.*}}/runtime/internal/runtime.String" %199)
// CHECK-NEXT:   br label %_llgo_71
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_71:                                         ; preds = %_llgo_70, %_llgo_61
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_72:                                         ; preds = %_llgo_63
// CHECK-NEXT:   %202 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %203 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %202, align 8
// CHECK-NEXT:   %204 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %203, 0
// CHECK-NEXT:   store ptr %204, ptr %15, align 8
// CHECK-NEXT:   %205 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %203, 2
// CHECK-NEXT:   %206 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %203, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %202)
// CHECK-NEXT:   %207 = extractvalue { ptr, ptr } %205, 1
// CHECK-NEXT:   %208 = extractvalue { ptr, ptr } %205, 0
// CHECK-NEXT:   %__llgo_funcval_code9 = call ptr asm "", "=r,0"(ptr %208)
// CHECK-NEXT:   call void %__llgo_funcval_code9(ptr {{(nest|swiftself)}} %207, %"{{.*}}/runtime/internal/runtime.String" %206)
// CHECK-NEXT:   br label %_llgo_73
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_73:                                         ; preds = %_llgo_72, %_llgo_63
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_74:                                         ; preds = %_llgo_65
// CHECK-NEXT:   %209 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %210 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %209, align 8
// CHECK-NEXT:   %211 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %210, 0
// CHECK-NEXT:   store ptr %211, ptr %15, align 8
// CHECK-NEXT:   %212 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %210, 2
// CHECK-NEXT:   %213 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %210, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %209)
// CHECK-NEXT:   %214 = extractvalue { ptr, ptr } %212, 1
// CHECK-NEXT:   %215 = extractvalue { ptr, ptr } %212, 0
// CHECK-NEXT:   %__llgo_funcval_code10 = call ptr asm "", "=r,0"(ptr %215)
// CHECK-NEXT:   call void %__llgo_funcval_code10(ptr {{(nest|swiftself)}} %214, %"{{.*}}/runtime/internal/runtime.String" %213)
// CHECK-NEXT:   br label %_llgo_75
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_75:                                         ; preds = %_llgo_74, %_llgo_65
// CHECK-NEXT:   br label %_llgo_53
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_76:                                         ; preds = %_llgo_22
// CHECK-NEXT:   %216 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %217 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %216, align 8
// CHECK-NEXT:   %218 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %217, 0
// CHECK-NEXT:   store ptr %218, ptr %15, align 8
// CHECK-NEXT:   %219 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %217, 2
// CHECK-NEXT:   %220 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %217, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %216)
// CHECK-NEXT:   %221 = extractvalue { ptr, ptr } %219, 1
// CHECK-NEXT:   %222 = extractvalue { ptr, ptr } %219, 0
// CHECK-NEXT:   %__llgo_funcval_code11 = call ptr asm "", "=r,0"(ptr %222)
// CHECK-NEXT:   call void %__llgo_funcval_code11(ptr {{(nest|swiftself)}} %221, %"{{.*}}/runtime/internal/runtime.String" %220)
// CHECK-NEXT:   br label %_llgo_77
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_77:                                         ; preds = %_llgo_76, %_llgo_22
// CHECK-NEXT:   br label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_78:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %223 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %224 = load { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %223, align 8
// CHECK-NEXT:   %225 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %224, 0
// CHECK-NEXT:   store ptr %225, ptr %15, align 8
// CHECK-NEXT:   %226 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %224, 2
// CHECK-NEXT:   %227 = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %224, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %223)
// CHECK-NEXT:   %228 = extractvalue { ptr, ptr } %226, 1
// CHECK-NEXT:   %229 = extractvalue { ptr, ptr } %226, 0
// CHECK-NEXT:   %__llgo_funcval_code12 = call ptr asm "", "=r,0"(ptr %229)
// CHECK-NEXT:   call void %__llgo_funcval_code12(ptr {{(nest|swiftself)}} %228, %"{{.*}}/runtime/internal/runtime.String" %227)
// CHECK-NEXT:   br label %_llgo_79
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_79:                                         ; preds = %_llgo_78, %_llgo_21
// CHECK-NEXT:   %230 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, align 8
// CHECK-NEXT:   %231 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %230, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %231)
// CHECK-NEXT:   %232 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %232, [label %_llgo_17, label %_llgo_20]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.complexOrder$1"(ptr {{(nest|swiftself)}} %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %5, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %1, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %5, 0
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, i64 1, 1
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %8, i64 1, 2
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %9, 1
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}/runtime/internal/runtime.Slice" %4, ptr %10, i64 %11, i64 16)
// CHECK-NEXT:   %13 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %12, ptr %13, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = add i64 48, %0
// CHECK-NEXT:   %2 = trunc i64 %1 to i32
// CHECK-NEXT:   %3 = sext i32 %2 to i64
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromInt64"(i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

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

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.label1(%"{{.*}}/runtime/internal/runtime.String" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %1)
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %2, %"{{.*}}/runtime/internal/runtime.String" %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.label2(%"{{.*}}/runtime/internal/runtime.String" %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %1)
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %3, %"{{.*}}/runtime/internal/runtime.String" %4)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %5, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %2)
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %6, %"{{.*}}/runtime/internal/runtime.String" %7)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %8
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.label3(%"{{.*}}/runtime/internal/runtime.String" %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %1)
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %4, %"{{.*}}/runtime/internal/runtime.String" %5)
// CHECK-NEXT:   %7 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %6, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %2)
// CHECK-NEXT:   %9 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %7, %"{{.*}}/runtime/internal/runtime.String" %8)
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %9, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 })
// CHECK-NEXT:   %11 = call %"{{.*}}/runtime/internal/runtime.String" @main.digit(i64 %3)
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %10, %"{{.*}}/runtime/internal/runtime.String" %11)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %12
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.Slice" @main.complexOrder()
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %2 = phi i64 [ -1, %_llgo_0 ], [ %3, %_llgo_2 ]
// CHECK-NEXT:   %3 = add i64 %2, 1
// CHECK-NEXT:   %4 = icmp slt i64 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %7 = icmp slt i64 %3, 0
// CHECK-NEXT:   %8 = icmp uge i64 %3, %6
// CHECK-NEXT:   %9 = or i1 %8, %7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %9, i64 %3, i1 true, i64 %6)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %5, i64 %3
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.String", ptr %10, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
