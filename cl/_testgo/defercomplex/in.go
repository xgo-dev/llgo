// LITTEST
package main

// Nested loop/branch defers must all use one function-level defer chain; the
// deferred recorder retains its result slice through a closure environment.
// CHECK: @[[CLEANUP_FINAL:[0-9]+]] = private unnamed_addr constant [13 x i8] c"cleanup-final"
// CHECK: @[[CLEANUP_BEFORE:[0-9]+]] = private unnamed_addr constant [19 x i8] c"cleanup-before-loop"
// CHECK: @[[EXIT_OUTER:[0-9]+]] = private unnamed_addr constant [10 x i8] c"exit-outer"
// CHECK: @[[POST_LOOP:[0-9]+]] = private unnamed_addr constant [9 x i8] c"post-loop"
// CHECK: @[[BRANCH_EVEN:[0-9]+]] = private unnamed_addr constant [11 x i8] c"branch-even"
// CHECK: @[[BRANCH_ODD:[0-9]+]] = private unnamed_addr constant [10 x i8] c"branch-odd"
// CHECK: @[[NESTED:[0-9]+]] = private unnamed_addr constant [6 x i8] c"nested"
// CHECK: @[[NESTED_TAIL:[0-9]+]] = private unnamed_addr constant [11 x i8] c"nested-tail"
// CHECK-LABEL: define %"{{.*}}Slice" @main.complexOrder(){{.*}} {
// CHECK: [[RESULT_SLOT:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: store ptr [[RESULT_SLOT]], ptr [[RECORD_ENV_SLOT:%[0-9]+]]
// CHECK: [[RECORD_FN:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.complexOrder$1", ptr undef }, ptr [[RECORD_ENV:%[0-9]+]], 1
// CHECK-NEXT: [[FINAL_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label1(%"{{.*}}String" { ptr @[[CLEANUP_FINAL]], i64 13 }, i64 0)
// CHECK-NEXT: [[PREVIOUS_DEFER:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[DEFER_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREVIOUS_DEFER]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[DEFER_FRAME]])
// CHECK: [[DEFER_HEAD:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[DEFER_FRAME]], i32 0, i32 5
// CHECK: [[OUTER_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label1(%"{{.*}}String" { ptr @[[EXIT_OUTER]], i64 10 }, i64 [[OUTER_I:%[0-9]+]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[OUTER_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[OUTER_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[POST_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label1(%"{{.*}}String" { ptr @[[POST_LOOP]], i64 9 }, i64 0)
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[POST_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[POST_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[EVEN_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label2(%"{{.*}}String" { ptr @[[BRANCH_EVEN]], i64 11 }, i64 [[OUTER_I]], i64 [[INNER_J:%[0-9]+]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[EVEN_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[EVEN_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[ODD_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label2(%"{{.*}}String" { ptr @[[BRANCH_ODD]], i64 10 }, i64 [[OUTER_I]], i64 [[INNER_J]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[ODD_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[ODD_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[NESTED_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label3(%"{{.*}}String" { ptr @[[NESTED]], i64 6 }, i64 [[OUTER_I]], i64 [[INNER_J]], i64 [[INNER_K:%[0-9]+]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[NESTED_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[NESTED_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[TAIL_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label3(%"{{.*}}String" { ptr @[[NESTED_TAIL]], i64 11 }, i64 [[OUTER_I]], i64 [[INNER_J]], i64 [[INNER_K]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[TAIL_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[TAIL_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: call void @"{{.*}}Rethrow"(ptr [[PREVIOUS_DEFER]])
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[FINAL_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[FINAL_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK-NEXT: [[BEFORE_LABEL:%[0-9]+]] = call %"{{.*}}String" @main.label1(%"{{.*}}String" { ptr @[[CLEANUP_BEFORE]], i64 19 }, i64 0)
// CHECK: store { ptr, ptr } [[RECORD_FN]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" [[BEFORE_LABEL]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[BEFORE_NODE:%[0-9]+]], ptr [[DEFER_HEAD]]
// CHECK: [[RUN_RECORD:%[0-9]+]] = load { ptr, i64, { ptr, ptr }, %"{{.*}}String" }, ptr [[RUN_NODE:%[0-9]+]]
// CHECK: [[RUN_FN:%[0-9]+]] = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}String" } [[RUN_RECORD]], 2
// CHECK-NEXT: [[RUN_LABEL:%[0-9]+]] = extractvalue { ptr, i64, { ptr, ptr }, %"{{.*}}String" } [[RUN_RECORD]], 3
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[RUN_NODE]])
// CHECK: call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %{{[0-9]+}}, %"{{.*}}String" [[RUN_LABEL]])
// CHECK: [[SAVED_DEFER:%[0-9]+]] = load %"{{.*}}Defer", ptr [[DEFER_FRAME]]
// CHECK-NEXT: [[RESTORED_DEFER:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[SAVED_DEFER]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[RESTORED_DEFER]])
// CHECK-LABEL: define void @"main.complexOrder$1"(ptr {{(nest|swiftself)}} %0, %"{{.*}}String" %1){{.*}} {
// CHECK: [[RECORD_ENV_VALUE:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[RECORD_RESULT_SLOT:%[0-9]+]] = extractvalue { ptr } [[RECORD_ENV_VALUE]], 0
// CHECK-NEXT: [[OLD_RESULT:%[0-9]+]] = load %"{{.*}}Slice", ptr [[RECORD_RESULT_SLOT]]
// CHECK: store %"{{.*}}String" %1, ptr %{{[0-9]+}}
// CHECK: [[APPEND_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[APPEND_SLICE:%[0-9]+]], 0
// CHECK-NEXT: [[APPEND_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[APPEND_SLICE]], 1
// CHECK-NEXT: [[NEW_RESULT:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}SliceAppend"(%"{{.*}}Slice" [[OLD_RESULT]], ptr [[APPEND_DATA]], i64 [[APPEND_LEN]], i64 16)
// CHECK-NEXT: [[UPDATED_RESULT_SLOT:%[0-9]+]] = extractvalue { ptr } [[RECORD_ENV_VALUE]], 0
// CHECK-NEXT: store %"{{.*}}Slice" [[NEW_RESULT]], ptr [[UPDATED_RESULT_SLOT]]
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ORDER:%[0-9]+]] = call %"{{.*}}Slice" @main.complexOrder()
// CHECK-NEXT: [[ORDER_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[ORDER]], 1
// CHECK: [[ORDER_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[ORDER]], 0
// CHECK: [[LABEL_PTR:%[0-9]+]] = getelementptr inbounds %"{{.*}}String", ptr [[ORDER_DATA]], i64 %{{[0-9]+}}
// CHECK-NEXT: [[LABEL:%[0-9]+]] = load %"{{.*}}String", ptr [[LABEL_PTR]]
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[LABEL]])

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
