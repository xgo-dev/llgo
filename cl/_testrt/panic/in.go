// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/panic.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 13 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
func main() {
	panic("panic message")
}
