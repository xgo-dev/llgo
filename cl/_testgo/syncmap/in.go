// LITTEST
package main

import (
	"fmt"
	"sync"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAP:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 96)
// CHECK: [[KEY1_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 1, ptr [[KEY1_DATA]]
// CHECK-NEXT: [[KEY1:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[KEY1_DATA]], 1
// CHECK-NEXT: [[HELLO_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr [[HELLO_DATA]]
// CHECK-NEXT: [[HELLO:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[HELLO_DATA]], 1
// CHECK-NEXT: call void @"sync.(*Map).Store"(ptr [[MAP]], %"{{.*}}eface" [[KEY1]], %"{{.*}}eface" [[HELLO]])
// CHECK: [[STRING_KEY_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 1 }, ptr [[STRING_KEY_DATA]]
// CHECK-NEXT: [[STRING_KEY:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[STRING_KEY_DATA]], 1
// CHECK-NEXT: [[VALUE100_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 100, ptr [[VALUE100_DATA]]
// CHECK-NEXT: [[VALUE100:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[VALUE100_DATA]], 1
// CHECK-NEXT: call void @"sync.(*Map).Store"(ptr [[MAP]], %"{{.*}}eface" [[STRING_KEY]], %"{{.*}}eface" [[VALUE100]])
// CHECK: [[LOOKUP_KEY_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 1 }, ptr [[LOOKUP_KEY_DATA]]
// CHECK-NEXT: [[LOOKUP_KEY:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[LOOKUP_KEY_DATA]], 1
// CHECK-NEXT: [[LOADED:%[0-9]+]] = call { %"{{.*}}eface", i1 } @"sync.(*Map).Load"(ptr [[MAP]], %"{{.*}}eface" [[LOOKUP_KEY]])
// CHECK-NEXT: [[LOADED_VALUE:%[0-9]+]] = extractvalue { %"{{.*}}eface", i1 } [[LOADED]], 0
// CHECK-NEXT: [[LOADED_OK:%[0-9]+]] = extractvalue { %"{{.*}}eface", i1 } [[LOADED]], 1
// CHECK-NEXT: [[PRINT_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK-NEXT: [[PRINT_VALUE_ADDR:%[0-9]+]] = getelementptr inbounds %"{{.*}}eface", ptr [[PRINT_DATA]], i64 0
// CHECK-NEXT: store %"{{.*}}eface" [[LOADED_VALUE]], ptr [[PRINT_VALUE_ADDR]]
// CHECK-NEXT: [[PRINT_OK_ADDR:%[0-9]+]] = getelementptr inbounds %"{{.*}}eface", ptr [[PRINT_DATA]], i64 1
// CHECK-NEXT: [[OK_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 1)
// CHECK-NEXT: store i1 [[LOADED_OK]], ptr [[OK_DATA]]
// CHECK-NEXT: [[OK_BOX:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_bool, ptr undef }, ptr [[OK_DATA]], 1
// CHECK-NEXT: store %"{{.*}}eface" [[OK_BOX]], ptr [[PRINT_OK_ADDR]]
// CHECK-NEXT: [[PRINT_ARGS0:%[0-9]+]] = insertvalue %"{{.*}}Slice" undef, ptr [[PRINT_DATA]], 0
// CHECK-NEXT: [[PRINT_ARGS1:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[PRINT_ARGS0]], i64 2, 1
// CHECK-NEXT: [[PRINT_ARGS:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[PRINT_ARGS1]], i64 2, 2
// CHECK-NEXT: call { i64, %"{{.*}}iface" } @fmt.Println(%"{{.*}}Slice" [[PRINT_ARGS]])
// CHECK-NEXT: call void @"sync.(*Map).Range"(ptr [[MAP]], { ptr, ptr } { ptr @"main.main$1", ptr null })
// CHECK-LABEL: define i1 @"main.main$1"(%"{{.*}}eface" %0, %"{{.*}}eface" %1){{.*}} {
// CHECK: [[RANGE_ARGS:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT: [[RANGE_KEY_ADDR:%[0-9]+]] = getelementptr inbounds %"{{.*}}eface", ptr [[RANGE_ARGS]], i64 0
// CHECK-NEXT: store %"{{.*}}eface" %0, ptr [[RANGE_KEY_ADDR]]
// CHECK-NEXT: [[RANGE_VALUE_ADDR:%[0-9]+]] = getelementptr inbounds %"{{.*}}eface", ptr [[RANGE_ARGS]], i64 1
// CHECK-NEXT: store %"{{.*}}eface" %1, ptr [[RANGE_VALUE_ADDR]]
// CHECK: [[RANGE_SLICE0:%[0-9]+]] = insertvalue %"{{.*}}Slice" undef, ptr [[RANGE_ARGS]], 0
// CHECK-NEXT: [[RANGE_SLICE1:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[RANGE_SLICE0]], i64 2, 1
// CHECK-NEXT: [[RANGE_SLICE:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[RANGE_SLICE1]], i64 2, 2
// CHECK-NEXT: call { i64, %"{{.*}}iface" } @fmt.Printf(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 7 }, %"{{.*}}Slice" [[RANGE_SLICE]])
// CHECK-NEXT: ret i1 true

func main() {
	var m sync.Map
	m.Store(1, "hello")
	m.Store("1", 100)
	v, ok := m.Load("1")
	fmt.Println(v, ok)
	m.Range(func(k, v interface{}) bool {
		fmt.Printf("%#v %v\n", k, v)
		return true
	})
}
