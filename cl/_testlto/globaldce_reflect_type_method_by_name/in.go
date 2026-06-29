// LITTEST
package main

import "reflect"

// CHECK-DAG: @"_llgo_{{.*}}globaldce_reflect_type_method_by_name.S" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility !{{[0-9]+}}
// CHECK-DAG: @"*_llgo_{{.*}}globaldce_reflect_type_method_by_name.S" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility !{{[0-9]+}}
// CHECK-LABEL: define void @"github.com/goplus/llgo/cl/_testlto/globaldce_reflect_type_method_by_name.main"
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect.Keep")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK-DAG: !"go.method.Keep:func() string"
// CHECK-DAG: !"go.method.Drop:func() string"
// CHECK-DAG: !"go.method.type.reflect.Keep"
// CHECK-DAG: !"go.method.type.reflect.Drop"
// SYMBOL-NOT: globaldce_reflect_type_method_by_name{{.*}}S{{.*}}Drop
// SYMBOL-DAG: globaldce_reflect_type_method_by_name{{.*}}S{{.*}}Keep
// SYMBOL-NOT: globaldce_reflect_type_method_by_name{{.*}}S{{.*}}Drop

type S struct{}

//go:noinline
func (S) Keep() string {
	return "keep"
}

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

func main() {
	m, ok := reflect.TypeOf(S{}).MethodByName("Keep")
	if !ok {
		panic("missing Keep")
	}
	out := m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())
}
