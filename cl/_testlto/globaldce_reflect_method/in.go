// LITTEST
package main

import "reflect"

// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.value.reflect.Keep")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.Method:func(int) reflect.Method")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.MethodByName:func(string) (reflect.Method, bool)")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect.Keep")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect.Missing")
// CHECK-DAG: !"go.method.Keep:func() string"
// CHECK-DAG: !"go.method.{{.*}}/globaldce_reflect_method.hidden:func() string"
// CHECK-DAG: !"go.method.value.reflect"
// CHECK-DAG: !"go.method.value.reflect.Keep"
// CHECK-DAG: !"go.method.type.reflect"
// CHECK-DAG: !"go.method.type.reflect.Keep"
// SYMBOL-NOT: globaldce_reflect_method{{.*}}S{{.*}}hidden
// SYMBOL-DAG: globaldce_reflect_method{{.*}}S{{.*}}Keep
// SYMBOL-NOT: globaldce_reflect_method{{.*}}S{{.*}}hidden

type S struct{}

//go:noinline
func (S) Keep() string {
	return "keep"
}

//go:noinline
func (S) hidden() string {
	return "hidden"
}

func callTypeMethod(method func(reflect.Type, int) reflect.Method, typ reflect.Type) reflect.Method {
	return method(typ, 0)
}

func callTypeMethodByName(method func(reflect.Type, string) (reflect.Method, bool), typ reflect.Type, name string) (reflect.Method, bool) {
	return method(typ, name)
}

func main() {
	out := reflect.ValueOf(S{}).MethodByName("Keep").Call(nil)
	println(out[0].String())

	m := reflect.TypeOf(S{}).Method(0)
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	method := reflect.Type.Method
	m = method(reflect.TypeOf(S{}), 0)
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	var methodVar func(reflect.Type, int) reflect.Method = reflect.Type.Method
	m = callTypeMethod(methodVar, reflect.TypeOf(S{}))
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	m, ok := reflect.TypeOf(S{}).MethodByName("Keep")
	if !ok {
		panic("missing Keep")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	methodByName := reflect.Type.MethodByName
	m, ok = methodByName(reflect.TypeOf(S{}), "Keep")
	if !ok {
		panic("missing Keep")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	var methodByNameVar func(reflect.Type, string) (reflect.Method, bool) = reflect.Type.MethodByName
	m, ok = callTypeMethodByName(methodByNameVar, reflect.TypeOf(S{}), "Keep")
	if !ok {
		panic("missing Keep")
	}
	out = m.Func.Call([]reflect.Value{reflect.ValueOf(S{})})
	println(out[0].String())

	if _, ok := reflect.TypeOf(S{}).MethodByName("Missing"); ok {
		panic("unexpected Missing")
	}
}
