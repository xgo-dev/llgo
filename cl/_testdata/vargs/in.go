// LITTEST
// Scope: common
package main

import _ "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

func main() {
	test(1, 2, 3)
}

func test(a ...any) {
	for _, v := range a {
		printf(cstr("%d\n"), v.(int))
	}
}

// Keep this check focused on the variadic slice, interface boxing, and
// assertion-to-C-varargs data flow. Bounds-loop details are owned elsewhere.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARGS:%[0-9]+]] = call noalias dereferenceable(48) ptr @"{{.*}}AllocZ"(i64 48)
// CHECK: [[BOX:%[0-9]+]] = call noalias dereferenceable(8) ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 1, ptr [[BOX]]
// CHECK-NEXT: [[EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[BOX]], 1
// CHECK: [[SLICE_LEN:%[0-9]+]] = insertvalue %"{{.*}}Slice" {{.*}}, i64 3, 1
// CHECK-NEXT: [[SLICE:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[SLICE_LEN]], i64 3, 2
// CHECK-NEXT: call void @main.test(%"{{.*}}Slice" [[SLICE]])

// CHECK-LABEL: define void @main.test(
// CHECK: [[LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: [[INDEX:%[0-9]+]] = phi i64
// CHECK: [[SLOT:%[0-9]+]] = getelementptr inbounds %"{{.*}}eface", ptr {{%[0-9]+}}, i64 {{%[0-9]+}}
// CHECK-NEXT: [[VALUE:%[0-9]+]] = load %"{{.*}}eface", ptr [[SLOT]]
// CHECK-NEXT: [[TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[VALUE]], 0
// CHECK-NEXT: [[IS_INT:%[0-9]+]] = icmp eq ptr [[TYPE]], @_llgo_int
// CHECK: [[DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[VALUE]], 1
// CHECK-NEXT: [[INT:%[0-9]+]] = load i64, ptr [[DATA]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[INT]])
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[TYPE]], ptr @_llgo_int)
