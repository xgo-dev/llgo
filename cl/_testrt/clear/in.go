// LITTEST
package clear

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/clear.Clear"(){{.*}} {
// CHECK: [[CLEAR_SLICE:%[0-9]+]] = insertvalue %"{{.*}}Slice" %{{[0-9]+}}, i64 4, 2
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.SliceClear"(ptr @"[]_llgo_int", %"{{.*}}Slice" [[CLEAR_SLICE]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}Slice" [[CLEAR_SLICE]])
// CHECK: [[CLEAR_MAP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_int", i64 4)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapClear"(ptr @"map[_llgo_int]_llgo_int", ptr [[CLEAR_MAP]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr [[CLEAR_MAP]])

func Clear() {
	a := []int{1, 2, 3, 4}
	clear(a)
	println(a)

	b := map[int]int{1: 1, 2: 2, 3: 3, 4: 4}
	clear(b)
	println(b)
}
