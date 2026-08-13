// LITTEST
package main

type T1 int

type T2 struct {
	v int
}

type T3[T any] struct {
	v T
}

type cacheKey struct {
	t1 T1
	t2 T2
	t3 T3[any]
	t4 *int
	t5 uintptr
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_main.cacheKey]_llgo_string", i64 0)
// CHECK: [[ASSIGN_KEY_VALUE:%[0-9]+]] = load %main.cacheKey, ptr [[ASSIGN_KEY_TMP:%[0-9]+]]
// CHECK-NEXT: [[ASSIGN_KEY:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT: store %main.cacheKey [[ASSIGN_KEY_VALUE]], ptr [[ASSIGN_KEY]]
// CHECK-NEXT: [[VALUE_SLOT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_main.cacheKey]_llgo_string", ptr [[MAP]], ptr [[ASSIGN_KEY]])
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr [[VALUE_SLOT]]
// CHECK: [[ACCESS_KEY_VALUE:%[0-9]+]] = load %main.cacheKey, ptr [[ACCESS_KEY_TMP:%[0-9]+]]
// CHECK-NEXT: [[ACCESS_KEY:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT: store %main.cacheKey [[ACCESS_KEY_VALUE]], ptr [[ACCESS_KEY]]
// CHECK-NEXT: [[ACCESS:%[0-9]+]] = call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"(ptr @"map[_llgo_main.cacheKey]_llgo_string", ptr [[MAP]], ptr [[ACCESS_KEY]])
// CHECK-NEXT: [[ACCESS_VALUE_PTR:%[0-9]+]] = extractvalue { ptr, i1 } [[ACCESS]], 0
// CHECK-NEXT: [[ACCESS_VALUE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[ACCESS_VALUE_PTR]]
// CHECK-NEXT: [[ACCESS_OK:%[0-9]+]] = extractvalue { ptr, i1 } [[ACCESS]], 1
// CHECK-NEXT: [[ACCESS_PAIR0:%[0-9]+]] = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" [[ACCESS_VALUE]], 0
// CHECK-NEXT: [[ACCESS_PAIR:%[0-9]+]] = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } [[ACCESS_PAIR0]], i1 [[ACCESS_OK]], 1
// CHECK-NEXT: [[PRINT_VALUE:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } [[ACCESS_PAIR]], 0
// CHECK-NEXT: [[PRINT_OK:%[0-9]+]] = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } [[ACCESS_PAIR]], 1
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" [[PRINT_VALUE]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 [[PRINT_OK]])
func main() {
	m := map[cacheKey]string{}
	m[cacheKey{0, T2{0}, T3[any]{0}, nil, 0}] = "world"
	v, ok := m[cacheKey{0, T2{0}, T3[any]{0}, nil, 0}]
	println(v, ok)
}
