// LITTEST
package main

import "reflect"

// CHECK-LABEL: define void @main.main
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.Method:func(int) reflect.Method")
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect")
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.MethodByName:func(string) (reflect.Method, bool)")
// CHECK: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.type.reflect.Metadata")
// SYMBOL-NOT: main{{.*}}S{{.*}}Drop

type I interface {
	Metadata() string
}

type S struct{}

//go:noinline
func (S) Drop() string {
	panic("Drop should be unreachable")
}

func main() {
	// Keep S's descriptor and method metadata live without using its method.
	println(reflect.TypeOf(S{}).String())

	t := reflect.TypeOf((*I)(nil)).Elem()
	m := t.Method(0)
	println(m.Name)
	println(m.Type.String())

	byName, ok := t.MethodByName("Metadata")
	println(ok)
	println(byName.Name)
}
