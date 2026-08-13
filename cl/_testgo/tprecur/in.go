// LITTEST
package main

func main() {
	recursive()
}

func recursive() {
	type T int
	if got, want := recur1[T](5), T(110); got != want {
		panic("error")
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT: call void @main.recursive()
// CHECK-NEXT: ret void

// CHECK-LABEL: define void @main.recursive(){{.*}} {
// CHECK: [[RECUR_RESULT:%.*]] = call i64 @"main.recur1[main.T.1.0]"(i64 5)
// CHECK-NEXT: [[RECUR_WRONG:%.*]] = icmp ne i64 [[RECUR_RESULT]], 110
// CHECK-NEXT: br i1 [[RECUR_WRONG]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" {{%.*}})
// CHECK-NEXT: unreachable

type Integer interface {
	~int | ~int32 | ~int64
}

func recur1[T Integer](n T) T {
	if n == 0 || n == 1 {
		return T(1)
	} else {
		return n * recur2(n-1)
	}
}

func recur2[T Integer](n T) T {
	list := make([]T, n)
	for i, _ := range list {
		list[i] = T(i + 1)
	}
	var sum T
	for _, elt := range list {
		sum += elt
	}
	return sum + recur1(n-1)
}

// recur1 preserves both base cases and passes n-1 to recur2 before multiplying.
// CHECK-LABEL: define linkonce i64 @"main.recur1[main.T.1.0]"(i64 %0){{.*}} {
// CHECK: [[IS_ZERO:%.*]] = icmp eq i64 %0, 0
// CHECK-NEXT: br i1 [[IS_ZERO]], label %{{.*}}, label %{{.*}}
// CHECK: ret i64 1
// CHECK: [[R1_PREV:%.*]] = sub i64 %0, 1
// CHECK-NEXT: [[R2_RESULT:%.*]] = call i64 @"main.recur2[main.T.1.0]"(i64 [[R1_PREV]])
// CHECK-NEXT: [[R1_RESULT:%.*]] = mul i64 %0, [[R2_RESULT]]
// CHECK-NEXT: ret i64 [[R1_RESULT]]
// CHECK: [[IS_ONE:%.*]] = icmp eq i64 %0, 1
// CHECK-NEXT: br i1 [[IS_ONE]], label %{{.*}}, label %{{.*}}

// recur2 builds [1..n], sums it, and adds recur1(n-1).
// CHECK-LABEL: define linkonce i64 @"main.recur2[main.T.1.0]"(i64 %0){{.*}} {
// CHECK: [[LIST:%.*]] = call %"{{.*}}Slice" @"{{.*}}MakeSlice"(i64 %0, i64 %0, i64 8)
// CHECK-NEXT: [[LIST_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[LIST]], 1
// CHECK: [[FILL_I0:%.*]] = phi i64 [ -1, %{{.*}} ], [ [[FILL_I:%.*]], %{{.*}} ]
// CHECK-NEXT: [[FILL_I]] = add i64 [[FILL_I0]], 1
// CHECK-NEXT: [[FILL_MORE:%.*]] = icmp slt i64 [[FILL_I]], [[LIST_LEN]]
// CHECK: [[FILL_VALUE:%.*]] = add i64 [[FILL_I]], 1
// CHECK: [[LIST_DATA:%.*]] = extractvalue %"{{.*}}Slice" [[LIST]], 0
// CHECK: [[LIST_LEN_FOR_STORE:%.*]] = extractvalue %"{{.*}}Slice" [[LIST]], 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{%.*}}, i64 [[FILL_I]], i1 true, i64 [[LIST_LEN_FOR_STORE]])
// CHECK-NEXT: [[LIST_ELEM:%.*]] = getelementptr inbounds i64, ptr [[LIST_DATA]], i64 [[FILL_I]]
// CHECK-NEXT: store i64 [[FILL_VALUE]], ptr [[LIST_ELEM]]
// CHECK: [[SUM:%.*]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_SUM:%.*]], %{{.*}} ]
// CHECK-NEXT: [[SUM_I0:%.*]] = phi i64 [ -1, %{{.*}} ], [ [[SUM_I:%.*]], %{{.*}} ]
// CHECK-NEXT: [[SUM_I]] = add i64 [[SUM_I0]], 1
// CHECK: [[SUM_DATA:%.*]] = extractvalue %"{{.*}}Slice" [[LIST]], 0
// CHECK: [[SUM_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[LIST]], 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{%.*}}, i64 [[SUM_I]], i1 true, i64 [[SUM_LEN]])
// CHECK-NEXT: [[SUM_ELEM_PTR:%.*]] = getelementptr inbounds i64, ptr [[SUM_DATA]], i64 [[SUM_I]]
// CHECK-NEXT: [[SUM_ELEM:%.*]] = load i64, ptr [[SUM_ELEM_PTR]]
// CHECK-NEXT: [[NEXT_SUM]] = add i64 [[SUM]], [[SUM_ELEM]]
// CHECK: [[R2_PREV:%.*]] = sub i64 %0, 1
// CHECK-NEXT: [[R1_TAIL:%.*]] = call i64 @"main.recur1[main.T.1.0]"(i64 [[R2_PREV]])
// CHECK-NEXT: [[R2_FINAL:%.*]] = add i64 [[SUM]], [[R1_TAIL]]
// CHECK-NEXT: ret i64 [[R2_FINAL]]
