// LITTEST
package main

// Check the four semantically distinct make([]T, len, cap) cases: valid,
// negative length, len greater than cap, and allocation-size overflow.
// CHECK-LABEL: define void @"main.init#1"(){{.*}} {
// CHECK: [[VALID1:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 2, i64 4, i64 8)
// CHECK-NEXT: [[VALID1_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[VALID1]], 1
// CHECK-NEXT: [[VALID1_BAD_LEN:%[0-9]+]] = icmp ne i64 [[VALID1_LEN]], 2
// CHECK: [[VALID1_CAP:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[VALID1]], 2
// CHECK-NEXT: [[VALID1_BAD_CAP:%[0-9]+]] = icmp ne i64 [[VALID1_CAP]], 4
// CHECK-LABEL: define void @"main.init#2"(){{.*}} {
// CHECK: [[VALID2:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 2, i64 4, i64 8)
// CHECK-NEXT: [[VALID2_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[VALID2]], 1
// CHECK-NEXT: [[VALID2_BAD_LEN:%[0-9]+]] = icmp ne i64 [[VALID2_LEN]], 2
// CHECK: [[VALID2_CAP:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[VALID2]], 2
// CHECK-NEXT: [[VALID2_BAD_CAP:%[0-9]+]] = icmp ne i64 [[VALID2_CAP]], 4
// CHECK-LABEL: define void @"main.init#3"(){{.*}} {
// CHECK: [[NEG_STATE:%[0-9]+]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @"main.init#3$1")
// CHECK-NEXT: call void @"main.init#3$1"()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[NEG_STATE]])
// CHECK: call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 -1, i64 -1, i64 8)
// CHECK-LABEL: define void @"main.init#3$1"(){{.*}} {
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.init#3$1", ptr [[NEG_TOKEN:%[0-9]+]])
// CHECK-NEXT: [[NEG_RECOVER:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[NEG_TOKEN]])
// CHECK-NEXT: [[NEG_MISSING:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[NEG_RECOVER]], %"{{.*}}eface" zeroinitializer)
// CHECK-NEXT: br i1 [[NEG_MISSING]], label %{{.*}}, label %{{.*}}
// CHECK-LABEL: define void @"main.init#4"(){{.*}} {
// CHECK: [[CAP_STATE:%[0-9]+]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @"main.init#4$1")
// CHECK-NEXT: call void @"main.init#4$1"()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[CAP_STATE]])
// CHECK: call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 2, i64 1, i64 8)
// CHECK-LABEL: define void @"main.init#4$1"(){{.*}} {
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.init#4$1", ptr [[CAP_TOKEN:%[0-9]+]])
// CHECK-NEXT: [[CAP_RECOVER:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[CAP_TOKEN]])
// CHECK-NEXT: [[CAP_MISSING:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[CAP_RECOVER]], %"{{.*}}eface" zeroinitializer)
// CHECK-NEXT: br i1 [[CAP_MISSING]], label %{{.*}}, label %{{.*}}
// CHECK-LABEL: define void @"main.init#5"(){{.*}} {
// CHECK: [[OVERFLOW_STATE:%[0-9]+]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @"main.init#5$1")
// CHECK-NEXT: call void @"main.init#5$1"()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[OVERFLOW_STATE]])
// CHECK: call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 9223372036854775807, i64 9223372036854775807, i64 8)
// CHECK-LABEL: define void @"main.init#5$1"(){{.*}} {
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.init#5$1", ptr [[OVERFLOW_TOKEN:%[0-9]+]])
// CHECK-NEXT: [[OVERFLOW_RECOVER:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[OVERFLOW_TOKEN]])
// CHECK-NEXT: [[OVERFLOW_MISSING:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[OVERFLOW_RECOVER]], %"{{.*}}eface" zeroinitializer)
// CHECK-NEXT: br i1 [[OVERFLOW_MISSING]], label %{{.*}}, label %{{.*}}

func main() {
}

func init() {
	var n int = 2
	buf := make([]int, n, n*2)
	if len(buf) != 2 || cap(buf) != 4 {
		panic("error")
	}
}

func init() {
	var n int32 = 2
	buf := make([]int, n, n*2)
	if len(buf) != 2 || cap(buf) != 4 {
		panic("error")
	}
}

func init() {
	defer func() {
		r := recover()
		if r == nil {
			println("must error")
		}
	}()
	var n int = -1
	buf := make([]int, n)
	_ = buf
}

func init() {
	defer func() {
		r := recover()
		if r == nil {
			println("must error")
		}
	}()
	var n int = 2
	buf := make([]int, n, n-1)
	_ = buf
}

func init() {
	defer func() {
		r := recover()
		if r == nil {
			println("must error")
		}
	}()
	var n int64 = 1<<63 - 1
	buf := make([]int, n)
	_ = buf
}
