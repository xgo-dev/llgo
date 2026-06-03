// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"error", align 1

func main() {
	recursive()
}

func recursive() {
	type T int
	if got, want := recur1[T](5), T(110); got != want {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecur.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tprecur.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tprecur.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecur.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/tprecur.recursive"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecur.recursive"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testgo/tprecur.recur1[{{.*}}/cl/_testgo/tprecur.T.1.0]"(i64 5)
// CHECK-NEXT:   %1 = icmp ne i64 %0, 110
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tprecur.recur1[{{.*}}/cl/_testgo/tprecur.T.1.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq i64 %0, 0
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   ret i64 1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %2 = sub i64 %0, 1
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/tprecur.recur2[{{.*}}/cl/_testgo/tprecur.T.1.0]"(i64 %2)
// CHECK-NEXT:   %4 = mul i64 %0, %3
// CHECK-NEXT:   ret i64 %4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = icmp eq i64 %0, 1
// CHECK-NEXT:   br i1 %5, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tprecur.recur2[{{.*}}/cl/_testgo/tprecur.T.1.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.MakeSlice"(i64 %0, i64 %0, i64 8)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %4 = add i64 %3, 1
// CHECK-NEXT:   %5 = icmp slt i64 %4, %2
// CHECK-NEXT:   br i1 %5, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %6 = add i64 %4, 1
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %9 = icmp slt i64 %4, 0
// CHECK-NEXT:   %10 = icmp uge i64 %4, %8
// CHECK-NEXT:   %11 = or i1 %10, %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds i64, ptr %7, i64 %4
// CHECK-NEXT:   store i64 %6, ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_3
// CHECK-NEXT:   %14 = phi i64 [ 0, %_llgo_3 ], [ %25, %_llgo_5 ]
// CHECK-NEXT:   %15 = phi i64 [ -1, %_llgo_3 ], [ %16, %_llgo_5 ]
// CHECK-NEXT:   %16 = add i64 %15, 1
// CHECK-NEXT:   %17 = icmp slt i64 %16, %13
// CHECK-NEXT:   br i1 %17, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %20 = icmp slt i64 %16, 0
// CHECK-NEXT:   %21 = icmp uge i64 %16, %19
// CHECK-NEXT:   %22 = or i1 %21, %20
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds i64, ptr %18, i64 %16
// CHECK-NEXT:   %24 = load i64, ptr %23, align 8
// CHECK-NEXT:   %25 = add i64 %14, %24
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %26 = sub i64 %0, 1
// CHECK-NEXT:   %27 = call i64 @"{{.*}}/cl/_testgo/tprecur.recur1[{{.*}}/cl/_testgo/tprecur.T.1.0]"(i64 %26)
// CHECK-NEXT:   %28 = add i64 %14, %27
// CHECK-NEXT:   ret i64 %28
// CHECK-NEXT: }
