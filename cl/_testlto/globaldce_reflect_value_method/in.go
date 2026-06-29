// LITTEST
package main

import "reflect"

// CHECK-DAG: @"_llgo_{{.*}}globaldce_reflect_value_method.S" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility !{{[0-9]+}}
// CHECK-DAG: @"*_llgo_{{.*}}globaldce_reflect_value_method.S" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility !{{[0-9]+}}
// CHECK-LABEL: define void @"github.com/goplus/llgo/cl/_testlto/globaldce_reflect_value_method.main"
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect")
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect.Keep")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK-NOT: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect")
// CHECK-DAG: !"go.method.Keep:func() string"
// CHECK-DAG: !"go.method.Drop:func() string"
// CHECK-DAG: !"go.method.value.reflect.Keep"
// CHECK-DAG: !"go.method.value.reflect.Drop"
// SYMBOL-NOT: globaldce_reflect_value_method{{.*}}S{{.*}}Drop
// SYMBOL-DAG: globaldce_reflect_value_method{{.*}}S{{.*}}Keep
// SYMBOL-NOT: globaldce_reflect_value_method{{.*}}S{{.*}}Drop

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
	out := reflect.ValueOf(S{}).MethodByName("Keep").Call(nil)
	println(out[0].String())
}
