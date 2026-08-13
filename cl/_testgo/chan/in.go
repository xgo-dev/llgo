// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// First channel: box/inspect one channel identity, capture its slot in the
// sender goroutine, then receive and print the transmitted value.
// CHECK: [[CH1_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[CH1:%.*]] = call ptr @"{{.*}}NewChan"(i64 8, i64 10)
// CHECK-NEXT: store ptr [[CH1]], ptr [[CH1_SLOT]]
// CHECK: [[CH1_BOX_DATA:%.*]] = load ptr, ptr [[CH1_SLOT]]
// CHECK-NEXT: [[CH1_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @"chan _llgo_int", ptr undef }, ptr [[CH1_BOX_DATA]], 1
// CHECK: [[CH1_PRINT_DATA:%.*]] = load ptr, ptr [[CH1_SLOT]]
// CHECK-NEXT: [[CH1_LEN_DATA:%.*]] = load ptr, ptr [[CH1_SLOT]]
// CHECK-NEXT: [[CH1_LEN:%.*]] = call i64 @"{{.*}}ChanLen"(ptr [[CH1_LEN_DATA]])
// CHECK: [[CH1_CAP_DATA:%.*]] = load ptr, ptr [[CH1_SLOT]]
// CHECK-NEXT: [[CH1_CAP:%.*]] = call i64 @"{{.*}}ChanCap"(ptr [[CH1_CAP_DATA]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[CH1_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[CH1_CAP]])
// CHECK: call void @"{{.*}}PrintEface"(%"{{.*}}eface" [[CH1_EFACE]])
// CHECK: [[SEND_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[CH1_SLOT]], ptr {{%.*}}
// CHECK: [[SEND_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[SEND_ENV]], 1
// CHECK: store { ptr, ptr } [[SEND_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$1", ptr {{%.*}}, i64 0)
// CHECK: [[CH1_RECV:%.*]] = load ptr, ptr [[CH1_SLOT]]
// CHECK: [[CH1_RECV_BUF:%.*]] = alloca i64
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[CH1_RECV]], ptr [[CH1_RECV_BUF]], i64 8)
// CHECK-NEXT: [[CH1_VALUE:%.*]] = load i64, ptr [[CH1_RECV_BUF]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[CH1_VALUE]])
// Second channel: capture it in a closer goroutine and preserve the receive ok
// bit alongside the zero value returned after close.
// CHECK: [[CH2_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[CH2:%.*]] = call ptr @"{{.*}}NewChan"(i64 8, i64 10)
// CHECK-NEXT: store ptr [[CH2]], ptr [[CH2_SLOT]]
// CHECK: [[CLOSE_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[CH2_SLOT]], ptr {{%.*}}
// CHECK: [[CLOSE_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr [[CLOSE_ENV]], 1
// CHECK: store { ptr, ptr } [[CLOSE_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$2", ptr {{%.*}}, i64 0)
// CHECK: [[CH2_RECV:%.*]] = load ptr, ptr [[CH2_SLOT]]
// CHECK: [[CH2_RECV_BUF:%.*]] = alloca i64
// CHECK: [[CH2_OK:%.*]] = call i1 @"{{.*}}ChanRecv"(ptr [[CH2_RECV]], ptr [[CH2_RECV_BUF]], i64 8)
// CHECK-NEXT: [[CH2_VALUE:%.*]] = load i64, ptr [[CH2_RECV_BUF]]
// CHECK: [[CH2_PAIR0:%.*]] = insertvalue { i64, i1 } undef, i64 [[CH2_VALUE]], 0
// CHECK-NEXT: [[CH2_PAIR:%.*]] = insertvalue { i64, i1 } [[CH2_PAIR0]], i1 [[CH2_OK]], 1
// CHECK-NEXT: [[CH2_PRINT_VALUE:%.*]] = extractvalue { i64, i1 } [[CH2_PAIR]], 0
// CHECK-NEXT: [[CH2_PRINT_OK:%.*]] = extractvalue { i64, i1 } [[CH2_PAIR]], 1
// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 [[CH2_PRINT_VALUE]])
// CHECK: call void @"{{.*}}PrintBool"(i1 [[CH2_PRINT_OK]])

// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[SEND_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[SEND_SLOT:%.*]] = extractvalue { ptr } [[SEND_CAPTURE]], 0
// CHECK-NEXT: [[SEND_CH:%.*]] = load ptr, ptr [[SEND_SLOT]]
// CHECK: [[SEND_BUF:%.*]] = alloca i64
// CHECK: store i64 100, ptr [[SEND_BUF]]
// CHECK-NEXT: call i1 @"{{.*}}ChanSend"(ptr [[SEND_CH]], ptr [[SEND_BUF]], i64 8)

// CHECK-LABEL: define void @"main.main$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[CLOSE_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[CLOSE_SLOT:%.*]] = extractvalue { ptr } [[CLOSE_CAPTURE]], 0
// CHECK-NEXT: [[CLOSE_CH:%.*]] = load ptr, ptr [[CLOSE_SLOT]]
// CHECK-NEXT: call void @"{{.*}}ChanClose"(ptr [[CLOSE_CH]])

func main() {
	ch := make(chan int, 10)
	var v any = ch
	println(ch, len(ch), cap(ch), v)
	go func() {
		ch <- 100
	}()
	n := <-ch
	println(n)

	ch2 := make(chan int, 10)
	go func() {
		close(ch2)
	}()
	n2, ok := <-ch2
	println(n2, ok)
}
