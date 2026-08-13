// LITTEST
package main

import (
	"reflect"
	"strings"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[FUNC_TYPE:[0-9]+]] = call %"g{{.*}}/runtime/internal/runtime.iface" @reflect.FuncOf(
// CHECK: %[[FUNC_VALUE:[0-9]+]] = call %reflect.Value @reflect.MakeFunc(%"g{{.*}}/runtime/internal/runtime.iface" %[[FUNC_TYPE]], { ptr, ptr } { ptr @"main.main$1", ptr null })
// CHECK: %[[FUNC_IFACE:[0-9]+]] = call %"g{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %[[FUNC_VALUE]])
// CHECK: %[[DYNAMIC_TYPE:[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.eface" %[[FUNC_IFACE]], 0
// CHECK: %[[MATCH:[0-9]+]] = call i1 @"g{{.*}}/runtime/internal/runtime.MatchesClosure"({{.*}}ptr %[[DYNAMIC_TYPE]])
// CHECK: br i1 %[[MATCH]]
// CHECK: %[[FUNC_DATA:[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.eface" %[[FUNC_IFACE]], 1
// CHECK: %[[FUNC_PAIR:[0-9]+]] = load { ptr, ptr }, ptr %[[FUNC_DATA]]
// CHECK: %[[CALL_DATA:[0-9]+]] = extractvalue { ptr, ptr } %[[FUNC_PAIR]], 1
// CHECK: %[[CALL_PTR:[0-9]+]] = extractvalue { ptr, ptr } %[[FUNC_PAIR]], 0
// CHECK: %[[CALL_CODE:__llgo_funcval_code]] = call ptr asm "", "=r,0"(ptr %[[CALL_PTR]])
// CHECK: %[[RESULT:[0-9]+]] = call %"g{{.*}}/runtime/internal/runtime.String" %[[CALL_CODE]](ptr {{(nest|swiftself)}} %[[CALL_DATA]], %"g{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 3 }, i64 2)
// CHECK: %[[EQUAL:[0-9]+]] = call i1 @"g{{.*}}/runtime/internal/runtime.StringEqual"(%"g{{.*}}/runtime/internal/runtime.String" %[[RESULT]],{{.*}})
// CHECK: %[[NOT_EQUAL:[0-9]+]] = xor i1 %[[EQUAL]], true
// CHECK: br i1 %[[NOT_EQUAL]]
func main() {
	typ := reflect.FuncOf([]reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf("")}, false)
	fn := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		r := strings.Repeat(args[0].String(), int(args[1].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	r := fn.Interface().(func(string, int) string)("abc", 2)
	if r != "abcabc" {
		panic("error")
	}
}

// CHECK-LABEL: define %"g{{.*}}/runtime/internal/runtime.Slice" @"main.main$1"(%"g{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK: [[ARG0_DATA:%[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK: [[ARG0_LEN:%[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: call void @"g{{.*}}/runtime/internal/runtime.CheckIndexRange"({{.*}}i64 0,{{.*}}i64 [[ARG0_LEN]])
// CHECK: [[ARG0_PTR:%[0-9]+]] = getelementptr inbounds %reflect.Value, ptr [[ARG0_DATA]], i64 0
// CHECK: [[ARG0_SAFE:%[0-9]+]] = call ptr @"g{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr [[ARG0_PTR]])
// CHECK-NEXT: [[ARG0:%[0-9]+]] = load %reflect.Value, ptr [[ARG0_SAFE]]
// CHECK-NEXT: [[TEXT:%[0-9]+]] = call %"g{{.*}}/runtime/internal/runtime.String" @reflect.Value.String(%reflect.Value [[ARG0]])
// CHECK: [[ARG1_DATA:%[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK: [[ARG1_LEN:%[0-9]+]] = extractvalue %"g{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: call void @"g{{.*}}/runtime/internal/runtime.CheckIndexRange"({{.*}}i64 1,{{.*}}i64 [[ARG1_LEN]])
// CHECK: [[ARG1_PTR:%[0-9]+]] = getelementptr inbounds %reflect.Value, ptr [[ARG1_DATA]], i64 1
// CHECK: [[ARG1_SAFE:%[0-9]+]] = call ptr @"g{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr [[ARG1_PTR]])
// CHECK-NEXT: [[ARG1:%[0-9]+]] = load %reflect.Value, ptr [[ARG1_SAFE]]
// CHECK-NEXT: [[COUNT:%[0-9]+]] = call i64 @reflect.Value.Int(%reflect.Value [[ARG1]])
// CHECK-NEXT: [[REPEATED:%[0-9]+]] = call %"g{{.*}}/runtime/internal/runtime.String" @strings.Repeat(%"g{{.*}}/runtime/internal/runtime.String" [[TEXT]], i64 [[COUNT]])
// CHECK: store %"g{{.*}}/runtime/internal/runtime.String" [[REPEATED]], ptr [[RESULT_TEXT:%[0-9]+]]
// CHECK-NEXT: [[RESULT_BOX:%[0-9]+]] = insertvalue %"g{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr [[RESULT_TEXT]], 1
// CHECK-NEXT: [[RESULT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"g{{.*}}/runtime/internal/runtime.eface" [[RESULT_BOX]])
// CHECK: [[RESULT_SLICE0:%[0-9]+]] = insertvalue %"g{{.*}}/runtime/internal/runtime.Slice" undef, ptr %{{[0-9]+}}, 0
// CHECK-NEXT: [[RESULT_SLICE1:%[0-9]+]] = insertvalue %"g{{.*}}/runtime/internal/runtime.Slice" [[RESULT_SLICE0]], i64 1, 1
// CHECK-NEXT: [[RESULT_SLICE:%[0-9]+]] = insertvalue %"g{{.*}}/runtime/internal/runtime.Slice" [[RESULT_SLICE1]], i64 1, 2
// CHECK-NEXT: ret %"g{{.*}}/runtime/internal/runtime.Slice" [[RESULT_SLICE]]
