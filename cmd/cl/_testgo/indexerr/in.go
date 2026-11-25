// LITTEST
// Scope: common
package main

// The fixture covers a matrix of fixed array/slice and signed/unsigned index
// lowering. Signed and unsigned indexes share a single unsigned comparison
// (idx >=u len); signed negatives fail as large unsigned values. Follow each
// predicate into the panic branch and each index into PanicIndex/PanicIndexU.
// CHECK-LABEL: define void @main.array(i64 %0){{.*}} {
// CHECK-NOT: icmp slt
// CHECK: %[[ARRAY_OOB:[0-9]+]] = icmp uge i64 %0, 2
// CHECK: br i1 %[[ARRAY_OOB]], label %{{_llgo_[0-9]+}}, label %{{_llgo_[0-9]+}}
// CHECK: call void @"{{.*}}PanicIndex"(i64 %0, i64 2)
// CHECK-NEXT: br label %{{_llgo_[0-9]+}}

// CHECK-LABEL: define void @main.array2(i64 %0){{.*}} {
// CHECK-NOT: icmp slt
// CHECK: %[[UARRAY_OOB:[0-9]+]] = icmp uge i64 %0, 2
// CHECK: br i1 %[[UARRAY_OOB]], label %{{_llgo_[0-9]+}}, label %{{_llgo_[0-9]+}}
// CHECK: call void @"{{.*}}PanicIndexU"(i64 %0, i64 2)
// CHECK-NEXT: br label %{{_llgo_[0-9]+}}

// Narrow signed indices must be sign-extended before the shared unsigned
// bounds predicate and before addressing the selected element.
// CHECK-LABEL: define i64 @main.narrowArray(i8 %0){{.*}} {
// CHECK: [[NARROW:%.*]] = sext i8 %0 to i64
// CHECK-NOT: icmp slt
// CHECK: [[NARROW_OOB:%.*]] = icmp uge i64 [[NARROW]], 2
// CHECK: call void @"{{.*}}PanicIndex"(i64 [[NARROW]], i64 2)
// CHECK: getelementptr inbounds i64, ptr %{{.*}}, i64 [[NARROW]]

// Slice bounds use the extracted dynamic length. The duplicate slice2 helper
// remains a runtime scenario; one lowering contract is sufficient here.
// CHECK-LABEL: define void @main.slice(i64 %0){{.*}} {
// CHECK: %[[SLICE_LEN:[0-9]+]] = extractvalue %"{{.*}}Slice" %{{[0-9]+}}, 1
// CHECK-NOT: icmp slt
// CHECK: %[[SLICE_OOB:[0-9]+]] = icmp uge i64 %0, %[[SLICE_LEN]]
// CHECK: br i1 %[[SLICE_OOB]], label %{{_llgo_[0-9]+}}, label %{{_llgo_[0-9]+}}
// CHECK: call void @"{{.*}}PanicIndex"(i64 %0, i64 %[[SLICE_LEN]])
// CHECK-NEXT: br label %{{_llgo_[0-9]+}}

func narrowArray(n int8) int {
	return [...]int{1, 2}[n]
}

// zeroMapLookup preserves the zero-length-array regression from _testlibgo/mapzero:
// the statically unreachable success edge must still lower the zero-value map
// lookup against a nil map pointer instead of crashing the compiler.
// CHECK-LABEL: define i64 @main.zeroMapLookup(i64 %0){{.*}} {
// CHECK: [[ZERO_OOB:%[0-9]+]] = icmp uge i64 %0, 0
// CHECK-NEXT: br i1 [[ZERO_OOB]], label %{{_llgo_[0-9]+}}, label %{{_llgo_[0-9]+}}
// CHECK: call void @"{{.*}}PanicIndex"(i64 %0, i64 0)
// CHECK: [[MAP_SLOT:%[0-9]+]] = call ptr @"{{.*}}MapAccess1Fast64"(ptr @"map[_llgo_int]_llgo_int", ptr null, i64 0)
// CHECK-NEXT: [[MAP_VALUE:%[0-9]+]] = load i64, ptr [[MAP_SLOT]]
// CHECK-NEXT: ret i64 [[MAP_VALUE]]
func zeroMapLookup(n int) int {
	return [0]map[int]int{}[n][0]
}

func main() {
	if narrowArray(1) != 2 {
		panic("narrow index")
	}
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
