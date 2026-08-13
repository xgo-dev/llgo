// LITTEST
package main

type point struct {
	x int
	y int
}

type N [2]int
type T *N
type S []int

// CHECK-LABEL: define void @main.main(){{.*}} {
// Array-of-struct selection: index 2 is loaded as a point, then both fields of
// that selected value are consumed.
// CHECK: %[[POINT:[0-9]+]] = alloca %main.point
// CHECK: %[[POINTS:[0-9]+]] = alloca [3 x %main.point]
// CHECK: %[[POINT2_INIT:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINTS]], i64 2
// CHECK: %[[POINT2_X:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINT2_INIT]], i32 0, i32 0
// CHECK: %[[POINT2_Y:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINT2_INIT]], i32 0, i32 1
// CHECK: store i64 5, ptr %[[POINT2_X]]
// CHECK: store i64 6, ptr %[[POINT2_Y]]
// CHECK: %[[POINT2:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINTS]], i64 2
// CHECK: %[[SELECTED_POINT:[0-9]+]] = load %main.point, ptr %[[POINT2]]
// CHECK: store %main.point %[[SELECTED_POINT]], ptr %[[POINT]]
// CHECK: %[[POINT_X:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINT]], i32 0, i32 0
// CHECK: load i64, ptr %[[POINT_X]]
// CHECK: %[[POINT_Y:[0-9]+]] = getelementptr inbounds %main.point, ptr %[[POINT]], i32 0, i32 1
// CHECK: load i64, ptr %[[POINT_Y]]

// Nested arrays select row 1 before indexing its two elements.
// CHECK: %[[ROW:[0-9]+]] = alloca [2 x i64]
// CHECK: %[[MATRIX:[0-9]+]] = alloca [2 x [2 x i64]]
// CHECK: %[[ROW1_INIT:[0-9]+]] = getelementptr inbounds [2 x i64], ptr %[[MATRIX]], i64 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"
// CHECK: %[[ROW1_ELEM0:[0-9]+]] = getelementptr inbounds i64, ptr %[[ROW1_INIT]], i64 0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"
// CHECK: %[[ROW1_ELEM1:[0-9]+]] = getelementptr inbounds i64, ptr %[[ROW1_INIT]], i64 1
// CHECK: store i64 3, ptr %[[ROW1_ELEM0]]
// CHECK: store i64 4, ptr %[[ROW1_ELEM1]]
// CHECK: %[[ROW1:[0-9]+]] = getelementptr inbounds [2 x i64], ptr %[[MATRIX]], i64 1
// CHECK: %[[SELECTED_ROW:[0-9]+]] = load [2 x i64], ptr %[[ROW1]]
// CHECK: store [2 x i64] %[[SELECTED_ROW]], ptr %[[ROW]]
// CHECK: getelementptr inbounds i64, ptr %[[ROW]], i64 0
// CHECK: getelementptr inbounds i64, ptr %[[ROW]], i64 1

// The SSA-known array index is folded to element 2 without losing the selected
// element load.
// CHECK: %[[INTS:[0-9]+]] = alloca [5 x i64]
// CHECK: %[[INT2_INIT:[0-9]+]] = getelementptr inbounds i64, ptr %[[INTS]], i64 2
// CHECK: store i64 3, ptr %[[INT2_INIT]]
// CHECK: %[[INT2:[0-9]+]] = getelementptr inbounds i64, ptr %[[INTS]], i64 2
// CHECK: load i64, ptr %[[INT2]]

// String indexes are byte loads converted through StringFromUint64. Capture
// each byte so an unrelated conversion cannot satisfy the check.
// CHECK: %[[BYTE2:[0-9]+]] = load i8, ptr getelementptr inbounds (i8, ptr @[[TEXT:[0-9]+]], i64 2)
// CHECK: %[[RUNE2:[0-9]+]] = zext i8 %[[BYTE2]] to i64
// CHECK: call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %[[RUNE2]])
// CHECK: %[[BYTE1:[0-9]+]] = load i8, ptr getelementptr inbounds (i8, ptr @[[TEXT]], i64 1)
// CHECK: %[[RUNE1:[0-9]+]] = zext i8 %[[BYTE1]] to i64
// CHECK: call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 %[[RUNE1]])

// Named pointer-to-array indexing and named-slice indexing use different
// lowering. The slice predicate, length and data pointer must stay associated.
// CHECK: %[[NAMED_ARRAY:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: %[[NAMED_ELEM_INIT:[0-9]+]] = getelementptr inbounds i64, ptr %[[NAMED_ARRAY]], i64 1
// CHECK: store i64 2, ptr %[[NAMED_ELEM_INIT]]
// CHECK: %[[NAMED_ELEM:[0-9]+]] = getelementptr inbounds i64, ptr %[[NAMED_ARRAY]], i64 1
// CHECK: load i64, ptr %[[NAMED_ELEM]]
// CHECK: %[[SLICE_DATA_RAW:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK: %[[SLICE0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %[[SLICE_DATA_RAW]], 0
// CHECK: %[[SLICE1:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE0]], i64 4, 1
// CHECK: %[[SLICE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE1]], i64 4, 2
// CHECK: %[[SLICE_DATA:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 0
// CHECK: %[[SLICE_LEN:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %[[SLICE]], 1
// CHECK: %[[SLICE_OOB:[0-9]+]] = icmp uge i64 1, %[[SLICE_LEN]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %[[SLICE_OOB]], i64 1, i1 true, i64 %[[SLICE_LEN]])
// CHECK: %[[SLICE_ELEM:[0-9]+]] = getelementptr inbounds i64, ptr %[[SLICE_DATA]], i64 1
// CHECK: load i64, ptr %[[SLICE_ELEM]]

// A zero array indexed at constant zero folds to its zero value.
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)

func main() {
	a := [...]point{{1, 2}, {3, 4}, {5, 6}}[2]
	println(a.x, a.y)

	b := [...][2]int{[2]int{1, 2}, [2]int{3, 4}}[1]
	println(b[0], b[1])

	var i int = 2
	println([...]int{1, 2, 3, 4, 5}[i])

	s := "123456"
	println(string(s[i]))
	println(string("123456"[1]))

	var n = N{1, 2}
	var t T = &n
	println(t[1])
	var s1 = S{1, 2, 3, 4}
	println(s1[1])

	println([2]int{}[0])
}
