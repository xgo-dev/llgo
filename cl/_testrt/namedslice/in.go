// LITTEST
package main

type MyBytes []byte

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/namedslice.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/namedslice.MyBytes", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: icmp eq ptr %{{[0-9]+}}, @"_llgo_github.com/goplus/llgo/cl/_testrt/namedslice.MyBytes"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK: }
func main() {
	var i any = MyBytes{}
	_, ok := i.(MyBytes)
	if !ok {
		panic("bad slice")
	}
}
