// LITTEST
package main

type data struct {
	s string
	c chan int
	m map[int]string
	a []int
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// First read every len/cap operation from the same zero-valued struct.
// CHECK: [[VALUE_DATA:%.*]] = alloca %main.data, align 8
// CHECK-NEXT: [[ZERO_DATA:%.*]] = alloca %main.data, align 8
// CHECK-NEXT: call void @llvm.memset.p0.i64(ptr [[ZERO_DATA]], i8 0, i64 56, i1 false)
// CHECK: [[ZERO_STRING:%.*]] = load %"{{.*}}String", ptr {{%.*}}
// CHECK-NEXT: [[ZERO_STRING_LEN:%.*]] = extractvalue %"{{.*}}String" [[ZERO_STRING]], 1
// CHECK: [[ZERO_CHAN:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[ZERO_CHAN_LEN:%.*]] = call i64 @"{{.*}}ChanLen"(ptr [[ZERO_CHAN]])
// CHECK: [[ZERO_MAP:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[ZERO_MAP_LEN:%.*]] = call i64 @"{{.*}}MapLen"(ptr [[ZERO_MAP]])
// CHECK: [[ZERO_SLICE:%.*]] = load %"{{.*}}Slice", ptr {{%.*}}
// CHECK-NEXT: [[ZERO_SLICE_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[ZERO_SLICE]], 1
// CHECK: [[ZERO_CHAN_FOR_CAP:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[ZERO_CHAN_CAP:%.*]] = call i64 @"{{.*}}ChanCap"(ptr [[ZERO_CHAN_FOR_CAP]])
// CHECK: [[ZERO_SLICE_FOR_CAP:%.*]] = load %"{{.*}}Slice", ptr {{%.*}}
// CHECK-NEXT: [[ZERO_SLICE_CAP:%.*]] = extractvalue %"{{.*}}Slice" [[ZERO_SLICE_FOR_CAP]], 2
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_STRING_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_CHAN_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_MAP_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_SLICE_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_CHAN_CAP]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZERO_SLICE_CAP]])
// Then construct and query the populated value, preserving channel/map/slice identities.
// CHECK: call void @llvm.memset.p0.i64(ptr [[VALUE_DATA]], i8 0, i64 56, i1 false)
// CHECK: [[VALUE_CHAN:%.*]] = call ptr @"{{.*}}NewChan"(i64 8, i64 2)
// CHECK: [[VALUE_MAP:%.*]] = call ptr @"{{.*}}MakeMap"(ptr @"map[_llgo_int]_llgo_string", i64 1)
// CHECK: [[VALUE_SLICE0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr {{%.*}}, 0
// CHECK-NEXT: [[VALUE_SLICE1:%.*]] = insertvalue %"{{.*}}Slice" [[VALUE_SLICE0]], i64 3, 1
// CHECK-NEXT: [[VALUE_SLICE:%.*]] = insertvalue %"{{.*}}Slice" [[VALUE_SLICE1]], i64 3, 2
// CHECK: store ptr [[VALUE_CHAN]], ptr {{%.*}}
// CHECK: store ptr [[VALUE_MAP]], ptr {{%.*}}
// CHECK: store %"{{.*}}Slice" [[VALUE_SLICE]], ptr {{%.*}}
// CHECK: [[VALUE_STRING:%.*]] = load %"{{.*}}String", ptr {{%.*}}
// CHECK-NEXT: [[VALUE_STRING_LEN:%.*]] = extractvalue %"{{.*}}String" [[VALUE_STRING]], 1
// CHECK: [[VALUE_CHAN_LOAD:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[VALUE_CHAN_LEN:%.*]] = call i64 @"{{.*}}ChanLen"(ptr [[VALUE_CHAN_LOAD]])
// CHECK: [[VALUE_MAP_LOAD:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[VALUE_MAP_LEN:%.*]] = call i64 @"{{.*}}MapLen"(ptr [[VALUE_MAP_LOAD]])
// CHECK: [[VALUE_SLICE_LOAD:%.*]] = load %"{{.*}}Slice", ptr {{%.*}}
// CHECK-NEXT: [[VALUE_SLICE_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[VALUE_SLICE_LOAD]], 1
// CHECK: [[VALUE_CHAN_CAP_LOAD:%.*]] = load ptr, ptr {{%.*}}
// CHECK-NEXT: [[VALUE_CHAN_CAP:%.*]] = call i64 @"{{.*}}ChanCap"(ptr [[VALUE_CHAN_CAP_LOAD]])
// CHECK: [[VALUE_SLICE_CAP_LOAD:%.*]] = load %"{{.*}}Slice", ptr {{%.*}}
// CHECK-NEXT: [[VALUE_SLICE_CAP:%.*]] = extractvalue %"{{.*}}Slice" [[VALUE_SLICE_CAP_LOAD]], 2
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_STRING_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_CHAN_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_MAP_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_SLICE_LEN]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_CHAN_CAP]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[VALUE_SLICE_CAP]])
func main() {
	d := &data{}
	println(len(d.s), len(d.c), len(d.m), len(d.a), cap(d.c), cap(d.a))
	v := &data{s: "hello", c: make(chan int, 2), m: map[int]string{1: "hello"}, a: []int{1, 2, 3}}
	println(len(v.s), len(v.c), len(v.m), len(v.a), cap(v.c), cap(v.a))
	checkChannelPoll()
}

// This fixture also runs on baremetal, where the receive-readiness hint is
// disabled. Nonblocking receives must still take the original locked path.
func checkChannelPoll() {
	var nilChannel chan int
	select {
	case <-nilChannel:
		panic("nil receive selected")
	default:
	}
	ch := make(chan int, 1)
	select {
	case <-ch:
		panic("empty receive selected")
	default:
	}
	ch <- 7
	select {
	case value, ok := <-ch:
		if value != 7 || !ok {
			panic("buffered receive corrupted")
		}
	default:
		panic("buffered receive missed")
	}
	close(ch)
	select {
	case value, ok := <-ch:
		if value != 0 || ok {
			panic("closed receive corrupted")
		}
	default:
		panic("closed receive missed")
	}
}
