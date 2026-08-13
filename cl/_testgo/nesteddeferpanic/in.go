// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.replacement()
// CHECK: call void @main.resumeOuterPanic()
// CHECK: call void @main.recoverThenPanic()

// recoverThenPanic: recover panic(2), remember it, then execute deferred panic(3).
// CHECK-LABEL: define void @main.recoverThenPanic(){{.*}} {
// CHECK: [[RTP_OUTER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.recoverThenPanic$1")
// CHECK: call void @"main.recoverThenPanic$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RTP_OUTER_STATE]])
// CHECK: call void @"main.recoverThenPanic$2"()

// CHECK-LABEL: define void @"main.recoverThenPanic$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.recoverThenPanic$1", ptr [[RTP_OUTER_SLOT:%.*]])
// CHECK: [[RTP_OUTER_VALUE:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RTP_OUTER_SLOT]])
// CHECK: [[RTP_OUTER_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[RTP_OUTER_VALUE]], 0
// CHECK: [[RTP_OUTER_IS_INT:%.*]] = icmp eq ptr [[RTP_OUTER_TYPE]], @_llgo_int
// CHECK: br i1 [[RTP_OUTER_IS_INT]], label %{{.*}}, label %{{.*}}
// CHECK: [[RTP_OUTER_DATA:%.*]] = extractvalue %"{{.*}}eface" [[RTP_OUTER_VALUE]], 1
// CHECK: [[RTP_OUTER_INT:%.*]] = load i64, ptr [[RTP_OUTER_DATA]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[RTP_OUTER_INT]])

// CHECK-LABEL: define void @"main.recoverThenPanic$2"(){{.*}} {
// CHECK: [[RTP_INNER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.recoverThenPanic$2$1")
// CHECK: call void @"main.recoverThenPanic$2$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RTP_INNER_STATE]])
// CHECK: store i64 2, ptr [[RTP_PANIC2_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[RTP_PANIC2:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[RTP_PANIC2_ADDR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[RTP_PANIC2]])

// CHECK-LABEL: define void @"main.recoverThenPanic$2$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.recoverThenPanic$2$1", ptr [[RTP_INNER_SLOT:%.*]])
// CHECK: [[RTP_RECOVERED2:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RTP_INNER_SLOT]])
// CHECK: [[RTP_RECOVERED2_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[RTP_RECOVERED2]], 0
// CHECK: [[RTP_RECOVERED2_IS_INT:%.*]] = icmp eq ptr [[RTP_RECOVERED2_TYPE]], @_llgo_int
// CHECK: [[RTP_RECOVERED2_DATA:%.*]] = extractvalue %"{{.*}}eface" [[RTP_RECOVERED2]], 1
// CHECK: [[RTP_OLD:%.*]] = load i64, ptr [[RTP_RECOVERED2_DATA]]
// CHECK: store i64 3, ptr [[RTP_PANIC3_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[RTP_PANIC3:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[RTP_PANIC3_ADDR]], 1
// CHECK: store %"{{.*}}eface" [[RTP_PANIC3]], ptr %{{.*}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[RTP_OLD]])
// CHECK: [[RTP_DEFERRED_PANIC:%.*]] = extractvalue { ptr, i64, %"{{.*}}eface" } %{{.*}}, 2
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[RTP_DEFERRED_PANIC]])

// replacement: panic(4) is superseded by a deferred panic(5), and the outer recover sees the latter.
// CHECK-LABEL: define void @main.replacement(){{.*}} {
// CHECK: [[REPL_OUTER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.replacement$1")
// CHECK: call void @"main.replacement$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[REPL_OUTER_STATE]])
// CHECK: call void @"main.replacement$2"()

// CHECK-LABEL: define void @"main.replacement$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.replacement$1", ptr [[REPL_SLOT:%.*]])
// CHECK: [[REPL_VALUE:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[REPL_SLOT]])
// CHECK: [[REPL_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[REPL_VALUE]], 0
// CHECK: [[REPL_IS_INT:%.*]] = icmp eq ptr [[REPL_TYPE]], @_llgo_int
// CHECK: [[REPL_DATA:%.*]] = extractvalue %"{{.*}}eface" [[REPL_VALUE]], 1
// CHECK: [[REPL_INT:%.*]] = load i64, ptr [[REPL_DATA]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[REPL_INT]])

