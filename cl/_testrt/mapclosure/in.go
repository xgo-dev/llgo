// LITTEST
package main

type Type interface {
	String() string
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.demo(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: [[DEMO_DATA:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK: [[DEMO_METHOD:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[DEMO_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[DEMO_METHOD]], 0
// CHECK-NEXT: [[DEMO_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[DEMO_PAIR0]], ptr [[DEMO_DATA]], 1
// CHECK: [[DEMO_RESULT:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" %{{[0-9]+}}(ptr %{{[0-9]+}})
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.String" [[DEMO_RESULT]]
func demo(t Type) string {
	return t.String()
}

type typ struct {
	s string
}

var (
	op = map[string]func(Type) string{
		"demo": demo,
	}
	list = []func(Type) string{demo}
)

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: [[OP_MAP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_closure${{[-A-Za-z0-9_]+}}", i64 1)
// CHECK: [[OP_SLOT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_closure${{[-A-Za-z0-9_]+}}", ptr [[OP_MAP]], ptr %{{[0-9]+}})
// CHECK-NEXT: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr [[OP_SLOT]]
// CHECK-NEXT: store ptr [[OP_MAP]], ptr @main.op
// CHECK: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr [[LIST_ELEM:%[0-9]+]]
// CHECK: store %"{{.*}}/runtime/internal/runtime.Slice" [[LIST:%[0-9]+]], ptr @main.list
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[TYP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: [[LOADED_MAP:%[0-9]+]] = load ptr, ptr @main.op
// CHECK: [[MAP_SLOT:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_string]_llgo_closure${{[-A-Za-z0-9_]+}}", ptr [[LOADED_MAP]], ptr %{{[0-9]+}})
// CHECK-NEXT: [[MAP_FN:%[0-9]+]] = load { ptr, ptr }, ptr [[MAP_SLOT]]
// CHECK-NEXT: [[LOADED_LIST:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.Slice", ptr @main.list
// CHECK-NEXT: [[LIST_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[LOADED_LIST]], 0
// CHECK-NEXT: [[LIST_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" [[LOADED_LIST]], 1
// CHECK: [[LIST_SLOT:%[0-9]+]] = getelementptr inbounds { ptr, ptr }, ptr [[LIST_DATA]], i64 0
// CHECK-NEXT: [[LIST_FN:%[0-9]+]] = load { ptr, ptr }, ptr [[LIST_SLOT]]
// CHECK-NEXT: [[ITAB1:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.typ")
// CHECK: [[MAP_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[MAP_FN]], 1
// CHECK-NEXT: [[MAP_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[MAP_FN]], 0
// CHECK: [[MAP_RESULT:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[MAP_ENV]], %"{{.*}}/runtime/internal/runtime.iface" [[MAP_ARG:%[0-9]+]])
// CHECK: [[ITAB2:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.typ")
// CHECK: [[LIST_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[LIST_FN]], 1
// CHECK-NEXT: [[LIST_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[LIST_FN]], 0
// CHECK: [[LIST_RESULT:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" %__llgo_funcval_code1(ptr {{(nest|swiftself)}} [[LIST_ENV]], %"{{.*}}/runtime/internal/runtime.iface" [[LIST_ARG:%[0-9]+]])
// CHECK-NEXT: [[SAME_RESULT:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" [[MAP_RESULT]], %"{{.*}}/runtime/internal/runtime.String" [[LIST_RESULT]])
// CHECK-NEXT: [[RESULT_MISMATCH:%[0-9]+]] = xor i1 [[SAME_RESULT]], true
// CHECK-NEXT: br i1 [[RESULT_MISMATCH]], label %{{.*}}, label %{{.*}}
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"main.(*typ).String"(ptr %0){{.*}} {
// CHECK: [[STRING_FIELD:%[0-9]+]] = getelementptr inbounds %main.typ, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[STRING_VALUE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.String", ptr [[STRING_FIELD]]
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.String" [[STRING_VALUE]]
func main() {
	t := &typ{"hello"}
	fn1 := op["demo"]
	fn2 := list[0]
	if fn1(t) != fn2(t) {
		panic("error")
	}
}

func (t *typ) String() string {
	return t.s
}
