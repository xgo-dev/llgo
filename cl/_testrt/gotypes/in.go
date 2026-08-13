// LITTEST
package main

// CHECK-LABEL: define void @main.foo(%"{{.*}}iface" %0){{.*}} {
// CHECK: ret void
func foo(bar) {
}

type base interface {
	f(m map[string]func())
}

type bar interface {
	base
	g(c chan func())
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.foo(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
func main() {
	foo(nil)
}