// CHECK-LABEL: define void @"main.replacement$2"(){{.*}} {
// CHECK: call void @"main.replacement$2$1"()
// CHECK: store i64 4, ptr [[REPL_PANIC4_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[REPL_PANIC4:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[REPL_PANIC4_ADDR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[REPL_PANIC4]])

// CHECK-LABEL: define void @"main.replacement$2$1"(){{.*}} {
// CHECK: store i64 5, ptr [[REPL_PANIC5_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[REPL_PANIC5:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[REPL_PANIC5_ADDR]], 1
// CHECK: store %"{{.*}}eface" [[REPL_PANIC5]], ptr %{{.*}}
// CHECK: [[REPL_DEFERRED_PANIC:%.*]] = extractvalue { ptr, i64, %"{{.*}}eface" } %{{.*}}, 2
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[REPL_DEFERRED_PANIC]])

// resumeOuterPanic: the deepest recover consumes panic(2), allowing panic(1) to resume outward.
// CHECK-LABEL: define void @main.resumeOuterPanic(){{.*}} {
// CHECK: [[RESUME_OUTER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.resumeOuterPanic$1")
// CHECK: call void @"main.resumeOuterPanic$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RESUME_OUTER_STATE]])
// CHECK: call void @"main.resumeOuterPanic$2"()

// CHECK-LABEL: define void @"main.resumeOuterPanic$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.resumeOuterPanic$1", ptr [[RESUME_OUTER_SLOT:%.*]])
// CHECK: [[RESUME_OUTER_VALUE:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RESUME_OUTER_SLOT]])
// CHECK: [[RESUME_OUTER_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[RESUME_OUTER_VALUE]], 0
// CHECK: icmp eq ptr [[RESUME_OUTER_TYPE]], @_llgo_int
// CHECK: [[RESUME_OUTER_DATA:%.*]] = extractvalue %"{{.*}}eface" [[RESUME_OUTER_VALUE]], 1
// CHECK: [[RESUME_OUTER_INT:%.*]] = load i64, ptr [[RESUME_OUTER_DATA]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[RESUME_OUTER_INT]])

// CHECK-LABEL: define void @"main.resumeOuterPanic$2"(){{.*}} {
// CHECK: call void @"main.resumeOuterPanic$2$1"()
// CHECK: store i64 1, ptr [[RESUME_PANIC1_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[RESUME_PANIC1:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[RESUME_PANIC1_ADDR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[RESUME_PANIC1]])

// CHECK-LABEL: define void @"main.resumeOuterPanic$2$1"(){{.*}} {
// CHECK: [[RESUME_INNER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.resumeOuterPanic$2$1$1")
// CHECK: call void @"main.resumeOuterPanic$2$1$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RESUME_INNER_STATE]])
// CHECK: store i64 2, ptr [[RESUME_PANIC2_ADDR:%[-A-Za-z0-9_.]+]]
// CHECK: [[RESUME_PANIC2:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[RESUME_PANIC2_ADDR]], 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[RESUME_PANIC2]])

// CHECK-LABEL: define void @"main.resumeOuterPanic$2$1$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.resumeOuterPanic$2$1$1", ptr [[RESUME_INNER_SLOT:%.*]])
// CHECK: [[RESUME_INNER_VALUE:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RESUME_INNER_SLOT]])
// CHECK: [[RESUME_INNER_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[RESUME_INNER_VALUE]], 0
// CHECK: icmp eq ptr [[RESUME_INNER_TYPE]], @_llgo_int
// CHECK: [[RESUME_INNER_DATA:%.*]] = extractvalue %"{{.*}}eface" [[RESUME_INNER_VALUE]], 1
// CHECK: [[RESUME_INNER_INT:%.*]] = load i64, ptr [[RESUME_INNER_DATA]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[RESUME_INNER_INT]])

func replacement() {
	defer func() {
		println("replacement", recover().(int))
	}()

	func() {
		for {
			defer func() {
				defer panic(5)
			}()
			break
		}
		panic(4)
	}()
}

func resumeOuterPanic() {
	defer func() {
		println("resume outer", recover().(int))
	}()

	func() {
		defer func() {
			defer func() {
				println("resume inner", recover().(int))
			}()
			panic(2)
		}()
		panic(1)
	}()
}

func recoverThenPanic() {
	defer func() {
		println("recover-then-panic outer", recover().(int))
	}()

	func() {
		defer func() {
			old := recover().(int)
			defer panic(3)
			println("recover-then-panic old", old)
		}()
		panic(2)
	}()
}

func main() {
	replacement()
	resumeOuterPanic()
	recoverThenPanic()
}
