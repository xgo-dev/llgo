// LITTEST
package clear

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/clear.Clear"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   store i64 2, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   store i64 3, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %0, i64 3
// CHECK-NEXT:   store i64 4, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 4, 1
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 4, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SliceClear"(ptr @"[]_llgo_int", %"{{.*}}/runtime/internal/runtime.Slice" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_int", i64 4)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %9, align 8
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %8, ptr %9)
// CHECK-NEXT:   store i64 1, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %11, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %8, ptr %11)
// CHECK-NEXT:   store i64 2, ptr %12, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 3, ptr %13, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %8, ptr %13)
// CHECK-NEXT:   store i64 3, ptr %14, align 8
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 4, ptr %15, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %8, ptr %15)
// CHECK-NEXT:   store i64 4, ptr %16, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.MapClear"(ptr @"map[_llgo_int]_llgo_int", ptr %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func Clear() {
	a := []int{1, 2, 3, 4}
	clear(a)
	println(a)

	b := map[int]int{1: 1, 2: 2, 3: 3, 4: 4}
	clear(b)
	println(b)
}
