// LITTEST
package main

// The fixture covers a matrix of fixed array/slice and signed/unsigned index
// lowering. Follow each predicate and index into CheckIndexRange; helper names
// alone would not prove that the right condition is being checked.
// CHECK-LABEL: define void @main.array(i64 %0){{.*}} {
// CHECK: %[[ARRAY_NEG:[0-9]+]] = icmp slt i64 %0, 0
// CHECK: %[[ARRAY_UPPER:[0-9]+]] = icmp uge i64 %0, 2
// CHECK: %[[ARRAY_OOB:[0-9]+]] = or i1 %[[ARRAY_UPPER]], %[[ARRAY_NEG]]
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[ARRAY_OOB]], i64 %0, i1 true, i64 2)

// CHECK-LABEL: define void @main.array2(i64 %0){{.*}} {
// CHECK-NOT: icmp slt
// CHECK: %[[UARRAY_OOB:[0-9]+]] = icmp uge i64 %0, 2
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[UARRAY_OOB]], i64 %0, i1 false, i64 2)

// The literal-index cases keep the signedness bit correct after constant
// folding. They are intentionally representative; defer/recover mechanics are
// runtime-tested and are not part of this IR contract.
// CHECK-LABEL: define void @"main.init#10"(){{.*}} {
// CHECK: %[[SLICE10_LEN:[0-9]+]] = extractvalue %"{{.*}}Slice" %{{[0-9]+}}, 1
// CHECK: %[[SLICE10_UPPER:[0-9]+]] = icmp uge i64 -1, %[[SLICE10_LEN]]
// CHECK: %[[SLICE10_OOB:[0-9]+]] = or i1 %[[SLICE10_UPPER]], true
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[SLICE10_OOB]], i64 -1, i1 true, i64 %[[SLICE10_LEN]])

// CHECK-LABEL: define void @"main.init#12"(){{.*}} {
// CHECK-NOT: icmp slt
// CHECK: %[[SLICE12_LEN:[0-9]+]] = extractvalue %"{{.*}}Slice" %{{[0-9]+}}, 1
// CHECK: %[[SLICE12_OOB:[0-9]+]] = icmp uge i64 2, %[[SLICE12_LEN]]
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[SLICE12_OOB]], i64 2, i1 false, i64 %[[SLICE12_LEN]])

// CHECK-LABEL: define void @"main.init#7"(){{.*}} {
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 true, i64 -1, i1 true, i64 2)

// CHECK-LABEL: define void @"main.init#9"(){{.*}} {
// CHECK-NOT: icmp slt
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 true, i64 2, i1 false, i64 2)

// Slice bounds use the extracted dynamic length. Both helpers are checked so
// one source scenario cannot silently lose its range check.
// CHECK-LABEL: define void @main.slice(i64 %0){{.*}} {
// CHECK: %[[SLICE_LEN:[0-9]+]] = extractvalue %"{{.*}}Slice" %{{[0-9]+}}, 1
// CHECK: %[[SLICE_NEG:[0-9]+]] = icmp slt i64 %0, 0
// CHECK: %[[SLICE_UPPER:[0-9]+]] = icmp uge i64 %0, %[[SLICE_LEN]]
// CHECK: %[[SLICE_OOB:[0-9]+]] = or i1 %[[SLICE_UPPER]], %[[SLICE_NEG]]
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[SLICE_OOB]], i64 %0, i1 true, i64 %[[SLICE_LEN]])

// CHECK-LABEL: define void @main.slice2(i64 %0){{.*}} {
// CHECK: %[[SLICE2_LEN:[0-9]+]] = extractvalue %"{{.*}}Slice" %{{[0-9]+}}, 1
// CHECK: %[[SLICE2_NEG:[0-9]+]] = icmp slt i64 %0, 0
// CHECK: %[[SLICE2_UPPER:[0-9]+]] = icmp uge i64 %0, %[[SLICE2_LEN]]
// CHECK: %[[SLICE2_OOB:[0-9]+]] = or i1 %[[SLICE2_UPPER]], %[[SLICE2_NEG]]
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 %[[SLICE2_OOB]], i64 %0, i1 true, i64 %[[SLICE2_LEN]])

func main() {
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("array -1 must error")
		}
	}()
	array(-1)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("array 2 must error")
		}
	}()
	array(2)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("array2 must error")
		}
	}()
	array2(2)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("slice -1 must error")
		}
	}()
	slice(-1)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("slice 2 must error")
		}
	}()
	slice(2)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("slice2 2 must error")
		}
	}()
	slice2(2)
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("2 must error")
		}
	}()
	a := [...]int{1, 2}
	var n = -1
	println(a[n])
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("-1 must error")
		}
	}()
	a := [...]int{1, 2}
	var n = 2
	println(a[n])
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("2 must error")
		}
	}()
	a := [...]int{1, 2}
	var n uint = 2
	println(a[n])
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("2 must error")
		}
	}()
	a := []int{1, 2}
	var n = -1
	println(a[n])
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("-1 must error")
		}
	}()
	a := []int{1, 2}
	var n = 2
	println(a[n])
}

func init() {
	defer func() {
		if r := recover(); r == nil {
			panic("2 must error")
		}
	}()
	a := []int{1, 2}
	var n uint = 2
	println(a[n])
}

func array(n int) {
	println([...]int{1, 2}[n])
}

func array2(n uint) {
	println([...]int{1, 2}[n])
}

func slice(n int) {
	println([]int{1, 2}[n])
}

func slice2(n int) {
	println([]int{1, 2}[n])
}
