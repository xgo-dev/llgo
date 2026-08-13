// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[SLICE4_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE4:%[0-9]+]], 1
// CHECK-NEXT: [[SHORT4:%[0-9]+]] = icmp slt i64 [[SLICE4_LEN]], 4
// CHECK-NEXT: br i1 [[SHORT4]], label %{{.*}}, label %{{.*}}
// CHECK: [[PANIC4_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE4]], 1
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 4, i64 [[PANIC4_LEN]])
// CHECK: [[SLICE4_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE4]], 0
// CHECK: [[ARRAY4:%[0-9]+]] = load [4 x i8], ptr [[SLICE4_DATA]]
// CHECK: [[ARRAY4_LAST:%[0-9]+]] = extractvalue [4 x i8] [[ARRAY4]], 3
// CHECK: [[EQUAL4_LAST:%[0-9]+]] = icmp eq i8 %{{[0-9]+}}, [[ARRAY4_LAST]]
// CHECK-NEXT: [[EQUAL4:%[0-9]+]] = and i1 %{{[0-9]+}}, [[EQUAL4_LAST]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[EQUAL4]])
// CHECK: [[SLICE2_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE2:%[0-9]+]], 1
// CHECK-NEXT: [[SHORT2:%[0-9]+]] = icmp slt i64 [[SLICE2_LEN]], 2
// CHECK-NEXT: br i1 [[SHORT2]], label %{{.*}}, label %{{.*}}
// CHECK: [[PANIC2_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE2]], 1
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 2, i64 [[PANIC2_LEN]])
// CHECK: [[SLICE2_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[SLICE2]], 0
// CHECK-NEXT: [[ARRAY2:%[0-9]+]] = load [2 x i8], ptr [[SLICE2_DATA]]
// CHECK: [[ARRAY2_LAST:%[0-9]+]] = extractvalue [2 x i8] [[ARRAY2]], 1
// CHECK: [[EQUAL2_LAST:%[0-9]+]] = icmp eq i8 [[ARRAY2_LAST]], %{{[0-9]+}}
// CHECK-NEXT: [[EQUAL2:%[0-9]+]] = and i1 %{{[0-9]+}}, [[EQUAL2_LAST]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[EQUAL2]])
func main() {
	array := [4]byte{1, 2, 3, 4}
	ptr := (*[4]byte)(array[:])
	println(array == *ptr)
	println(*(*[2]byte)(array[:]) == [2]byte{1, 2})
}
