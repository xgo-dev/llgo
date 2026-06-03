// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/gotypes.foo"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func foo(bar) {
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/gotypes.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/gotypes.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/gotypes.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type base interface {
	f(m map[string]func())
}

type bar interface {
	base
	g(c chan func())
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/gotypes.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/gotypes.foo"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	foo(nil)
}
