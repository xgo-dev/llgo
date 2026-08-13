// LITTEST
package main

// Blocking send selection and receive-with-default use different runtime
// entry points and result shapes.
// CHECK: @[[EXIT_TEXT:[0-9]+]] = private unnamed_addr constant [4 x i8] c"exit"
// CHECK: @[[CH1_TEXT:[0-9]+]] = private unnamed_addr constant [3 x i8] c"ch1"
// CHECK: @[[CH2_TEXT:[0-9]+]] = private unnamed_addr constant [3 x i8] c"ch2"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.send()
// CHECK-NEXT: call void @main.recv()
// CHECK-NEXT: ret void
// CHECK-LABEL: define void @main.recv(){{.*}} {
// CHECK: [[RECV_CH1_OBJ:%[0-9]+]] = call ptr @"{{.*}}NewChan"(i64 16, i64 0)
// CHECK: [[RECV_CH2_OBJ:%[0-9]+]] = call ptr @"{{.*}}NewChan"(i64 16, i64 0)
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$1", ptr %{{[0-9]+}}, i64 0)
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$2", ptr %{{[0-9]+}}, i64 0)
// CHECK: [[RECV_CH1:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: [[RECV_CH2:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[RECV_BUF1:%[0-9]+]] = alloca %"{{.*}}String"
// CHECK: [[RECV_OP1_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[RECV_CH1]], 0
// CHECK-NEXT: [[RECV_OP1_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP1_0]], ptr [[RECV_BUF1]], 1
// CHECK-NEXT: [[RECV_OP1_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP1_1]], i32 16, 2
// CHECK-NEXT: [[RECV_OP1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP1_2]], i1 false, 3
// CHECK: [[RECV_BUF2:%[0-9]+]] = alloca %"{{.*}}String"
// CHECK: [[RECV_OP2_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[RECV_CH2]], 0
// CHECK-NEXT: [[RECV_OP2_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP2_0]], ptr [[RECV_BUF2]], 1
// CHECK-NEXT: [[RECV_OP2_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP2_1]], i32 16, 2
// CHECK-NEXT: [[RECV_OP2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[RECV_OP2_2]], i1 false, 3
// CHECK: store %"{{.*}}ChanOp" [[RECV_OP1]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}ChanOp" [[RECV_OP2]], ptr %{{[0-9]+}}
// CHECK: [[RECV_CASES:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 2, 2
// CHECK-NEXT: [[TRY_SELECT:%[0-9]+]] = call { i64, i1, i1 } @"{{.*}}TrySelect"(%"{{.*}}Slice" [[RECV_CASES]])
// CHECK-NEXT: [[TRY_INDEX:%[0-9]+]] = extractvalue { i64, i1, i1 } [[TRY_SELECT]], 0
// CHECK-NEXT: [[TRY_OK:%[0-9]+]] = extractvalue { i64, i1, i1 } [[TRY_SELECT]], 1
// CHECK-NEXT: [[TRY_SELECTED:%[0-9]+]] = extractvalue { i64, i1, i1 } [[TRY_SELECT]], 2
// CHECK-NEXT: [[TRY_DISPATCH:%[0-9]+]] = select i1 [[TRY_SELECTED]], i64 [[TRY_INDEX]], i64 -1
// CHECK: [[RECV_DATA1:%[0-9]+]] = extractvalue %"{{.*}}ChanOp" [[RECV_OP1]], 1
// CHECK-NEXT: [[RECV_VALUE1:%[0-9]+]] = load %"{{.*}}String", ptr [[RECV_DATA1]]
// CHECK-NEXT: [[RECV_DATA2:%[0-9]+]] = extractvalue %"{{.*}}ChanOp" [[RECV_OP2]], 1
// CHECK-NEXT: [[RECV_VALUE2:%[0-9]+]] = load %"{{.*}}String", ptr [[RECV_DATA2]]
// CHECK: [[RECV_RESULT0:%[0-9]+]] = insertvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } undef, i64 [[TRY_DISPATCH]], 0
// CHECK-NEXT: [[RECV_RESULT1:%[0-9]+]] = insertvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT0]], i1 [[TRY_OK]], 1
// CHECK-NEXT: [[RECV_RESULT2:%[0-9]+]] = insertvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT1]], %"{{.*}}String" [[RECV_VALUE1]], 2
// CHECK-NEXT: [[RECV_RESULT:%[0-9]+]] = insertvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT2]], %"{{.*}}String" [[RECV_VALUE2]], 3
// CHECK-NEXT: [[RECV_CASE:%[0-9]+]] = extractvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT]], 0
// CHECK-NEXT: [[RECV_IS_CASE0:%[0-9]+]] = icmp eq i64 [[RECV_CASE]], 0
// CHECK: [[RECV_PRINT1:%[0-9]+]] = extractvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT]], 2
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[RECV_PRINT1]])
// CHECK: [[RECV_IS_CASE1:%[0-9]+]] = icmp eq i64 [[RECV_CASE]], 1
// CHECK: [[RECV_PRINT2:%[0-9]+]] = extractvalue { i64, i1, %"{{.*}}String", %"{{.*}}String" } [[RECV_RESULT]], 3
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[RECV_PRINT2]])
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @[[EXIT_TEXT]], i64 4 })
// CHECK-LABEL: define void @"main.recv$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[RECV1_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[RECV1_CH_SLOT:%[0-9]+]] = extractvalue { ptr } [[RECV1_ENV]], 0
// CHECK-NEXT: [[RECV1_CH:%[0-9]+]] = load ptr, ptr [[RECV1_CH_SLOT]]
// CHECK: store %"{{.*}}String" { ptr @[[CH1_TEXT]], i64 3 }, ptr [[RECV1_VALUE:%[0-9]+]]
// CHECK-NEXT: call i1 @"{{.*}}ChanSend"(ptr [[RECV1_CH]], ptr [[RECV1_VALUE]], i64 16)
// CHECK-LABEL: define void @"main.recv$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[RECV2_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[RECV2_CH_SLOT:%[0-9]+]] = extractvalue { ptr } [[RECV2_ENV]], 0
// CHECK-NEXT: [[RECV2_CH:%[0-9]+]] = load ptr, ptr [[RECV2_CH_SLOT]]
// CHECK: store %"{{.*}}String" { ptr @[[CH2_TEXT]], i64 3 }, ptr [[RECV2_VALUE:%[0-9]+]]
// CHECK-NEXT: call i1 @"{{.*}}ChanSend"(ptr [[RECV2_CH]], ptr [[RECV2_VALUE]], i64 16)
// CHECK-LABEL: define void @main.send(){{.*}} {
// CHECK: [[SEND_CH1_OBJ:%[0-9]+]] = call ptr @"{{.*}}NewChan"(i64 8, i64 0)
// CHECK: [[SEND_CH2_OBJ:%[0-9]+]] = call ptr @"{{.*}}NewChan"(i64 8, i64 0)
// CHECK: [[SEND_CH1:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK-NEXT: [[SEND_CH2:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: store i64 100, ptr [[SEND_BUF1:%[0-9]+]]
// CHECK-NEXT: [[SEND_OP1_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[SEND_CH1]], 0
// CHECK-NEXT: [[SEND_OP1_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP1_0]], ptr [[SEND_BUF1]], 1
// CHECK-NEXT: [[SEND_OP1_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP1_1]], i32 8, 2
// CHECK-NEXT: [[SEND_OP1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP1_2]], i1 true, 3
// CHECK: store i64 200, ptr [[SEND_BUF2:%[0-9]+]]
// CHECK-NEXT: [[SEND_OP2_0:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" undef, ptr [[SEND_CH2]], 0
// CHECK-NEXT: [[SEND_OP2_1:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP2_0]], ptr [[SEND_BUF2]], 1
// CHECK-NEXT: [[SEND_OP2_2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP2_1]], i32 8, 2
// CHECK-NEXT: [[SEND_OP2:%[0-9]+]] = insertvalue %"{{.*}}ChanOp" [[SEND_OP2_2]], i1 true, 3
// CHECK: store %"{{.*}}ChanOp" [[SEND_OP1]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}ChanOp" [[SEND_OP2]], ptr %{{[0-9]+}}
// CHECK: [[SEND_CASES:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 2, 2
// CHECK-NEXT: [[SEND_SELECT:%[0-9]+]] = call { i64, i1 } @"{{.*}}Select"(%"{{.*}}Slice" [[SEND_CASES]])
// CHECK-NEXT: [[SEND_CASE:%[0-9]+]] = extractvalue { i64, i1 } [[SEND_SELECT]], 0
// CHECK: [[SEND_DISPATCH:%[0-9]+]] = extractvalue { i64, i1 } %{{[0-9]+}}, 0
// CHECK-NEXT: [[SEND_IS_CASE0:%[0-9]+]] = icmp eq i64 [[SEND_DISPATCH]], 0
// CHECK: [[SEND_IS_CASE1:%[0-9]+]] = icmp eq i64 [[SEND_DISPATCH]], 1
// CHECK-LABEL: define void @"main.send$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[SEND1_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[SEND1_CH_SLOT:%[0-9]+]] = extractvalue { ptr } [[SEND1_ENV]], 0
// CHECK-NEXT: [[SEND1_CH:%[0-9]+]] = load ptr, ptr [[SEND1_CH_SLOT]]
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[SEND1_CH]], ptr [[SEND1_BUF:%[0-9]+]], i64 8)
// CHECK-NEXT: [[SEND1_VALUE:%[0-9]+]] = load i64, ptr [[SEND1_BUF]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[SEND1_VALUE]])
// CHECK-LABEL: define void @"main.send$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[SEND2_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[SEND2_CH_SLOT:%[0-9]+]] = extractvalue { ptr } [[SEND2_ENV]], 0
// CHECK-NEXT: [[SEND2_CH:%[0-9]+]] = load ptr, ptr [[SEND2_CH_SLOT]]
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[SEND2_CH]], ptr [[SEND2_BUF:%[0-9]+]], i64 8)
// CHECK-NEXT: [[SEND2_VALUE:%[0-9]+]] = load i64, ptr [[SEND2_BUF]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[SEND2_VALUE]])

func main() {
	send()
	recv()
}

func send() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		println(<-ch1)
	}()
	go func() {
		println(<-ch2)
	}()

	select {
	case ch1 <- 100:
	case ch2 <- 200:
	}
}

func recv() {
	c1 := make(chan string)
	c2 := make(chan string)
	go func() {
		c1 <- "ch1"
	}()
	go func() {
		c2 <- "ch2"
	}()

	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-c1:
			println(msg1)
		case msg2 := <-c2:
			println(msg2)
		default:
			println("exit")
		}
	}
}
