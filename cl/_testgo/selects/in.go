// LITTEST
package main

// Select cases are nondeterministic at runtime; check only that both the main
// function and its goroutine lower their case table through runtime.Select.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[C1:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK: [[C2:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK: [[C3:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK: [[C4:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK: [[MAIN_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[MAIN_ENV:%[0-9]+]], 1
// CHECK: store { ptr, ptr } [[MAIN_CLOSURE]], ptr %{{[0-9]+}}
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %{{[0-9]+}}, i64 0)
// CHECK: [[MAIN_C1:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr [[MAIN_C1]], ptr [[C1_VALUE:%[0-9]+]], i64 0)
// CHECK: [[MAIN_C2:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[MAIN_RECV2_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[MAIN_C2]], 0
// CHECK-NEXT: [[MAIN_RECV2_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV2_0]], ptr [[MAIN_RECV2_BUF:%[0-9]+]], 1
// CHECK-NEXT: [[MAIN_RECV2_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV2_1]], i32 0, 2
// CHECK-NEXT: [[MAIN_RECV2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV2_2]], i1 false, 3
// CHECK: [[MAIN_RECV4_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[C4]], 0
// CHECK-NEXT: [[MAIN_RECV4_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV4_0]], ptr [[MAIN_RECV4_BUF:%[0-9]+]], 1
// CHECK-NEXT: [[MAIN_RECV4_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV4_1]], i32 0, 2
// CHECK-NEXT: [[MAIN_RECV4:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[MAIN_RECV4_2]], i1 false, 3
// CHECK: store %"{{.*}}ChanOp" [[MAIN_RECV2]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}ChanOp" [[MAIN_RECV4]], ptr %{{[0-9]+}}
// CHECK: [[MAIN_CASES:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 2, 2
// CHECK-NEXT: [[MAIN_SELECT:%[0-9]+]] = call { i64, i1 } @"{{.*}}/runtime/internal/runtime.Select"(%"{{.*}}Slice" [[MAIN_CASES]])
// CHECK-NEXT: [[MAIN_SELECTED:%[0-9]+]] = extractvalue { i64, i1 } [[MAIN_SELECT]], 0
// CHECK: [[MAIN_DISPATCH:%[0-9]+]] = extractvalue { i64, i1, {}, {} } %{{[0-9]+}}, 0
// CHECK-NEXT: [[MAIN_CASE0:%[0-9]+]] = icmp eq i64 [[MAIN_DISPATCH]], 0
// CHECK: [[MAIN_CASE1:%[0-9]+]] = icmp eq i64 [[MAIN_DISPATCH]], 1
// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[GOR_ENV:%[0-9]+]] = load { ptr, ptr, ptr }, ptr %0
// CHECK-NEXT: [[GOR_C1_SLOT:%[0-9]+]] = extractvalue { ptr, ptr, ptr } [[GOR_ENV]], 0
// CHECK-NEXT: [[GOR_C1:%[0-9]+]] = load ptr, ptr [[GOR_C1_SLOT]]
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr [[GOR_C1]], ptr [[GOR_C1_BUF:%[0-9]+]], i64 0)
// CHECK: [[GOR_C2_SLOT:%[0-9]+]] = extractvalue { ptr, ptr, ptr } [[GOR_ENV]], 1
// CHECK-NEXT: [[GOR_C2:%[0-9]+]] = load ptr, ptr [[GOR_C2_SLOT]]
// CHECK-NEXT: [[GOR_C3_SLOT:%[0-9]+]] = extractvalue { ptr, ptr, ptr } [[GOR_ENV]], 2
// CHECK-NEXT: [[GOR_C3:%[0-9]+]] = load ptr, ptr [[GOR_C3_SLOT]]
// CHECK: [[GOR_SEND2_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[GOR_C2]], 0
// CHECK-NEXT: [[GOR_SEND2_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_SEND2_0]], ptr [[GOR_SEND2_BUF:%[0-9]+]], 1
// CHECK-NEXT: [[GOR_SEND2_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_SEND2_1]], i32 0, 2
// CHECK-NEXT: [[GOR_SEND2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_SEND2_2]], i1 true, 3
// CHECK: [[GOR_RECV3_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[GOR_C3]], 0
// CHECK-NEXT: [[GOR_RECV3_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_RECV3_0]], ptr [[GOR_RECV3_BUF:%[0-9]+]], 1
// CHECK-NEXT: [[GOR_RECV3_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_RECV3_1]], i32 0, 2
// CHECK-NEXT: [[GOR_RECV3:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[GOR_RECV3_2]], i1 false, 3
// CHECK: store %"{{.*}}ChanOp" [[GOR_SEND2]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}ChanOp" [[GOR_RECV3]], ptr %{{[0-9]+}}
// CHECK: [[GOR_CASES:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 2, 2
// CHECK-NEXT: [[GOR_SELECT:%[0-9]+]] = call { i64, i1 } @"{{.*}}/runtime/internal/runtime.Select"(%"{{.*}}Slice" [[GOR_CASES]])
// CHECK-NEXT: [[GOR_SELECTED:%[0-9]+]] = extractvalue { i64, i1 } [[GOR_SELECT]], 0
// CHECK: [[GOR_DISPATCH:%[0-9]+]] = extractvalue { i64, i1, {} } %{{[0-9]+}}, 0
// CHECK-NEXT: [[GOR_CASE0:%[0-9]+]] = icmp eq i64 [[GOR_DISPATCH]], 0
// CHECK: [[GOR_CASE1:%[0-9]+]] = icmp eq i64 [[GOR_DISPATCH]], 1

func main() {
	c1 := make(chan struct{}, 1)
	c2 := make(chan struct{}, 1)
	c3 := make(chan struct{}, 1)
	c4 := make(chan struct{}, 1)

	go func() {
		<-c1
		println("<-c1")

		select {
		case c2 <- struct{}{}:
			println("c2<-")
		case <-c3:
			println("<-c3")
		}
	}()

	c1 <- struct{}{}
	println("c1<-")

	select {
	case <-c2:
		println("<-c2")
	case <-c4:
		println("<-c4")
	}
}
